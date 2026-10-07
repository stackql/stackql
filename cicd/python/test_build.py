import json
import os
import unittest
from unittest.mock import patch

import build


class DependencyInventoryBuildTests(unittest.TestCase):
    def test_cross_build_runs_generator_on_host_for_target(self):
        go_env = {
            'GOOS': 'darwin', 'GOARCH': 'arm64',
            'GOHOSTOS': 'windows', 'GOHOSTARCH': 'amd64',
        }
        with patch.dict(os.environ, {'GOOS': 'darwin', 'GOARCH': 'arm64'}):
            with patch.object(build.subprocess, 'check_output', return_value=json.dumps(go_env)):
                with patch.object(build.subprocess, 'call', return_value=0) as run:
                    self.assertEqual(build.generate_dependency_inventory(), 0)
                    command = run.call_args.args[0]
                    self.assertEqual(command[-2:], ['-targets', 'darwin/arm64'])
                    self.assertEqual(run.call_args.kwargs['env']['GOOS'], 'windows')
                    self.assertEqual(run.call_args.kwargs['env']['GOARCH'], 'amd64')
                    self.assertEqual(os.environ['GOOS'], 'darwin')
                    self.assertEqual(os.environ['GOARCH'], 'arm64')

    def test_generation_failure_prevents_binary_build(self):
        with patch.dict(os.environ, {}):
            with patch.object(build, 'generate_dependency_inventory', return_value=7):
                with patch.object(build.subprocess, 'call') as run:
                    self.assertEqual(build.build_stackql(False), 7)
                    run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
