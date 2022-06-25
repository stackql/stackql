
import base64
import json
import os

_exe_name = 'stackql'

IS_WINDOWS = '0'
if os.name == 'nt':
  IS_WINDOWS = '1'
  _exe_name = _exe_name + '.exe'

REPOSITORY_ROOT = os.path.abspath(os.path.join(__file__, '..', '..', '..', '..'))

ROBOT_TEST_ROOT = os.path.abspath(os.path.join(__file__, '..'))

ROBOT_PROD_REG_DIR = os.path.abspath(os.path.join(ROBOT_TEST_ROOT, 'registry', 'prod'))
ROBOT_DEV_REG_DIR = os.path.abspath(os.path.join(ROBOT_TEST_ROOT, 'registry', 'dev'))
ROBOT_MOCKED_REG_DIR = os.path.abspath(os.path.join(ROBOT_TEST_ROOT, 'registry', 'mocked'))

ROBOT_INTEGRATION_TEST_ROOT = os.path.abspath(os.path.join(__file__, '..', 'integration'))

MOCKSERVER_PORT_REGISTRY = 1094

def get_output_from_local_file(fp :str) -> str:
  with open(os.path.join(REPOSITORY_ROOT, fp), 'r') as f:
    return f.read().strip()

def get_unix_path(pathStr :str) -> str:
  return pathStr.replace('\\', '/')


_PROD_REGISTRY_URL :str = "https://cdn.statically.io/gh/stackql/stackql-provider-registry/main/providers"
_DEV_REGISTRY_URL :str = "https://cdn.statically.io/gh/stackql/stackql-provider-registry/dev/providers"
_MOCKED_REGISTRY_URL :str = f"http://localhost:{MOCKSERVER_PORT_REGISTRY}/gh/stackql/stackql-provider-registry/main/providers"

REPOSITORY_ROOT_UNIX = get_unix_path(REPOSITORY_ROOT)
REGISTRY_ROOT_CANONICAL   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry'))
REGISTRY_ROOT_MOCKED   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry-mocked'))
REGISTRY_ROOT_EXPERIMENTAL   = get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'registry-advanced'))
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
_REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG    = { 
  "url": f"file://{get_unix_path(REGISTRY_ROOT_EXPERIMENTAL)}",
  "localDocRoot": f"{get_unix_path(REGISTRY_ROOT_EXPERIMENTAL)}",
  "useEmbedded": False,
  "verifyConfig": {
    "nopVerify": True 
  } 
}
_REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG    = { 
  "url": f"file://{get_unix_path(REGISTRY_ROOT_CANONICAL)}",
  "localDocRoot": f"{get_unix_path(REGISTRY_ROOT_CANONICAL)}",
  "srcPrefix": "registry-verb-matching-src",
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
  }, 
  "aws": { 
    "type": "aws_signing_v4",
    "credentialsfilepath": get_unix_path(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'aws', 'functional-test-dummy-aws-key.txt')),
     "keyID": "NON_SECRET" 
  },
  "github": { 
    "type": "basic", 
    "credentialsenvvar": "GITHUB_SECRET_KEY" 
  },
  "k8s": { 
    "credentialsenvvar": "K8S_SECRET_KEY",
    "type": "api_key",
    "valuePrefix": "Bearer " 
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

def get_registry_cfg(url :str, local_root :str, nop_verify :bool) -> dict:
  registry   = { 
    "url": url,
    "verifyConfig": {
      "nopVerify": nop_verify 
    } 
  }
  if local_root != "":
    registry["localDocRoot"] = local_root
  return registry

PG_SRV_MTLS_CFG_STR :str = json.dumps(_mTLS_CFG)

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'okta', 'api-key.txt'), 'r') as f:
    OKTA_SECRET_STR = f.read()

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'github', 'github-key.txt'), 'r') as f:
    GITHUB_SECRET_STR = f.read()

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'k8s', 'k8s-token.txt'), 'r') as f:
    K8S_SECRET_STR = f.read()


REGISTRY_PROD_CFG_STR = json.dumps(get_registry_cfg(_PROD_REGISTRY_URL, ROBOT_PROD_REG_DIR, False))
REGISTRY_DEV_CFG_STR = json.dumps(get_registry_cfg(_DEV_REGISTRY_URL, ROBOT_DEV_REG_DIR, False))

