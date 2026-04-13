#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

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
