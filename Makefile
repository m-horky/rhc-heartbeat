.ONESHELL:
.SHELLFLAGS := -e -c

VERSION := $(shell rpmspec rhc-heartbeat.spec --query --srpm --queryformat '%{version}')
LDFLAGS := -ldflags "-X github.com/redhatinsights/rhc/pkg/version.Version=$(VERSION)"
RPMBUILD_CHECK_OPTION := $(if $(filter 1,$(NOCHECK)),--without check)

.PHONY: check
check:
	golangci-lint fmt --diff
	go test ./...
	go vet ./...
	golangci-lint run

.PHONY: fmt
fmt:
	golangci-lint fmt
	golangci-lint run --fix

.PHONY: build
build:
	mkdir -p build
	go build -o build/config ./examples/config
	go build -o build/heartbeat ./examples/heartbeat
	go build -o build/remote-write-upload ./examples/remote-write-upload
	go build -o build/rhc-heartbeat ./cmd/rhc-heartbeat

.PHONY: server
server:
	podman run -it --rm --replace \
	    --name prometheus \
	    --network podman \
	    -p "0.0.0.0:9090:9090/tcp" \
	    -v "./test/prometheus.yml:/etc/prometheus/prometheus.yml:ro,Z" \
	    "docker.io/prom/prometheus:latest" \
	    --config.file=/etc/prometheus/prometheus.yml \
	    --web.enable-remote-write-receiver

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
	rm -rf build/
	rm -rf vendor/
	rm -rf rhc-heartbeat-*.tar.*
