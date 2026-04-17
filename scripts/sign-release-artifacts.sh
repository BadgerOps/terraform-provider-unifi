#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <version> [dist_dir]" >&2
  exit 1
fi

if [[ -z "${GPG_PRIVATE_KEY:-}" ]]; then
  echo "GPG_PRIVATE_KEY is required" >&2
  exit 1
fi

if [[ -z "${PASSPHRASE:-}" ]]; then
  echo "PASSPHRASE is required" >&2
  exit 1
fi

version="$1"
dist_dir="${2:-dist/release}"
provider_name="terraform-provider-unifi"
checksum_file="${dist_dir}/${provider_name}_${version}_SHA256SUMS"
signature_file="${checksum_file}.sig"

if [[ ! -f "${checksum_file}" ]]; then
  echo "missing checksum file: ${checksum_file}" >&2
  exit 1
fi

gnupg_home="$(mktemp -d)"
trap 'rm -rf "${gnupg_home}"' EXIT
chmod 700 "${gnupg_home}"
export GNUPGHOME="${gnupg_home}"

printf '%s\n' "${GPG_PRIVATE_KEY}" | gpg --batch --import

fingerprint="$(
  gpg --batch --with-colons --list-secret-keys |
    awk -F: '$1 == "fpr" { print $10; exit }'
)"

if [[ -z "${fingerprint}" ]]; then
  echo "unable to determine imported GPG key fingerprint" >&2
  exit 1
fi

gpg \
  --batch \
  --yes \
  --pinentry-mode loopback \
  --passphrase "${PASSPHRASE}" \
  --local-user "${fingerprint}" \
  --detach-sign \
  --output "${signature_file}" \
  "${checksum_file}"
