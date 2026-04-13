#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TFPLUGINDOCS_VERSION="${TFPLUGINDOCS_VERSION:-v0.24.0}"

cd "${ROOT_DIR}"

if [[ -x /usr/bin/terraform ]]; then
  export PATH="/usr/bin:${PATH}"
elif command -v terraform >/dev/null 2>&1 && terraform version 2>/dev/null | grep -q '^Terraform v'; then
  :
else
  echo "Terraform CLI not found in PATH; tfplugindocs may download one automatically." >&2
fi

terraform fmt -recursive examples

go run "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@${TFPLUGINDOCS_VERSION}" generate \
  --provider-dir "${ROOT_DIR}" \
  --provider-name unifi
