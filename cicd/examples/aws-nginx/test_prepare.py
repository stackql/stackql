"""Offline checks of executable image defaults and explicit pinning."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import Mock
from image_inputs import resolve_image, PARAMETER


class PrepareInputsTest(unittest.TestCase):
    def prepare(self, *extra):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            public = root / 'existing.pub'
            public.write_text('ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFixture offline-test\n')
            case = root / 'case'
            result = subprocess.run([
                sys.executable, str(Path(__file__).with_name('prepare.py')),
                '--case', str(case), '--region', 'ap-southeast-2', '--image-id', 'ami-00000000000000001', '--zone-a', 'ap-southeast-2a', '--zone-b', 'ap-southeast-2b',
                '--ssh-cidr', '203.0.113.7/32', '--public-key', str(public), *extra,
            ], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            return json.loads((case / 'inputs.json').read_text())

    def test_prepared_image_is_concrete(self):
        self.assertEqual(self.prepare()['image_id'], 'ami-00000000000000001')

    def test_default_resolves_once_before_launch(self):
        runner = Mock(return_value=Mock(stdout='ami-00000000000000002\n'))
        self.assertEqual(resolve_image(None, 'ap-southeast-2', runner), 'ami-00000000000000002')
        runner.assert_called_once()
        self.assertIn(PARAMETER, runner.call_args.args[0])
        self.assertIn('ap-southeast-2', runner.call_args.args[0])

    def test_pinned_image_never_looks_up_latest(self):
        runner = Mock()
        self.assertEqual(resolve_image('ami-00000000000000001', 'ap-southeast-2', runner), 'ami-00000000000000001')
        runner.assert_not_called()

    def test_moving_reference_is_rejected(self):
        with self.assertRaises(ValueError):
            resolve_image('resolve:ssm:' + PARAMETER, 'ap-southeast-2')

    def test_separate_deployments_do_not_recycle_client_tokens(self):
        first, second = self.prepare(), self.prepare()
        self.assertNotEqual(first['client_token'], second['client_token'])
        self.assertLessEqual(len(first['client_token']), 64)

    def test_existing_inputs_are_not_rotated(self):
        with tempfile.TemporaryDirectory() as temporary:
            case = Path(temporary)
            original = '{"image_id":"ami-00000000000000001","client_token":"persisted-token"}'
            (case / 'inputs.json').write_text(original)
            result = subprocess.run([
                sys.executable, str(Path(__file__).with_name('prepare.py')),
                '--case', str(case), '--region', 'ap-southeast-2',
                '--zone-a', 'ap-southeast-2a', '--zone-b', 'ap-southeast-2b',
                '--ssh-cidr', '203.0.113.7/32',
            ], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('case already prepared', result.stderr)
            self.assertEqual((case / 'inputs.json').read_text(), original)

    def test_explicit_image_can_be_pinned(self):
        image = 'ami-00000000000000001'
        self.assertEqual(self.prepare('--image-id', image)['image_id'], image)


if __name__ == '__main__':
    unittest.main()
