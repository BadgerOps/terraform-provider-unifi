# Publishing candidates and stable releases

The **Release** workflow is manual. Merging a PR or updating the changelog does not publish a release. Both candidates and stable promotions run the reusable Go (including lint), generated documentation, and Terraform example checks against the exact source commit before building and signing packages. The publishing job also installs the packaged provider from its filesystem mirror and loads its schema before signing.

## Prerequisites

- Merge the release workflow into `master` before the first dispatch.
- Configure the existing `GPG_PRIVATE_KEY` and `PASSPHRASE` Actions secrets, and register the public signing key with the Terraform Registry.
- Keep the Registry's GitHub release integration configured for this repository.
- Dispatch from `master`. Publishing needs repository write access; contributors can propose changes through normal PRs.
- Add changes to `[Unreleased]` in `CHANGELOG.md`. Keep README/example version references on the latest stable version during candidate testing.

All examples below use `0.6.0` illustratively. Select the appropriate next stable version for the actual changes. Versions must be newer than every published stable version. RC numbers start at 1, without leading zeros.

## Publish a candidate

1. Merge the changes to test, with useful `[Unreleased]` notes, into `master`.
2. Open **Actions → Release → Run workflow**, choose `master`, set `version` to `0.6.0-rc.1`, and leave `candidate` empty. Equivalently:

   ```shell
   gh workflow run release.yml --ref master -f version=0.6.0-rc.1
   ```

3. Inspect the workflow's validation and publishing jobs. It builds from the dispatch commit, creates a draft with all platform archives, manifest, checksums, signature, and mirror bundle, then publishes it as a GitHub prerelease without changing the latest stable release.
4. Check the release's source SHA and assets. After Registry ingestion, use a fresh Terraform directory with the exact RC constraint to verify a real Registry install, signature/checksum acceptance, `terraform validate`, and a controller-backed plan. This manual end-to-end verification is required before inviting other testers; local mirror tests do not prove Registry ingestion.
5. Share the release and [tester guide](../docs/guides/release-candidates.md). Record known issues in the release notes or linked issues, and collect reports with controller and Network versions. Allow a few days for testing across different controllers, adjusting the window to the risk of the changes.

Fixes go through PRs. Publish `0.6.0-rc.2` from the updated `master` when needed. Never move a published tag or replace its artifacts. Normal stable version constraints will not select an RC.

## Promote an approved candidate

After reviewing CI, controller lifecycle tests, and community reports, dispatch:

```shell
gh workflow run release.yml --ref master \
  -f version=0.6.0 \
  -f candidate=v0.6.0-rc.2
```

The workflow requires an existing published prerelease of the same base version. It verifies that the tag still matches the full source SHA recorded by our RC workflow and that the source is in the dispatch commit's history. It rebuilds **that exact candidate commit**, reruns checks, and embeds the stable version in the binary. Newer `master` changes are excluded. These are newly built stable packages, not renamed RC archives; the toolchain/dependency checks run again.

The stable release gets a new `v0.6.0` tag and becomes GitHub's latest release. Its notes come from `[Unreleased]` at the candidate commit, not from current `master`. The RC remains available. A stable release cannot be published directly from arbitrary untested source through this workflow.

After publication, submit a housekeeping PR on `master` to move **only the promoted candidate's entries** into a dated `[0.6.0]` changelog section. Leave later changes under `[Unreleased]`, run `make sync-version`, and regenerate docs. This updates stable install examples without changing the candidate source used for promotion. The published source tag still contains the candidate's `[Unreleased]` section; the GitHub release carries the versioned release notes.

## Failures and retries

- Validation, build, or signing failures before creating a release can be retried after resolving the cause. Source fixes require a new dispatch from the fixed commit; do not assume “re-run jobs” uses new code.
- Existing tags or releases, including drafts, cause preparation to fail. There is no overwrite or force-push path.
- If publishing fails after draft creation, inspect the draft and uploaded signed artifacts. A maintainer can finish publishing a complete draft after checking its source and assets. For an incomplete unpublished draft, remove it and any unpublished tag before retrying. Never delete or reuse a published version; publish a new candidate instead.
- GitHub Actions serializes release runs. Avoid queueing many requests: GitHub concurrency can replace an older pending run with a newer pending run. A running release is not canceled by this workflow.
- A successful GitHub release is not proof of Registry ingestion. If installation cannot find the version, inspect Registry publishing errors and signing configuration before announcing it.

Live controller credentials are not required by release CI. Controller-backed validation is performed separately on test infrastructure. Never upload state, saved plans, or credentials as test reports.
