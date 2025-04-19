
import argparse
import copy
import glob
import json
import os
import re
import subprocess
import sys

from enum import Enum
from typing import AnyStr, Callable, Iterable, TextIO, Tuple, TypeVar, List

StringOrBytes = TypeVar('StringOrBytes', bytes, str)

TEST_ERRORS_COUNT :int = 0
TESTS_IN_ERROR :List[str] = []
TESTS_SUCCEEDED :List[str] = []

class TestStatus(Enum):
    success = 1
    failed  = 2


CURDIR :str = os.path.dirname(os.path.realpath(__file__))
TEST_ROOT_DIR :str = os.path.abspath(os.path.join(CURDIR, '..', '..'))
REPOSITORY_ROOT_DIR :str = os.path.abspath(os.path.join(CURDIR, '..', '..', '..'))
PROVIDER_REGISTRY_ROOT_DIR :str = os.path.abspath(os.path.join(TEST_ROOT_DIR, 'registry')).replace(os.sep, '/')
TEST_GENERATOR_DEFINITIONS_ROOT :str = os.path.join(TEST_ROOT_DIR, 'test-generators')
TEST_GENERATOR_ALWAYS_ROOT :str = os.path.join(TEST_GENERATOR_DEFINITIONS_ROOT, 'always')
TEST_GENERATOR_ASSETS_ROOT :str = os.path.join(TEST_ROOT_DIR, 'assets')
DEFAULT_RUN_DIR :str = os.path.abspath(os.path.join(REPOSITORY_ROOT_DIR, 'build'))
DEFAULT_APP_DIR :str = os.path.abspath(os.path.join(TEST_ROOT_DIR, '.stackql'))
DEFAULT_CONFIG_FILE :str = os.path.abspath(os.path.join(TEST_ROOT_DIR, '.stackqlrc'))
DEFAULT_DB_FILE :str = os.path.abspath(os.path.join(TEST_ROOT_DIR, 'db/tmp/python-tests-tmp-db.sqlite')).replace(os.sep, '/')
DEFAULT_REGISTRY_CFG :str = f'{{ "url": "file://{PROVIDER_REGISTRY_ROOT_DIR}", "localDocRoot": "{PROVIDER_REGISTRY_ROOT_DIR}",  "useEmbedded": false, "verifyConfig": {{ "nopVerify": true }} }}'
DEFAULT_EXECUTABLE :str = 'stackql'

DEFAULT_SQL_BACKEND_CONFIG : str = f'{{ "dsn": "{DEFAULT_DB_FILE}"  }}'

TEST_COUNT :int = 0
DEFAULT_TEST_NAME_TEMPLATE :str = 'TEST #{}'

EMPTY_PATTERN    :str = '^$'
JSON_ARR_PATTERN :str = '^\[.*\]$'
JSON_OBJ_PATTERN :str = '^{.*}$'



parser = argparse.ArgumentParser(description='Process some test config.')
parser.add_argument(
    '--rundir', 
    type=str,
    default=DEFAULT_RUN_DIR,
    help='directory containing executable'
)
parser.add_argument(
    '--appdir',
    type=str,
    default=DEFAULT_APP_DIR,
    help='directory containing config and cache'
)
parser.add_argument(
    '--configfile',
    type=str,
    default=DEFAULT_CONFIG_FILE,
    help='directory containing config and cache'
)
parser.add_argument(
    '--executable',
    type=str,
    default=DEFAULT_EXECUTABLE,
    help='name of executable file (no directory / path)'
)
parser.add_argument(
    '--additionalintegrationtestdir',
    type=str,
    default='',
    help='opt-in path to additional directory containing only test configs; can be used to run live integration tests.'
)
parser.add_argument(
    '--loglevel',
    type=str,
    default=os.environ.get('STACKQL_TEST_LOG_LEVEL', 'warn'),
    help='log level'
)
parser.add_argument(
    '--testgoogleproject',
    type=str,
    default=os.environ.get('STACKQL_TEST_GOOGLE_PROJECT', 'lab-kr-network-01'),
    help='google project to use in testing'
)
parser.add_argument(
    '--testgooglezone',
    type=str,
    default=os.environ.get('STACKQL_TEST_GOOGLE_ZONE', 'australia-southeast1-b'),
    help='google zone to use in testing'
)
parser.add_argument(
    '--verbosetesting',
    type=bool,
    default=os.environ.get('STACKQL_TEST_VERBOSE', False),
    help='enable verbose outputs for tests'
)
parser.add_argument(
    '--sqlbackend',
    type=str,
    default=DEFAULT_SQL_BACKEND_CONFIG,
    help='db file to use as starting point'
)
parser.add_argument(
    '--registry',
    type=str,
    default=DEFAULT_REGISTRY_CFG,
    help='registry config'
)

