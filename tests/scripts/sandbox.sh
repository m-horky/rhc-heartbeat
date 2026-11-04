#!/usr/bin/env bash
# Manage the per-plan sandbox root used by the local plan.
#
# create: allocate the sandbox outside of TMT_PLAN_DATA and export it via
#         TMT_PLAN_ENVIRONMENT_FILE.
# clean:  remove the exported sandbox root at the end of the plan.

set -euo pipefail

usage() {
    printf 'Usage: %s create|clean\n' "$0" >&2
    exit 2
}

create() {
    : "${TMT_PLAN_ENVIRONMENT_FILE:?TMT_PLAN_ENVIRONMENT_FILE is not set}"

    # Keep the sandbox out of TMT_PLAN_DATA: tmt pulls that directory back
    # after every step, which would copy the generated certificates and keys
    # around and keep them as plan artifacts.
    root=$(mktemp --directory --tmpdir=/var/tmp rhc-heartbeat-sandbox.XXXXXX)

    printf 'SANDBOX_ROOT=%s\n' "$root" >>"$TMT_PLAN_ENVIRONMENT_FILE"
}

clean() {
    case "${SANDBOX_ROOT:-}" in
        /var/tmp/rhc-heartbeat-sandbox.*)
            rm -rf -- "$SANDBOX_ROOT"
            ;;
        '')
            # The sandbox was never created, there is nothing to remove.
            ;;
        *)
            printf 'Refusing to remove unexpected sandbox root: %s\n' "$SANDBOX_ROOT" >&2
            exit 1
            ;;
    esac
}

case "${1:-}" in
    create)
        create
        ;;
    clean)
        clean
        ;;
    *)
        usage
        ;;
esac
