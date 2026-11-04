.ONESHELL:
.SHELLFLAGS := -e -c

VERSION := $(shell rpmspec rhc-heartbeat.spec --query --srpm --queryformat '%{version}')
LDFLAGS := -ldflags "-X github.com/redhatinsights/rhc/pkg/version.Version=$(VERSION)"
RPMBUILD_CHECK_OPTION := $(if $(filter 1,$(NOCHECK)),--without check)
ARTIFACTS_DIR := $(CURDIR)/.tmt-artifacts

.PHONY: check
check:
	golangci-lint fmt --diff
	go test ./...
	go vet ./...
	golangci-lint run
	python3 -m compileall -q tests
	tmt lint

.PHONY: fmt
fmt:
	golangci-lint fmt
	golangci-lint run --fix

.PHONY: build
build:
	mkdir -p build
	CGO_ENABLED=0 go build -o build/heartbeat ./cmd/rhc-heartbeat
	CGO_ENABLED=0 go build -o build/heartbeat-rhsm ./cmd/heartbeat-rhsm

# Copy build artifacts into a non-git-ignored directory, so tmt copies them over
# to the test environment.
.PHONY: _copy_tmt_artifacts
_copy_tmt_artifacts:
	@if [ ! -d build ]; then echo "no such directory: build/" >&2; exit 1; fi
	rm -rf "$(ARTIFACTS_DIR)"
	cp -a build "$(ARTIFACTS_DIR)"

.PHONY: server
server:
	podman run -it --rm --replace \
	    --name prometheus \
	    --network podman \
	    -p "0.0.0.0:9090:9090/tcp" \
	    -v "./tests/scripts/prometheus.yml:/etc/prometheus/prometheus.yml:ro,Z" \
	    "docker.io/prom/prometheus:latest" \
	    --config.file=/etc/prometheus/prometheus.yml \
	    --web.enable-remote-write-receiver

.PHONY: test-local
test-local: _copy_tmt_artifacts
	trap 'rm -rf "$(ARTIFACTS_DIR)"' EXIT; \
	tmt --feeling-safe run -vvv -a --scratch \
	    provision --how local plan --name '^/plans/local/'

.PHONY: test-container
test-container: _copy_tmt_artifacts
	trap 'rm -rf "$(ARTIFACTS_DIR)"' EXIT; \
	tmt run -vvv -a --scratch plan --name '^/plans/container/'

.PHONY: test-vm
test-vm: _copy_tmt_artifacts
	trap 'rm -rf "$(ARTIFACTS_DIR)"' EXIT; \
	tmt run -vvv -a --scratch plan --name '^/plans/vm/'

.PHONY: archive
archive:
	git archive --prefix rhc-heartbeat-$(VERSION)/ --format tar.gz HEAD > rhc-heartbeat-$(VERSION).tar.gz

# Generate -vendor tarball to be used as .spec's Source1, containing dependencies.
# On Fedora, this could be done by go-vendor-tools package, but CentOS Stream and RHEL
# do not have it available yet.
.PHONY: archive-deps
archive-deps:
	go mod vendor
	tar --create --bzip2 \
		--file "$(CURDIR)/rhc-heartbeat-$(VERSION)-vendor.tar.bz2" \
		--sort name \
		--mtime="@$${SOURCE_DATE_EPOCH:-0}" \
		--owner 0 --group 0 --numeric-owner \
		go.mod go.sum vendor/

.PHONY: srpm
srpm: archive archive-deps
	rpmbuild \
	--define "_sourcedir $$(pwd)" \
	-bs rhc-heartbeat.spec

.PHONY: rpm
rpm: srpm
	mkdir -p "$$(pwd)/build"
	rpmbuild $(RPMBUILD_CHECK_OPTION) \
	--define "_sourcedir $$(pwd)" \
	--define "_rpmdir $$(pwd)/build" \
	-bb rhc-heartbeat.spec
	ls -R1p "$$(pwd)/build/"

.PHONY: clean
clean:
	rm -rf "$(ARTIFACTS_DIR)"
	rm -rf build/
	rm -rf vendor/
	rm -rf rhc-heartbeat-*.tar.*
