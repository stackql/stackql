import argparse
import collections
import hashlib
import os
import re
import sys

from robot.api import ExecutionResult, SuiteVisitor, TestSuiteBuilder


_SHARD_GROUP_TAG_PREFIX = 'shard-group:'

# Report directories are named after the uploaded artifact, which encodes the
# (platform, db backend) group plus the shard coordinates of the run.
_REPORT_DIR_PATTERN = re.compile(
    r'^robot-output-(?P<group>.+)-shard-(?P<index>\d+)-of-(?P<count>\d+)$'
)

_OUTPUT_FILE_NAME = 'output.xml'

_NOT_RUN_STATUS = 'NOT RUN'


class ShardByTest(SuiteVisitor):
    """Select one deterministic shard while keeping tagged tests together."""

    def __init__(self, shard_index, shard_count):
        self.shard_index = int(shard_index)
        self.shard_count = int(shard_count)
        if self.shard_count < 1:
            raise ValueError('shard count must be positive')
        if not 1 <= self.shard_index <= self.shard_count:
            raise ValueError('shard index must be between 1 and the shard count')

    def start_suite(self, suite):
        suite.tests = [
            test for test in suite.tests
            if self._shard_for(self._shard_key(test)) == self.shard_index
        ]

    def end_suite(self, suite):
        suite.suites = [child for child in suite.suites if child.test_count > 0]

    def _shard_for(self, test_name):
        digest = hashlib.sha256(test_name.encode('utf-8')).digest()
        return int.from_bytes(digest[:8], 'big') % self.shard_count + 1

    def _shard_key(self, test):
        group_tags = [
            str(tag).lower() for tag in test.tags
            if str(tag).lower().startswith(_SHARD_GROUP_TAG_PREFIX)
        ]
        if len(group_tags) > 1:
            raise ValueError(f'test {test.full_name} has multiple shard group tags')
        return group_tags[0] if group_tags else test.full_name


def _expected_tests(source):
    """Every test in the source tree, parsed with no shard modifier applied."""
    suite = TestSuiteBuilder().build(source)
    return collections.Counter(test.full_name for test in suite.all_tests)


def _collect_report_dirs(reports_dir):
    """Map group name to {shard index: (shard count, report dir)}."""
    errors = []
    groups = collections.defaultdict(dict)
    if not os.path.isdir(reports_dir):
        return groups, [f'reports dir not found: {reports_dir}']
    for entry in sorted(os.listdir(reports_dir)):
        matched = _REPORT_DIR_PATTERN.match(entry)
        if matched is None:
            errors.append(f'unrecognised entry in reports dir: {entry}')
            continue
        index = int(matched.group('index'))
        if index in groups[matched.group('group')]:
            errors.append(f'duplicate report for shard {index}: {entry}')
            continue
        groups[matched.group('group')][index] = (
            int(matched.group('count')),
            os.path.join(reports_dir, entry),
        )
    return groups, errors


def _verify_group(group, shards, expected):
    """Assert that the shards of one group ran exactly the expected tests."""
    errors = []
    shard_counts = {count for count, _ in shards.values()}
    if len(shard_counts) != 1:
        return [f'{group}: inconsistent shard counts {sorted(shard_counts)}']
    shard_count = shard_counts.pop()
    absent_shards = sorted(set(range(1, shard_count + 1)) - set(shards))
    surplus_shards = sorted(set(shards) - set(range(1, shard_count + 1)))
    if absent_shards:
        errors.append(f'{group}: no report for shards {absent_shards} of {shard_count}')
    if surplus_shards:
        errors.append(f'{group}: reports for out of range shards {surplus_shards} of {shard_count}')
    observed = collections.Counter()
    for index in sorted(shards):
        output_path = os.path.join(shards[index][1], _OUTPUT_FILE_NAME)
        if not os.path.isfile(output_path):
            errors.append(f'{group}: shard {index} has no {_OUTPUT_FILE_NAME}')
            continue
        shard_tests = list(ExecutionResult(output_path).suite.all_tests)
        print(f'{group}: shard {index} of {shard_count} reported {len(shard_tests)} tests')
        for test in shard_tests:
            observed[test.full_name] += 1
            if test.status == _NOT_RUN_STATUS:
                errors.append(f'{group}: test not run: {test.full_name}')
    for name in sorted((expected - observed).elements()):
        errors.append(f'{group}: expected test absent from every shard: {name}')
    for name in sorted((observed - expected).elements()):
        errors.append(f'{group}: test reported more often than expected: {name}')
    expected_total = sum(expected.values())
    observed_total = sum(observed.values())
    print(f'{group}: {observed_total} tests reported, {expected_total} expected')
    if observed_total != expected_total:
        errors.append(
            f'{group}: total test count {observed_total} does not match expected {expected_total}'
        )
    return errors


def verify_shards(source, reports_dir, expected_groups):
    """Return the list of reasons the shard reports fail to cover the source.

    Each expected group must account for every test in the source exactly
    once across its shards, by name and by total count. An empty list
    means verified; anything that prevents verification is itself an error.
    """
    expected = _expected_tests(source)
    if not expected:
        return [f'no tests found in {source}']
    groups, errors = _collect_report_dirs(reports_dir)
    for group in sorted(set(expected_groups) - set(groups)):
        errors.append(f'{group}: no reports found')
    for group in sorted(set(groups) - set(expected_groups)):
        errors.append(f'{group}: reports found for a group that is not expected')
    for group in sorted(set(groups) & set(expected_groups)):
        errors.extend(_verify_group(group, groups[group], expected))
    return errors


def main():
    parser = argparse.ArgumentParser(
        description='Assert that sharded robot runs cover every test in the source.'
    )
    parser.add_argument('--source', required=True)
    parser.add_argument('--reports-dir', required=True)
    parser.add_argument('--expected-group', action='append', required=True)
    args = parser.parse_args()
    errors = verify_shards(args.source, args.reports_dir, args.expected_group)
    for error in errors:
        print(f'SHARD VERIFICATION FAILURE: {error}', file=sys.stderr)
    if errors:
        sys.exit(1)
    print('shard verification PASSED')


if __name__ == '__main__':
    main()
