
import base64
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
REGISTRY_ROOT_CANONICAL   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry'))
REGISTRY_ROOT_MOCKED   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry-mocked'))
REGISTRY_ROOT_DEPRECATED   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry-deprecated'))
STACKQL_EXE     = get_unix_path(os.path.join(REPOSITORY_ROOT, 'build', _exe_name))
_REGISTRY_NO_VERIFY_CFG    = { 
  "url": f"file://{get_unix_path(REGISTRY_ROOT_MOCKED)}",
  "localDocRoot": f"{get_unix_path(REGISTRY_ROOT_MOCKED)}",
  "useEmbedded": False,
  "verifyConfig": {
    "nopVerify": True 
  } 
}
_REGISTRY_CANONICAL_CFG    = { 
  "url": f"file://{get_unix_path(REGISTRY_ROOT_CANONICAL)}",
  "localDocRoot": f"{get_unix_path(REGISTRY_ROOT_CANONICAL)}",
  "verifyConfig": {
    "nopVerify": False 
  } 
}
_REGISTRY_DEPRECATED_CFG    = { 
  "url": f"file://{get_unix_path(REGISTRY_ROOT_DEPRECATED)}",
  "localDocRoot": f"{get_unix_path(REGISTRY_ROOT_DEPRECATED)}",
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
STACKQL_PG_SERVER_KEY_PATH   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_server_key.pem"))
STACKQL_PG_SERVER_CERT_PATH  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_server_cert.pem"))
STACKQL_PG_CLIENT_KEY_PATH   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_client_key.pem"))
STACKQL_PG_CLIENT_CERT_PATH  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_client_cert.pem"))
STACKQL_PG_RUBBISH_KEY_PATH  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_rubbish_key.pem"))
STACKQL_PG_RUBBISH_CERT_PATH :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_rubbish_cert.pem"))

with open(os.path.join(REPOSITORY_ROOT, 'test', 'server', 'mtls', 'credentials', 'pg_client_cert.pem'), 'rb') as f:
  _CLIENT_CERT_ENCODED :str = base64.b64encode(f.read()).decode('utf-8')

_mTLS_CFG :dict = { 
  "keyFilePath": STACKQL_PG_SERVER_KEY_PATH,
  "certFilePath": STACKQL_PG_SERVER_CERT_PATH,
  "clientCAs": [ 
    _CLIENT_CERT_ENCODED
  ] 
}

PG_SRV_MTLS_CFG_STR :str = json.dumps(_mTLS_CFG)

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'okta', 'api-key.txt'), 'r') as f:
    OKTA_SECRET_STR = f.read()

REGISTRY_NO_VERIFY_CFG_STR = json.dumps(_REGISTRY_NO_VERIFY_CFG)
REGISTRY_CANONICAL_CFG_STR = json.dumps(_REGISTRY_CANONICAL_CFG)
REGISTRY_DEPRECATED_CFG_STR = json.dumps(_REGISTRY_DEPRECATED_CFG)
AUTH_CFG_STR = json.dumps(_AUTH_CFG)
SHOW_PROVIDERS_STR = "show providers;"
SHOW_OKTA_SERVICES_FILTERED_STR  = "show services from okta like 'app%';"
SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR  = "show resources from okta.application like 'gr%';"
JSON_INIT_FILE_PATH = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-gcp-expectations.json')
MOCKSERVER_JAR = os.path.join(REPOSITORY_ROOT, 'test', 'downloads', 'mockserver-netty-5.12.0-shaded.jar')
MOCKSERVER_PORT = 1080

PG_SRV_PORT_MTLS = 5476
PG_SRV_PORT_UNENCRYPTED = 5477

PSQL_EXE :str = os.environ.get('PSQL_EXE', 'psql')

PSQL_CLIENT_HOST :str = "127.0.0.1"

PSQL_MTLS_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH} sslkey={STACKQL_PG_CLIENT_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"

PSQL_MTLS_INVALID_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_RUBBISH_CERT_PATH} sslkey={STACKQL_PG_RUBBISH_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"

PSQL_UNENCRYPTED_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_UNENCRYPTED} user=myuser dbname=mydatabase"

SELECT_CONTAINER_SUBNET_AGG_DESC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"
SELECT_CONTAINER_SUBNET_AGG_ASC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange asc;"

SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-desc.txt'))

SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-asc.txt'))

GET_IAM_POLICY_AGG_ASC_INPUT_FILE = os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'input', 'select-exec-dependent-org-iam-policy.iql')

GET_IAM_POLICY_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'cloudresourcemanager', 'select-exec-getiampolicy-agg.csv'))