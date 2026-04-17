#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TFPLUGINDOCS_VERSION="${TFPLUGINDOCS_VERSION:-v0.24.0}"
CACHE_ROOT="${BADGEROPS_TOOL_CACHE_ROOT:-${ROOT_DIR}/.cache/tooling}"
PROVIDER_HOST="registry.terraform.io"
PROVIDER_NAMESPACE="badgerops"
PROVIDER_NAME="unifi"

cd "${ROOT_DIR}"

PROVIDER_VERSION="$(
  awk '
    match($0, /^## \[([0-9]+\.[0-9]+\.[0-9]+)\]/, parts) {
      print parts[1]
      exit
    }
  ' CHANGELOG.md
)"

if [[ -z "${PROVIDER_VERSION}" ]]; then
  echo "unable to determine provider version from CHANGELOG.md" >&2
  exit 1
fi

mkdir -p \
  "${CACHE_ROOT}/go-tmp" \
  "${CACHE_ROOT}/go-build" \
  "${CACHE_ROOT}/go-mod"

export GOTMPDIR="${BADGEROPS_GOTMPDIR:-${CACHE_ROOT}/go-tmp}"
export TMPDIR="${BADGEROPS_TMPDIR:-${CACHE_ROOT}/go-tmp}"
export GOCACHE="${BADGEROPS_GOCACHE:-${CACHE_ROOT}/go-build}"
export GOMODCACHE="${BADGEROPS_GOMODCACHE:-${CACHE_ROOT}/go-mod}"

if [[ -x /usr/bin/terraform ]]; then
  export PATH="/usr/bin:${PATH}"
elif command -v terraform >/dev/null 2>&1 && terraform version 2>/dev/null | grep -q '^Terraform v'; then
  :
else
  echo "Terraform CLI not found in PATH; tfplugindocs may download one automatically." >&2
fi

terraform fmt -recursive examples

schema_tmp_dir="$(mktemp -d "${TMPDIR}/tfplugindocs-schema.XXXXXX")"
trap 'rm -rf "${schema_tmp_dir}"' EXIT

target_os="$(go env GOOS)"
target_arch="$(go env GOARCH)"
mirror_dir="${schema_tmp_dir}/mirror/${PROVIDER_HOST}/${PROVIDER_NAMESPACE}/${PROVIDER_NAME}/${PROVIDER_VERSION}/${target_os}_${target_arch}"
schema_file="${schema_tmp_dir}/providers-schema.json"
terraformrc_file="${schema_tmp_dir}/terraformrc"

mkdir -p "${mirror_dir}"

go build -o "${mirror_dir}/terraform-provider-${PROVIDER_NAME}_v${PROVIDER_VERSION}" .

cat > "${schema_tmp_dir}/provider.tf" <<EOF
terraform {
  required_providers {
    ${PROVIDER_NAME} = {
      source  = "${PROVIDER_NAMESPACE}/${PROVIDER_NAME}"
      version = "${PROVIDER_VERSION}"
    }
  }
}

provider "${PROVIDER_NAME}" {}
EOF

cat > "${terraformrc_file}" <<EOF
provider_installation {
  filesystem_mirror {
    path    = "${schema_tmp_dir}/mirror"
    include = ["${PROVIDER_NAMESPACE}/${PROVIDER_NAME}"]
  }
  direct {
    exclude = ["${PROVIDER_NAMESPACE}/${PROVIDER_NAME}"]
  }
}
EOF

TF_CLI_CONFIG_FILE="${terraformrc_file}" terraform -chdir="${schema_tmp_dir}" init -backend=false >/dev/null
TF_CLI_CONFIG_FILE="${terraformrc_file}" terraform -chdir="${schema_tmp_dir}" providers schema -json > "${schema_file}"

perl -0pi -e 's/"registry\.terraform\.io\/badgerops\/unifi"/"unifi"/g' "${schema_file}"

go run "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@${TFPLUGINDOCS_VERSION}" generate \
  --provider-dir "${ROOT_DIR}" \
  --provider-name "${PROVIDER_NAME}" \
  --providers-schema "${schema_file}"

# Keep the generated provider index on a stable markdown EOF shape. tfplugindocs
# can emit an extra trailing blank line here, which otherwise causes docs drift.
perl -0pi -e 's/\n*\z/\n\n/s' "${ROOT_DIR}/docs/index.md"
