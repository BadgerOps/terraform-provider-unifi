#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST_PATH="${ROOT_DIR}/internal/openapi/spec/manifest.json"
REPO_BASE_URL="https://www.ui.com/downloads/unifi/debian"
SUITE="stable"
COMPONENT="ubiquiti"
ARCH="amd64"
PACKAGE_NAME="unifi"

usage() {
  cat <<'EOF'
usage: check-openapi-upstream.sh [--manifest PATH] [--repo-base-url URL] [--suite NAME] [--component NAME] [--arch NAME] [--package-name NAME]

Checks the current stable UniFi package feed, downloads the latest package for the
selected architecture, extracts api-docs/integration.json, and compares it with the
committed OpenAPI snapshot manifest.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --manifest)
      MANIFEST_PATH="$2"
      shift 2
      ;;
    --repo-base-url)
      REPO_BASE_URL="$2"
      shift 2
      ;;
    --suite)
      SUITE="$2"
      shift 2
      ;;
    --component)
      COMPONENT="$2"
      shift 2
      ;;
    --arch)
      ARCH="$2"
      shift 2
      ;;
    --package-name)
      PACKAGE_NAME="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ ! -f "${MANIFEST_PATH}" ]]; then
  echo "manifest not found: ${MANIFEST_PATH}" >&2
  exit 1
fi

required_tools=(curl gzip python3 sha256sum find mktemp tar)
for tool in "${required_tools[@]}"; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "required tool not found: ${tool}" >&2
    exit 1
  fi
done

extract_deb() {
  local package_file="$1"
  local extract_dir="$2"

  if command -v dpkg-deb >/dev/null 2>&1; then
    dpkg-deb -x "${package_file}" "${extract_dir}"
    return 0
  fi

  if ! command -v ar >/dev/null 2>&1; then
    echo "required tool not found: ar" >&2
    return 1
  fi

  local ar_dir="${work_dir}/ar"
  mkdir -p "${ar_dir}"
  (
    cd "${ar_dir}"
    ar x "${package_file}"
  )

  local data_archive=""
  for candidate in data.tar.zst data.tar.xz data.tar.gz data.tar; do
    if [[ -f "${ar_dir}/${candidate}" ]]; then
      data_archive="${ar_dir}/${candidate}"
      break
    fi
  done

  if [[ -z "${data_archive}" ]]; then
    echo "unable to locate data archive in deb package" >&2
    return 1
  fi

  case "${data_archive}" in
    *.tar.zst)
      tar --zstd -xf "${data_archive}" -C "${extract_dir}"
      ;;
    *.tar.xz)
      tar -xJf "${data_archive}" -C "${extract_dir}"
      ;;
    *.tar.gz)
      tar -xzf "${data_archive}" -C "${extract_dir}"
      ;;
    *.tar)
      tar -xf "${data_archive}" -C "${extract_dir}"
      ;;
    *)
      echo "unsupported data archive format: ${data_archive}" >&2
      return 1
      ;;
  esac
}

packages_url="${REPO_BASE_URL}/dists/${SUITE}/${COMPONENT}/binary-${ARCH}/Packages.gz"

cache_root="${ROOT_DIR}/.cache/openapi-upstream-check"
mkdir -p "${cache_root}"
work_dir="$(mktemp -d "${cache_root}/run.XXXXXX")"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

mapfile -t manifest_fields < <(
  python3 - "${MANIFEST_PATH}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    manifest = json.load(fh)

print(manifest["upstream"]["api_version"])
print(manifest["upstream"]["openapi_version"])
print(manifest["upstream"]["source_package"]["version"])
print(manifest["snapshot"]["sha256"])
PY
)

current_api_version="${manifest_fields[0]}"
current_openapi_version="${manifest_fields[1]}"
current_source_package_version="${manifest_fields[2]}"
current_snapshot_sha256="${manifest_fields[3]}"

