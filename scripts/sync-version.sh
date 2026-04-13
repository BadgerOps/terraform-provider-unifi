#!/usr/bin/env bash

set -euo pipefail

mode="fix"
if [[ "${1:-}" == "--check" ]]; then
  mode="check"
elif [[ $# -gt 0 ]]; then
  echo "usage: $0 [--check]" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

version="$(
  awk '
    match($0, /^## \[([0-9]+\.[0-9]+\.[0-9]+)\]/, parts) {
      print parts[1]
      exit
    }
  ' CHANGELOG.md
)"

if [[ -z "${version}" ]]; then
  echo "unable to determine current release version from CHANGELOG.md" >&2
  exit 1
fi

managed_files=(
  "README.md"
  "examples/basic-site/main.tf"
)

tmp_dir=""
if [[ "${mode}" == "check" ]]; then
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' EXIT

  for file in "${managed_files[@]}"; do
    cp "${file}" "${tmp_dir}/$(echo "${file}" | tr '/' '_')"
  done
fi

perl -0pi -e '
  s/(source = "badgerops\/unifi"\n      version = ")\d+\.\d+\.\d+(")/${1}'"${version}"'${2}/g;
  s/(go build -o terraform-provider-unifi_v)\d+\.\d+\.\d+( \.)/${1}'"${version}"'${2}/g;
  s/(make release-artifacts VERSION=)\d+\.\d+\.\d+/${1}'"${version}"'/g;
' README.md

perl -0pi -e '
  s/(source  = "badgerops\/unifi"\n      version = ")\d+\.\d+\.\d+(")/${1}'"${version}"'${2}/g;
' examples/basic-site/main.tf

if [[ "${mode}" == "check" ]]; then
  changed=0
  for file in "${managed_files[@]}"; do
    if ! cmp -s "${tmp_dir}/$(echo "${file}" | tr '/' '_')" "${file}"; then
      changed=1
      break
    fi
  done

  if [[ "${changed}" -eq 1 ]]; then
    echo "version references were updated to ${version}; stage the changes and re-run the commit" >&2
    git diff -- "${managed_files[@]}" >&2
    exit 1
  fi
fi