REGISTRY_MOCKED_CFG_STR = json.dumps(get_registry_cfg(_MOCKED_REGISTRY_URL, ROBOT_MOCKED_REG_DIR, False))

REGISTRY_NO_VERIFY_CFG_STR = json.dumps(_REGISTRY_NO_VERIFY_CFG)
REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR = json.dumps(_REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG)
REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG_STR = json.dumps(_REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG)
REGISTRY_CANONICAL_CFG_STR = json.dumps(_REGISTRY_CANONICAL_CFG)
REGISTRY_DEPRECATED_CFG_STR = json.dumps(_REGISTRY_DEPRECATED_CFG)
AUTH_CFG_STR = json.dumps(_AUTH_CFG)
SHOW_PROVIDERS_STR = "show providers;"
SHOW_OKTA_SERVICES_FILTERED_STR  = "show services from okta like 'app%';"
SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR  = "show resources from okta.application like 'gr%';"
SHOW_METHODS_GITHUB_REPOS_REPOS = "show methods in github.repos.repos;"
DESCRIBE_GITHUB_REPOS_PAGES = "describe github.repos.pages;"
MOCKSERVER_JAR = os.path.join(REPOSITORY_ROOT, 'test', 'downloads', 'mockserver-netty-5.12.0-shaded.jar')

JSON_INIT_FILE_PATH_GOOGLE = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-gcp-expectations.json')
MOCKSERVER_PORT_GOOGLE = 1080

JSON_INIT_FILE_PATH_OKTA = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-okta-expectations.json')
MOCKSERVER_PORT_OKTA = 1090

JSON_INIT_FILE_PATH_AWS = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-aws-expectations.json')
MOCKSERVER_PORT_AWS = 1091

JSON_INIT_FILE_PATH_K8S = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-k8s-expectations.json')
MOCKSERVER_PORT_K8S = 1092

JSON_INIT_FILE_PATH_GITHUB = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-github-expectations.json')
MOCKSERVER_PORT_GITHUB = 1093

JSON_INIT_FILE_PATH_REGISTRY = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-registry-expectations.json')

PG_SRV_PORT_MTLS = 5476
PG_SRV_PORT_UNENCRYPTED = 5477

PSQL_EXE :str = os.environ.get('PSQL_EXE', 'psql')

PSQL_CLIENT_HOST :str = "127.0.0.1"

PSQL_MTLS_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH} sslkey={STACKQL_PG_CLIENT_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"

PSQL_MTLS_INVALID_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_RUBBISH_CERT_PATH} sslkey={STACKQL_PG_RUBBISH_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"

PSQL_UNENCRYPTED_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_UNENCRYPTED} user=myuser dbname=mydatabase"

SELECT_CONTAINER_SUBNET_AGG_DESC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"
SELECT_CONTAINER_SUBNET_AGG_ASC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange asc;"
SELECT_ACCELERATOR_TYPES_DESC = "select  kind, name  from  google.compute.acceleratorTypes where project = 'testing-project' and zone = 'australia-southeast1-a' order by name desc;"

SELECT_AWS_VOLUMES = "select VolumeId, Encrypted, Size from aws.ec2.volumes where region = 'ap-southeast-1' order by VolumeId asc;"
CREATE_AWS_VOLUME = "insert into aws.ec2.volumes(AvailabilityZone, Size, region) select 'ap-southeast-1a', 10, 'ap-southeast-1';"

SELECT_GITHUB_REPOS_PAGES_SINGLE = "select url from github.repos.pages where owner = 'dummyorg' and repo = 'dummyapp.io';"
SELECT_GITHUB_REPOS_IDS_ASC = "select id from github.repos.repos where org = 'dummyorg' order by id ASC;"

SELECT_GITHUB_REPOS_FILTERED_SINGLE = "select id, name from github.repos.repos where org = 'dummyorg' and name = 'dummyapp.io';"

SELECT_GITHUB_SCIM_USERS = "select JSON_EXTRACT(name, '$.givenName') || ' ' || JSON_EXTRACT(name, '$.familyName') as name, userName, externalId, id from github.scim.users where org = 'dummyorg' order by id asc;"

SELECT_OKTA_APPS = "select name, status, label, id from okta.application.apps apps where apps.subdomain = 'example-subdomain' order by name asc;"

