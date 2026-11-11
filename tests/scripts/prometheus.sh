#!/bin/bash
# Manage the Prometheus Remote Write receiver used by the tmt plans.
#
# check: validate the receiver at PROMETHEUS_URL.
# start: run the receiver as a systemd unit.
# stop:  stop the receiver systemd unit.

set -euo pipefail

: "${TMT_PLAN_DATA:?TMT_PLAN_DATA is not set}"

IMAGE="${PROMETHEUS_IMAGE:-docker.io/prom/prometheus:latest}"
PORT="${PROMETHEUS_PORT:-19090}"
CONTAINER="heartbeat-test-prometheus"
UNIT="heartbeat-test-prometheus.service"
UNIT_FILE="/etc/systemd/system/$UNIT"

fail() {
    printf 'Prometheus: %s\n' "$*" >&2
    exit 1
}

write_environment() {
    local url=$1
    : "${TMT_PLAN_ENVIRONMENT_FILE:?TMT_PLAN_ENVIRONMENT_FILE is not set}"

    printf 'PROMETHEUS_URL=%s\n' "$url" >> "$TMT_PLAN_ENVIRONMENT_FILE"
}

wait_healthy() {
    local url=$1 deadline=$((SECONDS + 90))

    while ((SECONDS < deadline)); do
        if curl --noproxy '*' --fail --silent --show-error \
            "$url/-/healthy" >/dev/null 2>&1; then
            return 0
        fi

        sleep 1
    done

    printf 'Prometheus did not become healthy at %s\n' "$url" >&2
    exit 1
}

check() {
    local url="${PROMETHEUS_URL:-}"

    if [ -z "$url" ]; then
        fail "PROMETHEUS_URL is not set."
    fi

    url="${url%/}"

    if ! curl --noproxy '*' --fail --silent --show-error --max-time 10 \
        "$url/-/healthy" >/dev/null; then
        fail "Prometheus Remote Write is not running at $url."
    fi

    write_environment "$url"
}

start() {
    local config podman

    : "${TMT_TREE:?TMT_TREE is not set}"
    config="$TMT_TREE/tests/scripts/prometheus.yml"

    podman=$(command -v podman)
    podman pull "$IMAGE"

    cat >"$UNIT_FILE" <<EOF
[Unit]
Description=Prometheus receiver for rhc-heartbeat tests
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
ExecStart=$podman run --replace --name $CONTAINER -p $PORT:9090 -v $config:/etc/prometheus/prometheus.yml:ro,Z $IMAGE --config.file=/etc/prometheus/prometheus.yml --web.enable-remote-write-receiver
ExecStop=$podman stop $CONTAINER
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$UNIT"
    systemctl restart "$UNIT"

    wait_healthy "http://127.0.0.1:$PORT"
    write_environment "http://127.0.0.1:$PORT"
}

stop() {
    systemctl disable --now "$UNIT" >/dev/null 2>&1 || true
}

case "${1:-}" in
    check)
        check
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    *)
        printf 'Usage: %s check|start|stop\n' "$0" >&2
        exit 2
        ;;
esac
