#!/usr/bin/env python3


import argparse
import json
import os
import subprocess


def build_stackql(verbose :bool) -> int:
    os.environ['BUILDMAJORVERSION'] = os.environ.get('BUILDMAJORVERSION', '1')
    os.environ['BUILDMINORVERSION'] = os.environ.get('BUILDMINORVERSION', '1')
    os.environ['BUILDPATCHVERSION'] = os.environ.get('BUILDPATCHVERSION', '1')
    os.environ['CGO_ENABLED'] = os.environ.get('CGO_ENABLED', '1')
    return subprocess.call(
        f'go build {"-x -v" if verbose else ""} --tags "json1 sqleanall" -ldflags "-X github.com/stackql/stackql/internal/stackql/cmd.BuildMajorVersion={os.environ.get("BUILDMAJORVERSION")} '
        f'-X github.com/stackql/stackql/internal/stackql/cmd.BuildMinorVersion={os.environ.get("BUILDMINORVERSION")} '
        f'-X github.com/stackql/stackql/internal/stackql/cmd.BuildPatchVersion={os.environ.get("BUILDPATCHVERSION")} '
        f'-X github.com/stackql/stackql/internal/stackql/cmd.BuildCommitSHA={os.environ.get("BUILDCOMMITSHA", "")} '
        f'-X github.com/stackql/stackql/internal/stackql/cmd.BuildShortCommitSHA={os.environ.get("BUILDCOMMITSHA", "")[0:7 or None]} '
        f"-X 'github.com/stackql/stackql/internal/stackql/cmd.BuildDate={os.environ.get('BUILDDATE', '')}' "
        f"-X 'stackql/internal/stackql/planbuilder.PlanCacheEnabled={os.environ.get('PLANCACHEENABLED', '')}' "
        f'-X github.com/stackql/stackql/internal/stackql/cmd.BuildPlatform={os.environ.get("BUILDPLATFORM", "")}" '
        '-o build/ ./stackql',
        shell=True
    )


def unit_test_stackql(verbose :bool) -> int:
    return subprocess.call(
        f'go test -timeout 1200s {"-v" if verbose else ""} --tags "json1 sqleanall"  ./...',
        shell=True
    )

def sanitise_val(val :any) -> str:
    if isinstance(val, bool):
        return str(val).lower()
    return str(val)


def run_robot_mocked_functional_tests_stackql(*args, **kwargs) -> int:
    variables = ' '.join([f'--variable {key}:{sanitise_val(value)} ' for key, value in kwargs.get("variables", {}).items() ])
    return subprocess.call(
        'robot '
        f'{variables} ' 
        '-d test/robot/functional '
        'test/robot/functional',
        shell=True
    )

def run_robot_integration_tests_stackql(*args, **kwargs) -> int:
    variables = ' '.join([f'--variable {key}:{sanitise_val(value)} ' for key, value in kwargs.get("variables", {}).items()])
    return subprocess.call(
        'robot '
        f'{variables} ' 
        '-d test/robot/integration '
        'test/robot/integration',
        shell=True
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--verbose', action='store_true')
    parser.add_argument('--build', action='store_true')
    parser.add_argument('--test', action='store_true')
    parser.add_argument('--robot-test', action='store_true')
    parser.add_argument('--robot-test-integration', action='store_true')
    parser.add_argument('--config', type=json.loads, default={})
    args = parser.parse_args()
    ret_code = 0
    if args.build:
        ret_code = build_stackql(args.verbose)
        if ret_code != 0:
            exit(ret_code)
    if args.test:
        ret_code = unit_test_stackql(args.verbose)
        if ret_code != 0:
            exit(ret_code)
    if args.robot_test:
        ret_code = run_robot_mocked_functional_tests_stackql(**args.config)
        if ret_code != 0:
            exit(ret_code)
    if args.robot_test_integration:
        ret_code = run_robot_integration_tests_stackql(**args.config)
        if ret_code != 0:
            exit(ret_code)
    exit(ret_code)


if __name__ == '__main__':
    main()
