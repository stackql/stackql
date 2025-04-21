
import argparse
import json
import re

from typing import List, Optional

class _Tag(object):

    def __init__(
        self,
        raw_tag: str,
        is_integration: bool = False,
        is_release: bool = False,
        is_hotfix: bool = False,
        is_prerelease: bool = False,
        is_build: bool = False,
        is_invalid: bool = False,
        is_regression: bool = False,
        is_robot: bool = False,
        repository_shorthand: Optional[str] = None,
        run_type: Optional[str] = None,
        run_id: Optional[str] = None,
    ):
        self._raw_tag = raw_tag
        self._is_integration = is_integration
        self._is_release = is_release
        self._is_hotfix = is_hotfix
        self._is_prerelease = is_prerelease
        self._is_build = is_build
        self._is_invalid = is_invalid
        self._is_regression = is_regression
        self._is_robot = is_robot
        self._repository_shorthand = repository_shorthand if repository_shorthand else ''
        self._run_type = run_type if run_type else ''
        self._run_id = run_id if run_id else ''


    def is_robot(self) -> bool:
        return self._is_robot
    
    def is_regression(self) -> bool:
        return self._is_regression
    
    def get_run_id(self) -> str:
        return self._run_id
    
    def get_run_type(self) -> str:
        return self._run_type
    
    def get_repository_shorthand(self) -> str:
        return self._repository_shorthand
    
    def _to_dict(self) -> dict:
        return {
            'raw_tag': self._raw_tag,
            'is_integration': self._is_integration,
            'is_release': self._is_release,
            'is_hotfix': self._is_hotfix,
            'is_prerelease': self._is_prerelease,
            'is_build': self._is_build,
            'is_invalid': self._is_invalid,
            'is_regression': self._is_regression,
            'is_robot': self._is_robot,
            'repository_shorthand': self.get_repository_shorthand(),
            'run_type': self._run_type,
            'run_id': self._run_id,
        }
    
    def json(self) -> str:
        return json.dumps(self._to_dict())


class _TagParser(object):

    _BUILD_ONLY_TAG_PATTERN = re.compile(r'^build-(?P<descriptor>[^-]*)-(?P<seq>\d+).*$')
    _BUILD_RELEASE_TAG_PATTERN = re.compile(r'^build-release-(?P<descriptor>[^-]*)-(?P<seq>\d+).*$')
    _SCENARIO_ONLY_TAG_PATTERN = re.compile(r'^scenario-(?P<run_id>[^-]*)-(?P<run_type>[^-]*)-(?P<repository_shorthand>[^-]*)$')
    _ROBOT_TAG_PATTERN = re.compile(r'^robot-(?P<run_id>[^-]*)-(?P<run_type>[^-]*)-(?P<repository_shorthand>[^-]*)-(?P<seq>\d+)$')
    _ROBOT_ONLY_TAG_PATTERN = re.compile(r'^robot-(?P<descriptor>[^-]*)-(?P<seq>\d+).*$')
    _REGRESSION_ONLY_TAG_PATTERN = re.compile(r'^regression-(?P<descriptor>[^-]*)-(?P<seq>\d+).*$')

    def __init__(
        self,
        raw_tag: str,
        permitted_types: List[str]
    ):
        self._raw_tag = raw_tag
        self._permitted_types = permitted_types

    def _parse_build_tag(self) -> _Tag:
        match = self._BUILD_ONLY_TAG_PATTERN.match(self._raw_tag)
        if match:
            return _Tag(raw_tag=self._raw_tag, is_build=True)
        match = self._BUILD_RELEASE_TAG_PATTERN.match(self._raw_tag)
        if match:
            return _Tag(raw_tag=self._raw_tag, is_build=True)
        return None
    
    def _parse_scenario_tag(self) -> _Tag:
        match = self._SCENARIO_ONLY_TAG_PATTERN.match(self._raw_tag)
        run_id = match.group('run_id')
        run_type = match.group('run_type')
        repository_shorthand = match.group('repository_shorthand')
        if run_id and run_type:
            return _Tag(raw_tag=self._raw_tag, is_integration=True, run_id=run_id, run_type=run_type, repository_shorthand=repository_shorthand)
        return None
    
    def _parse_robot_tag(self) -> _Tag:
        match = self._ROBOT_TAG_PATTERN.match(self._raw_tag)
        if match:
            return _Tag(raw_tag=self._raw_tag, is_robot=True)
        match = self._ROBOT_ONLY_TAG_PATTERN.match(self._raw_tag)
        if match:
            return _Tag(raw_tag=self._raw_tag, is_robot=True)
        return None
    
    def _parse_regression_tag(self) -> _Tag:
        match = self._REGRESSION_ONLY_TAG_PATTERN.match(self._raw_tag)
        if match:
            return _Tag(raw_tag=self._raw_tag, is_regression=True)
        return None

    def parse(self) -> _Tag:
        if not self._permitted_types:
            raise ValueError('No permitted types specified')
        for t in self._permitted_types:
            if t == 'build':
                tag = self._parse_build_tag()
                if tag:
                    return tag
            elif t == 'scenario':
                tag = self._parse_scenario_tag()
                if tag:
                    return tag
            elif t == 'robot':
                tag = self._parse_robot_tag()
                if tag:
                    return tag
            elif t == 'regression':
                tag = self._parse_regression_tag()
                if tag:
                    return tag
        raise ValueError(f'Raw tag: "{self._raw_tag}" is incompatible with permitted types: {" ".join(self._permitted_types)}')

def _parse_args() -> argparse.Namespace:
    """
    Parse the arguments.
    """
    parser = argparse.ArgumentParser(description='Handle and interpret git tags.')
    parser.add_argument('tag', help='The tag to parse', type=str)
    parser.add_argument('--parse-registry-tag', help='Opt-in to parse a registry tag', action=argparse.BooleanOptionalAction)
    parser.add_argument('--parse-scenario-tag', help='Opt-in to parse a scenario tag', action=argparse.BooleanOptionalAction)
    return parser.parse_args()

def main():
    args = _parse_args()
    if args.parse_registry_tag:
        tag_parser = _TagParser(args.tag, ['robot', 'regression'])
        tag = tag_parser.parse()
        if tag.is_robot():
            print(tag.json())
            return
        elif tag.is_regression():
            print(tag.json())
            return
    if args.parse_scenario_tag:
        tag_parser = _TagParser(args.tag, ['scenario'])
        tag = tag_parser.parse()
        if tag.get_run_id() and tag.get_run_type():
            print(tag.json())
            return
        else:
            raise ValueError(f'Invalid scenario tag inferred: {tag.json()}')
    raise ValueError('No action specified')

if __name__ == '__main__':
    main()

