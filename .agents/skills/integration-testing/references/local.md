# Local execution environment

The suite runs in the agent's own container. The agent cannot manage the Prometheus server, the system clock, or reboots: Podman is not available in the container, and the host running it is not disposable.

## Prometheus

The Prometheus server is managed by the operator and expected at `http://10.88.0.1:9090` (see `docs/TESTING.md`). Verify that it is reachable before running scenarios, for example:

    curl -fsS http://10.88.0.1:9090/-/healthy

If the instance cannot be accessed, report that to the operator and ask them to enable it (they may use `make server`); do not attempt to start, stop, or reconfigure it yourself.

## Capability restrictions

Scenarios that require managing the system clock or rebooting the machine (see the respective scenarios in `test/features/tier-3.feature`) cannot be executed in this environment. Report them as not executable locally instead of simulating, approximating, or silently skipping them.

If the system is not registered with subscription-manager, simulate the identity certificate pair with `test/scripts/identity.sh` as shown in `docs/TESTING.md`.
