import base64
import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location("prepare_release", Path(__file__).parents[1] / "prepare_release.py")
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.previous = Path.cwd()
        os.chdir(self.directory.name)
        self.addCleanup(self.directory.cleanup)
        self.addCleanup(os.chdir, self.previous)
        release.git("init", "-q")
        release.git("config", "user.email", "test@example.com")
        release.git("config", "user.name", "Release Test")
        self.commit("Candidate change")
        self.source = release.git("rev-parse", "HEAD")
        self.stable = {"tag_name": "v0.5.5", "draft": False, "prerelease": False}
        self.candidate = {
            "tag_name": "v0.6.0-rc.1", "draft": False, "prerelease": True,
            "target_commitish": self.source,
        }

    def commit(self, note):
        Path("CHANGELOG.md").write_text(f"# Changelog\n\n## [Unreleased]\n\n### Added\n\n- {note}\n\n## [0.5.5]\n\n- Old release\n")
        release.git("add", "CHANGELOG.md")
        release.git("commit", "-qm", note)

    def test_rc_from_current_commit_with_unreleased_notes(self):
        result = release.prepare("0.6.0-rc.1", "", [self.stable])
        self.assertEqual(result["source_sha"], self.source)
        self.assertEqual(result["prerelease"], "true")
        notes = base64.b64decode(result["notes_base64"]).decode()
        self.assertIn("Candidate change", notes)
        self.assertIn('version = "= 0.6.0-rc.1"', notes)
        self.assertNotIn("Old release", notes)

    def test_stable_uses_candidate_source_and_notes_after_master_advances(self):
        release.git("tag", "v0.6.0-rc.1")
        self.commit("Untested later change")
        result = release.prepare("0.6.0", "v0.6.0-rc.1", [self.stable, self.candidate])
        self.assertEqual(result["source_sha"], self.source)
        self.assertEqual(result["prerelease"], "false")
        notes = base64.b64decode(result["notes_base64"]).decode()
        self.assertIn("Candidate change", notes)
        self.assertNotIn("Untested later change", notes)

    def test_reject_invalid_versions(self):
        for version in ["v0.6.0", "0.6", "01.6.0", "0.6.0-rc.0", "0.6.0-rc.01",
                        "0.6.0-beta.1", "0.6.0+build", "0.6.0\n", "$(echo bad)", "../../bad"]:
            with self.subTest(version=version), self.assertRaises(ValueError):
                release.prepare(version, "", [self.stable])

    def test_reject_existing_tag_or_release_including_drafts(self):
        for draft in [False, True]:
            with self.subTest(draft=draft), self.assertRaisesRegex(ValueError, "already exists"):
                release.prepare("0.6.0-rc.1", "", [{**self.candidate, "draft": draft}])
        release.git("tag", "v0.6.0-rc.1")
        with self.assertRaisesRegex(ValueError, "already exists"):
            release.prepare("0.6.0-rc.1", "", [])

    def test_reject_old_or_already_stable_version(self):
        for version in ["0.5.4-rc.1", "0.5.5-rc.1", "0.5.5"]:
            with self.subTest(version=version), self.assertRaisesRegex(ValueError, "newer|already exists"):
                release.prepare(version, "", [self.stable])
        # Compare numerically, not lexically.
        release.prepare("0.10.0-rc.1", "", [{**self.stable, "tag_name": "v0.9.0"}])

    def test_reject_incorrect_promotion_inputs(self):
        for candidate in ["", "0.6.0-rc.1", "v0.7.0-rc.1", "v0.6.0"]:
            with self.subTest(candidate=candidate), self.assertRaises(ValueError):
                release.prepare("0.6.0", candidate, [self.stable, self.candidate])
        with self.assertRaisesRegex(ValueError, "empty"):
            release.prepare("0.6.0-rc.2", "v0.6.0-rc.1", [self.candidate])

    def test_reject_unpublished_or_non_rc_candidate(self):
        release.git("tag", "v0.6.0-rc.1")
        for candidates in [[], [{**self.candidate, "draft": True}], [{**self.candidate, "prerelease": False}]]:
            with self.subTest(candidates=candidates), self.assertRaisesRegex(ValueError, "published"):
                release.prepare("0.6.0", "v0.6.0-rc.1", candidates)

    def test_reject_moved_candidate_tag(self):
        self.commit("Changed source")
        release.git("tag", "v0.6.0-rc.1")
        with self.assertRaisesRegex(ValueError, "recorded source SHA"):
            release.prepare("0.6.0", "v0.6.0-rc.1", [self.candidate])

    def test_reject_candidate_not_in_master_history(self):
        self.commit("Unmerged candidate")
        release.git("tag", "v0.6.0-rc.1")
        candidate = {**self.candidate, "target_commitish": release.git("rev-parse", "HEAD")}
        release.git("reset", "--hard", self.source)
        with self.assertRaises(subprocess.CalledProcessError):
            release.prepare("0.6.0", "v0.6.0-rc.1", [candidate])

    def test_dispatch_requires_master_before_any_api_call(self):
        with mock.patch.dict(os.environ, {"GITHUB_REF": "refs/heads/feature"}), \
             mock.patch.object(release.subprocess, "check_output") as command:
            with self.assertRaisesRegex(ValueError, "master"):
                release.main()
            command.assert_not_called()

    def test_main_writes_single_line_outputs_from_paginated_releases(self):
        output = Path("outputs")
        environment = {"GITHUB_REF": "refs/heads/master", "GITHUB_REPOSITORY": "owner/repo",
                       "RELEASE_VERSION": "0.6.0-rc.1", "CANDIDATE_TAG": "",
                       "GITHUB_OUTPUT": str(output)}
        real_command = subprocess.check_output

        def command(args, **kwargs):
            if args[0] == "gh":
                self.assertIn("--paginate", args)
                self.assertIn("--slurp", args)
                return release.json.dumps([[self.stable], []])
            return real_command(args, **kwargs)

        with mock.patch.dict(os.environ, environment), \
             mock.patch.object(release.subprocess, "check_output", side_effect=command):
            release.main()
        values = dict(line.split("=", 1) for line in output.read_text().splitlines())
        self.assertEqual(values["source_sha"], self.source)
        self.assertEqual(values["version"], "0.6.0-rc.1")
        self.assertEqual(len(values), 4)

    def test_reject_empty_unreleased_notes(self):
        for changelog in ["# Changelog", "## [Unreleased]\n\n### Added\n\n## [0.5.5]\n- Old release"]:
            with self.subTest(changelog=changelog), self.assertRaisesRegex(ValueError, "nonempty"):
                release.unreleased_notes(changelog)


if __name__ == "__main__":
    unittest.main()
