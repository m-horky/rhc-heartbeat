---
name: integration-testing
description: Agentically run the tmt integration suite. Load when asked to run integration tests.
---

Treat `docs/TESTING.md` as shared guidance, not a turnkey script: adapt its examples to the available environment. Treat the test metadata under `tests/` as the behavioral expectations; report contradictory steps or unavailable prerequisites rather than silently changing them.

## Execution environments

The suite can run in three environments; read the matching resource before executing anything:

- Local machine: `make test-local` runs `/plans/local` on the runner with the locally built binary. The agent cannot manage the system clock or reboot the machine. See `references/local.md`.
- Container: the default. `make test` runs `/plans/container` in a CentOS Stream container from the RPMs. See `references/local.md`.
- Virtual machine: `make test-vm` provisions a CentOS Stream VM and installs the RPMs. This is the only environment that can run tier 3. If libvirt is not available, use a disposable remote VM over SSH and see `references/remote.md`.

Default to the local environment. Use the VM environment only when the task requires systemd, SELinux or reboot behavior, or when the operator supplies an SSH host.

Whichever environment is used, verify access to the Prometheus instance before running scenarios, and report to the operator when it cannot be accessed.

## Rules

- Do not read files (Go or otherwise) this skill, its resources, or `TESTING.md` do not mention.
- Do not use Sourcegraph or other codebase navigation tools.
- Do not change any program files unless explicitly asked.
- Prefer `tmt run`, `tmt try` and the bundled helper scripts over ad-hoc harnesses.

## Recommendations

- Store command output in a file, so it can be referenced later.
- Use `--scratch` for repeatable runs and `tmt clean` for leftovers.
