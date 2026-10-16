---
name: integration-testing
description: Agentically run integration suite. Load when asked to run integration tests.
---

Treat `docs/TESTING.md` as shared guidance, not a turnkey script: adapt its examples to the available environment. Treat `test/features/` as behavioral expectations; report contradictory steps or unavailable prerequisites rather than silently changing them.

## Execution environments

The suite can run in two environments; read the matching resource before executing anything:

- Local (default): the suite runs in the agent's own container. The agent cannot manage the Prometheus server, the system clock, or reboots. See `references/local.md`.
- Remote over SSH: the suite runs on a disposable remote VM that the agent fully controls, including the Prometheus server, the system clock, and reboots. Requires an SSH host (IP address or hostname) supplied by the operator. See `references/remote.md`.

Default to the local environment. Use the remote environment only when the operator supplies an SSH host; if they request remote testing without one, ask for the host first.

Whichever environment is used, verify access to the Prometheus instance before running scenarios, and report to the operator when it cannot be accessed.

## Rules

- Do not read files (Go or otherwise) this skill, its resources, or `TESTING.md` do not mention.
- Do not use Sourcegraph or other codebase navigation tools.
- Do not change any program files unless explicitly asked.
- If the skill bundles helper scripts, prefer them over implementing ad-hoc ones.

## Recommendations

- Store command output in a file, so it can be referenced later.

## Files

The following Gherkin files are in scope:

!find test/features -type f -print

The following scripts are used during testing:

!find test/scripts -type f -print
build/rhc-heartbeat

## Helper scripts

`scripts/` directory of this skill has Python helpers for scenarios Prometheus alone cannot serve. Run all with proxy environment scrubbed (see above). Docstrings document usage, ports, certificate generation commands.

- `http500.py`: replies HTTP 500 to every POST; rejecting Remote Write endpoint.
- `authproxy.py`: HTTP proxy enforcing Basic proxy auth (`good:pass`); forwards authorized requests, returns 407 otherwise. Remember non-loopback endpoint requirement.
- `tlsforward.py`: HTTPS endpoint forwarding to Prometheus for CA verification scenarios; cert and key selected with `TLS_CERT`/`TLS_KEY`. Server cert must carry `subjectAltName` (Go rejects CN-only), trusted CA configured on client via `rhsm.conf [rhsm] repo_ca_cert`.
