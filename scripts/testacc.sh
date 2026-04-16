#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CACHE_ROOT="${BADGEROPS_TOOL_CACHE_ROOT:-${ROOT_DIR}/.cache/tooling}"
cd "${ROOT_DIR}"

mkdir -p \
  "${CACHE_ROOT}/go-tmp" \
  "${CACHE_ROOT}/go-build" \
  "${CACHE_ROOT}/go-mod"

export GOTMPDIR="${BADGEROPS_GOTMPDIR:-${CACHE_ROOT}/go-tmp}"
export TMPDIR="${BADGEROPS_TMPDIR:-${CACHE_ROOT}/go-tmp}"
export GOCACHE="${BADGEROPS_GOCACHE:-${CACHE_ROOT}/go-build}"
export GOMODCACHE="${BADGEROPS_GOMODCACHE:-${CACHE_ROOT}/go-mod}"

if [[ -f .envrc ]] && command -v direnv >/dev/null 2>&1 && [[ -z "${BADGEROPS_TESTACC_DIRENV:-}" ]]; then
  exec env BADGEROPS_TESTACC_DIRENV=1 direnv exec . "$0" "$@"
fi

if [[ -z "${TF_ACC_TERRAFORM_PATH:-}" ]]; then
  if [[ -x /usr/bin/terraform ]]; then
    export TF_ACC_TERRAFORM_PATH=/usr/bin/terraform
  elif command -v terraform >/dev/null 2>&1 && terraform version 2>/dev/null | grep -q '^Terraform v'; then
    export TF_ACC_TERRAFORM_PATH
    TF_ACC_TERRAFORM_PATH="$(command -v terraform)"
  fi
fi

go test -v ./internal/provider -run '^TestAccLive' "$@"