args = parser.parse_args()
executable_path = f'{args.rundir}/{args.executable}'

INVOCATION_BASE_ARGS = [
    executable_path,
    f'--configfile={args.configfile}',
    '--offline',
    # '--provider=google',
    f'--approot={args.appdir}',
    f'--loglevel={args.loglevel}',
    f'--sqlBackend={args.sqlbackend}',
    f'--registry={args.registry}'
]

INVOCATION_SIMPLE_ARGS = INVOCATION_BASE_ARGS + [ 'exec' ]

COMPUTE_RSC_STR = "compute"

GOOGLE_PROV_STR = "google"

BASIC_USE_STMT = f"USE {GOOGLE_PROV_STR};"

SHOW_PROVIDERS_STMT = "SHOW PROVIDERS;"

SHOW_SERVICES_STMT = f"SHOW SERVICES FROM {GOOGLE_PROV_STR};"

SHOW_RESOURCES_STMT = f"SHOW RESOURCES FROM {GOOGLE_PROV_STR}.{COMPUTE_RSC_STR};"

SHOW_ALT_RESOURCES_STMT = f"{BASIC_USE_STMT}; SHOW RESOURCES from {GOOGLE_PROV_STR}.compute;"

SHOW_EXTENDED_PROVIDERS_STMT = "SHOW EXTENDED PROVIDERS;"

SHOW_EXTENDED_SERVICES_STMT = f"SHOW EXTENDED SERVICES FROM {GOOGLE_PROV_STR};"

SHOW_EXTENDED_RESOURCES_STMT = f"SHOW EXTENDED RESOURCES FROM {GOOGLE_PROV_STR}.compute;"

SHOW_ALT_EXTENDED_RESOURCES_STMT = f"{BASIC_USE_STMT}; SHOW EXTENDED RESOURCES FROM {GOOGLE_PROV_STR}.compute;"

DESCRIBE_RESOURCE_STMT = f"DESCRIBE {GOOGLE_PROV_STR}.{COMPUTE_RSC_STR}.instances;"

DESCRIBE_EXTENDED_RESOURCE_STMT = f"DESCRIBE EXTENDED {GOOGLE_PROV_STR}.{COMPUTE_RSC_STR}.instances;"

def print_prez_layer(message :StringOrBytes, file :TextIO=sys.stdout):
    print('', file=file)
    print(message, file=file)
    print('', file=file)

def summary_print_prez_layer(messages :Iterable[StringOrBytes], file :TextIO=sys.stdout):
    print_prez_layer("#" * 24)
    for message in messages:
        print_prez_layer(message)
    print_prez_layer("#" * 24)


def print_verbose_outputs(messages :Iterable[StringOrBytes], file :TextIO=sys.stdout):
    for message in messages:
        print_prez_layer(message, file=file)

def get_output_contents(filename) -> List[AnyStr]:
    with open(filename, 'rt', encoding='utf-8') as f:
        return [ line for line in f.readlines() ]


def test_completion_msg(*args, **kwargs):
    global TEST_COUNT
    print_prez_layer('TEST ENDED: ' + kwargs.get('name', DEFAULT_TEST_NAME_TEMPLATE.format(TEST_COUNT)), file=sys.stderr)


def handle_test_failure(*args, **kwargs):
    global TEST_ERRORS_COUNT
    global TESTS_IN_ERROR
    TEST_ERRORS_COUNT += 1
    TESTS_IN_ERROR.append(kwargs.get('name', 'nameless test'))


