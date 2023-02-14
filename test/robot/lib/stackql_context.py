
import base64
import json
import os
import typing
import copy

from typed_python_responses import SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED

_exe_name = 'stackql'

IS_WINDOWS = '0'
if os.name == 'nt':
  IS_WINDOWS = '1'
  _exe_name = _exe_name + '.exe'

_DOCKER_REG_PATH :str = '/opt/stackql/registry' 

_PG_SCHEMA_PHYSICAL_TABLES = "stackql_raw"
_PG_SCHEMA_INTEL = "stackql_intel"

_BUILD_MAJOR_VERSION = os.environ.get('BUILDMAJORVERSION', '1')
_BUILD_MINOR_VERSION = os.environ.get('BUILDMINORVERSION', '1')
_BUILD_PATCH_VERSION = os.environ.get('BUILDPATCHVERSION', '1')

_SHELL_WELCOME_MSG = """
""" + f"stackql Command Shell {_BUILD_MAJOR_VERSION}.{_BUILD_MINOR_VERSION}.{_BUILD_PATCH_VERSION}" + """
Copyright (c) 2021, stackql studios. All rights reserved.
Welcome to the interactive shell for running stackql commands.
---
"""

_AZURE_INTEGRATION_TESTING_SUB_ID :str = os.environ.get('AZURE_INTEGRATION_TESTING_SUB_ID', '10001000-1000-1000-1000-100010001000')

_AZURE_VM_SIZES_ENUMERATION :str = f"SELECT * FROM azure.compute.virtual_machine_sizes WHERE location = 'Australia East' AND subscriptionId = '{_AZURE_INTEGRATION_TESTING_SUB_ID}';"

_TEST_APP_CACHE_ROOT = os.path.join("test", ".stackql")

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

DB_INTERNAL_CFG_LAX :str = "{ \"tableRegex\": \"(?i)^(?:public\\\\.)?(?:pg_.*|current_schema|information_schema)\" }"

_TEST_APP_CACHE_ROOT = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", ".stackql"))

ROBOT_TEST_ROOT = os.path.abspath(os.path.join(__file__, '..'))

ROBOT_PROD_REG_DIR = os.path.abspath(os.path.join(ROBOT_TEST_ROOT, 'registry', 'prod'))
ROBOT_DEV_REG_DIR = os.path.abspath(os.path.join(ROBOT_TEST_ROOT, 'registry', 'dev'))
ROBOT_MOCKED_REG_DIR = os.path.abspath(os.path.join(ROBOT_TEST_ROOT, 'registry', 'mocked'))

ROBOT_INTEGRATION_TEST_ROOT = os.path.abspath(os.path.join(__file__, '..', 'integration'))

_NAMESPACES_TTL_SIMPLE = '{ "analytics": { "ttl": 86400, "regex": "^(?:stackql_analytics_)?(?P<objectName>.*)$", "template": "stackql_analytics_{{ .objectName }}" } }'
_NAMESPACES_TTL_TRANSPARENT = '{ "analytics": { "ttl": 86400, "regex": "^(?P<objectName>.*)$", "template": "stackql_analytics_{{ .objectName }}" } }'
_NAMESPACES_TTL_SPECIALCASE_TRANSPARENT = '{ "analytics": { "ttl": 86400, "regex": "^(?P<objectName>github.*)$", "template": "stackql_analytics_{{ .objectName }}" } }'

_GC_CFG_EAGER = '{ "isEager": true }'

_SQL_BACKEND = '{ "isEager": true }'

NAMESPACES_TTL_SIMPLE = _NAMESPACES_TTL_SIMPLE.replace(' ', '')
NAMESPACES_TTL_TRANSPARENT = _NAMESPACES_TTL_TRANSPARENT.replace(' ', '')
NAMESPACES_TTL_SPECIALCASE_TRANSPARENT = _NAMESPACES_TTL_SPECIALCASE_TRANSPARENT.replace(' ', '')

MOCKSERVER_PORT_REGISTRY = 1094

def get_output_from_local_file(fp :str) -> str:
  with open(os.path.join(REPOSITORY_ROOT, fp), 'r') as f:
    return f.read().strip()

def get_json_from_local_file(fp :str) -> typing.Any:
  with open(os.path.join(REPOSITORY_ROOT, fp), 'r') as f:
    return json.load(f)

def get_unix_path(pathStr :str) -> str:
  return pathStr.replace('\\', '/')


_PROD_REGISTRY_URL :str = "https://cdn.statically.io/gh/stackql/stackql-provider-registry/main/providers"
_DEV_REGISTRY_URL :str = "https://cdn.statically.io/gh/stackql/stackql-provider-registry/dev/providers"

REPOSITORY_ROOT_UNIX = get_unix_path(REPOSITORY_ROOT)
STACKQL_EXE     = ' '.join(get_unix_path(os.path.join(REPOSITORY_ROOT, 'build', _exe_name)).splitlines())

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
_REGISTRY_CANONICAL_NO_VERIFY = RegistryCfg(
  get_unix_path(os.path.join('test', 'registry')),
  nop_verify=True
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
  },
  "azure": { 
    "type": "api_key",
    "valuePrefix": "Bearer ",
    "credentialsenvvar": "AZ_ACCESS_TOKEN"
  },
  "sumologic": {
    "type": "basic",
    "credentialsenvvar": "SUMO_CREDS"
  }
}

_AUTH_PLUS_EXTERNAL_POSTGRES = copy.deepcopy(_AUTH_CFG)

_AUTH_PLUS_EXTERNAL_POSTGRES["pgi"] = { 
  "type": "sql_data_source::postgres",
  "sqlDataSource": {
    "dsn": "postgres://stackql:stackql@127.0.0.1:8432" 
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
  },
  "azure": { 
    "type": "api_key",
    "valuePrefix": "Bearer ",
    "credentialsenvvar": "AZ_ACCESS_TOKEN"
  },
  "sumologic": {
    "type": "basic",
    "credentialsenvvar": "SUMO_CREDS"
  }
}

_AUTH_PLUS_EXTERNAL_POSTGRES_DOCKER = copy.deepcopy(_AUTH_CFG_DOCKER)

_AUTH_PLUS_EXTERNAL_POSTGRES_DOCKER["pgi"] = { 
  "type": "sql_data_source::postgres",
  "sqlDataSource": {
    "dsn": "postgres://stackql:stackql@host.docker.internal:8432" 
  } 
}

_AUTH_CFG_INTEGRATION={ 
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
  },
  "azure": { 
    "type": "azure_default"
  },
  "sumologic": {
    "type": "basic",
    "credentialsenvvar": "SUMO_CREDS"
  }
}
_AUTH_CFG_INTEGRATION_DOCKER={ 
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
  },
  "azure": { 
    "type": "azure_default"
  },
  "sumologic": {
    "type": "basic",
    "credentialsenvvar": "SUMO_CREDS"
  }
}
STACKQL_PG_SERVER_KEY_PATH   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_server_key.pem"))
STACKQL_PG_SERVER_CERT_PATH  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_server_cert.pem"))
STACKQL_PG_CLIENT_KEY_PATH   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_client_key.pem"))
STACKQL_PG_CLIENT_CERT_PATH  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_client_cert.pem"))
STACKQL_PG_SERVER_CERT_PATH_UNIX  :str = get_unix_path(STACKQL_PG_SERVER_CERT_PATH)
STACKQL_PG_CLIENT_KEY_PATH_UNIX   :str = get_unix_path(STACKQL_PG_CLIENT_KEY_PATH)
STACKQL_PG_CLIENT_CERT_PATH_UNIX  :str = get_unix_path(STACKQL_PG_CLIENT_CERT_PATH)
STACKQL_PG_RUBBISH_KEY_PATH  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_rubbish_key.pem"))
STACKQL_PG_RUBBISH_CERT_PATH :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "server", "mtls", "credentials", "pg_rubbish_cert.pem"))
STACKQL_PG_SERVER_KEY_PATH_DOCKER   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_server_key.pem"))
STACKQL_PG_SERVER_CERT_PATH_DOCKER  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_server_cert.pem"))
STACKQL_PG_CLIENT_KEY_PATH_DOCKER   :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_client_key.pem"))
STACKQL_PG_CLIENT_CERT_PATH_DOCKER  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_client_cert.pem"))
STACKQL_PG_RUBBISH_KEY_PATH_DOCKER  :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_rubbish_key.pem"))
STACKQL_PG_RUBBISH_CERT_PATH_DOCKER :str = os.path.abspath(os.path.join(REPOSITORY_ROOT, "vol", "srv", "credentials", "pg_rubbish_cert.pem"))

def get_sql_dialect_from_sql_backend_str(sql_backend_str :str) -> str:
  if sql_backend_str == 'postgres_tcp':
    return 'postgres'
  return 'sqlite'

def get_analytics_db_init_path(sql_backend_str :str) -> str:
  sql_dialect = get_sql_dialect_from_sql_backend_str(sql_backend_str)
  return os.path.abspath(os.path.join(REPOSITORY_ROOT, "test", "db", sql_dialect,  "cache_setup.sql"))


