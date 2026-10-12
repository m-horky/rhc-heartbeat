#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: query.sh [-u url] [-q matcher]... [-l limit] [-h]

Query Prometheus for samples matching the supplied label matchers.

Options:
  -u  Prometheus API URL (default: http://10.88.0.1:9090/api/v1/query)
  -q  Label matcher to constrain the query; may be specified more than once
  -l  Maximum number of results (default: 1)
  -h  Show this help message

Example:
  query.sh -q 'system_uuid="7ba0121b-feac-45ef-b9fe-abb9f6602db2"' -q 'kind="ping"' -l 10
EOF
}

parse_args() {
  URL="http://10.88.0.1:9090/api/v1/query"
  LIMIT=1
  MATCHERS=()

  while (($#)); do
    case "$1" in
      -u)
        if (($# < 2)); then
          echo "error: -u requires a URL" >&2
          exit 2
        fi
        URL=$2
        shift 2
        ;;
      -q)
        if (($# < 2)); then
          echo "error: -q requires a matcher" >&2
          exit 2
        fi
        MATCHERS+=("$2")
        shift 2
        ;;
      -l)
        if (($# < 2)); then
          echo "error: -l requires a result limit" >&2
          exit 2
        fi
        LIMIT=$2
        shift 2
        ;;
      -h)
        usage
        exit 0
        ;;
      --)
        shift
        if (($#)); then
          echo "error: unexpected argument: $1" >&2
          exit 2
        fi
        ;;
      -*)
        echo "error: unknown option: $1" >&2
        usage >&2
        exit 2
        ;;
      *)
        echo "error: unexpected argument: $1" >&2
        usage >&2
        exit 2
        ;;
    esac
  done

  if [[ ! $LIMIT =~ ^[1-9][0-9]*$ ]]; then
    echo "error: limit must be a positive integer" >&2
    exit 2
  fi
}

main() {
  local query selector matcher separator=""
  parse_args "$@"

  if ! command -v curl >/dev/null 2>&1 || ! command -v jq >/dev/null 2>&1; then
    echo "error: curl and jq are required" >&2
    exit 1
  fi

  selector="rhc_heartbeat_time_monotonic_seconds"
  if ((${#MATCHERS[@]})); then
    selector+="{"
    for matcher in "${MATCHERS[@]}"; do
      if [[ -n $separator ]]; then
        selector+=','
      fi
      selector+=$matcher
      separator=,
    done
    selector+="}"
  fi
  query="${selector} and topk(${LIMIT}, timestamp(${selector}))"

  local response
  if ! response=$(curl --fail --silent --show-error --get \
    --data-urlencode "query=${query}" \
    --data-urlencode "limit=${LIMIT}" \
    "${URL}"); then
    echo "error: Prometheus query request failed" >&2
    exit 1
  fi

  if ! jq -e '.status == "success"' >/dev/null <<<"${response}"; then
    jq -r '.error // "Prometheus returned an unsuccessful query response"' <<<"${response}" >&2
    exit 1
  fi

  jq -c '.data.result[]' <<<"${response}"
}

main "$@"
