#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: identity.sh [-f] [-h] [directory]

Generate a short-lived client certificate and private key in directory.

Options:
  -f  Force generation, replacing any existing certificate or key
  -h  Show this help message
EOF
}

parse_args() {
  FORCE=false
  DIR=""
  local dir=""

  while (($#)); do
    case "$1" in
      -f)
        FORCE=true
        shift
        ;;
      -h)
        usage
        exit 0
        ;;
      --)
        shift
        if (($# > 1)); then
          echo "error: only one directory may be specified" >&2
          exit 2
        fi
        dir=${1:-"."}
        break
        ;;
      -*)
        echo "error: unknown option: $1" >&2
        usage >&2
        exit 2
        ;;
      *)
        if [[ -n "${dir}" ]]; then
          echo "error: only one directory may be specified" >&2
          exit 2
        fi
        dir=$1
        shift
        ;;
    esac
  done

  DIR=${dir:-"."}
}

main() {
  local CERT KEY UUID
  parse_args "$@"

  if ! command -v openssl >/dev/null 2>&1; then
    echo "error: openssl is required." >&2
    exit 1
  fi

  mkdir -p -- "${DIR}"

  CERT="${DIR}/cert.pem"
  KEY="${DIR}/key.pem"
  if [[ -e "${CERT}" || -e "${KEY}" ]]; then
    if [[ "${FORCE}" != true ]]; then
      echo "error: certificate pair already exists under ${DIR}" >&2
      exit 1
    fi
    rm -f -- "${CERT}" "${KEY}"
  fi

  UUID=$(cat /proc/sys/kernel/random/uuid)

  umask 077
  openssl req -x509 -newkey rsa:4096 -sha256 -nodes -days 1 \
    -keyout "${KEY}" -out "${CERT}" \
    -subj "/O=tests/CN=${UUID}" \
    -addext "extendedKeyUsage=clientAuth" 2>/dev/null

  echo "${UUID}"
}

main "$@"