ANALYTICS_DB_INIT_PATH_DOCKER :str = get_unix_path(os.path.join('/opt', 'stackql', "db", "cache_setup.sql"))

def get_analytics_db_init_path_unix(sql_backend_str :str) ->str:
  return get_unix_path(get_analytics_db_init_path(sql_backend_str))

_SQL_BACKEND_POSTGRES_DOCKER_DSN :str = 'postgres://stackql:stackql@postgres_stackql:5432/stackql'


def get_analytics_sql_backend(execution_env :str, sql_backend_str :str) -> str:
  if execution_env == 'native':
    return f'{{ "dbInitFilepath": "{get_analytics_db_init_path_unix(sql_backend_str)}" }}'.replace(' ', '')
  if execution_env == 'docker':
    if sql_backend_str == 'postgres_tcp':
      return f'{{ "dbEngine": "postgres_tcp", "dsn": "{_SQL_BACKEND_POSTGRES_DOCKER_DSN}", "sqlDialect": "postgres", "dbInitFilepath": "{ANALYTICS_DB_INIT_PATH_DOCKER}", "schemata": {{ "tableSchema": "{_PG_SCHEMA_PHYSICAL_TABLES}", "intelViewSchema": "{_PG_SCHEMA_INTEL}", "opsViewSchema": "stackql_ops" }} }}'.replace(' ', '')
    return f'{{ "dbInitFilepath": "{ANALYTICS_DB_INIT_PATH_DOCKER}" }}'.replace(' ', '')


def get_canonical_sql_backend(execution_env :str, sql_backend_str :str) -> str:
  if execution_env == 'native':
    return '{}'
  if execution_env == 'docker':
    if sql_backend_str == 'postgres_tcp':
      return f'{{ "dbEngine": "postgres_tcp", "dsn": "{_SQL_BACKEND_POSTGRES_DOCKER_DSN}", "sqlDialect": "postgres", "schemata": {{ "tableSchema": "{_PG_SCHEMA_PHYSICAL_TABLES}", "intelViewSchema": "{_PG_SCHEMA_INTEL}", "opsViewSchema": "stackql_ops" }} }}'.replace(' ', '')
    return '{}'


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

def get_object_count_dict(count :int) -> dict:
  """
  Blasted type inference in golang SQL lib is not flash.
  """
  return { "object_count": f"{count}" }

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

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'azure', 'azure-token.txt'), 'r') as f:
    AZURE_SECRET_STR = f.read()

with open(os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'credentials', 'dummy', 'sumologic', 'sumologic-token.txt'), 'r') as f:
    SUMOLOGIC_SECRET_STR = f.read()

REGISTRY_PROD_CFG_STR = json.dumps(get_registry_cfg(_PROD_REGISTRY_URL, ROBOT_PROD_REG_DIR, False))
REGISTRY_DEV_CFG_STR = json.dumps(get_registry_cfg(_DEV_REGISTRY_URL, ROBOT_DEV_REG_DIR, False))

AUTH_CFG_STR = json.dumps(_AUTH_CFG)
AUTH_CFG_STR_DOCKER = json.dumps(_AUTH_CFG_DOCKER)
AUTH_PLUS_EXTERNAL_POSTGRES = json.dumps(_AUTH_PLUS_EXTERNAL_POSTGRES)
AUTH_PLUS_EXTERNAL_POSTGRES_DOCKER = json.dumps(_AUTH_PLUS_EXTERNAL_POSTGRES_DOCKER)
AUTH_CFG_INTEGRATION_STR = json.dumps(_AUTH_CFG_INTEGRATION)
AUTH_CFG_INTEGRATION_STR_DOCKER = json.dumps(_AUTH_CFG_INTEGRATION_DOCKER)
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

JSON_INIT_FILE_PATH_AZURE = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-azure-expectations.json')
MOCKSERVER_PORT_AZURE = 1095

JSON_INIT_FILE_PATH_SUMOLOGIC = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-sumologic-expectations.json')
MOCKSERVER_PORT_SUMOLOGIC = 1096

JSON_INIT_FILE_PATH_REGISTRY = os.path.join(REPOSITORY_ROOT, 'test', 'mockserver', 'expectations', 'static-registry-expectations.json')

PG_SRV_PORT_MTLS = 5476
PG_SRV_PORT_MTLS_WITH_NAMESPACES = 5486
PG_SRV_PORT_MTLS_WITH_EAGER_GC = 5496
PG_SRV_PORT_UNENCRYPTED = 5477

PG_SRV_PORT_DOCKER_MTLS = 5576
PG_SRV_PORT_DOCKER_MTLS_WITH_NAMESPACES = 5586
PG_SRV_PORT_DOCKER_MTLS_WITH_EAGER_GC = 5596
PG_SRV_PORT_DOCKER_UNENCRYPTED = 5577

PSQL_EXE :str = os.environ.get('PSQL_EXE', 'psql')

PSQL_CLIENT_HOST :str = "127.0.0.1"

CREATE_DISKS_VIEW_NO_PRIMARY_ALIAS = "create view cross_cloud_disks_not_aliased as select 'aws' as vendor, volumeId as name, volumeType as type, status, size from aws.ec2.volumes where region = 'ap-southeast-2' union select 'google' as vendor, name, split_part(split_part(type, '/', 11), '-', 2) as type, status, sizeGb as size from google.compute.disks where project = 'testing-project' and zone = 'australia-southeast1-a' ;"
CREATE_DISKS_VIEW_PRIMARY_ALIAS = "create view cross_cloud_disks_aliased as select 'google' as vendor, name, split_part(split_part(type, '/', 11), '-', 2) as type, status, sizeGb as size from google.compute.disks where project = 'testing-project' and zone = 'australia-southeast1-a' union select 'aws' as vendor, volumeId as name, volumeType as type, status, size from aws.ec2.volumes where region = 'ap-southeast-2' ;"

PSQL_MTLS_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH} sslkey={STACKQL_PG_CLIENT_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"
PSQL_MTLS_CONN_STR_UNIX :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH_UNIX} sslkey={STACKQL_PG_CLIENT_KEY_PATH_UNIX} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_UNIX} dbname=mydatabase"

PSQL_MTLS_CONN_STR_UNIX_WITH_NAMESPACES :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS_WITH_NAMESPACES} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH_UNIX} sslkey={STACKQL_PG_CLIENT_KEY_PATH_UNIX} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_UNIX} dbname=mydatabase"
PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS_WITH_EAGER_GC} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH_UNIX} sslkey={STACKQL_PG_CLIENT_KEY_PATH_UNIX} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_UNIX} dbname=mydatabase"
PSQL_MTLS_INVALID_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_RUBBISH_CERT_PATH} sslkey={STACKQL_PG_RUBBISH_KEY_PATH} sslrootcert={STACKQL_PG_SERVER_CERT_PATH} dbname=mydatabase"

PSQL_UNENCRYPTED_CONN_STR :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_UNENCRYPTED} user=myuser dbname=mydatabase"
POSTGRES_URL_UNENCRYPTED_CONN :str = f"postgresql://myuser:mypass@{PSQL_CLIENT_HOST}:{PG_SRV_PORT_UNENCRYPTED}/mydatabase"

PSQL_MTLS_CONN_STR_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH_DOCKER} sslkey={STACKQL_PG_CLIENT_KEY_PATH_DOCKER} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_DOCKER} dbname=mydatabase"
PSQL_MTLS_CONN_STR_WITH_NAMESPACES_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS_WITH_NAMESPACES} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH_DOCKER} sslkey={STACKQL_PG_CLIENT_KEY_PATH_DOCKER} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_DOCKER} dbname=mydatabase"
PSQL_MTLS_CONN_STR_WITH_EAGER_GC_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS_WITH_EAGER_GC} user=myuser sslmode=verify-full sslcert={STACKQL_PG_CLIENT_CERT_PATH_DOCKER} sslkey={STACKQL_PG_CLIENT_KEY_PATH_DOCKER} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_DOCKER} dbname=mydatabase"
PSQL_MTLS_INVALID_CONN_STR_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_MTLS} user=myuser sslmode=verify-full sslcert={STACKQL_PG_RUBBISH_CERT_PATH_DOCKER} sslkey={STACKQL_PG_RUBBISH_KEY_PATH_DOCKER} sslrootcert={STACKQL_PG_SERVER_CERT_PATH_DOCKER} dbname=mydatabase"
PSQL_UNENCRYPTED_CONN_STR_DOCKER :str = f"host={PSQL_CLIENT_HOST} port={PG_SRV_PORT_UNENCRYPTED} user=myuser dbname=mydatabase"
POSTGRES_URL_UNENCRYPTED_CONN_DOCKER :str = f"postgresql://myuser:mypass@{PSQL_CLIENT_HOST}:{PG_SRV_PORT_UNENCRYPTED}/mydatabase"

