import unittest

import attach_build_artifacts as attach


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