def test_presentation(test_callable):
    def inner(*args, **kwargs):
        global TEST_COUNT
        global TESTS_SUCCEEDED
        TEST_COUNT += 1
        print_prez_layer('BEGINNING TEST: ' + kwargs.get('name', DEFAULT_TEST_NAME_TEMPLATE.format(TEST_COUNT)), file=sys.stderr)
        test_status, outputs, returncode = test_callable(*args, **kwargs)
        if returncode != 0 or test_status == TestStatus.failed:
            handle_test_failure(*args, **kwargs)
            for output in outputs:
                for o in output.splitlines():
                    print_prez_layer(o)
            test_completion_msg(*args, **kwargs)
            return
        expected = kwargs.get('expected')
        output_for_test :List[AnyStr] = []
        if (not kwargs.get('test_output_file') or kwargs.get('test_output_file') == 'stdout'):
            for output in outputs:
                for o in output.splitlines():
                    output_for_test.append(o)
        else:
            output_for_test = get_output_contents(kwargs.get('test_output_file'))
        if test_status == TestStatus.success and expected:
            test_has_failed = False
            j = 0
            for i in range(len(output_for_test)):
                if re.match(expected[j], output_for_test[i]):
                    j += 1
                    if j == len(expected):
                        break
            if j == len(expected):
                TESTS_SUCCEEDED.append(kwargs.get('name', 'nameless test'))
                print_prez_layer('assertion succeeded', file=sys.stderr)
            else:
                handle_test_failure(*args, **kwargs)
                print_prez_layer('assertion #{} failed: "{}" unmatched in any output!!!'.format(j, expected[j]), file=sys.stderr)
                print_prez_layer(f'failure output: {" ".join(output_for_test)}')
        else:
            TESTS_SUCCEEDED.append(kwargs.get('name', 'nameless test'))
            print_prez_layer('no assertion test succeeded', file=sys.stderr)
        if kwargs.get('verbose'):
            print_verbose_outputs(outputs, file=sys.stderr)
        test_completion_msg(*args, **kwargs)
    return inner


@test_presentation
def integration_test(*args, **kwargs) -> Tuple[TestStatus, Iterable[StringOrBytes], int]:
    try:
        child = subprocess.Popen([
                *args
            ],
            stdout=subprocess.PIPE, 
            stderr=subprocess.PIPE,
            encoding='utf-8'
        )
        return (
            TestStatus.success, 
            child.communicate(),
            child.returncode
        )
    except Exception as e:
        print_prez_layer('Exception caught: ' +  str(e), file=sys.stderr)
        return (TestStatus.failed, ('Exception caught', str(e)), 0)


def standard_integration_tests(*args, **kwargs) -> Iterable[Tuple[TestStatus, Iterable[StringOrBytes]]]:
    ret_val = []
    ret_val += [
        integration_test(
            *INVOCATION_SIMPLE_ARGS,
            '-o=json',
            *args,
            **kwargs
        ) 
    ]
    return ret_val


