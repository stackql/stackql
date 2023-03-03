#!/usr/bin/env python3


import argparse
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


def run_robot_mocked_functional_tests_stackql(should_run_docker_external_tests :bool, concurrency_limit :int) -> int:
    should_run_docker_external_tests_str = 'true' if should_run_docker_external_tests else 'false'
    return subprocess.call(
        'robot '
        f'--variable SHOULD_RUN_DOCKER_EXTERNAL_TESTS:{should_run_docker_external_tests_str} '
        f'--variable CONCURRENCY_LIMIT:{concurrency_limit} ' 
        '-d test/robot/functional '
        'test/robot/functional',
        shell=True
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--verbose', action='store_true')
    parser.add_argument('--build', action='store_true')
    parser.add_argument('--test', action='store_true')
    parser.add_argument('--robot-test', action='store_true')
    parser.add_argument('--robot-test-aggressively-concurrent', action='store_true')
    parser.add_argument('--robot-test-docker-external', action='store_true')
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
        ret_code = run_robot_mocked_functional_tests_stackql(args.robot_test_docker_external, -1 if args.robot_test_aggressively_concurrent else 1)
        if ret_code != 0:
            exit(ret_code)
    exit(ret_code)


if __name__ == '__main__':
    main()
