.DEFAULT_GOAL := check

all: check

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
	go build -o build/otlp-upload ./examples/otlp-upload
	go build -o build/rhc-heartbeat ./cmd/rhc-heartbeat

.PHONY: server
server:
	podman run -it \
	--name otel --replace \
	--network podman \
	-v ./test/otelcol.yaml:/etc/otelcol/config.yaml:ro,Z \
	ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:latest
