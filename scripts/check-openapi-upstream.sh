#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST_PATH="${ROOT_DIR}/internal/openapi/spec/manifest.json"
FIRMWARE_API_URL="https://fw-update.ubnt.com/api/firmware-latest"
PRODUCT="unifi-native"
CHANNEL="release"
PLATFORM="uos-deb11-arm64"
SAVE_SPEC_PATH=""

usage() {
  cat <<'EOF'
usage: check-openapi-upstream.sh [--manifest PATH] [--firmware-api-url URL] [--product NAME] [--channel NAME] [--platform NAME] [--save-spec PATH]

Asks the Ubiquiti firmware API for the latest UniFi Network package on the selected
release channel, downloads it, extracts api-docs/integration.json when the package
ships it, and compares that document with the committed OpenAPI snapshot manifest.

The firmware API replaced the apt Packages index as the source because the apt
stable suite stopped listing packages when it moved to the 10.5 line.

When the latest package does not ship an OpenAPI document, the package metadata is
reported but no OpenAPI drift is raised. The provider is generated from the OpenAPI
snapshot, not from the application package version alone.

--save-spec copies the extracted api-docs/integration.json to PATH so the committed
snapshot can be refreshed from the same artifact the check inspected.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --manifest)
      MANIFEST_PATH="$2"
      shift 2
      ;;
    --firmware-api-url)
      FIRMWARE_API_URL="$2"
      shift 2
      ;;
    --product)
      PRODUCT="$2"
      shift 2
      ;;
    --channel)
      CHANNEL="$2"
      shift 2
      ;;
    --platform)
      PLATFORM="$2"
      shift 2
      ;;
    --save-spec)
      SAVE_SPEC_PATH="$2"
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

required_tools=(curl python3 sha256sum find mktemp tar)
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

packages_url="${FIRMWARE_API_URL}?filter=eq~~product~~${PRODUCT}&filter=eq~~channel~~${CHANNEL}"

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

firmware_index_file="${work_dir}/firmware-latest.json"
curl -fsSL -o "${firmware_index_file}" "${packages_url}"

package_fields_output="$(
  python3 - "${firmware_index_file}" "${PLATFORM}" <<'PY'
import json
import sys

index_path = sys.argv[1]
platform = sys.argv[2]

with open(index_path, "r", encoding="utf-8") as fh:
    index = json.load(fh)

entries = index.get("_embedded", {}).get("firmware", [])
for entry in entries:
    if entry.get("platform") == platform:
        print(entry.get("version", "").removeprefix("v"))
        print(entry.get("_links", {}).get("data", {}).get("href", ""))
        print(entry.get("sha256_checksum", ""))
        break
else:
    platforms = ", ".join(sorted({entry.get("platform", "") for entry in entries})) or "none"
    raise SystemExit(f"platform not found in firmware index: {platform} (available: {platforms})")
PY
)"

mapfile -t package_fields <<< "${package_fields_output}"

latest_package_version="${package_fields[0]}"
package_url="${package_fields[1]}"
latest_package_sha256="${package_fields[2]}"

if [[ -z "${latest_package_version}" || -z "${package_url}" || -z "${latest_package_sha256}" ]]; then
  echo "unable to resolve package metadata for ${PRODUCT} on ${PLATFORM}" >&2
  exit 1
fi

package_file="${work_dir}/${PRODUCT}.deb"
curl -fsSL -o "${package_file}" "${package_url}"

downloaded_package_sha256="$(sha256sum "${package_file}" | awk '{print $1}')"
if [[ "${downloaded_package_sha256}" != "${latest_package_sha256}" ]]; then
  echo "package checksum mismatch for ${package_url}: expected ${latest_package_sha256}, got ${downloaded_package_sha256}" >&2
  exit 1
fi

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

  if [[ -n "${SAVE_SPEC_PATH}" ]]; then
    cp "${integration_path}" "${SAVE_SPEC_PATH}"
  fi
elif [[ -n "${SAVE_SPEC_PATH}" ]]; then
  echo "package ${latest_package_version} does not ship api-docs/integration.json; nothing to save" >&2
  exit 1
fi

changed_fields=()
if [[ "${snapshot_source}" == "packaged-api-docs" ]]; then
  if [[ "${current_api_version}" != "${latest_api_version}" ]]; then
    changed_fields+=("api_version")
  fi
  if [[ "${current_openapi_version}" != "${latest_openapi_version}" ]]; then
    changed_fields+=("openapi_version")
  fi
  if [[ "${current_snapshot_sha256}" != "${latest_snapshot_sha256}" ]]; then
    changed_fields+=("snapshot_sha256")
  fi
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