SELECT_CONTAINER_SUBNET_AGG_DESC = "select ipCidrRange, sum(5) cc  from  google.container.\"projects.aggregated.usableSubnetworks\" where projectsId = 'testing-project' group by ipCidrRange having sum(5) >= 5 order by ipCidrRange desc;"
SELECT_CONTAINER_SUBNET_AGG_ASC = "select ipCidrRange, sum(5) cc  from  google.container.\"projects.aggregated.usableSubnetworks\" where projectsId = 'testing-project' group by ipCidrRange having sum(5) >= 5 order by ipCidrRange asc;"
SELECT_ACCELERATOR_TYPES_DESC = "select  kind, name, maximumCardsPerInstance  from  google.compute.acceleratorTypes where project = 'testing-project' and zone = 'australia-southeast1-a' order by name desc;"
SELECT_ACCELERATOR_TYPES_DESC_FROM_INTEL_VIEWS = "select  kind, name  from  stackql_intel.\"google.compute.acceleratorTypes\" where project = 'testing-project' and zone like '%%australia-southeast1-a' order by name desc;"
SELECT_ACCELERATOR_TYPES_DESC_FROM_INTEL_VIEWS_SUBQUERY = "SELECT name AS name, count(kind) AS \"COUNT(kind)\" FROM (SELECT *    from stackql_intel.\"google.compute.acceleratorTypes\"    limit 80) AS virtual_table GROUP BY name ORDER BY \"COUNT(kind)\" DESC LIMIT 1000;"
SELECT_MACHINE_TYPES_DESC = "select name from google.compute.machineTypes where project = 'testing-project' and zone = 'australia-southeast1-a' order by name desc;"
SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY = "SELECT etag FROM google.compute.instances_iam_policies WHERE project = 'testing-project' AND zone = 'australia-southeast1-a' AND resource = '000000001';"

SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL = "SELECT DISTINCT EventTime, Identifier from aws.cloud_control.resource_requests where data__ResourceRequestStatusFilter='{}' and region = 'ap-southeast-1' order by Identifier, EventTime;"

SELECT_AZURE_COMPUTE_PUBLIC_KEYS = "select id, location from azure.compute.ssh_public_keys where subscriptionId = '10001000-1000-1000-1000-100010001000' ORDER BY id ASC;"
SELECT_AZURE_COMPUTE_VIRTUAL_MACHINES = "SELECT id, name FROM azure.compute.virtual_machines WHERE resourceGroupName = 'stackql-ops-cicd-dev-01' AND subscriptionId = '10001000-1000-1000-1000-100010001000' ORDER BY name ASC;"

SHOW_TRANSACTION_ISOLATION_LEVEL = "show transaction isolation level"
SELECT_HSTORE_DETAILS = "SELECT t.oid, typarray FROM pg_type t JOIN pg_namespace ns ON typnamespace = ns.oid WHERE typname = 'hstore'"

SHOW_TRANSACTION_ISOLATION_LEVEL_JSON_EXPECTED = [{"transaction_isolation": "read committed"}]
SELECT_HSTORE_DETAILS_JSON_EXPECTED = []

SHOW_TRANSACTION_ISOLATION_LEVEL_TUPLES_EXPECTED = [("read committed",)]
SELECT_HSTORE_DETAILS_TUPLES_EXPECTED = []

SELECT_POSTGRES_CATALOG_JOIN_TUPLE_EXPECTED = ("__iql__.control.gc.rings",)

SELECT_AZURE_COMPUTE_PUBLIC_KEYS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'azure', 'compute', 'ssh-public-keys-list.txt'))
SELECT_AZURE_COMPUTE_VIRTUAL_MACHINES_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'azure', 'compute', 'vm-list.txt'))

SELECT_EXTERNAL_INFORMATION_SCHEMA_ORDERED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'external_sources', 'select_information_schema_single_table_ordered.txt'))
SELECT_EXTERNAL_INFORMATION_SCHEMA_FILTERED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'external_sources', 'select_information_schema_single_table_filtered.txt'))
SELECT_EXTERNAL_INFORMATION_SCHEMA_INNER_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'external_sources', 'select_information_schema_inner_join.txt'))

SELECT_AZURE_COMPUTE_PUBLIC_KEYS_JSON_EXPECTED = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'azure', 'compute', 'ssh-public-keys-list.json'))
SELECT_AZURE_COMPUTE_VIRTUAL_MACHINES_JSON_EXPECTED = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'azure', 'compute', 'vm-list.json'))
SELECT_AZURE_COMPUTE_BILLING_ACCOUNTS_JSON_EXPECTED = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'azure', 'billing', 'billing-account-list.json'))

SELECT_AWS_S3_BUCKET_LOCATIONS = "select LocationConstraint from aws.s3.bucket_locations where region = 'ap-southeast-1' and bucket = 'stackql-trial-bucket-01';"
SELECT_AWS_S3_BUCKETS = "select Name, CreationDate from  aws.s3.buckets where region = 'ap-southeast-1' order by Name ASC;"
SELECT_AWS_S3_OBJECTS = "select \"Key\", Size, StorageClass from  aws.s3.objects where region = 'ap-southeast-1' and bucket = 'stackql-trial-bucket-01' order by \"Key\" ASC;"
SELECT_AWS_S3_OBJECTS_NULL = "select \"Key\", Size, StorageClass from  aws.s3.objects where region = 'ap-southeast-2' and bucket = 'stackql-trial-bucket-02' order by \"Key\" ASC;"
SELECT_AWS_EC2_VPN_GATEWAYS_NULL = "select vpnGatewayId, amazonSideAsn from aws.ec2.vpn_gateways where region = 'ap-southeast-1' order by vpnGatewayId ASC;"
SELECT_AWS_VOLUMES = "select VolumeId, Encrypted, Size from aws.ec2.volumes where region = 'ap-southeast-1' order by VolumeId asc;"
SELECT_AWS_IAM_USERS_ASC = "select UserName, Arn from aws.iam.users WHERE region = 'us-east-1' order by UserName ASC;"
CREATE_AWS_VOLUME = """insert into aws.ec2.volumes(AvailabilityZone, Size, region, TagSpecification) select 'ap-southeast-1a', JSON(10), 'ap-southeast-1', JSON('[ { "ResourceType": "volume", "Tag": [ { "Key": "stack", "Value": "production" }, { "Key": "name", "Value": "multi-tag-volume" } ] } ]');"""
CREATE_AWS_CLOUD_CONTROL_LOG_GROUP = """insert into aws.cloud_control.resources(region, data__TypeName, data__DesiredState) select 'ap-southeast-1', 'AWS::Logs::LogGroup', string('{ "LogGroupName": "LogGroupResourceExampleThird", "RetentionInDays":90}');"""
SELECT_AWS_CLOUD_CONTROL_VPCS_DESC = "select Identifier, Properties from aws.cloud_control.resources where region = 'ap-southeast-1' and data__TypeName = 'AWS::EC2::VPC' order by Identifier desc;"
SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION = "SELECT JSON_EXTRACT(Properties, '$.Arn') as Arn FROM aws.cloud_control.resources WHERE region = 'ap-southeast-2' and data__TypeName = 'AWS::S3::Bucket' and data__Identifier = 'stackql-trial-bucket-01';"
SELECT_AWS_CLOUD_CONTROL_BUCKET_VIEW_PROJECTION = "select Arn from aws.pseudo_s3.s3_bucket_listing where data__Identifier = 'stackql-trial-bucket-01' ;"
SELECT_AWS_CLOUD_CONTROL_BUCKET_VIEW_STAR = "select * from aws.pseudo_s3.s3_bucket_listing where data__Identifier = 'stackql-trial-bucket-01' ;"
SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION_DEFECTIVE = "SELECT JSON_EXTRACT(Arn, '$.Properties') as Arn FROM aws.cloud_control.resources WHERE region = 'ap-southeast-2' and data__TypeName = 'AWS::S3::Bucket' and data__Identifier = 'stackql-trial-bucket-01';"
GET_AWS_CLOUD_CONTROL_VPCS_DESC = "select Identifier, Properties from aws.cloud_control.resources where region = 'ap-southeast-1' and data__TypeName = 'AWS::EC2::VPC' and data__Identifier = 'CloudControlExample';"
GET_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP = """select TypeName, OperationStatus, StatusMessage, Identifier, RequestToken from aws.cloud_control.resource_requests where data__RequestToken = 'abc001' and region = 'ap-southeast-1';"""
SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC = "select TypeName, OperationStatus, StatusMessage, Identifier, RequestToken from aws.cloud_control.resource_requests where data__ResourceRequestStatusFilter='{}' and region = 'ap-southeast-1' order by RequestToken desc;"
UPDATE_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP = """update aws.cloud_control.resources set data__PatchDocument = string('[{"op":"replace","path":"/RetentionInDays","value":180}]') WHERE region = 'ap-southeast-1' AND data__TypeName = 'AWS::Logs::LogGroup' AND data__Identifier = 'LogGroupResourceExampleThird';"""
UPDATE_AWS_EC2_VOLUME = "update aws.ec2.volumes set Size = 12 WHERE region = 'ap-southeast-1' AND VolumeId = 'vol-000000000000001';"