def simple_test_suite():
    """
    Integration test suite to be called on all builds
    """
    standard_integration_tests(
        BASIC_USE_STMT,
        verbose = args.verbosetesting,
        name = 'Verbose simple USE test + assertion'
    )
    standard_integration_tests(
        SHOW_PROVIDERS_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple SHOW PROVIDERS test + assertion'
    )
    standard_integration_tests(
        SHOW_SERVICES_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple SHOW SERVICES test + assertion'
    )
    standard_integration_tests(
        SHOW_RESOURCES_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple SHOW RESOURCES test + assertion'
    )
    standard_integration_tests(
        SHOW_ALT_RESOURCES_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple alt SHOW RESOURCES test + assertion'
    )
    standard_integration_tests(
        SHOW_EXTENDED_PROVIDERS_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple SHOW EXTENDED PROVIDERS test + assertion'
    )
    standard_integration_tests(
        SHOW_EXTENDED_SERVICES_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple SHOW EXTENDED SERVICES test + assertion'
    )
    standard_integration_tests(
        SHOW_EXTENDED_RESOURCES_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple SHOW EXTENDED RESOURCES test + assertion'
    )
    standard_integration_tests(
        SHOW_ALT_EXTENDED_RESOURCES_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple SHOW EXTENDED RESOURCES test + assertion'
    )
    standard_integration_tests(
        DESCRIBE_RESOURCE_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple DESCRIBE RESOURCE test + assertion'
    )
    standard_integration_tests(
        DESCRIBE_EXTENDED_RESOURCE_STMT,
        expected = [ JSON_ARR_PATTERN ],
        verbose = args.verbosetesting,
        name = 'Verbose simple DESCRIBE EXTENDED RESOURCE test + assertion'
    )
    _DESCRIBE_RESOURCE_INPUT_FILE = os.path.join(TEST_ROOT_DIR, 'inputs', 'describe-google-compute.iql')
    _DESCRIBE_RESOURCE_OUTPUT_FILE = os.path.join(TEST_ROOT_DIR, 'outputs', 'describe-google-compute.json')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=json',
        '--provider=google',
        '-i={}'.format(_DESCRIBE_RESOURCE_INPUT_FILE),
        '-f={}'.format(_DESCRIBE_RESOURCE_OUTPUT_FILE),
        'exec',
        expected = [ JSON_ARR_PATTERN ],
        test_output_file = _DESCRIBE_RESOURCE_OUTPUT_FILE,
        verbose = args.verbosetesting,
        name = 'Verbose output file based DESCRIBE RESOURCE test + assertion'
    )
    _SHOW_PROVIDERS_INPUT_FILE = os.path.join(TEST_ROOT_DIR, 'inputs', 'show-providers.iql')
    _SHOW_PROVIDERS_OUTPUT_FILE = os.path.join(TEST_ROOT_DIR, 'outputs', 'show-providers.json')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=json',
        '-i={}'.format(_SHOW_PROVIDERS_INPUT_FILE),
        '-f={}'.format(_SHOW_PROVIDERS_OUTPUT_FILE),
        'exec',
        expected = [ JSON_ARR_PATTERN ],
        test_output_file = _SHOW_PROVIDERS_OUTPUT_FILE,
        verbose = args.verbosetesting,
        name = 'Verbose output file based SHOW PROVIDERS test + assertion'
    )
    _SHOW_PROVIDERS_OUTPUT_FILE_CSV = os.path.join(TEST_ROOT_DIR, 'outputs', 'show-providers.csv')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=csv',
        '-i={}'.format(_SHOW_PROVIDERS_INPUT_FILE),
        '-f={}'.format(_SHOW_PROVIDERS_OUTPUT_FILE_CSV),
        'exec',
        expected = [ 'name' ],
        test_output_file = _SHOW_PROVIDERS_OUTPUT_FILE_CSV,
        verbose = args.verbosetesting,
        name = 'Verbose output CSV file based SHOW PROVIDERS test + assertion'
    )
    _SHOW_PROVIDERS_OUTPUT_FILE_ALT_CSV = os.path.join(TEST_ROOT_DIR, 'outputs', 'show-providers-alt.csv')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=csv',
        '-d=;',
        '-i={}'.format(_SHOW_PROVIDERS_INPUT_FILE),
        '-f={}'.format(_SHOW_PROVIDERS_OUTPUT_FILE_ALT_CSV),
        'exec',
        expected = [ 'name', 'google' ],
        test_output_file = _SHOW_PROVIDERS_OUTPUT_FILE_ALT_CSV,
        verbose = args.verbosetesting,
        name = 'Verbose output alt-deimited CSV file based SHOW PROVIDERS test + assertion'
    )
    _SHOW_EXTENDED_SERVICES_OUTPUT_FILE_CSV = os.path.join(TEST_ROOT_DIR, 'outputs', 'show-services.csv')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=csv',
        '-f={}'.format(_SHOW_EXTENDED_SERVICES_OUTPUT_FILE_CSV),
        'exec',
        SHOW_EXTENDED_SERVICES_STMT,
        expected = [ 'id,name,title,description' ],
        test_output_file = _SHOW_EXTENDED_SERVICES_OUTPUT_FILE_CSV,
        verbose = args.verbosetesting,
        name = 'Verbose output CSV file based SHOW EXTENDED SERVICES test + assertion'
    )
    _SHOW_EXTENDED_SERVICES_OUTPUT_FILE_ALT_CSV = os.path.join(TEST_ROOT_DIR, 'outputs', 'show-services-alt.csv')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=csv',
        '-d=;',
        '-f={}'.format(_SHOW_EXTENDED_SERVICES_OUTPUT_FILE_ALT_CSV),
        'exec',
        SHOW_EXTENDED_SERVICES_STMT,
        expected = [ 'id;name;title;description' ],
        test_output_file = _SHOW_EXTENDED_SERVICES_OUTPUT_FILE_ALT_CSV,
        verbose = args.verbosetesting,
        name = 'Verbose output alt-deimited CSV file based SHOW EXTENDED SERVICES test + assertion'
    )
    _SHOW_EXTENDED_SERVICES_FILTERED_OUTPUT_FILE_ALT_CSV = os.path.join(TEST_ROOT_DIR, 'outputs', 'show-services-filtered-alt.csv')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=csv',
        '-d=;',
        '-f={}'.format(_SHOW_EXTENDED_SERVICES_OUTPUT_FILE_ALT_CSV),
        'exec',
        "show extended services from google where description like 'Provides natural language%' and version = 'v1'",
        expected = [ 'id;name;title;description;version;preferred', 'language:v1;.*' ],
        test_output_file = _SHOW_EXTENDED_SERVICES_OUTPUT_FILE_ALT_CSV,
        verbose = args.verbosetesting,
        name = 'Verbose output, filtered, alt-deimited CSV file based SHOW EXTENDED SERVICES test + assertion'
    )
    _SHOW_EXTENDED_RESOURCES_FILTERED_OUTPUT_FILE_ALT_CSV = os.path.join(TEST_ROOT_DIR, 'outputs', 'show-resourcces-filtered-alt.csv')
    integration_test(
        *INVOCATION_BASE_ARGS,
        '-o=csv',
        '-d=;',
        '-f={}'.format(_SHOW_EXTENDED_RESOURCES_FILTERED_OUTPUT_FILE_ALT_CSV),
        'exec',
        "show extended resources from google.compute where name = 'resourcePolicies' and id like '%.resourcePol%';",
        expected = [ 'name;id;description', 'resourcePolicies;.*' ],
        test_output_file = _SHOW_EXTENDED_RESOURCES_FILTERED_OUTPUT_FILE_ALT_CSV,
        verbose = args.verbosetesting,
        name = 'Verbose output, filtered, alt-deimited CSV file based SHOW EXTENDED RESOURCES test + assertion'
    )


