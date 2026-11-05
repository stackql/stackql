import unittest
from unittest import mock

import attach_build_artifacts as attach


class GhTests(unittest.TestCase):
    def test_gh_decodes_output_as_utf8(self):
        # Regression: on Windows the default cp1252 decoding of gh output crashed
        # the stdout reader thread on the first non-cp1252 byte.
        completed = mock.Mock(returncode=0, stdout='{"sha": "f7b4116"}', stderr='')
        with mock.patch.object(attach.subprocess, 'run', return_value=completed) as run:
            self.assertEqual(attach.gh('api', 'repos/stackql/stackql/commits/v1.0.888'),
                             '{"sha": "f7b4116"}')
        self.assertEqual(run.call_args.kwargs.get('encoding'), 'utf-8')
        self.assertEqual(run.call_args.kwargs.get('errors'), 'replace')

    def test_tag_commit_requests_only_the_sha(self):
        with mock.patch.object(attach, 'gh', return_value='f7b41161\n') as gh:
            self.assertEqual(attach.tag_commit('stackql/stackql', 'v1.0.888'), 'f7b41161')
        gh.assert_called_once_with('api', 'repos/stackql/stackql/commits/v1.0.888', '--jq', '.sha')


class VersionTests(unittest.TestCase):
    def test_accepts_with_and_without_v_prefix(self):
        self.assertEqual(attach.parse_version('0.12.778'), '0.12.778')
        self.assertEqual(attach.parse_version('v0.12.778'), '0.12.778')
        self.assertEqual(attach.build_number('0.12.778'), 778)

    def test_rejects_malformed_versions(self):
        for raw in ('0.12', 'v0.12.778.1', 'latest', ''):
            with self.assertRaises(attach.ReleaseError):
                attach.parse_version(raw)


class SidecarTests(unittest.TestCase):
    def test_sha256sum_format(self):
        digest = 'ab' * 32
        self.assertEqual(attach.sidecar_line(digest, 'stackql_linux_amd64.zip'),
                         f'{digest}  stackql_linux_amd64.zip\n')


class PlanTests(unittest.TestCase):
    def test_absent_asset_is_uploaded(self):
        self.assertEqual(attach.plan_asset('ab' * 32, None), 'upload')

    def test_identical_attached_asset_is_skipped(self):
        self.assertEqual(attach.plan_asset('ab' * 32, 'sha256:' + 'ab' * 32), 'skip')

    def test_attached_asset_with_different_bytes_fails(self):
        with self.assertRaises(attach.ReleaseError):
            attach.plan_asset('ab' * 32, 'sha256:' + 'cd' * 32)

    def test_attached_asset_without_digest_fails(self):
        with self.assertRaises(attach.ReleaseError):
            attach.plan_asset('ab' * 32, '')


if __name__ == '__main__':
    unittest.main()
