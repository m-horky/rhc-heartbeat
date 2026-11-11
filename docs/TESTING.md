# Integration testing

The integration suite is implemented with `tmt`. The Python test scripts define the scenarios.

To execute the tests, make sure you have the dependencies installed: `make`, `python3`, `tmt`, `openssl`, `curl`. For container plan: `podman`. For VM plan: `libvirt`, `virt-install`, `qemu-kvm`.

## Layout

- `tests/tierN/`: test implementations,
- `tests/testlib/`: shared Python helpers,
- `tests/scripts/`: helper and plan support scripts, configuration,
- `plans/*.fmf`: test environment plans.
- `.tmt-artifacts`: temporary copy of files from `build/` (TMT ignores gitignored files, and builds are required for local testing).

## Build artifacts

At provision time, tmt only copies files that aren't gitignored, so `build/` with locally built artifacts would never reach the test environment. The Make target `_copy_tmt_artifacts` copies the builds into a temporary `.tmt-artifacts` directory which is deleted after the testing finished.

When testing locally, always run `make build` (local) or `make rpm` (container, vm) before triggering the tests.

## Environments

Local tests run tier 1 and tier 2 tests with mocked directory paths, so the host system is not altered.

```shell
$ export PROMETHEUS_URL=http://10.88.0.1:9090
$ make test-local
```

Container tests require `.rpm`s and run tier 1 and tier 2 tests.

```shell
$ export PROMETHEUS_URL=http://10.88.0.1:9090
$ make test-container
```

VM tests require `.rpm`s and run all test tiers.

```shell
$ make test-vm
```

## Architecture

### Test run internals

When running locally, we don't want to execute against the real filesystem. The `sandbox-root` prepare step generates a new temporary directory `HEARTBEAT_ROOT` which is exported into the test environment via `TMT_PLAN_ENVIRONMENT_FILE`.

Each test receives a `TestContext` object that contains appropriate paths -- real filesystem if `HEARTBEAT_ROOT` is not set, or filesystem with this custom root if set.

## Selecting plans and tests

```shell
# Only tier 1 in a container
$ tmt run -a plan --name '^/plans/container/tier1$'

# Only tier 1 on the local machine
$ make build _copy_tmt_artifacts
$ tmt --feeling-safe run -a \
    provision --how local plan --name '^/plans/local/tier1$'

# Interactive guest with the tests from the current directory
$ tmt try centos-stream-10@virtual
```

## Debugging

- `python3 -m tests.scripts.query` can be used to query the Prometheus database.
- `python3 -m tests.scripts.identity [DIR=.]` can be used to generate identity keypair into DIR.
- `make --dry-run test-container` shows what Make would run.
- `tmt run --last report -vvv` shows details of the last test.
- `tmt run login` opens a shell on the provisioned guest.
- `tmt try centos-stream-10@virtual -t /tests/tier3` provisions a guest and keeps it available for experiments.
- `tmt clean runs` removes old run workdirs, `tmt clean guests` stops leftover guests.

Test logs and submitted files are stored under `$TMT_TEST_DATA` and are pulled back into the tmt run workdir.
