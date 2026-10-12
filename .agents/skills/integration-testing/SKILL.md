---
name: integration-testing
description: Agentically run integration suite. Load when asked to run integration tests.
---

Treat `docs/TESTING.md` as shared guidance, not a turnkey script: adapt its examples to the available environment. Assume the Prometheus server is already running; if it is not, ask the operator to enable it, since the agent cannot start it. Treat `test/features/` as behavioral expectations; report contradictory steps or unavailable prerequisites rather than silently changing them.

## Rules

- Do not read files (Go or otherwise) this skill or `TESTING.md` do not mention.
- Do not use Sourcegraph or other codebase navigation tools.
- Do not change any program files unless explicitly asked.
- You are in a container, Podman is not available. Assume the Prometheus server is running.

## Recommendations

- Store command output in a file, so it can be referenced later.

## Files

The following Gherkin files are in scope:

!find test/features -type f -print

The following scripts are used during testing:

!find test/scripts -type f -print
build/rhc-heartbeat
