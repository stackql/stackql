
import base64
import json
import os

_exe_name = 'stackql'

IS_WINDOWS = '0'
if os.name == 'nt':
  IS_WINDOWS = '1'
  _exe_name = _exe_name + '.exe'

_DOCKER_REG_PATH :str = '/opt/stackql/registry' 

class RegistryCfg:

  def __init__(
    self,
    local_path :str,
    remote_url :str = '', 
    nop_verify :bool = False,
    src_prefix :str = '',
    is_null_registry :bool = False,
  ) -> None:
    self.local_path :str = local_path
    self.remote_url :str = remote_url
    self.nop_verify :bool = nop_verify
    self.src_prefix :str = src_prefix
    self.is_null_registry :bool = is_null_registry

  def _get_local_path(self, execution_environment :str) -> str:
    if self.local_path == '':
      return ''
    if execution_environment == "docker":
      return _DOCKER_REG_PATH
    return os.path.join(REPOSITORY_ROOT_UNIX, self.local_path)
  
  def _get_url(self, execution_environment :str) -> str:
    if self.remote_url != '':
      return self.remote_url
    if execution_environment == "docker":
      return f'file://{_DOCKER_REG_PATH}'
    return f'file://{os.path.join(REPOSITORY_ROOT_UNIX, self.local_path)}'
  
  def get_config_str(self, execution_environment :str) -> str:
    if self.is_null_registry:
      return ''
    cfg_dict = {
      "url": self._get_url(execution_environment)
    }
    if self._get_local_path(execution_environment) != "":
      cfg_dict["localDocRoot"] = self._get_local_path(execution_environment)
    if self.nop_verify:
      cfg_dict['verifyConfig'] = {
        'nopVerify': True
      }
    if self.src_prefix != '':
      cfg_dict['srcPrefix'] = self.src_prefix
    return json.dumps(cfg_dict)

  def get_source_path_for_docker(self) -> str:
    return self.local_path
      


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

REPOSITORY_ROOT_UNIX = get_unix_path(REPOSITORY_ROOT)
STACKQL_EXE     = get_unix_path(os.path.join(REPOSITORY_ROOT, 'build', _exe_name))

def get_registry_mocked(execution_env :str) -> RegistryCfg:
  return RegistryCfg(
    "",
    remote_url=get_registry_mock_url(execution_env),
    nop_verify=True
  )
_REGISTRY_NULL = RegistryCfg(
  '',
  is_null_registry=True
)
_REGISTRY_NO_VERIFY = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry-mocked')),
  nop_verify=True
)
_REGISTRY_EXPERIMENTAL_NO_VERIFY = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry-advanced')),
  nop_verify=True
)
_REGISTRY_EXPERIMENTAL_DOCKER_NO_VERIFY = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry-advanced-docker')),
  nop_verify=True
)
_REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry')),
  src_prefix="registry-verb-matching-src",
  nop_verify=True
)
_REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_DOCKER = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry')),
  src_prefix="registry-verb-matching-src-docker",
  nop_verify=True
)
_REGISTRY_CANONICAL = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry')),
  nop_verify=False
)
_REGISTRY_DEPRECATED = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry-deprecated')),
  nop_verify=False
)
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
_AUTH_CFG_DOCKER={ 
  "google": { 
    "credentialsfilepath": get_unix_path(os.path.join('/opt', 'stackql', 'credentials', 'dummy', 'google', 'docker-functional-test-dummy-sa-key.json')),
    "type": "service_account"
  }, 
  "okta": { 
    "credentialsenvvar": "OKTA_SECRET_KEY",
    "type": "api_key" 
  }, 
  "aws": { 
    "type": "aws_signing_v4",
    "credentialsfilepath": get_unix_path(os.path.join('/opt', 'stackql', 'credentials', 'dummy', 'aws', 'functional-test-dummy-aws-key.txt')),
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
STACKQL_PG_SERVER_KEY_PATH_DOCKER   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_server_key.pem"))
STACKQL_PG_SERVER_CERT_PATH_DOCKER  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_server_cert.pem"))
STACKQL_PG_CLIENT_KEY_PATH_DOCKER   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_client_key.pem"))
STACKQL_PG_CLIENT_CERT_PATH_DOCKER  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_client_cert.pem"))
STACKQL_PG_RUBBISH_KEY_PATH_DOCKER  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_rubbish_key.pem"))
STACKQL_PG_RUBBISH_CERT_PATH_DOCKER :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_rubbish_cert.pem"))

with open(os.path.join(REPOSITORY_ROOT, 'test', 'server', 'mtls', 'credentials', 'pg_client_cert.pem'), 'rb') as f:
  _CLIENT_CERT_ENCODED :str = base64.b64encode(f.read()).decode('utf-8')

# with open(os.path.join(REPOSITORY_ROOT, 'vol', 'srv', 'credentials', 'pg_client_cert.pem'), 'rb') as f:
#   _DOCKER_CLIENT_CERT_ENCODED :str = base64.b64encode(f.read()).decode('utf-8')

_mTLS_CFG :dict = { 
  "keyFilePath": STACKQL_PG_SERVER_KEY_PATH,
  "certFilePath": STACKQL_PG_SERVER_CERT_PATH,
  "clientCAs": [ 
    _CLIENT_CERT_ENCODED
  ] 
}

_mTLS_CFG_DOCKER :dict = { 
  "keyFilePath": "/opt/stackql/srv/credentials/pg_server_key.pem",
  "certFilePath": "/opt/stackql/srv/credentials/pg_server_cert.pem",
  "clientCAs": [ 
    "'\$(base64 -w 0 /opt/stackql/srv/credentials/pg_client_cert.pem)'"
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

PG_SRV_MTLS_DOCKER_CFG_STR :str = json.dumps(_mTLS_CFG_DOCKER).replace('\\$', '\$')

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'okta', 'api-key.txt'), 'r') as f:
    OKTA_SECRET_STR = f.read()

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'github', 'github-key.txt'), 'r') as f:
    GITHUB_SECRET_STR = f.read()

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'k8s', 'k8s-token.txt'), 'r') as f:
    K8S_SECRET_STR = f.read()


