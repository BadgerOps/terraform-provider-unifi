#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <version> [dist_dir]" >&2
  exit 1
fi

version="$1"
dist_dir="${2:-dist/release}"
repo_root="$(pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

provider_type="unifi"
provider_host="registry.terraform.io"
provider_namespace="badgerops"
provider_name="terraform-provider-${provider_type}"
manifest_source="${repo_root}/terraform-registry-manifest.json"

if [[ ! -f "${manifest_source}" ]]; then
  echo "missing Terraform Registry manifest file: ${manifest_source}" >&2
  exit 1
fi

platforms=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

rm -rf "${dist_dir}"
mkdir -p "${dist_dir}"

mirror_root="${work_dir}/terraform-mirror/${provider_host}/${provider_namespace}/${provider_type}/${version}"

for platform in "${platforms[@]}"; do
  goos="${platform%/*}"
  goarch="${platform#*/}"
  target="${goos}_${goarch}"
  ext=""
  if [[ "${goos}" == "windows" ]]; then
    ext=".exe"
  fi

  build_dir="${work_dir}/build/${target}"
  archive_name="${provider_name}_${version}_${target}.zip"
  binary_name="${provider_name}_v${version}${ext}"

  mkdir -p "${build_dir}" "${mirror_root}/${target}"

  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${version}" \
      -o "${build_dir}/${binary_name}" \
      .

  (
    cd "${build_dir}"
    zip -q -9 "${repo_root}/${dist_dir}/${archive_name}" "${binary_name}"
  )

  cp "${dist_dir}/${archive_name}" "${mirror_root}/${target}/${archive_name}"
done

manifest_asset="${provider_name}_${version}_manifest.json"
cp "${manifest_source}" "${dist_dir}/${manifest_asset}"

mirror_bundle="${dist_dir}/${provider_name}_${version}_terraform-mirror.tar.gz"
tar -C "${work_dir}" -czf "${mirror_bundle}" terraform-mirror

(
  cd "${dist_dir}"
  mapfile -t checksum_inputs < <(printf '%s\n' *.zip "${manifest_asset}" | sort)
  sha256sum "${checksum_inputs[@]}" > "${provider_name}_${version}_SHA256SUMS"
)
