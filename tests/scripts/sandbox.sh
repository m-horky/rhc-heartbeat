#!/usr/bin/env bash
# Create the per-plan sandbox root used by the local plan.

set -euo pipefail

: "${TMT_PLAN_DATA:?TMT_PLAN_DATA is not set}"
: "${TMT_PLAN_ENVIRONMENT_FILE:?TMT_PLAN_ENVIRONMENT_FILE is not set}"

root=$(mktemp --directory --tmpdir="$TMT_PLAN_DATA" sandbox.XXXXXX)

printf 'HEARTBEAT_ROOT=%s\n' "$root" >>"$TMT_PLAN_ENVIRONMENT_FILE"