REGISTRY_PROD_CFG_STR = json.dumps(get_registry_cfg(_PROD_REGISTRY_URL, ROBOT_PROD_REG_DIR, False))
REGISTRY_DEV_CFG_STR = json.dumps(get_registry_cfg(_DEV_REGISTRY_URL, ROBOT_DEV_REG_DIR, False))

AUTH_CFG_STR = json.dumps(_AUTH_CFG)
AUTH_CFG_STR_DOCKER = json.dumps(_AUTH_CFG_DOCKER)
SHOW_PROVIDERS_STR = "show providers;"
SHOW_OKTA_SERVICES_FILTERED_STR  = "show services from okta like 'app%';"
SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR  = "show resources from okta.application like 'gr%';"
SHOW_METHODS_GITHUB_REPOS_REPOS = "show methods in github.repos.repos;"
DESCRIBE_GITHUB_REPOS_PAGES = "describe github.repos.pages;"
DESCRIBE_AWS_EC2_INSTANCES = "describe aws.ec2.instances;"
DESCRIBE_AWS_EC2_DEFAULT_KMS_KEY_ID = "describe aws.ec2.ebs_default_kms_key_id;"
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

PG_SRV_PORT_DOCKER_MTLS = 5576
PG_SRV_PORT_DOCKER_UNENCRYPTED = 5577

PSQL_EXE :str = os.environ.get('PSQL_EXE', 'psql')

PSQL_CLIENT_HOST :str = "127.0.0.1"

PSQL_MTLS_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH} sslkey={STACKQL_PG_CLIENT_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"
PSQL_MTLS_INVALID_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_RUBBISH_CERT_PATH} sslkey={STACKQL_PG_RUBBISH_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"
PSQL_UNENCRYPTED_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_UNENCRYPTED} user=myuser dbname=mydatabase"

PSQL_MTLS_CONN_STR_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH_DOCKER} sslkey={STACKQL_PG_CLIENT_KEY_PATH_DOCKER} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_DOCKER} dbname=mydatabase"
PSQL_MTLS_INVALID_CONN_STR_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_RUBBISH_CERT_PATH_DOCKER} sslkey={STACKQL_PG_RUBBISH_KEY_PATH_DOCKER} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_DOCKER} dbname=mydatabase"
PSQL_UNENCRYPTED_CONN_STR_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_UNENCRYPTED} user=myuser dbname=mydatabase"

