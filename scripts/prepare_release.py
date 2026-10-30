#!/usr/bin/env python3
"""Resolve an explicit RC or stable promotion without creating tags or releases."""

import base64
import json
import os
from pathlib import Path
import re
import subprocess
import sys

VERSION = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-rc\.([1-9][0-9]*))?")


def parse_version(value):
    match = VERSION.fullmatch(value)
    if not match:
        raise ValueError("version must be X.Y.Z or X.Y.Z-rc.N (N >= 1), without a v prefix")
    return tuple(int(part) for part in match.groups()[:3]), match.group(4)


def git(*args):
    return subprocess.check_output(["git", *args], text=True).strip()


def unreleased_notes(changelog):
    match = re.search(r"^## \[Unreleased\][^\n]*\n(.*?)(?=^## \[|\Z)", changelog, re.M | re.S)
    if not match or not re.search(r"^[-*] \S", match.group(1), re.M):
        raise ValueError("CHANGELOG.md needs a nonempty [Unreleased] section")
    return match.group(1).strip()


def prepare(version, candidate, releases):
    base, rc = parse_version(version)
    tag = "v" + version
    existing_tags = git("tag", "--list").splitlines()
    if tag in existing_tags or any(r["tag_name"] == tag for r in releases):
        raise ValueError(f"{tag} already exists; never overwrite a release or tag")
    stable_versions = []
    for release in releases:
        if release["draft"] or release["prerelease"]:
            continue
        try:
            released_base, released_rc = parse_version(release["tag_name"].removeprefix("v"))
        except ValueError:
            continue
        if released_rc is None:
            stable_versions.append(released_base)
    if stable_versions and base <= max(stable_versions):
        raise ValueError("release version must be newer than every published stable version")

    if rc:
        if candidate:
            raise ValueError("candidate must be empty when publishing an RC")
        source = git("rev-parse", "HEAD")
    else:
        if not candidate.startswith("v"):
            raise ValueError("stable publication requires a candidate tag such as v0.6.0-rc.1")
        candidate_base, candidate_rc = parse_version(candidate[1:])
        if candidate_base != base or not candidate_rc:
            raise ValueError("candidate must be an RC of the requested stable version")
        release = next((r for r in releases if r["tag_name"] == candidate), None)
        if not release or release["draft"] or not release["prerelease"]:
            raise ValueError("candidate must be a published GitHub prerelease")
        source = git("rev-parse", "--verify", f"refs/tags/{candidate}^{{commit}}")
        # Our workflow records the full source SHA in target_commitish. Reject a
        # moved tag or a manually created candidate without that provenance.
        if release["target_commitish"] != source:
            raise ValueError("candidate tag does not match its release's recorded source SHA")
        git("merge-base", "--is-ancestor", source, "HEAD")

    changelog = git("show", f"{source}:CHANGELOG.md")
    notes = unreleased_notes(changelog)
    if rc:
        intro = f"Release candidate **{tag}** for opt-in testing; not a stable release."
    else:
        intro = f"Stable release **{tag}**, promoted from **{candidate}**."
    notes = f"{intro}\n\nSource commit: `{source}`\n\n{notes}\n"
    if rc:
        notes += f'''
## Testing this candidate

Use `source = "badgerops/unifi"` and `version = "= {version}"` in
`required_providers`, then run `terraform init -upgrade`, `terraform validate`,
and `terraform plan`. Disable local development overrides for this provider.
The Registry can take a few minutes to ingest the release.

Review the changes listed above and exercise the affected resources on a test
site. Report the RC version, Terraform version, controller model, UniFi OS and
Network versions, affected resources, and sanitized diagnostics in a GitHub issue.
Never include API keys, WiFi passwords, state files, or saved plan files.

See the [tester guide](https://github.com/BadgerOps/terraform-provider-unifi/blob/{source}/docs/guides/release-candidates.md)
and [open issues](https://github.com/BadgerOps/terraform-provider-unifi/issues) for testing guidance and known issues. This candidate has not yet completed
community validation; a successful plan does not establish apply-time behavior.
'''
    return {
        "version": version,
        "source_sha": source,
        "prerelease": "true" if rc else "false",
        "notes_base64": base64.b64encode(notes.encode()).decode(),
    }


def main():
    if os.environ.get("GITHUB_REF") != "refs/heads/master":
        raise ValueError("dispatch the Release workflow from master")
    pages = json.loads(subprocess.check_output([
        "gh", "api", "--paginate", "--slurp",
        f"repos/{os.environ['GITHUB_REPOSITORY']}/releases?per_page=100",
    ], text=True))
    result = prepare(os.environ["RELEASE_VERSION"], os.environ.get("CANDIDATE_TAG", ""),
                     [release for page in pages for release in page])
    with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
        for key, value in result.items():
            output.write(f"{key}={value}\n")
    print(f"Validated {result['version']} at {result['source_sha']}")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, subprocess.CalledProcessError) as error:
        print(f"Release validation failed: {error}", file=sys.stderr)
        sys.exit(1)
