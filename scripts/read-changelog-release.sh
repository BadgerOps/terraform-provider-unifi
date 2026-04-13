#!/usr/bin/env bash

set -euo pipefail

changelog="${1:-CHANGELOG.md}"

if [[ ! -f "${changelog}" ]]; then
  echo "missing changelog: ${changelog}" >&2
  exit 1
fi

version="$(
  awk '
    match($0, /^## \[([0-9]+\.[0-9]+\.[0-9]+)\]/, parts) {
      print parts[1]
      exit
    }
  ' "${changelog}"
)"

if [[ -z "${version}" ]]; then
  echo "unable to find a semver release heading in ${changelog}" >&2
  exit 1
fi

notes="$(
  awk -v version="${version}" '
    $0 ~ "^## \\[" version "\\]" { in_section = 1; next }
    in_section && /^## \[/ { exit }
    in_section { print }
  ' "${changelog}"
)"

notes="$(printf '%s\n' "${notes}" | sed '1{/^[[:space:]]*$/d;}; :a; /^[[:space:]]*$/{$d;N;ba;};')"

if [[ -z "${notes}" ]]; then
  echo "release ${version} in ${changelog} has no notes" >&2
  exit 1
fi

printf 'version=%s\n' "${version}"
printf 'notes<<__CHANGELOG__\n%s\n__CHANGELOG__\n' "${notes}"
