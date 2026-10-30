#!/usr/bin/env bash

# Install the packaged provider through its mirror, without controller access.
set -euo pipefail

version="${1:?usage: smoke-test-release.sh <version> [dist_dir]}"
dist_dir="$(realpath "${2:-dist/release}")"
provider_name="terraform-provider-unifi"
test_dir="$(mktemp -d)"
trap 'rm -rf "${test_dir}"' EXIT

(cd "${dist_dir}" && sha256sum -c "${provider_name}_${version}_SHA256SUMS")
tar -xzf "${dist_dir}/${provider_name}_${version}_terraform-mirror.tar.gz" -C "${test_dir}"
platform="$(go env GOOS)_$(go env GOARCH)"
archive="${provider_name}_${version}_${platform}.zip"
cmp "${dist_dir}/${archive}" \
  "${test_dir}/terraform-mirror/registry.terraform.io/badgerops/unifi/${archive}"

cat > "${test_dir}/main.tf" <<HCL
terraform {
  required_providers {
    unifi = {
      source  = "badgerops/unifi"
      version = "= ${version}"
    }
  }
}
HCL
cat > "${test_dir}/terraformrc" <<HCL
provider_installation {
  filesystem_mirror {
    path = "${test_dir}/terraform-mirror"
    include = ["badgerops/unifi"]
  }
}
HCL
export TF_CLI_CONFIG_FILE="${test_dir}/terraformrc"
export TF_DATA_DIR="${test_dir}/.terraform"
terraform -chdir="${test_dir}" init -backend=false -input=false
terraform -chdir="${test_dir}" validate
terraform -chdir="${test_dir}" providers schema -json > "${test_dir}/schema.json"
python3 - "${test_dir}/schema.json" <<'PY'
import json
import sys
with open(sys.argv[1]) as source:
    schema = json.load(source)
assert schema['provider_schemas']['registry.terraform.io/badgerops/unifi']['resource_schemas']
print('Packaged provider installed and returned its resource schema.')
PY