def run_integration_test_generator(generator :dict, parent_test_file_path :str, index :int):
    invocation_base_args = copy.deepcopy(INVOCATION_BASE_ARGS)
    if generator.get("testwitoutapicalls"):
      invocation_base_args.append(f'--testwitoutapicalls={generator.get("testwitoutapicalls")}')
    if generator.get("credentialsfilepath"):
      kp :str = generator.get("credentialsfilepath") if generator.get("credentialsfilepath").startswith("/") else os.path.abspath(os.path.join(TEST_GENERATOR_ASSETS_ROOT, generator.get("credentialsfilepath")))
      invocation_base_args.append(f'--credentialsfilepath={kp}')
    if generator.get("iqldata"):
      idt :str = generator.get("iqldata") if generator.get("iqldata").startswith("/") else os.path.abspath(os.path.join(TEST_GENERATOR_ASSETS_ROOT, generator.get("iqldata")))
      invocation_base_args.append(f'--iqldata={idt}')
    if generator.get("headless"):
      invocation_base_args.append("-H")
    if generator.get("dry_run"):
      invocation_base_args.append("--dryrun")
    output_format :str = generator.get("output", "csv")
    instruction :str = generator.get("instruction", "exec")
    fallback_name :str = f'{parent_test_file_path}_{index}'
    output_file :str = 'stdout' if generator.get("output_file") == 'stdout' else os.path.join(TEST_GENERATOR_ASSETS_ROOT, generator.get("output_file", fallback_name + '.' + output_format))
    if generator.get("query"):
      integration_test(
          *invocation_base_args,
          f'-o={output_format}',
          f'-d={generator.get("delimiter", ",")}',
          f'-f={output_file}',
          instruction,
          f'{generator.get("query", "")}',
          expected = generator.get("expected", []),
          test_output_file = output_file,
          verbose = not(not(generator.get("verbose"))),
          name = f'{parent_test_file_path}: {generator.get("name", "#" + str(index))}'
      )
    elif generator.get("input_file"):
        input_file :str = generator.get("input_file") if generator.get("input_file").startswith("/") else os.path.join(TEST_GENERATOR_ASSETS_ROOT, generator.get("input_file"))
        if generator.get("external_tmpl_ctx_file"):
            tmpl_ctx_file :str = os.path.join(TEST_GENERATOR_ASSETS_ROOT, generator.get("external_tmpl_ctx_file"))
            integration_test(
                *invocation_base_args,
                f'-o={output_format}',
                f'-d={generator.get("delimiter", ",")}',
                f'-f={output_file}',
                instruction,
                f'-i={input_file}',
                f'-q={tmpl_ctx_file}',
                expected = generator.get("expected", []),
                test_output_file = output_file,
                verbose = not(not(generator.get("verbose"))),
                name = f'{parent_test_file_path}: {generator.get("name", "#" + str(index))}'
            )
        else:
            integration_test(
                *invocation_base_args,
                f'-o={output_format}',
                f'-d={generator.get("delimiter", ",")}',
                f'-f={output_file}',
                instruction,
                f'-i={input_file}',
                expected = generator.get("expected", []),
                test_output_file = output_file,
                verbose = not(not(generator.get("verbose"))),
                name = f'{parent_test_file_path}: {generator.get("name", "#" + str(index))}'
            )
    else:
      exit(1)