SELECT_CONTAINER_SUBNET_AGG_DESC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"
SELECT_CONTAINER_SUBNET_AGG_ASC = "select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange asc;"
SELECT_ACCELERATOR_TYPES_DESC = "select  kind, name  from  google.compute.acceleratorTypes where project = 'testing-project' and zone = 'australia-southeast1-a' order by name desc;"
SELECT_MACHINE_TYPES_DESC = "select name from google.compute.machineTypes where project = 'testing-project' and zone = 'australia-southeast1-a' order by name desc;"
SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY = "SELECT eTag FROM google.compute.instances_iam_policies WHERE project = 'testing-project' AND zone = 'australia-southeast1-a' AND resource = '000000001';"

SELECT_AWS_VOLUMES = "select VolumeId, Encrypted, Size from aws.ec2.volumes where region = 'ap-southeast-1' order by VolumeId asc;"
CREATE_AWS_VOLUME = """insert into aws.ec2.volumes(AvailabilityZone, Size, region, TagSpecification) select 'ap-southeast-1a', JSON(10), 'ap-southeast-1', JSON('[ { "ResourceType": "volume", "Tag": [ { "Key": "stack", "Value": "production" }, { "Key": "name", "Value": "multi-tag-volume" } ] } ]');"""
CREATE_AWS_CLOUD_CONTROL_LOG_GROUP = """insert into aws.cloud_control.resources(region, data__TypeName, data__DesiredState) select 'ap-southeast-1', 'AWS::Logs::LogGroup', string('{ "LogGroupName": "LogGroupResourceExampleThird", "RetentionInDays":90}');"""
SELECT_AWS_CLOUD_CONTROL_VPCS_DESC = "select Identifier, Properties from aws.cloud_control.resources where region = 'ap-southeast-1' and data__TypeName = 'AWS::EC2::VPC' order by Identifier desc;"

SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC = "select TypeName, OperationStatus, StatusMessage, Identifier, RequestToken from aws.cloud_control.resource_requests where data__ResourceRequestStatusFilter='{}' and region = 'ap-southeast-1' order by RequestToken desc;"

SELECT_GITHUB_REPOS_PAGES_SINGLE = "select url from github.repos.pages where owner = 'dummyorg' and repo = 'dummyapp.io';"
SELECT_GITHUB_REPOS_IDS_ASC = "select id from github.repos.repos where org = 'dummyorg' order by id ASC;"
SELECT_GITHUB_BRANCHES_NAMES_DESC = "select name from github.repos.branches where owner = 'dummyorg' and repo = 'dummyapp.io' order by name desc;"
SELECT_GITHUB_REPOS_FILTERED_SINGLE = "select id, name from github.repos.repos where org = 'dummyorg' and name = 'dummyapp.io';"
SELECT_GITHUB_SCIM_USERS = "select JSON_EXTRACT(name, '$.givenName') || ' ' || JSON_EXTRACT(name, '$.familyName') as name, userName, externalId, id from github.scim.users where org = 'dummyorg' order by id asc;"
SELECT_GITHUB_SAML_IDENTITIES = "select guid, JSON_EXTRACT(samlIdentity, '$.nameId') AS saml_id, JSON_EXTRACT(user, '$.login') AS github_login from github.scim.saml_ids where org = 'dummyorg' order by JSON_EXTRACT(user, '$.login') asc;"
SELECT_GITHUB_TAGS_COUNT = "select count(*) as ct from github.repos.tags where owner = 'dummyorg' and repo = 'dummyapp.io';"
SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS = "select substr(su.userName, 1, instr(su.userName, '@') - 1), su.externalId, su.id, u.login, u.two_factor_authentication AS is_two_fa_enabled from github.scim.users su inner join github.users.users u ON substr(su.userName, 1, instr(su.userName, '@') - 1) = u.username and substr(su.userName, 1, instr(su.userName, '@') - 1) = u.login where su.org = 'dummyorg' order by su.id asc;"
SELECT_GITHUB_ORGS_MEMBERS = "select om.login from github.orgs.members om where om.org = 'dummyorg' order by om.login desc;"

