
import json
import os

_exe_name = 'stackql'

if os.name == 'nt':
  _exe_name = _exe_name + '.exe'

REPOSITORY_ROOT = os.path.abspath(os.path.join(__file__, '..', '..', '..', '..'))

def get_output_from_local_file(fp :str) -> str:
  with open(os.path.join(REPOSITORY_ROOT, fp), 'r') as f:
    return f.read().strip()

def get_unix_path(pathStr :str) -> str:
  return pathStr.replace('\\', '/')


REPOSITORY_ROOT_UNIX = get_unix_path(REPOSITORY_ROOT)
REGISTRY_ROOT_MOCKED   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry-mocked'))
REGISTRY_ROOT_EMBEDDED   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry-embedded-decompressed'))
STACKQL_EXE     = get_unix_path(os.path.join(REPOSITORY_ROOT, 'build', _exe_name))
_REGISTRY_NO_VERIFY_CFG    = { 
  "url": f"file://{get_unix_path(REGISTRY_ROOT_MOCKED)}",
  "localDocRoot": f"{get_unix_path(REGISTRY_ROOT_MOCKED)}",
  "useEmbedded": False,
  "verifyConfig": {
    "nopVerify": True 
  } 
}
_REGISTRY_EMBEDDED_CFG    = { 
  "url": f"file://{get_unix_path(REGISTRY_ROOT_EMBEDDED)}",
  "localDocRoot": f"{get_unix_path(REGISTRY_ROOT_EMBEDDED)}",
  "useEmbedded": True,
  "verifyConfig": {
    "nopVerify": False 
  } 
}
_AUTH_CFG={ 
  "google": { 
    "credentialsfilepath": get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'google', 'functional-test-dummy-sa-key.json')),
    "type": "service_account"
  }, 
  "okta": { 
    "credentialsenvvar": "OKTA_SECRET_KEY",
    "type": "api_key" 
  } 
}

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'okta', 'api-key.txt'), 'r') as f:
    OKTA_SECRET_STR = f.read()

REGISTRY_NO_VERIFY_CFG_STR = json.dumps(_REGISTRY_NO_VERIFY_CFG)
REGISTRY_EMBEDDED_CFG_STR = json.dumps(_REGISTRY_EMBEDDED_CFG)
AUTH_CFG_STR = json.dumps(_AUTH_CFG)
SHOW_PROVIDERS_STR = "show providers;"
SHOW_OKTA_SERVICES_FILTERED_STR  = "show services from okta like 'app%';"
SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR  = "show resources from okta.application like 'gr%';"
JSON_INIT_FILE_PATH = os.path.join(REPOSITORY_ROOT, 'test', 'server', 'expectations', 'static-gcp-expectations.json')
MOCKSERVER_JAR = os.path.join(REPOSITORY_ROOT, 'test', 'downloads', 'mockserver-netty-5.12.0-shaded.jar')


SELECT_CONTAINER_SUBNET_AGG_DESC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"
SELECT_CONTAINER_SUBNET_AGG_ASC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange asc;"

SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-desc.txt'))

SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-asc.txt'))

GET_IAM_POLICY_AGG_ASC_INPUT_FILE = os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'input', 'select-exec-dependent-org-iam-policy.iql')

GET_IAM_POLICY_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'cloudresourcemanager', 'select-exec-getiampolicy-agg.csv'))