def run_integration_test_dir(dirpath :str):
    for test_file in os.listdir(dirpath):
        with open(os.path.join(dirpath, test_file), 'rt', encoding='utf-8') as f:
            test_definitions :dict = json.load(f)
        integration_tests_config :List[dict] = test_definitions.get("integration_tests", None)
        if integration_tests_config:
            i = 0
            for test_config in integration_tests_config:
                run_integration_test_generator(test_config, test_file, i)
                i += 1


def run_integration_test_generators(additional_test_dir :str):
    run_integration_test_dir(TEST_GENERATOR_ALWAYS_ROOT)
    if additional_test_dir != '':
        add_dir :str = additional_test_dir if additional_test_dir.startswith("/") else os.path.abspath(os.path.join(TEST_GENERATOR_DEFINITIONS_ROOT, additional_test_dir))
        run_integration_test_dir(add_dir)


def prepare_output_dirs():
    fileList = glob.glob(f'{TEST_GENERATOR_ASSETS_ROOT}/*.csv', recursive=True)
    for filePath in fileList:
        try:
            os.remove(filePath)
        except OSError:
            print(f"Error while deleting file {filePath}")

def main(args):
    prepare_output_dirs()
    simple_test_suite()
    run_integration_test_generators(args.additionalintegrationtestdir)
    if TEST_ERRORS_COUNT > 0:
        messages = [
            'TEST SUMMARY',
            'The following tests succeeded:',
            f'Test suite FAILED with {TEST_ERRORS_COUNT} failing tests out of {TEST_COUNT} total',
            'The following tests FAILED:'
        ]
        messages[2:2] = [ f'    ++ {msg}' for msg in TESTS_SUCCEEDED ]
        messages += [ f'    -- {msg}' for msg in TESTS_IN_ERROR ]
        summary_print_prez_layer(
            messages
        )
        exit(1)
    else:
        messages = [
            'TEST SUMMARY',
            'The following tests succeeded:',
            f'Test suite PASSED; all {TEST_COUNT} tests succeeded'
        ]
        messages[2:2] = [ f'    ++ {msg}' for msg in TESTS_SUCCEEDED ]
        summary_print_prez_layer(
            messages
        )


if __name__ == '__main__':
    main(args)