SELECT_OKTA_APPS = "select name, status, label, id from okta.application.apps apps where apps.subdomain = 'example-subdomain' order by name asc;"
SELECT_OKTA_USERS_ASC = "select JSON_EXTRACT(ou.profile, '$.login') as login, ou.status from okta.user.users ou WHERE ou.subdomain = 'dummyorg' order by JSON_EXTRACT(ou.profile, '$.login') asc;"

SELECT_CONTRIVED_GCP_OKTA_JOIN = "select d1.name, d1.id, d2.name as d2_name, d2.status, d2.label, d2.id as d2_id from google.compute.disks d1 inner join okta.application.apps d2 on d1.name = d2.label where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and d2.subdomain = 'dev-79923018-admin' order by d1.name ASC;"

SELECT_GITHUB_OKTA_SAML_JOIN = "select JSON_EXTRACT(saml.samlIdentity, '$.username') as saml_username, om.login as github_login, ou.status as okta_status from github.scim.saml_ids saml INNER JOIN okta.user.users ou ON JSON_EXTRACT(saml.samlIdentity, '$.username') = JSON_EXTRACT(ou.profile, '$.login') INNER JOIN github.orgs.members om ON JSON_EXTRACT(saml.user, '$.login') = om.login where ou.subdomain = 'dummyorg' AND om.org = 'dummyorg' AND saml.org = 'dummyorg' order by om.login desc;"

SELECT_CONTRIVED_GCP_THREE_WAY_JOIN = "select d1.name as n, d1.id, n1.description, s1.description as s1_description from google.compute.disks d1 inner join google.compute.networks n1 on d1.name = n1.name inner join google.compute.subnetworks s1 on d1.name = s1.name where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and n1.project = 'testing-project' and s1.project = 'testing-project' and s1.region = 'australia-southeast1' ;"

SELECT_CONTRIVED_GCP_SELF_JOIN = "select d1.name as n, d1.id, d2.id as d2_id from google.compute.disks d1 inner join google.compute.disks d2 on d1.id = d2.id where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and d2.project = 'testing-project' and d2.zone = 'australia-southeast1-b' order by d1.name ASC;"

SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-desc.txt'))

SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'container', 'agg-subnetworks-allowed', 'table', 'simple-count-grouped-variant-asc.txt'))

SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'contrived-gcp-okta-join.txt'))

SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL = "select u.name, om.login, u.two_factor_authentication AS is_two_fa_enabled from github.orgs.members om inner join github.users.users u on om.login = u.login AND u.username = om.login where om.org = 'dummyorg' order by u.name desc;"

SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS = "show insert into google.iam.service_accounts;"
SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR = "show insert into google.compute.instances_iam_policies;"

SELECT_CONTRIVED_GCP_THREE_WAY_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'contrived-three-way-gcp-join.txt'))

SELECT_CONTRIVED_GCP_SELF_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'contrived-gcp-self-join.txt'))

SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN = "SELECT i.name as instance_name, i.status as instance_status, c.sha as commit_sha, JSON_EXTRACT(c.commit, '$.author.email') as author_email, DATE(JSON_EXTRACT(c.commit, '$.author.date')) as commit_date FROM github.repos.commits c INNER JOIN google.compute.instances i ON JSON_EXTRACT(i.labels, '$.sha') = c.sha WHERE c.owner = 'dummyorg' AND c.repo = 'dummyapp.io' AND i.project = 'testing-project' AND i.zone = 'australia-southeast1-a';"

SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'gcp-github-labelled-instances-commits.txt'))

SELECT_ACCELERATOR_TYPES_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'simple-select', 'compute-accelerator-type', 'select-zone-list-desc.txt'))

SELECT_MACHINE_TYPES_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'compute', 'instance-type-list-names-paginated-desc.txt'))

SELECT_OKTA_APPS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'simple-select', 'okta', 'apps', 'select-apps-asc.txt'))
SELECT_OKTA_USERS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'okta', 'select-users-asc.txt'))

SELECT_AWS_VOLUMES_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'volumes', 'select-volumes-asc.txt'))
SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-list-vpcs-desc.txt'))
SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-list-operations-desc.txt'))

SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-pages.txt'))
SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-ids-asc.txt'))
SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-single-filtered.txt'))
SELECT_GITHUB_SCIM_USERS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'scim', 'select-github-scim-users.txt'))
SELECT_GITHUB_SAML_IDENTITIES_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'scim', 'select-github-saml-identities.txt'))
SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-branches-names-desc.txt'))
SELECT_GITHUB_TAGS_COUNT_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-tags-count.txt'))
SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'joins', 'select-github-sequential-join.txt'))
SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'joins', 'select-github-sequential-join-with-functions.txt'))
SELECT_GITHUB_OKTA_SAML_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'github-saml-members-okta-users.txt'))
SELECT_GITHUB_ORGS_MEMBERS_PAGE_LIMITED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'orgs', 'page-limited-members.txt'))
SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'compute', 'instance-iam-policy-projection.txt'))

SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'show', 'show-insert-google-iam-service-accounts.txt'))
SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'show', 'show-insert-google-compute-instances-iam-error.txt'))


GET_IAM_POLICY_AGG_ASC_INPUT_FILE = os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'input', 'select-exec-dependent-org-iam-policy.iql')
GET_IAM_POLICY_AGG_ASC_INPUT_FILE_DOCKER = os.path.join('/opt', 'stackql', 'input', 'select-exec-dependent-org-iam-policy.iql')

GET_IAM_POLICY_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'cloudresourcemanager', 'select-exec-getiampolicy-agg.csv'))

SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'show', 'show-methods-github-repo-repo.txt'))

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'cloudresourcemanager', 'projects-getiampolicy-roles-asc.txt'))

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' and role like '%owner' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' and role = 'roles/owner' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'cloudresourcemanager', 'projects-getiampolicy-roles-asc-filtered.txt'))

SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS :bytes =  b"""SELECT i.zone, i.name, i.machineType, i.deletionProtection, '[{"subnetwork":"' || JSON_EXTRACT(i.networkInterfaces, '$[0].subnetwork') || '"}]', '[{"boot": true, "initializeParams": { "diskSizeGb": "' || JSON_EXTRACT(i.disks, '$[0].diskSizeGb') || '", "sourceImage": "' || d.sourceImage || '"}}]', i.labels FROM google.compute.instances i INNER JOIN google.compute.disks d ON i.name = d.name WHERE i.project = 'testing-project' AND i.zone = 'australia-southeast1-a' AND d.project = 'testing-project' AND d.zone = 'australia-southeast1-a' AND i.name LIKE '%' order by i.name DESC;"""

SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'joins', 'disks-instances-rewritten.txt'))

def get_select_k8s_nodes_asc(execution_env :str) -> str:
  k8s_host = '127.0.0.1'
  if execution_env == 'docker':
    k8s_host = 'host.docker.internal'
  return f"select name, uid, creationTimestamp from k8s.core_v1.node where cluster_addr = '{k8s_host}:{MOCKSERVER_PORT_K8S}' order by name asc;"

def get_registry_mock_url(execution_env :str) -> str:
  host = 'localhost'
  if execution_env == 'docker':
    host = 'host.docker.internal'
  return f"http://{host}:{MOCKSERVER_PORT_REGISTRY}/gh/stackql/stackql-provider-registry/main/providers"

SELECT_K8S_NODES_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'k8s', 'select-nodes-asc.txt'))

REGISTRY_LIST = "registry list;"
REGISTRY_GOOGLE_PROVIDER_LIST = "registry list google;"
REGISTRY_LIST_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'registry', 'all-providers-list.txt'))
REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'registry', 'google-list.txt'))