UPDATE_GITHUB_ORG = "update github.orgs.orgs set data__description = 'Some silly description.' WHERE  org = 'dummyorg';"

SELECT_GITHUB_REPOS_PAGES_SINGLE = "select url from github.repos.pages where owner = 'dummyorg' and repo = 'dummyapp.io';"
SELECT_GITHUB_REPOS_IDS_ASC = "select id from github.repos.repos where org = 'dummyorg' order by id ASC;"
SELECT_GITHUB_BRANCHES_NAMES_DESC = "select name from github.repos.branches where owner = 'dummyorg' and repo = 'dummyapp.io' order by name desc;"
SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS = "select name, split_part(teams_url, '/', 4) as extracted_team, regexp_replace((JSON_EXTRACT(owner, '$.url')), '^https://[^/]+/[^/]+/', 'username = ') as user_suffix, nlike('%docusaurus%', name) as is_docusaurus, unicode_version() as unicode_lib_version from github.repos.repos where org = 'dummyorg' order by name ASC;"
SELECT_GITHUB_REPOS_FILTERED_SINGLE = "select id, name from github.repos.repos where org = 'dummyorg' and name = 'dummyapp.io';"
SELECT_GITHUB_SCIM_USERS = "select JSON_EXTRACT(name, '$.givenName') || ' ' || JSON_EXTRACT(name, '$.familyName') as name, userName, externalId, id from github.scim.users where org = 'dummyorg' order by id asc;"
SELECT_GITHUB_SAML_IDENTITIES = "select guid, JSON_EXTRACT(samlIdentity, '$.nameId') AS saml_id, JSON_EXTRACT(user, '$.login') AS github_login from github.scim.saml_ids where org = 'dummyorg' order by JSON_EXTRACT(user, '$.login') asc;"
SELECT_GITHUB_TAGS_COUNT = "select count(*) as ct from github.repos.tags where owner = 'dummyorg' and repo = 'dummyapp.io';"
SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS = "select substr(su.userName, 1, instr(su.userName, '@') - 1), su.externalId, su.id, u.login, u.two_factor_authentication AS is_two_fa_enabled from github.scim.users su inner join github.users.users u ON substr(su.userName, 1, instr(su.userName, '@') - 1) = u.username and substr(su.userName, 1, instr(su.userName, '@') - 1) = u.login where su.org = 'dummyorg' order by su.id asc;"
SELECT_GITHUB_ORGS_MEMBERS = "select om.login from github.orgs.members om where om.org = 'dummyorg' order by om.login desc;"
SELECT_GITHUB_JOIN_IN_PARAMS = "select r.name, col.login, col.type, col.role_name from github.repos.collaborators col inner join github.repos.repos r ON col.repo = r.name where col.owner = 'dummyorg' and r.org = 'dummyorg' order by r.name, col.login desc;"
SELECT_GITHUB_JOIN_IN_PARAMS_SPECIALCASE = "select r.id, r.name, col.login, col.type, col.role_name from github.repos.collaborators col inner join github.repos.repos r ON col.repo = r.name where col.owner = 'specialcaseorg' and r.org = 'specialcaseorg' order by r.name, col.login desc;"

SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_SIMPLE = "select r.name, col.login, col.type, col.role_name from stackql_analytics_github.repos.collaborators col inner join stackql_analytics_github.repos.repos r ON col.repo = r.name where col.owner = 'stackql' and r.org = 'stackql' order by r.name, col.login desc;"
SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_TRANSPARENT = "select r.name, col.login, col.type, col.role_name from github.repos.collaborators col inner join github.repos.repos r ON col.repo = r.name where col.owner = 'stackql' and r.org = 'stackql' order by r.name, col.login desc;"

SELECT_OKTA_APPS = "select name, status, label, id from okta.application.apps apps where apps.subdomain = 'example-subdomain' order by name asc;"
SELECT_OKTA_USERS_ASC = "select JSON_EXTRACT(ou.profile, '$.login') as login, ou.status from okta.user.users ou WHERE ou.subdomain = 'dummyorg' order by JSON_EXTRACT(ou.profile, '$.login') asc;"

PURGE_CONSERVATIVE = "PURGE CONSERVATIVE;"

PURGE_CONSERVATIVE_RESPONSE_JSON = [{'message': "PURGE of type 'conservative' successfully completed"}]

_SHOW_INSERT_GOOGLE_BIGQUERY_DATASET = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'simple-templating', 'insert-bigquery-datasets.iql'))
_SHOW_INSERT_GOOGLE_CONTAINER_CLUSTERS = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'simple-templating', 'insert-container-clusters.iql'))

def get_native_query_row_count_from_table(table_name :str, sql_backend_str :str) -> str:
  if sql_backend_str == 'postgres_tcp':
    return f"NATIVEQUERY 'SELECT COUNT(*) as object_count FROM \"{_PG_SCHEMA_PHYSICAL_TABLES}\".\"{table_name}\"' ;"
  return f"NATIVEQUERY 'SELECT COUNT(*) as object_count FROM \"{table_name}\"' ;"


def get_native_table_count_by_name(table_name :str, sql_backend_str :str) -> str:
  return f"NATIVEQUERY 'SELECT COUNT(*) as object_count FROM sqlite_master where type = 'table' and name = '{table_name}' ;"


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
SELECT_OKTA_APPS_ASC_EXPECTED_JSON = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'simple-select', 'okta', 'apps', 'select-apps-asc.json'))
SELECT_OKTA_USERS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'okta', 'select-users-asc.txt'))


SELECT_SOME_VIEW_EXPECTED_JSON = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'views', 'select-some-view.json'))

SELECT_CROSS_CLOUD_DISKS_VIEW_EXPECTED_JSON = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'views', 'select-cross-cloud-disks.json'))

SELECT_POSTGRES_CATALOG_JOIN = "SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relkind in ('r', 'p');"

DELETE_AWS_CLOUD_CONTROL_LOG_GROUP = "delete from aws.cloud_control.resources where region = 'ap-southeast-1' and data__TypeName = 'AWS::Logs::LogGroup' and data__Identifier = 'LogGroupResourceExampleThird';"

SELECT_AWS_VOLUMES_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'ec2', 'select-volumes-asc.txt'))
SELECT_AWS_EC2_VPN_GATEWAYS_NULL_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'ec2', 'select-vpn-gateways-empty.txt'))
SELECT_AWS_IAM_USERS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'iam', 'select-users-asc.txt'))
SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-list-vpcs-desc.txt'))
GET_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-get-vpcs-desc.txt'))
SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_JSON_EXPECTED = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-list-vpcs-desc.json'))
SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-bucket-detail-projection.txt'))
SELECT_AWS_CLOUD_CONTROL_BUCKET_VIEW_STAR_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-bucket-detail-star.txt'))
SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION_JSON_EXPECTED = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-bucket-detail-projection.json'))
GET_AWS_CLOUD_CONTROL_VPCS_DESC_JSON_EXPECTED = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-get-vpcs-desc.json'))
SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-list-operations-desc.txt'))
GET_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-get-operation-desc.txt'))
SELECT_AWS_S3_OBJECTS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 's3', 'select-objects.txt'))
SELECT_AWS_S3_OBJECTS_NULL_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 's3', 'select-objects-empty.txt'))
SELECT_AWS_S3_BUCKETS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 's3', 'select-buckets.txt'))
SELECT_AWS_S3_BUCKET_LOCATIONS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 's3', 'select-bucket-locations.txt'))
VIEW_SELECT_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-bucket-detail.txt'))
VIEW_SELECT_STAR_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-star-bucket-detail.txt'))
AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_FILTERED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-projection-bucket-view-response-filtered-only.txt'))
AWS_CC_VIEW_SELECT_STAR_BUCKET_FILTERED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-star-bucket-view-response-filtered-only.txt'))
AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_COMPLEX_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-projection-bucket-view-parameter-pushed-and-response-filtered.txt'))
AWS_CC_VIEW_SELECT_STAR_BUCKET_COMPLEX_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aws', 'cloud_control', 'select-star-bucket-view-parameter-pushed-and-response-filtered.txt'))

SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-pages.txt'))
SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-ids-asc.txt'))
SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-functions.txt'))
SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-repos-single-filtered.txt'))
SELECT_GITHUB_SCIM_USERS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'scim', 'select-github-scim-users.txt'))
SELECT_GITHUB_SAML_IDENTITIES_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'scim', 'select-github-saml-identities.txt'))
SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-branches-names-desc.txt'))
SELECT_GITHUB_TAGS_COUNT_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'repos', 'select-github-tags-count.txt'))
SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'joins', 'select-github-sequential-join.txt'))
SELECT_GITHUB_JOIN_IN_PARAMS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'joins', 'select-github-join-on-path-param.txt'))
SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'joins', 'select-github-sequential-join-with-functions.txt'))
SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'joins', 'analytics-repositories-collaborators.txt'))
SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_SPECIALCASE_JSON_EXPECTED = get_json_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'joins', 'specialcase-firstlook-repositories-collaborators.json'))
SELECT_GITHUB_OKTA_SAML_JOIN_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'joins', 'inner', 'github-saml-members-okta-users.txt'))
SELECT_GITHUB_ORGS_MEMBERS_PAGE_LIMITED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'github', 'orgs', 'page-limited-members.txt'))
SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'compute', 'instance-iam-policy-projection.txt'))

SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'show', 'show-insert-google-iam-service-accounts.txt'))
SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'show', 'show-insert-google-compute-instances-iam-error.txt'))

SELECT_SUMOLOGIC_COLLECTORS_IDS = 'select id from sumologic.collectors.collectors order by id desc;'
SELECT_SUMOLOGIC_COLLECTORS_IDS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'sumologic', 'select-collectors-desc.txt'))

GET_IAM_POLICY_AGG_ASC_INPUT_FILE = os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'input', 'select-exec-dependent-org-iam-policy.iql')
GET_IAM_POLICY_AGG_ASC_INPUT_FILE_DOCKER = os.path.join('/opt', 'stackql', 'input', 'select-exec-dependent-org-iam-policy.iql')

_FILE_QUERY_PARSER_TEST_POSTGRES_CASTING = os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'input', 'parser-testing', 'postgres-casting-query.sql')
_FILE_QUERY_PARSER_TEST_KEYWORD_QUOTING = os.path.join(REPOSITORY_ROOT, 'test', 'assets', 'input', 'parser-testing', 'keyword-quoting-query.sql')

_QUERY_PARSER_TEST_POSTGRES_CASTING = get_output_from_local_file(_FILE_QUERY_PARSER_TEST_POSTGRES_CASTING)
_QUERY_PARSER_TEST_KEYWORD_QUOTING = get_output_from_local_file(_FILE_QUERY_PARSER_TEST_KEYWORD_QUOTING)

GET_IAM_POLICY_AGG_ASC_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'aggregated-select', 'google', 'cloudresourcemanager', 'select-exec-getiampolicy-agg.csv'))

SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'show', 'show-methods-github-repo-repo.txt'))

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'cloudresourcemanager', 'projects-getiampolicy-roles-asc.txt'))

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' and role like '%owner' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED = "SELECT role, members, condition from google.cloudresourcemanager.project_iam_policies where projectsId = 'testproject' and role = 'roles/owner' order by role asc;"

SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'cloudresourcemanager', 'projects-getiampolicy-roles-asc-filtered.txt'))

SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS :bytes =  b"""SELECT i.zone, i.name, i.machineType, i.deletionProtection, '[{"subnetwork":"' || JSON_EXTRACT(i.networkInterfaces, '$[0].subnetwork') || '"}]', '[{"boot": true, "initializeParams": { "diskSizeGb": "' || JSON_EXTRACT(i.disks, '$[0].diskSizeGb') || '", "sourceImage": "' || d.sourceImage || '"}}]', i.labels FROM google.compute.instances i INNER JOIN google.compute.disks d ON i.name = d.name WHERE i.project = 'testing-project' AND i.zone = 'australia-southeast1-a' AND d.project = 'testing-project' AND d.zone = 'australia-southeast1-a' AND i.name LIKE '%' order by i.name DESC;"""

SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED = get_output_from_local_file(os.path.join('test', 'assets', 'expected', 'google', 'joins', 'disks-instances-rewritten.txt'))

_CREATE_SOME_VIEW = "create or replace view some_view as select id, name, url from github.repos.repos where org = 'stackql' order by name;"

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


def get_db_setup_src(sql_backend_str :str) -> str:
  if sql_backend_str == 'postgres_tcp':
    return './test/db/postgres'
  return './test/db/sqlite'



