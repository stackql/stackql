import hashlib

from robot.api import SuiteVisitor


_SHARD_GROUP_TAG_PREFIX = 'shard-group:'


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
