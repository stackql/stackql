import mistune

from typing import List, Tuple

import subprocess, os, sys, shutil, io

import json

import jinja2

from tabulate import tabulate

import argparse


"""
Intentions:

  - Support markdown parsing.
  - Support sequential markdown code block execution, leveraging [info strings](https://spec.commonmark.org/0.30/#info-string).
"""


def parse_args() -> argparse.Namespace:
    """
    Parse the arguments.
    """
    parser = argparse.ArgumentParser(description='Create a token.')
    parser.add_argument('--test-root', type=str, help='The test root.', default=os.path.join(_REPOSITORY_ROOT_PATH, 'docs', 'walkthroughs'))
    return parser.parse_args()


_REPOSITORY_ROOT_PATH = os.path.abspath(os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', '..'))


def eprint(*args, **kwargs):
    print(*args, file=sys.stderr, **kwargs)


_TEMPLATE_ENV = jinja2.Environment(
    trim_blocks=True,
    block_start_string='<%',
    block_end_string='%>',
    variable_start_string='<<',
    variable_end_string='>>'
)

class ASTNode(object):

    _STACKQL_SHELL_INVOCATION: str = 'stackql-shell'
    _BASH: str = 'bash'
    _SETUP: str = 'setup'
    _TEARDOWN: str = 'teardown'
    _BLOCK_CODE: str = 'block_code'
    _EXPECTATION: str = 'expectation'
    _STDOUT: str = 'stdout'
    _STDERR: str = 'stderr'
    _TABLE_CONTAINNS_DATA: str = 'table-contains-data'
    _REGEX: str = 'regex'

    def __init__(self, node: dict):
        self.node = node
        self.children = []
        self._local_vars = {}
        for candidate in self._get_annotations():
            split_list = candidate.split("=", 1)
            if len(split_list) == 2:
                self._local_vars[split_list[0]] = split_list[1]
        if 'children' in node:
            for child in node['children']:
                self.children.append(ASTNode(child))

    def get_type(self) -> str:
        return self.node.get('type', '')

    def get_text(self) -> str:
        return self.node.get('raw', '').strip()

    def is_executable(self) -> bool:
        return self.get_type() == self._BLOCK_CODE

    def is_expectation(self) -> bool:
        return self.get_type() == self._BLOCK_CODE and self._EXPECTATION in self._get_annotations()
    
    def get_expectation_metadata(self) -> dict:
        return self.node.get('attrs', {}).get('info', '').split(' ')
    
    def _has_annotation(self, annotation: str) -> List[str]:
        return annotation in self._get_annotations()
    
    def has_annotation(self, annotation: str) -> bool:
        return self._has_annotation(annotation)
    
    def _get_annotations(self) -> List[str]:
        return self.node.get('attrs', {}).get('info', '').split(' ')
    
    def is_stackql_shell_invocation(self) -> bool:
        return self._STACKQL_SHELL_INVOCATION in self._get_annotations()
    
    def is_bash(self) -> bool:
        return self._BASH in self._get_annotations()
    
    def is_setup(self) -> bool:
        return self._SETUP in self._get_annotations()
    
    def is_teardown(self) -> bool:
        return self._TEARDOWN in self._get_annotations()

    def get_execution_language(self) -> str:
        return self.node.get('lang', '')
    
    def expand(self) -> str:
        tpl = _TEMPLATE_ENV.from_string(self.get_text())
        rv = tpl.render(**self._local_vars)
        return rv

    def __str__(self):
        return json.dumps(self.node, indent=2)
    
    def __repr__(self):
        return self.__str__()
    

class MdAST(object):

    def __init__(self, ast: List[ASTNode], path: str):
        self._ast: List[ASTNode] = ast
        self._path: str = path

    def get_name(self) -> str:
        return os.path.basename(self._path)

    def get_ordered(self) -> List[ASTNode]:
        return self._ast
    
    def expand(self, node: ASTNode) -> str:
        return node.expand()
    
    def __str__(self):
        return json.dumps([node.node for node in self._ast], indent=2)
    
    def __repr__(self):
        return self.__str__()



class MdParser(object):

    def parse_markdown_file(self, file_path: str, lang=None) -> MdAST:
        markdown: mistune.Markdown = mistune.create_markdown(renderer='ast')
        with open(file_path, 'r') as f:
            txt = f.read()
        raw_list: List[dict] = markdown(txt)
        return MdAST([ASTNode(node) for node in raw_list], file_path)
    
class Expectation(object):

    _STDOUT_TABLE_CONTAINS_DATA: str = 'stdout-table-contains-data'
    _STDOUT_CONTAINS_ALL:        str = 'stdout-contains-all'

    def __init__(self, node: ASTNode):
        self._node: ASTNode = node

    def _contains_nonempty_table(self, s: str) -> bool:
        required_run_length: int = 5
        lines = s.split('\n')
        # print(f'lines: {lines}')
        if len(lines) < required_run_length:
            return False
        run_length: int = 0
        for line in lines:
            if line.startswith('|'):
                run_length += 1
                if run_length == required_run_length:
                    return True
            else:
                run_length = 0
        return False

    def passes_stdout(self, actual_stdout: str) -> bool:
        if self._node.has_annotation(self._STDOUT_TABLE_CONTAINS_DATA):
            eprint(f'running expectation check: {self._STDOUT_TABLE_CONTAINS_DATA}')
            return self._contains_nonempty_table(actual_stdout)
        if self._node.has_annotation(self._STDOUT_CONTAINS_ALL):
            eprint(f'running expectation check: {self._STDOUT_CONTAINS_ALL}')
            return self._node.get_text() in actual_stdout
        return True
    
    def passes_stderr(self, actual_stderr: str) -> bool:
        return True
    
    def __str__(self):
        return str(self._node)
    
    def __repr__(self):
        return self.__str__()

class WorkloadDTO(object):

    def __init__(
        self,
        name: str,
        setup: str,
        in_session: List[str],
        teardown: str,
        expectations: List[Expectation]
    ):
        self._name = name
        self._setup = setup
        self._in_session = in_session
        self._teardown = teardown
        self._expectations = expectations

    def get_name(self) -> str:
        return self._name

    def get_setup(self) -> List[str]:
        return self._setup
    
    def get_in_session(self) -> List[str]:
        return self._in_session
    
    def get_teardown(self) -> List[str]:
        return self._teardown
    
    def get_expectations(self) -> List[Expectation]:
        return self._expectations
    
    def __str__(self):
        return f'Setup: {self._setup}\nIn Session: {self._in_session}\nTeardown: {self._teardown}\nExpectations: {self._expectations}'
    
    def __repr__(self):
        return self.__str__()

class MdOrchestrator(object):

    def __init__(
        self,
        parser: MdParser, 
        max_setup_blocks: int = 1,
        max_invocations_blocks: int = 1,
        max_teardown_blocks: int = 1,
        setup_contains_shell_invocation: bool = True
    ):
        self._parser = parser
        self._max_setup_blocks = max_setup_blocks
        self._max_invocations_blocks = max_invocations_blocks
        self._max_teardown_blocks = max_teardown_blocks
        self._setup_contains_shell_invocation = setup_contains_shell_invocation

    def orchestrate(self, file_path: str) -> WorkloadDTO:
        setup_count: int = 0
        teardown_count: int = 0
        invocation_count: int = 0
        ast = self._parser.parse_markdown_file(file_path)
        # print(f'AST: {ast}')
        setup_str: str = f'cd {_REPOSITORY_ROOT_PATH};\n'
        in_session_commands: List[str] = []
        teardown_str: str = f'cd {_REPOSITORY_ROOT_PATH};\n'
        expectations: List[Expectation] = []
        for node in ast.get_ordered():
            if node.is_expectation():
                expectations.append(Expectation(node))
                continue
            elif node.is_executable():
                if node.is_setup():
                    if setup_count < self._max_setup_blocks:
                        setup_str += ast.expand(node)
                        setup_count += 1
                    else:
                        raise KeyError(f'Maximum setup blocks exceeded: {self._max_setup_blocks}')
                elif node.is_teardown():
                    if teardown_count < self._max_teardown_blocks:
                        teardown_str += ast.expand(node)
                        teardown_count += 1
                    else:
                        raise KeyError(f'Maximum teardown blocks exceeded: {self._max_teardown_blocks}')
                elif node.is_stackql_shell_invocation():
                    if invocation_count < self._max_invocations_blocks:
                        all_commands: str = ast.expand(node).split('\n\n')
                        in_session_commands += all_commands
                        invocation_count += 1
                    else:
                        raise KeyError(f'Maximum invocation blocks exceeded: {self._max_invocations_blocks}')
        return WorkloadDTO(ast.get_name(), setup_str, in_session_commands, teardown_str, expectations)

class WalkthroughResult:

  def __init__(
    self,
    name: str,
    stdout_str :str,
    stderr_str :str, 
    rc :int,
    passes_stdout: bool,
    passes_stderr: bool
  ) -> None:
    self.name: str = name 
    self.stdout :str = stdout_str
    self.stderr :str = stderr_str
    self.rc = rc
    self.passes_stdout_check = passes_stdout
    self.passes_stderr_check = passes_stderr

class SimpleRunner(object):

    def __init__(self, workload: WorkloadDTO):
        self._workload = workload

    def run(self) -> WalkthroughResult:
        bash_path = shutil.which('bash')
        pr: subprocess.Popen = subprocess.Popen(
            self._workload.get_setup(),
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            shell=True,
            executable=bash_path
        )
        for cmd in self._workload.get_in_session():
            pr.stdin.write(f"{cmd}\n".encode(sys.getdefaultencoding()))
            pr.stdin.flush()
        stdout_bytes, stderr_bytes = pr.communicate()

        stdout_str: str = stdout_bytes.decode(sys.getdefaultencoding())
        stderr_str: str = stderr_bytes.decode(sys.getdefaultencoding())

        fails_stdout: bool = False
        fails_stderr: bool = False

        for expectation in self._workload.get_expectations():
            passes_stdout: bool = expectation.passes_stdout(stdout_str)
            passes_stderr: bool = expectation.passes_stderr(stderr_str)
            if not passes_stdout:
                fails_stdout = True
            if not passes_stderr:
                fails_stderr = True
            print('---')
            print(f'Expectation: {expectation}')
            print('stdout:')
            print(stdout_str)
            print('')
            print('stderr:')
            print(stderr_str)
            print('')
            print(f'Passes stdout: {passes_stdout}')
            print(f'Passes stderr: {passes_stderr}')
            print('---')
        return WalkthroughResult(
            self._workload.get_name(),
            stdout_str,
            stderr_str,
            pr.returncode,
            not fails_stdout,
            not fails_stderr
        )

class AllWalkthroughsRunner(object):
    
    def __init__(self):
        md_parser = MdParser()
        self._orchestrator: MdOrchestrator = MdOrchestrator(md_parser)

    def run_all(self, walkthrough_inodes: List[str], recursive=True, skip_readme=True) -> List[WalkthroughResult]:
        results: List[WalkthroughResult] = []
        for inode_path in walkthrough_inodes:
            is_dir = os.path.isdir(inode_path)
            if is_dir:
                for root, dirs, files in os.walk(inode_path):
                    for file in files:
                        if skip_readme and file == 'README.md':
                            eprint(f'Skipping README.md')
                            continue
                        file_path = os.path.join(root, file)
                        print(f'File path: {file_path}')
                        workload: WorkloadDTO = self._orchestrator.orchestrate(file_path)
                        e2e: SimpleRunner = SimpleRunner(workload)
                        result = e2e.run()
                        results.append(result)
                continue
            is_file = os.path.isfile(inode_path)
            if is_file:
                if skip_readme and file == 'README.md':
                    eprint(f'Skipping README.md')
                    continue
                file_path = inode_path
                workload: WorkloadDTO = self._orchestrator.orchestrate(file_path)
                e2e: SimpleRunner = SimpleRunner(workload)
                result = e2e.run()
                results.append(result)
                continue
            raise FileNotFoundError(f'Path not tractable: {inode_path}')
        return results

def _collate_results(results: List[WalkthroughResult]) -> bool:
    failed: int = 0
    for result in results:
        if result.rc != 0 or not result.passes_stdout_check or not result.passes_stderr_check:
            failed += 1
    print(f'Failed test count: {failed}')
    print(tabulate([[result.name, result.rc, result.passes_stdout_check, result.passes_stderr_check] for result in results], headers=['Test Name', 'Return Code', 'Passes Stdout Checks', 'Passes Stderr Checks']))
    return failed == 0

def run_tests(root_dir: str) -> List[WalkthroughResult]:
    """
    Run all tests.
    A decent entry point for a test harness.

    :param root_dir: The root directory.

    :return: The results.
    """
    runner: AllWalkthroughsRunner = AllWalkthroughsRunner()
    results: List[WalkthroughResult] = runner.run_all([root_dir])
    return results


def _process_tests(root_dir: str) -> List[WalkthroughResult]:
    results: List[WalkthroughResult] = run_tests(root_dir)
    if _collate_results(results):
        print('All tests passed.')
        sys.exit(0)
    print('Some tests failed.')
    sys.exit(1)

def _main() -> None:
    args :argparse.Namespace = parse_args()
    _process_tests(args.test_root)


if __name__ == '__main__':
    _main()
