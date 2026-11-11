# Local and container execution

There are two non-VM plans:

- `/plans/local`: runs on the runner, uses `build/heartbeat` staged in
  `.tmt-artifacts/` and a temporary sandbox root.
- `/plans/container`: runs in a CentOS Stream container, uses the RPMs from
  `build/` and real system paths.

Both cover tier 1 and tier 2.

## Prometheus

The local and container plans use an externally managed receiver. They do not
start Prometheus; the `prepare` phase only checks that `PROMETHEUS_URL` is set
and healthy. Start one with `make server` and export an address reachable from
the test environment before running:

    export PROMETHEUS_URL=http://127.0.0.1:9090
    curl --noproxy '*' -fsS "$PROMETHEUS_URL/-/healthy"

For container tests, use the host's non-loopback address or another address
reachable from inside the container, not `127.0.0.1`. If the receiver is not
reachable, report the error to the operator and ask them to start it; do not
guess another endpoint.

The VM plan starts its own Podman container, so it does not need an external
receiver or `PROMETHEUS_URL`.

## Commands

    make test-local      # tier 1 and 2 on the local machine
    make test-container  # tier 1 and 2 in a CentOS Stream container
    make test-dry        # validate metadata and plan selection

The local target requires `--feeling-safe`, which the Makefile supplies.
It stages `build/` into `.tmt-artifacts/` so tmt copies the artifacts into
the plan work tree; `/plans/local` reads the binary from there. Do not add
`.tmt-artifacts/` to `.gitignore`, otherwise tmt skips it again.

## Capability restrictions

The local and container plans cannot run tier 3: there is no systemd
session, no SELinux confinement and no reboot. Do not simulate these
scenarios outside `/plans/vm`; run it on a disposable VM instead.

The local plan must not write outside its temporary root. The test library
sets the `RHC_HEARTBEAT_*` path overrides for that reason. Do not run the
binary with system paths on the runner.
