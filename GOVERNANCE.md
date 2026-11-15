# Repository governance

## Merge policy

Pull requests use squash merging so each merged change has one focused commit on `master`. Required checks must pass and the branch must be current before merge. Direct pushes to `master` are reserved for repository recovery.

Merged branches are deleted automatically. Auto-merge may be used after the required review and checks are present.

## Maintainer recovery

The repository owner is the emergency bypass maintainer. Bypass is limited to restoring CI, reverting a broken or vulnerable release, or repairing repository rules that prevent normal review. The owner must open or reference an issue, explain the bypass in the resulting commit or pull request, and return the repository to the protected workflow immediately afterward.

Adding another maintainer should use organization team ownership rather than granting broad individual access. Until the repository is organization-owned and a second trusted maintainer is appointed, the documented owner bypass is the recovery path.

## Optional GitHub features

GitHub Projects and the repository wiki are intentionally disabled. Roadmap work is tracked with issues, labels, and milestones; maintained documentation lives in the repository.