packages_gz_file="${work_dir}/Packages.gz"
packages_file="${work_dir}/Packages"
curl -fsSL -o "${packages_gz_file}" "${packages_url}"
gzip -dc "${packages_gz_file}" > "${packages_file}"

mapfile -t package_fields < <(
  python3 - "${packages_file}" "${PACKAGE_NAME}" <<'PY'
import sys

packages_path = sys.argv[1]
package_name = sys.argv[2]

with open(packages_path, "r", encoding="utf-8") as fh:
    content = fh.read()

for stanza in content.split("\n\n"):
    fields = {}
    for line in stanza.splitlines():
        if not line or line.startswith(" "):
            continue
        key, sep, value = line.partition(":")
        if not sep:
            continue
        fields[key] = value.strip()
    if fields.get("Package") == package_name:
        print(fields.get("Version", ""))
        print(fields.get("Filename", ""))
        print(fields.get("SHA256", ""))
        break
else:
    raise SystemExit(f"package not found in Packages index: {package_name}")
PY
)

latest_package_version="${package_fields[0]}"
latest_package_filename="${package_fields[1]}"
latest_package_sha256="${package_fields[2]}"

if [[ -z "${latest_package_version}" || -z "${latest_package_filename}" ]]; then
  echo "unable to resolve package metadata for ${PACKAGE_NAME}" >&2
  exit 1
fi

package_url="${REPO_BASE_URL}/${latest_package_filename}"
package_file="${work_dir}/${PACKAGE_NAME}.deb"
curl -fsSL -o "${package_file}" "${package_url}"

extract_dir="${work_dir}/extract"
mkdir -p "${extract_dir}"
extract_deb "${package_file}" "${extract_dir}"

integration_path="$(find "${extract_dir}" -path '*/api-docs/integration.json' -print -quit)"
snapshot_source="package-version"
latest_api_version="${latest_package_version%%-*}"
latest_openapi_version="unknown"
latest_snapshot_sha256="unavailable"

if [[ -n "${integration_path}" ]]; then
  latest_snapshot_sha256="$(sha256sum "${integration_path}" | awk '{print $1}')"

  mapfile -t spec_fields < <(
    python3 - "${integration_path}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    spec = json.load(fh)

print(spec["info"]["version"])
print(spec["openapi"])
PY
  )

  latest_api_version="${spec_fields[0]}"
  latest_openapi_version="${spec_fields[1]}"
  snapshot_source="packaged-api-docs"
fi

changed_fields=()
if [[ "${current_api_version}" != "${latest_api_version}" ]]; then
  changed_fields+=("api_version")
fi
if [[ "${latest_openapi_version}" != "unknown" && "${current_openapi_version}" != "${latest_openapi_version}" ]]; then
  changed_fields+=("openapi_version")
fi
if [[ "${latest_snapshot_sha256}" != "unavailable" && "${current_snapshot_sha256}" != "${latest_snapshot_sha256}" ]]; then
  changed_fields+=("snapshot_sha256")
fi

update_available="false"
if [[ "${#changed_fields[@]}" -gt 0 ]]; then
  update_available="true"
fi

comparison_fields="none"
if [[ "${#changed_fields[@]}" -gt 0 ]]; then
  comparison_fields="$(IFS=,; echo "${changed_fields[*]}")"
fi

cat <<EOF
current_api_version=${current_api_version}
current_openapi_version=${current_openapi_version}
current_source_package_version=${current_source_package_version}
current_snapshot_sha256=${current_snapshot_sha256}
latest_api_version=${latest_api_version}
latest_openapi_version=${latest_openapi_version}
latest_package_version=${latest_package_version}
latest_package_sha256=${latest_package_sha256}
latest_snapshot_sha256=${latest_snapshot_sha256}
packages_url=${packages_url}
package_url=${package_url}
snapshot_source=${snapshot_source}
update_available=${update_available}
comparison_fields=${comparison_fields}
EOF
