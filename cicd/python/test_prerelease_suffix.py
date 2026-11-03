import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from prerelease_suffix import prerelease_suffix


class PrereleaseSuffixTests(unittest.TestCase):
    def test_branch_policy(self):
        for branch, live, expected in [
            ("stable/v0.13", "stable/v0.13", ""),
            ("stable/v0.12", "stable/v0.13", ""),
            ("stable/older/maintenance", "stable/v0.13", ""),
            ("main", "stable/v0.13", "-rc"),
            ("main", "main", ""),
            ("feature/example", "stable/v0.13", "-alpha"),
            ("develop", "main", "-alpha"),
            ("version-next", "main", "-alpha"),
            ("production", "production", ""),
        ]:
            with self.subTest(branch=branch, live=live):
                self.assertEqual(prerelease_suffix(branch, live), expected)

    def test_tags_do_not_infer_their_source_branch_from_the_name(self):
        for tag in ("build-release-test-1", "main", "stable/v0.13", "unpublished-commit"):
            with self.subTest(tag=tag):
                self.assertEqual(prerelease_suffix(tag, "stable/v0.13", "tag"), "-beta")

    def test_cli_reads_github_tag_context(self):
        with tempfile.TemporaryDirectory() as directory:
            github_env = Path(directory) / "github-env"
            result = subprocess.run(
                [sys.executable, str(Path(__file__).with_name("prerelease_suffix.py")), "build-release-test-1", "main"],
                env=dict(os.environ, GITHUB_ENV=str(github_env), GITHUB_REF_TYPE="tag"),
                capture_output=True, text=True, check=True,
            )
            self.assertEqual(result.stdout, "-beta\n")
            self.assertEqual(github_env.read_text(encoding="utf-8"), "BUILDPRERELEASESUFFIX=-beta\n")

    def test_cli_exports_empty_and_nonempty_suffixes(self):
        with tempfile.TemporaryDirectory() as directory:
            github_env = Path(directory) / "github-env"
            github_env.write_text("EXISTING=value\n", encoding="utf-8")
            for branch, expected in [("main", "-rc"), ("stable/v0.12", ""), ("feature/test", "-alpha")]:
                result = subprocess.run(
                    [sys.executable, str(Path(__file__).with_name("prerelease_suffix.py")), branch, "stable/v0.13"],
                    env=dict(os.environ, GITHUB_ENV=str(github_env), GITHUB_REF_TYPE="branch"), capture_output=True, text=True, check=True,
                )
                self.assertEqual(result.stdout, expected + "\n")
            self.assertEqual(github_env.read_text(encoding="utf-8"),
                             "EXISTING=value\nBUILDPRERELEASESUFFIX=-rc\nBUILDPRERELEASESUFFIX=\nBUILDPRERELEASESUFFIX=-alpha\n")


if __name__ == "__main__":
    unittest.main()