SELECT_CONTRIVED_GCP_OKTA_JOIN = "select d1.name, d1.id, d2.name as d2_name, d2.status, d2.label, d2.id as d2_id from google.compute.disks d1 inner join okta.application.apps d2 on d1.name = d2.label where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and d2.subdomain = 'dev-79923018-admin' order by d1.name ASC;"

SELECT_CONTRIVED_GCP_THREE_WAY_JOIN = "select d1.name as n, d1.id, n1.description, s1.description as s1_description from google.compute.disks d1 inner join google.compute.networks n1 on d1.name = n1.name inner join google.compute.subnetworks s1 on d1.name = s1.name where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and n1.project = 'testing-project' and s1.project = 'testing-project' and s1.region = 'australia-southeast1' ;"

SELECT_CONTRIVED_GCP_SELF_JOIN = "select d1.name as n, d1.id, d2.id as d2_id from google.compute.disks d1 inner join google.compute.disks d2 on d1.id = d2.id where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and d2.project = 'testing-project' and d2.zone = 'australia-southeast1-b' order by d1.name ASC;"

SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-desc.txt'))

SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-asc.txt'))

SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'contrived-gcp-okta-join.txt'))

SELECT_CONTRIVED_GCP_THREE_WAY_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'contrived-three-way-gcp-join.txt'))

SELECT_CONTRIVED_GCP_SELF_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'contrived-gcp-self-join.txt'))

SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN = "SELECT i.name as instance_name, i.status as instance_status, c.sha as commit_sha, JSON_EXTRACT(c.commit, '$.author.email') as author_email, DATE(JSON_EXTRACT(c.commit, '$.author.date')) as commit_date FROM github.repos.commits c INNER JOIN google.compute.instances i ON JSON_EXTRACT(i.labels, '$.sha') = c.sha WHERE c.owner = 'dummyorg' AND c.repo = 'dummyapp.io' AND i.project = 'testing-project' AND i.zone = 'australia-southeast1-a';"

SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'gcp-github-labelled-instances-commits.txt'))

SELECT_ACCELERATOR_TYPES_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'simple-select', 'compute-accelerator-type', 'select-zone-list-desc.txt'))

SELECT_OKTA_APPS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'simple-select', 'okta', 'apps', 'select-apps-asc.txt'))

SELECT_AWS_VOLUMES_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'volumes', 'select-volumes-asc.txt'))

SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-pages.txt'))
SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-ids-asc.txt'))
SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-single-filtered.txt'))
SELECT_GITHUB_SCIM_USERS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'scim', 'select-github-scim-users.txt'))

GET_IAM_POLICY_AGG_ASC_INPUT_FILE = os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'input', 'select-exec-dependent-org-iam-policy.iql')

GET_IAM_POLICY_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'cloudresourcemanager', 'select-exec-getiampolicy-agg.csv'))

SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'show', 'show-methods-github-repo-repo.txt'))

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'cloudresourcemanager', 'projects-getiampolicy-roles-asc.txt'))

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' and role like '%owner' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' and role = 'roles/owner' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'cloudresourcemanager', 'projects-getiampolicy-roles-asc-filtered.txt'))

SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS :bytes =  b"""SELECT i.zone, i.name, i.machineType, i.deletionProtection, '[{"subnetwork":"' || JSON_EXTRACT(i.networkInterfaces, '$[0].subnetwork') || '"}]', '[{"boot": true, "initializeParams": { "diskSizeGb": "' || JSON_EXTRACT(i.disks, '$[0].diskSizeGb') || '", "sourceImage": "' || d.sourceImage || '"}}]', i.labels FROM google.compute.instances i INNER JOIN google.compute.disks d ON i.name = d.name WHERE i.project = 'testing-project' AND i.zone = 'australia-southeast1-a' AND d.project = 'testing-project' AND d.zone = 'australia-southeast1-a' AND i.name LIKE '%' order by i.name DESC;"""

SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'joins', 'disks-instances-rewritten.txt'))

SELECT_K8S_NODES_ASC = f"select name, uid, creationTimestamp from k8s.core_v1.node where cluster_addr = '127.0.0.1:{MOCKSERVER_PORT_K8S}' order by name asc;"
SELECT_K8S_NODES_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'k8s', 'select-nodes-asc.txt'))

REGISTRY_LIST = "registry list;"
REGISTRY_GOOGLE_PROVIDER_LIST = "registry list google;"
REGISTRY_LIST_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'registry', 'all-providers-list.txt'))
REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'registry', 'google-list.txt'))