def get_variables(execution_env :str, sql_backend_str :str):
  NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_ONE = get_native_query_row_count_from_table('okta.application.apps.Application.generation_1', sql_backend_str)
  NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_TWO = get_native_query_row_count_from_table('okta.application.apps.Application.generation_2', sql_backend_str)
  rv = {
    ## general config
    'AZURE_SECRET_STR':                               AZURE_SECRET_STR,
    'BUILDMAJORVERSION':                              _BUILD_MAJOR_VERSION,
    'BUILDMINORVERSION':                              _BUILD_MINOR_VERSION,
    'BUILDPATCHVERSION':                              _BUILD_PATCH_VERSION,
    'DB_INTERNAL_CFG_LAX':                            DB_INTERNAL_CFG_LAX,
    'DB_SETUP_SRC':                                   get_db_setup_src(sql_backend_str),
    'GC_CFG_EAGER':                                   _GC_CFG_EAGER,
    'GITHUB_SECRET_STR':                              GITHUB_SECRET_STR,
    'IS_WINDOWS':                                     IS_WINDOWS,
    'K8S_SECRET_STR':                                 K8S_SECRET_STR,
    'MOCKSERVER_JAR':                                 MOCKSERVER_JAR,
    'MOCKSERVER_PORT_AWS':                            MOCKSERVER_PORT_AWS,
    'MOCKSERVER_PORT_AZURE':                          MOCKSERVER_PORT_AZURE,
    'MOCKSERVER_PORT_GITHUB':                         MOCKSERVER_PORT_GITHUB,
    'MOCKSERVER_PORT_GOOGLE':                         MOCKSERVER_PORT_GOOGLE,
    'MOCKSERVER_PORT_K8S':                            MOCKSERVER_PORT_K8S,
    'MOCKSERVER_PORT_OKTA':                           MOCKSERVER_PORT_OKTA,
    'MOCKSERVER_PORT_REGISTRY':                       MOCKSERVER_PORT_REGISTRY,
    'MOCKSERVER_PORT_SUMOLOGIC':                      MOCKSERVER_PORT_SUMOLOGIC,
    'NAMESPACES_TTL_SIMPLE':                          NAMESPACES_TTL_SIMPLE,
    'NAMESPACES_TTL_SPECIALCASE_TRANSPARENT':         NAMESPACES_TTL_SPECIALCASE_TRANSPARENT,
    'NAMESPACES_TTL_TRANSPARENT':                     NAMESPACES_TTL_TRANSPARENT,
    'OKTA_SECRET_STR':                                OKTA_SECRET_STR,
    'PG_SRV_MTLS_DOCKER_CFG_STR':                     PG_SRV_MTLS_DOCKER_CFG_STR,
    'PG_SRV_PORT_DOCKER_MTLS':                        PG_SRV_PORT_DOCKER_MTLS,
    'PG_SRV_PORT_DOCKER_MTLS_WITH_EAGER_GC':          PG_SRV_PORT_DOCKER_MTLS_WITH_EAGER_GC,
    'PG_SRV_PORT_DOCKER_MTLS_WITH_NAMESPACES':        PG_SRV_PORT_DOCKER_MTLS_WITH_NAMESPACES,
    'PG_SRV_PORT_DOCKER_UNENCRYPTED':                 PG_SRV_PORT_DOCKER_UNENCRYPTED,
    'PG_SRV_PORT_MTLS':                               PG_SRV_PORT_MTLS,
    'PG_SRV_PORT_MTLS_WITH_EAGER_GC':                 PG_SRV_PORT_MTLS_WITH_EAGER_GC,
    'PG_SRV_PORT_MTLS_WITH_NAMESPACES':               PG_SRV_PORT_MTLS_WITH_NAMESPACES,
    'PG_SRV_PORT_UNENCRYPTED':                        PG_SRV_PORT_UNENCRYPTED,
    'POSTGRES_URL_UNENCRYPTED_CONN':                  POSTGRES_URL_UNENCRYPTED_CONN,
    'PSQL_CLIENT_HOST':                               PSQL_CLIENT_HOST,
    'PSQL_EXE':                                       PSQL_EXE,
    'REGISTRY_ROOT_CANONICAL':                        _REGISTRY_CANONICAL,
    'REGISTRY_ROOT_DEPRECATED':                       _REGISTRY_DEPRECATED,
    'REGISTRY_CANONICAL_CFG_STR':                     _REGISTRY_CANONICAL,
    'REGISTRY_CANONICAL_NO_VERIFY_CFG_STR':           _REGISTRY_CANONICAL_NO_VERIFY,
    'REGISTRY_DEPRECATED_CFG_STR':                    _REGISTRY_DEPRECATED,
    'REGISTRY_MOCKED_CFG_STR':                        get_registry_mocked(execution_env),
    'REGISTRY_NO_VERIFY_CFG_STR':                     _REGISTRY_NO_VERIFY,
    'REGISTRY_NULL':                                  _REGISTRY_NULL,
    'REPOSITORY_ROOT':                                REPOSITORY_ROOT,
    'SQL_BACKEND_CFG_STR_ANALYTICS':                  get_analytics_sql_backend(execution_env, sql_backend_str),
    'SQL_BACKEND_CFG_STR_CANONICAL':                  get_canonical_sql_backend(execution_env, sql_backend_str),
    'STACKQL_EXE':                                    STACKQL_EXE,
    'SUMOLOGIC_SECRET_STR':                           SUMOLOGIC_SECRET_STR,
    ## queries and expectations
    'AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_COMPLEX_EXPECTED':                  AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_COMPLEX_EXPECTED,
    'AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_FILTERED_EXPECTED':                 AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_FILTERED_EXPECTED,
    'AWS_CC_VIEW_SELECT_STAR_BUCKET_COMPLEX_EXPECTED':                        AWS_CC_VIEW_SELECT_STAR_BUCKET_COMPLEX_EXPECTED,
    'AWS_CC_VIEW_SELECT_STAR_BUCKET_FILTERED_EXPECTED':                       AWS_CC_VIEW_SELECT_STAR_BUCKET_FILTERED_EXPECTED,
    'AWS_CLOUD_CONTROL_METHOD_SIGNATURE_CMD_ARR':                             [ SELECT_AWS_CLOUD_CONTROL_VPCS_DESC, GET_AWS_CLOUD_CONTROL_VPCS_DESC ],
    'AWS_CLOUD_CONTROL_METHOD_SIGNATURE_CMD_ARR_EXPECTED':                    SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_JSON_EXPECTED + GET_AWS_CLOUD_CONTROL_VPCS_DESC_JSON_EXPECTED,
    'AWS_CLOUD_CONTROL_BUCKET_DETAIL_PROJECTION_DEFECTIVE_CMD_ARR':           [ SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION_DEFECTIVE, SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION ],
    'AWS_CLOUD_CONTROL_BUCKET_DETAIL_PROJECTION_DEFECTIVE_CMD_ARR_EXPECTED':  SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION_JSON_EXPECTED,
    'AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_PROJECTION':                        SELECT_AWS_CLOUD_CONTROL_BUCKET_VIEW_PROJECTION,
    'AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_PROJECTION_EXPECTED':               SELECT_AWS_CLOUD_CONTROL_BUCKET_PROJECTION_EXPECTED,
    'AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_STAR':                              SELECT_AWS_CLOUD_CONTROL_BUCKET_VIEW_STAR,
    'AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_STAR_EXPECTED':                     SELECT_AWS_CLOUD_CONTROL_BUCKET_VIEW_STAR_EXPECTED,
    'AZURE_VM_SIZES_ENUMERATION':                                             _AZURE_VM_SIZES_ENUMERATION,
    'CREATE_AWS_VOLUME':                                                      CREATE_AWS_VOLUME,
    'CREATE_AWS_CLOUD_CONTROL_LOG_GROUP':                                     CREATE_AWS_CLOUD_CONTROL_LOG_GROUP,
    'DELETE_AWS_CLOUD_CONTROL_LOG_GROUP':                                     DELETE_AWS_CLOUD_CONTROL_LOG_GROUP,
    'DESCRIBE_AWS_EC2_INSTANCES':                                             DESCRIBE_AWS_EC2_INSTANCES,
    'DESCRIBE_AWS_EC2_DEFAULT_KMS_KEY_ID':                                    DESCRIBE_AWS_EC2_DEFAULT_KMS_KEY_ID,
    'DESCRIBE_GITHUB_REPOS_PAGES':                                            DESCRIBE_GITHUB_REPOS_PAGES,
    'GET_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP':                                GET_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP,
    'GET_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP_EXPECTED':                       GET_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP_EXPECTED,
    'GET_AWS_CLOUD_CONTROL_VPCS_DESC':                                        GET_AWS_CLOUD_CONTROL_VPCS_DESC,
    'GET_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED':                               GET_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED,
    'GET_IAM_POLICY_AGG_ASC_EXPECTED':                                        GET_IAM_POLICY_AGG_ASC_EXPECTED,
    'PG_CLIENT_SETUP_QUERIES':                                                [ SHOW_TRANSACTION_ISOLATION_LEVEL, SELECT_HSTORE_DETAILS ],
    'PG_CLIENT_SETUP_QUERIES_JSON_EXPECTED':                                  SHOW_TRANSACTION_ISOLATION_LEVEL_JSON_EXPECTED + SELECT_HSTORE_DETAILS_JSON_EXPECTED,
    'PG_CLIENT_SETUP_QUERIES_TUPLES_EXPECTED':                                SHOW_TRANSACTION_ISOLATION_LEVEL_TUPLES_EXPECTED + SELECT_HSTORE_DETAILS_TUPLES_EXPECTED,
    'QUERY_PARSER_TEST_KEYWORD_QUOTING':                                      _QUERY_PARSER_TEST_KEYWORD_QUOTING,
    'QUERY_PARSER_TEST_POSTGRES_CASTING':                                     _QUERY_PARSER_TEST_POSTGRES_CASTING,
    'REGISTRY_GOOGLE_PROVIDER_LIST':                                          REGISTRY_GOOGLE_PROVIDER_LIST,
    'REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED':                                 REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED,
    'REGISTRY_LIST':                                                          REGISTRY_LIST,
    'REGISTRY_LIST_EXPECTED':                                                 REGISTRY_LIST_EXPECTED,
    'SELECT_ACCELERATOR_TYPES_DESC':                                          SELECT_ACCELERATOR_TYPES_DESC,
    'SELECT_ACCELERATOR_TYPES_DESC_EXPECTED':                                 SELECT_ACCELERATOR_TYPES_DESC_EXPECTED,
    'SELECT_ACCELERATOR_TYPES_DESC_SEQUENCE':                                 [ SELECT_ACCELERATOR_TYPES_DESC, SELECT_ACCELERATOR_TYPES_DESC_FROM_INTEL_VIEWS, SELECT_ACCELERATOR_TYPES_DESC_FROM_INTEL_VIEWS_SUBQUERY ],
    'SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_EXPECTED':      SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_EXPECTED,
    'SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_SIMPLE':        SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_SIMPLE,
    'SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_TRANSPARENT':   SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_TRANSPARENT,
    'SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL':                                [ SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL ],
    'SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED':                       SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED,
    'SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC':                               SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC,
    'SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC_EXPECTED':                      SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC_EXPECTED,
    'SELECT_AWS_CLOUD_CONTROL_VPCS_DESC':                                     SELECT_AWS_CLOUD_CONTROL_VPCS_DESC,
    'SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED':                            SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED,
    'SELECT_AWS_EC2_VPN_GATEWAYS_NULL':                                       SELECT_AWS_EC2_VPN_GATEWAYS_NULL,
    'SELECT_AWS_EC2_VPN_GATEWAYS_NULL_EXPECTED':                              SELECT_AWS_EC2_VPN_GATEWAYS_NULL_EXPECTED,
    'SELECT_AWS_IAM_USERS_ASC':                                               SELECT_AWS_IAM_USERS_ASC,
    'SELECT_AWS_IAM_USERS_ASC_EXPECTED':                                      SELECT_AWS_IAM_USERS_ASC_EXPECTED,
    'SELECT_AWS_S3_BUCKET_LOCATIONS':                                         SELECT_AWS_S3_BUCKET_LOCATIONS,
    'SELECT_AWS_S3_BUCKET_LOCATIONS_EXPECTED':                                SELECT_AWS_S3_BUCKET_LOCATIONS_EXPECTED,
    'SELECT_AWS_S3_BUCKETS':                                                  SELECT_AWS_S3_BUCKETS,
    'SELECT_AWS_S3_BUCKETS_EXPECTED':                                         SELECT_AWS_S3_BUCKETS_EXPECTED,
    'SELECT_AWS_S3_OBJECTS':                                                  SELECT_AWS_S3_OBJECTS,
    'SELECT_AWS_S3_OBJECTS_EXPECTED':                                         SELECT_AWS_S3_OBJECTS_EXPECTED,
    'SELECT_AWS_S3_OBJECTS_NULL':                                             SELECT_AWS_S3_OBJECTS_NULL,
    'SELECT_AWS_S3_OBJECTS_NULL_EXPECTED':                                    SELECT_AWS_S3_OBJECTS_NULL_EXPECTED,
    'SELECT_AWS_VOLUMES':                                                     SELECT_AWS_VOLUMES,
    'SELECT_AWS_VOLUMES_ASC_EXPECTED':                                        SELECT_AWS_VOLUMES_ASC_EXPECTED,
    'SELECT_CONTAINER_SUBNET_AGG_ASC':                                        SELECT_CONTAINER_SUBNET_AGG_ASC,
    'SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED':                               SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED,
    'SELECT_CONTAINER_SUBNET_AGG_DESC':                                       SELECT_CONTAINER_SUBNET_AGG_DESC,
    'SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED':                              SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED,
    'SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN':                        SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN,
    'SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN_EXPECTED':               SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN_EXPECTED,
    'SELECT_CONTRIVED_GCP_OKTA_JOIN':                                         SELECT_CONTRIVED_GCP_OKTA_JOIN,
    'SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED':                                SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED,
    'SELECT_CONTRIVED_GCP_SELF_JOIN':                                         SELECT_CONTRIVED_GCP_SELF_JOIN,
    'SELECT_CONTRIVED_GCP_SELF_JOIN_EXPECTED':                                SELECT_CONTRIVED_GCP_SELF_JOIN_EXPECTED,
    'SELECT_CONTRIVED_GCP_THREE_WAY_JOIN':                                    SELECT_CONTRIVED_GCP_THREE_WAY_JOIN,
    'SELECT_CONTRIVED_GCP_THREE_WAY_JOIN_EXPECTED':                           SELECT_CONTRIVED_GCP_THREE_WAY_JOIN_EXPECTED,
    'SELECT_CROSS_CLOUD_DISKS_VIEW_EXPECTED_JSON':                            SELECT_CROSS_CLOUD_DISKS_VIEW_EXPECTED_JSON,
    'SELECT_EXTERNAL_INFORMATION_SCHEMA_FILTERED_EXPECTED':                   SELECT_EXTERNAL_INFORMATION_SCHEMA_FILTERED_EXPECTED,
    'SELECT_EXTERNAL_INFORMATION_SCHEMA_INNER_JOIN_EXPECTED':                 SELECT_EXTERNAL_INFORMATION_SCHEMA_INNER_JOIN_EXPECTED,
    'SELECT_EXTERNAL_INFORMATION_SCHEMA_ORDERED_EXPECTED':                    SELECT_EXTERNAL_INFORMATION_SCHEMA_ORDERED_EXPECTED,
    'SELECT_GITHUB_BRANCHES_NAMES_DESC':                                      SELECT_GITHUB_BRANCHES_NAMES_DESC,
    'SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED':                             SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED,
    'SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL':                                SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL,
    'SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED':                       SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED,
    'SELECT_GITHUB_JOIN_IN_PARAMS':                                           SELECT_GITHUB_JOIN_IN_PARAMS,
    'SELECT_GITHUB_JOIN_IN_PARAMS_EXPECTED':                                  SELECT_GITHUB_JOIN_IN_PARAMS_EXPECTED,
    'SELECT_GITHUB_OKTA_SAML_JOIN':                                           SELECT_GITHUB_OKTA_SAML_JOIN,
    'SELECT_GITHUB_OKTA_SAML_JOIN_EXPECTED':                                  SELECT_GITHUB_OKTA_SAML_JOIN_EXPECTED,
    'SELECT_GITHUB_ORGS_MEMBERS':                                             SELECT_GITHUB_ORGS_MEMBERS,
    'SELECT_GITHUB_ORGS_MEMBERS_PAGE_LIMITED_EXPECTED':                       SELECT_GITHUB_ORGS_MEMBERS_PAGE_LIMITED_EXPECTED,
    'SELECT_GITHUB_REPOS_FILTERED_SINGLE':                                    SELECT_GITHUB_REPOS_FILTERED_SINGLE,
    'SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED':                           SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED,
    'SELECT_GITHUB_REPOS_IDS_ASC':                                            SELECT_GITHUB_REPOS_IDS_ASC,
    'SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED':                                   SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED,
    'SELECT_GITHUB_REPOS_PAGES_SINGLE':                                       SELECT_GITHUB_REPOS_PAGES_SINGLE,
    'SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED':                              SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED,
    'SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS':                              SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS,
    'SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS_EXPECTED':                     SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS_EXPECTED,
    'SELECT_GITHUB_SAML_IDENTITIES':                                          SELECT_GITHUB_SAML_IDENTITIES,
    'SELECT_GITHUB_SAML_IDENTITIES_EXPECTED':                                 SELECT_GITHUB_SAML_IDENTITIES_EXPECTED,
    'SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS':                                 SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS,
    'SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS_EXPECTED':                        SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS_EXPECTED,
    'SELECT_GITHUB_SCIM_USERS':                                               SELECT_GITHUB_SCIM_USERS,
    'SELECT_GITHUB_SCIM_USERS_EXPECTED':                                      SELECT_GITHUB_SCIM_USERS_EXPECTED,
    'SELECT_GITHUB_TAGS_COUNT':                                               SELECT_GITHUB_TAGS_COUNT,
    'SELECT_GITHUB_TAGS_COUNT_EXPECTED':                                      SELECT_GITHUB_TAGS_COUNT_EXPECTED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY':                           SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED':       SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED':                  SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED':         SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED,
    'SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED':             SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED,
    'SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY':                              SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY,
    'SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_EXPECTED':                     SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_EXPECTED,
    'SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS':                     SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS,
    'SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED':            SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED,
    'SELECT_K8S_NODES_ASC':                                                   get_select_k8s_nodes_asc(execution_env),
    'SELECT_K8S_NODES_ASC_EXPECTED':                                          SELECT_K8S_NODES_ASC_EXPECTED,
    'SELECT_MACHINE_TYPES_DESC':                                              SELECT_MACHINE_TYPES_DESC,
    'SELECT_MACHINE_TYPES_DESC_EXPECTED':                                     SELECT_MACHINE_TYPES_DESC_EXPECTED,
    'SELECT_OKTA_APPS':                                                       SELECT_OKTA_APPS,
    'SELECT_OKTA_APPS_ASC_EXPECTED':                                          SELECT_OKTA_APPS_ASC_EXPECTED,
    'SELECT_OKTA_USERS_ASC':                                                  SELECT_OKTA_USERS_ASC,
    'SELECT_OKTA_USERS_ASC_EXPECTED':                                         SELECT_OKTA_USERS_ASC_EXPECTED,
    'SELECT_POSTGRES_BACKEND_PID_ARR':                                        [ 'SELECT pg_backend_pid();' ],
    'SELECT_POSTGRES_CATALOG_JOIN_ARR':                                       [ SELECT_POSTGRES_CATALOG_JOIN ],
    'SELECT_POSTGRES_CATALOG_JOIN_TUPLE_EXPECTED':                            SELECT_POSTGRES_CATALOG_JOIN_TUPLE_EXPECTED,
    'SELECT_SUMOLOGIC_COLLECTORS_IDS':                                        SELECT_SUMOLOGIC_COLLECTORS_IDS,
    'SELECT_SUMOLOGIC_COLLECTORS_IDS_EXPECTED':                               SELECT_SUMOLOGIC_COLLECTORS_IDS_EXPECTED,
    'SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD':                            [ SELECT_AZURE_COMPUTE_VIRTUAL_MACHINES, SELECT_AZURE_COMPUTE_PUBLIC_KEYS ],
    'SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_EXPECTED':                   _SHELL_WELCOME_MSG + SELECT_AZURE_COMPUTE_VIRTUAL_MACHINES_EXPECTED + '\n' + SELECT_AZURE_COMPUTE_PUBLIC_KEYS_EXPECTED,
    'SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_JSON_EXPECTED':              SELECT_AZURE_COMPUTE_VIRTUAL_MACHINES_JSON_EXPECTED + SELECT_AZURE_COMPUTE_PUBLIC_KEYS_JSON_EXPECTED,
    'SHELL_COMMANDS_AZURE_BILLING_PATH_SPLIT_GUARD':                          [ "select name from azure.billing.accounts order by name desc;" ],
    'SHELL_COMMANDS_AZURE_BILLING_PATH_SPLIT_GUARD_JSON_EXPECTED':            SELECT_AZURE_COMPUTE_BILLING_ACCOUNTS_JSON_EXPECTED,
    'SHELL_COMMANDS_DISKS_VIEW_ALIASED_SEQUENCE':                             [ CREATE_DISKS_VIEW_PRIMARY_ALIAS, "select * from cross_cloud_disks_aliased order by name desc;", "drop view cross_cloud_disks_aliased;" ],
    'SHELL_COMMANDS_DISKS_VIEW_ALIASED_SEQUENCE_JSON_EXPECTED':               [ { "message": "DDL execution completed" } ] + SELECT_CROSS_CLOUD_DISKS_VIEW_EXPECTED_JSON + [ { "message": "DDL execution completed" } ],
    'SHELL_COMMANDS_DISKS_VIEW_NOT_ALIASED_SEQUENCE':                         [ CREATE_DISKS_VIEW_NO_PRIMARY_ALIAS, "select * from cross_cloud_disks_not_aliased order by name desc;", "drop view cross_cloud_disks_not_aliased;" ],
    'SHELL_COMMANDS_DISKS_VIEW_NOT_ALIASED_SEQUENCE_JSON_EXPECTED':           [ { "message": "DDL execution completed" } ] + SELECT_CROSS_CLOUD_DISKS_VIEW_EXPECTED_JSON + [ { "message": "DDL execution completed" } ],
    'SHELL_COMMANDS_GC_SEQUENCE_CANONICAL':                                   [ SELECT_OKTA_APPS, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_TWO, PURGE_CONSERVATIVE, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_TWO, SELECT_OKTA_APPS, SELECT_OKTA_APPS, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_TWO, PURGE_CONSERVATIVE, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_TWO ],
    'SHELL_COMMANDS_GC_SEQUENCE_CANONICAL_JSON_EXPECTED':                     SELECT_OKTA_APPS_ASC_EXPECTED_JSON + [ get_object_count_dict(5)] + PURGE_CONSERVATIVE_RESPONSE_JSON + [ get_object_count_dict(0) ] + SELECT_OKTA_APPS_ASC_EXPECTED_JSON + SELECT_OKTA_APPS_ASC_EXPECTED_JSON + [ get_object_count_dict(10)] + PURGE_CONSERVATIVE_RESPONSE_JSON + [get_object_count_dict(0) ],
    'SHELL_COMMANDS_GC_SEQUENCE_EAGER':                                       [ SELECT_OKTA_APPS, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_ONE, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_ONE, SELECT_OKTA_APPS, SELECT_OKTA_APPS, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_ONE, NATIVEQUERY_OKTA_APPS_ROW_COUNT_DISCO_ID_ONE ],
    'SHELL_COMMANDS_GC_SEQUENCE_EAGER_JSON_EXPECTED':                         SELECT_OKTA_APPS_ASC_EXPECTED_JSON + [ get_object_count_dict(0)] + [ get_object_count_dict(0) ] + SELECT_OKTA_APPS_ASC_EXPECTED_JSON + SELECT_OKTA_APPS_ASC_EXPECTED_JSON + [ get_object_count_dict(0)] + [get_object_count_dict(0) ],
    'SHELL_COMMANDS_SPECIALCASE_REPEATED_CACHED':                             [ SELECT_GITHUB_JOIN_IN_PARAMS_SPECIALCASE, SELECT_GITHUB_JOIN_IN_PARAMS_SPECIALCASE ],
    'SHELL_COMMANDS_SPECIALCASE_REPEATED_CACHED_JSON_EXPECTED':               SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_SPECIALCASE_JSON_EXPECTED + SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_SPECIALCASE_JSON_EXPECTED,
    'SHELL_COMMANDS_VIEW_HANDLING_SEQUENCE':                                  [ _CREATE_SOME_VIEW, "select * from some_view;", "drop view some_view;" ],
    'SHELL_COMMANDS_VIEW_HANDLING_SEQUENCE_JSON_EXPECTED':                    [ { "message": "DDL execution completed" } ] + SELECT_SOME_VIEW_EXPECTED_JSON + [ { "message": "DDL execution completed" } ],
    'SHELL_SESSION_SIMPLE_COMMANDS':                                          [ SELECT_GITHUB_BRANCHES_NAMES_DESC ],
    'SHELL_SESSION_SIMPLE_EXPECTED':                                          _SHELL_WELCOME_MSG + SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED,
    'SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR':                   SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR,
    'SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR':                   SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR,
    'SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR_EXPECTED':          SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR_EXPECTED,
    'SHOW_INSERT_GOOGLE_BIGQUERY_DATASET':                                    _SHOW_INSERT_GOOGLE_BIGQUERY_DATASET,
    'SHOW_INSERT_GOOGLE_CONTAINER_CLUSTERS':                                  _SHOW_INSERT_GOOGLE_CONTAINER_CLUSTERS,
    'SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS':                                SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS,
    'SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS':                                SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS,
    'SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS_EXPECTED':                       SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS_EXPECTED,
    'SHOW_METHODS_GITHUB_REPOS_REPOS':                                        SHOW_METHODS_GITHUB_REPOS_REPOS,
    'SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED':                               SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED,
    'SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR':                           SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR,
    'SHOW_OKTA_SERVICES_FILTERED_STR':                                        SHOW_OKTA_SERVICES_FILTERED_STR,
    'SHOW_PROVIDERS_STR':                                                     SHOW_PROVIDERS_STR,
    'UPDATE_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP':                             UPDATE_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP,
    'UPDATE_AWS_EC2_VOLUME':                                                  UPDATE_AWS_EC2_VOLUME,
    'UPDATE_GITHUB_ORG':                                                      UPDATE_GITHUB_ORG,
    'VIEW_SELECT_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED':                   VIEW_SELECT_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED,
    'VIEW_SELECT_STAR_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED':              VIEW_SELECT_STAR_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED,
  }
  if execution_env == 'docker':
    rv['AUTH_CFG_STR']                                  = AUTH_CFG_STR_DOCKER
    rv['AUTH_PLUS_EXTERNAL_POSTGRES']                   = AUTH_PLUS_EXTERNAL_POSTGRES_DOCKER
    rv['AUTH_CFG_STR_INTEGRATION']                      = AUTH_CFG_INTEGRATION_STR_DOCKER
    rv['GET_IAM_POLICY_AGG_ASC_INPUT_FILE']             = GET_IAM_POLICY_AGG_ASC_INPUT_FILE_DOCKER
    rv['JSON_INIT_FILE_PATH_AWS']                       = JSON_INIT_FILE_PATH_AWS
    rv['JSON_INIT_FILE_PATH_AZURE']                     = JSON_INIT_FILE_PATH_AZURE
    rv['JSON_INIT_FILE_PATH_GITHUB']                    = JSON_INIT_FILE_PATH_GITHUB
    rv['JSON_INIT_FILE_PATH_GOOGLE']                    = JSON_INIT_FILE_PATH_GOOGLE
    rv['JSON_INIT_FILE_PATH_K8S']                       = JSON_INIT_FILE_PATH_K8S
    rv['JSON_INIT_FILE_PATH_OKTA']                      = JSON_INIT_FILE_PATH_OKTA
    rv['JSON_INIT_FILE_PATH_REGISTRY']                  = JSON_INIT_FILE_PATH_REGISTRY
    rv['JSON_INIT_FILE_PATH_SUMOLOGIC']                 = JSON_INIT_FILE_PATH_SUMOLOGIC
    rv['PG_SRV_MTLS_CFG_STR']                           = PG_SRV_MTLS_CFG_STR
    rv['PSQL_MTLS_CONN_STR']                            = PSQL_MTLS_CONN_STR_DOCKER
    rv['PSQL_MTLS_CONN_STR_UNIX']                       = PSQL_MTLS_CONN_STR_DOCKER
    rv['PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC']         = PSQL_MTLS_CONN_STR_WITH_EAGER_GC_DOCKER
    rv['PSQL_MTLS_CONN_STR_UNIX_WITH_NAMESPACES']       = PSQL_MTLS_CONN_STR_WITH_NAMESPACES_DOCKER
    rv['PSQL_MTLS_INVALID_CONN_STR']                    = PSQL_MTLS_INVALID_CONN_STR_DOCKER
    rv['PSQL_UNENCRYPTED_CONN_STR']                     = PSQL_UNENCRYPTED_CONN_STR_DOCKER
    rv['REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR']       = _REGISTRY_EXPERIMENTAL_DOCKER_NO_VERIFY
    rv['REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG_STR'] = _REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_DOCKER
  else:
    rv['AUTH_CFG_STR']                                  = AUTH_CFG_STR
    rv['AUTH_PLUS_EXTERNAL_POSTGRES']                   = AUTH_PLUS_EXTERNAL_POSTGRES
    rv['AUTH_CFG_STR_INTEGRATION']                      = AUTH_CFG_INTEGRATION_STR
    rv['GET_IAM_POLICY_AGG_ASC_INPUT_FILE']             = GET_IAM_POLICY_AGG_ASC_INPUT_FILE
    rv['JSON_INIT_FILE_PATH_AWS']                       = JSON_INIT_FILE_PATH_AWS
    rv['JSON_INIT_FILE_PATH_AZURE']                     = JSON_INIT_FILE_PATH_AZURE
    rv['JSON_INIT_FILE_PATH_GITHUB']                    = JSON_INIT_FILE_PATH_GITHUB
    rv['JSON_INIT_FILE_PATH_GOOGLE']                    = JSON_INIT_FILE_PATH_GOOGLE
    rv['JSON_INIT_FILE_PATH_K8S']                       = JSON_INIT_FILE_PATH_K8S
    rv['JSON_INIT_FILE_PATH_OKTA']                      = JSON_INIT_FILE_PATH_OKTA
    rv['JSON_INIT_FILE_PATH_REGISTRY']                  = JSON_INIT_FILE_PATH_REGISTRY
    rv['JSON_INIT_FILE_PATH_SUMOLOGIC']                 = JSON_INIT_FILE_PATH_SUMOLOGIC
    rv['PG_SRV_MTLS_CFG_STR']                           = PG_SRV_MTLS_CFG_STR
    rv['PSQL_MTLS_CONN_STR']                            = PSQL_MTLS_CONN_STR
    rv['PSQL_MTLS_CONN_STR_UNIX']                       = PSQL_MTLS_CONN_STR_UNIX
    rv['PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC']         = PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC
    rv['PSQL_MTLS_CONN_STR_UNIX_WITH_NAMESPACES']       = PSQL_MTLS_CONN_STR_UNIX_WITH_NAMESPACES
    rv['PSQL_MTLS_INVALID_CONN_STR']                    = PSQL_MTLS_INVALID_CONN_STR
    rv['PSQL_UNENCRYPTED_CONN_STR']                     = PSQL_UNENCRYPTED_CONN_STR
    rv['REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR']       = _REGISTRY_EXPERIMENTAL_NO_VERIFY
    rv['REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG_STR'] = _REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY
  return rv