def get_variables(execution_env :str):
  rv = {
    ## general config
    'GITHUB_SECRET_STR':                              GITHUB_SECRET_STR,
    'K8S_SECRET_STR':                                 K8S_SECRET_STR,
    'MOCKSERVER_JAR':                                 MOCKSERVER_JAR,
    'MOCKSERVER_PORT_AWS':                            MOCKSERVER_PORT_AWS,
    'MOCKSERVER_PORT_GITHUB':                         MOCKSERVER_PORT_GITHUB,
    'MOCKSERVER_PORT_GOOGLE':                         MOCKSERVER_PORT_GOOGLE,
    'MOCKSERVER_PORT_K8S':                            MOCKSERVER_PORT_K8S,
    'MOCKSERVER_PORT_OKTA':                           MOCKSERVER_PORT_OKTA,
    'MOCKSERVER_PORT_REGISTRY':                       MOCKSERVER_PORT_REGISTRY,
    'OKTA_SECRET_STR':                                OKTA_SECRET_STR,
    'PG_SRV_MTLS_DOCKER_CFG_STR':                     PG_SRV_MTLS_DOCKER_CFG_STR,
    'PG_SRV_PORT_DOCKER_MTLS':                        PG_SRV_PORT_DOCKER_MTLS,
    'PG_SRV_PORT_DOCKER_UNENCRYPTED':                 PG_SRV_PORT_DOCKER_UNENCRYPTED,
    'PG_SRV_PORT_MTLS':                               PG_SRV_PORT_MTLS,
    'PG_SRV_PORT_UNENCRYPTED':                        PG_SRV_PORT_UNENCRYPTED,
    'PSQL_CLIENT_HOST':                               PSQL_CLIENT_HOST,
    'PSQL_EXE':                                       PSQL_EXE,
    'REGISTRY_ROOT_CANONICAL':                        _REGISTRY_CANONICAL,
    'REGISTRY_ROOT_DEPRECATED':                       _REGISTRY_DEPRECATED,
    'REGISTRY_CANONICAL_CFG_STR':                     _REGISTRY_CANONICAL,
    'REGISTRY_DEPRECATED_CFG_STR':                    _REGISTRY_DEPRECATED,
    'REGISTRY_MOCKED_CFG_STR':                        get_registry_mocked(execution_env),
    'REGISTRY_NO_VERIFY_CFG_STR':                     _REGISTRY_NO_VERIFY,
    'REGISTRY_NULL':                                  _REGISTRY_NULL,
    'REPOSITORY_ROOT':                                REPOSITORY_ROOT,
    'STACKQL_EXE':                                    STACKQL_EXE,
    ## queries and expectations
    'CREATE_AWS_VOLUME':                                                    CREATE_AWS_VOLUME,
    'CREATE_AWS_CLOUD_CONTROL_LOG_GROUP':                                   CREATE_AWS_CLOUD_CONTROL_LOG_GROUP,
    'DESCRIBE_AWS_EC2_INSTANCES':                                           DESCRIBE_AWS_EC2_INSTANCES,
    'DESCRIBE_AWS_EC2_DEFAULT_KMS_KEY_ID':                                  DESCRIBE_AWS_EC2_DEFAULT_KMS_KEY_ID,
    'DESCRIBE_GITHUB_REPOS_PAGES':                                          DESCRIBE_GITHUB_REPOS_PAGES,
    'GET_IAM_POLICY_AGG_ASC_EXPECTED':                                      GET_IAM_POLICY_AGG_ASC_EXPECTED,
    'REGISTRY_GOOGLE_PROVIDER_LIST':                                        REGISTRY_GOOGLE_PROVIDER_LIST,
    'REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED':                               REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED,
    'REGISTRY_LIST':                                                        REGISTRY_LIST,
    'REGISTRY_LIST_EXPECTED':                                               REGISTRY_LIST_EXPECTED,
    'SELECT_ACCELERATOR_TYPES_DESC':                                        SELECT_ACCELERATOR_TYPES_DESC,
    'SELECT_ACCELERATOR_TYPES_DESC_EXPECTED':                               SELECT_ACCELERATOR_TYPES_DESC_EXPECTED,
    'SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC':                             SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC,
    'SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC_EXPECTED':                    SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC_EXPECTED,
    'SELECT_AWS_CLOUD_CONTROL_VPCS_DESC':                                   SELECT_AWS_CLOUD_CONTROL_VPCS_DESC,
    'SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED':                          SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED,
    'SELECT_AWS_VOLUMES':                                                   SELECT_AWS_VOLUMES,
    'SELECT_AWS_VOLUMES_ASC_EXPECTED':                                      SELECT_AWS_VOLUMES_ASC_EXPECTED,
    'SELECT_CONTAINER_SUBNET_AGG_ASC':                                      SELECT_CONTAINER_SUBNET_AGG_ASC,
    'SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED':                             SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED,
    'SELECT_CONTAINER_SUBNET_AGG_DESC':                                     SELECT_CONTAINER_SUBNET_AGG_DESC,
    'SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED':                            SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED,
    'SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN':                      SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN,
    'SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN_EXPECTED':             SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN_EXPECTED,
    'SELECT_CONTRIVED_GCP_OKTA_JOIN':                                       SELECT_CONTRIVED_GCP_OKTA_JOIN,
    'SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED':                              SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED,
    'SELECT_CONTRIVED_GCP_SELF_JOIN':                                       SELECT_CONTRIVED_GCP_SELF_JOIN,
    'SELECT_CONTRIVED_GCP_SELF_JOIN_EXPECTED':                              SELECT_CONTRIVED_GCP_SELF_JOIN_EXPECTED,
    'SELECT_CONTRIVED_GCP_THREE_WAY_JOIN':                                  SELECT_CONTRIVED_GCP_THREE_WAY_JOIN,
    'SELECT_CONTRIVED_GCP_THREE_WAY_JOIN_EXPECTED':                         SELECT_CONTRIVED_GCP_THREE_WAY_JOIN_EXPECTED,
    'SELECT_GITHUB_BRANCHES_NAMES_DESC':                                    SELECT_GITHUB_BRANCHES_NAMES_DESC,
    'SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED':                           SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED,
    'SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL':                              SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL,
    'SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL':                              SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL,
    'SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED':                     SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED,
    'SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED':                     SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED,
    'SELECT_GITHUB_OKTA_SAML_JOIN':                                         SELECT_GITHUB_OKTA_SAML_JOIN,
    'SELECT_GITHUB_OKTA_SAML_JOIN_EXPECTED':                                SELECT_GITHUB_OKTA_SAML_JOIN_EXPECTED,
    'SELECT_GITHUB_ORGS_MEMBERS':                                           SELECT_GITHUB_ORGS_MEMBERS,
    'SELECT_GITHUB_ORGS_MEMBERS_PAGE_LIMITED_EXPECTED':                     SELECT_GITHUB_ORGS_MEMBERS_PAGE_LIMITED_EXPECTED,
    'SELECT_GITHUB_REPOS_FILTERED_SINGLE':                                  SELECT_GITHUB_REPOS_FILTERED_SINGLE,
    'SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED':                         SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED,
    'SELECT_GITHUB_REPOS_IDS_ASC':                                          SELECT_GITHUB_REPOS_IDS_ASC,
    'SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED':                                 SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED,
    'SELECT_GITHUB_REPOS_PAGES_SINGLE':                                     SELECT_GITHUB_REPOS_PAGES_SINGLE,
    'SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED':                            SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED,
    'SELECT_GITHUB_SAML_IDENTITIES':                                        SELECT_GITHUB_SAML_IDENTITIES,
    'SELECT_GITHUB_SAML_IDENTITIES_EXPECTED':                               SELECT_GITHUB_SAML_IDENTITIES_EXPECTED,
    'SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS':                               SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS,
    'SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS_EXPECTED':                      SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS_EXPECTED,
    'SELECT_GITHUB_SCIM_USERS':                                             SELECT_GITHUB_SCIM_USERS,
    'SELECT_GITHUB_SCIM_USERS_EXPECTED':                                    SELECT_GITHUB_SCIM_USERS_EXPECTED,
    'SELECT_GITHUB_TAGS_COUNT':                                             SELECT_GITHUB_TAGS_COUNT,
    'SELECT_GITHUB_TAGS_COUNT_EXPECTED':                                    SELECT_GITHUB_TAGS_COUNT_EXPECTED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY':                         SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED':     SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED':                SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED':       SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED':           SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED,
    'SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY':                            SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY,
    'SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_EXPECTED':                   SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_EXPECTED,
    'SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS':                   SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS,
    'SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED':          SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED,
    'SELECT_K8S_NODES_ASC':                                                 get_select_k8s_nodes_asc(execution_env),
    'SELECT_K8S_NODES_ASC_EXPECTED':                                        SELECT_K8S_NODES_ASC_EXPECTED,
    'SELECT_MACHINE_TYPES_DESC':                                            SELECT_MACHINE_TYPES_DESC,
    'SELECT_MACHINE_TYPES_DESC_EXPECTED':                                   SELECT_MACHINE_TYPES_DESC_EXPECTED,
    'SELECT_OKTA_APPS':                                                     SELECT_OKTA_APPS,
    'SELECT_OKTA_APPS_ASC_EXPECTED':                                        SELECT_OKTA_APPS_ASC_EXPECTED,
    'SELECT_OKTA_USERS_ASC':                                                SELECT_OKTA_USERS_ASC,
    'SELECT_OKTA_USERS_ASC_EXPECTED':                                       SELECT_OKTA_USERS_ASC_EXPECTED,
    'SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR':                 SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR,
    'SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR':                 SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR,
    'SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR_EXPECTED':        SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR_EXPECTED,
    'SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS':                              SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS,
    'SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS':                              SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS,
    'SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS_EXPECTED':                     SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS_EXPECTED,
    'SHOW_METHODS_GITHUB_REPOS_REPOS':                                      SHOW_METHODS_GITHUB_REPOS_REPOS,
    'SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED':                             SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED,
    'SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR':                         SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR,
    'SHOW_OKTA_SERVICES_FILTERED_STR':                                      SHOW_OKTA_SERVICES_FILTERED_STR,
    'SHOW_PROVIDERS_STR':                                                   SHOW_PROVIDERS_STR,
  }
  if execution_env == 'docker':
    rv['AUTH_CFG_STR']                                  = AUTH_CFG_STR_DOCKER
    rv['GET_IAM_POLICY_AGG_ASC_INPUT_FILE']             = GET_IAM_POLICY_AGG_ASC_INPUT_FILE_DOCKER
    rv['JSON_INIT_FILE_PATH_AWS']                       = JSON_INIT_FILE_PATH_AWS
    rv['JSON_INIT_FILE_PATH_GITHUB']                    = JSON_INIT_FILE_PATH_GITHUB
    rv['JSON_INIT_FILE_PATH_GOOGLE']                    = JSON_INIT_FILE_PATH_GOOGLE
    rv['JSON_INIT_FILE_PATH_K8S']                       = JSON_INIT_FILE_PATH_K8S
    rv['JSON_INIT_FILE_PATH_OKTA']                      = JSON_INIT_FILE_PATH_OKTA
    rv['JSON_INIT_FILE_PATH_REGISTRY']                  = JSON_INIT_FILE_PATH_REGISTRY
    rv['PG_SRV_MTLS_CFG_STR']                           = PG_SRV_MTLS_CFG_STR
    rv['PSQL_MTLS_CONN_STR']                            = PSQL_MTLS_CONN_STR_DOCKER
    rv['PSQL_MTLS_INVALID_CONN_STR']                    = PSQL_MTLS_INVALID_CONN_STR_DOCKER
    rv['PSQL_UNENCRYPTED_CONN_STR']                     = PSQL_UNENCRYPTED_CONN_STR_DOCKER
    rv['REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR']       = _REGISTRY_EXPERIMENTAL_DOCKER_NO_VERIFY
    rv['REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG_STR'] = _REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_DOCKER
  else: 
    rv['AUTH_CFG_STR']                                  = AUTH_CFG_STR
    rv['GET_IAM_POLICY_AGG_ASC_INPUT_FILE']             = GET_IAM_POLICY_AGG_ASC_INPUT_FILE
    rv['JSON_INIT_FILE_PATH_AWS']                       = JSON_INIT_FILE_PATH_AWS
    rv['JSON_INIT_FILE_PATH_GITHUB']                    = JSON_INIT_FILE_PATH_GITHUB
    rv['JSON_INIT_FILE_PATH_GOOGLE']                    = JSON_INIT_FILE_PATH_GOOGLE
    rv['JSON_INIT_FILE_PATH_K8S']                       = JSON_INIT_FILE_PATH_K8S
    rv['JSON_INIT_FILE_PATH_OKTA']                      = JSON_INIT_FILE_PATH_OKTA
    rv['JSON_INIT_FILE_PATH_REGISTRY']                  = JSON_INIT_FILE_PATH_REGISTRY
    rv['PG_SRV_MTLS_CFG_STR']                           = PG_SRV_MTLS_CFG_STR
    rv['PSQL_MTLS_CONN_STR']                            = PSQL_MTLS_CONN_STR
    rv['PSQL_MTLS_INVALID_CONN_STR']                    = PSQL_MTLS_INVALID_CONN_STR
    rv['PSQL_UNENCRYPTED_CONN_STR']                     = PSQL_UNENCRYPTED_CONN_STR
    rv['REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR']       = _REGISTRY_EXPERIMENTAL_NO_VERIFY
    rv['REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG_STR'] = _REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY
  return rv