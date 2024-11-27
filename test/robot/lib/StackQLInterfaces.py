

from asyncio import subprocess
import json
import os
import time
import typing

from robot.api.deco import keyword, library
from robot.libraries.BuiltIn import BuiltIn
from robot.libraries.Collections import Collections
from robot.libraries.Process import Process
from robot.libraries.OperatingSystem import OperatingSystem 

from stackql_context import RegistryCfg, _TEST_APP_CACHE_ROOT, PSQL_EXE, SQLITE_EXE
from ShellSession import ShellSession
from psycopg_client import PsycoPGClient
from psycopg2_client import PsycoPG2Client
from sqlalchemy_client import SQLAlchemyClient

SQL_BACKEND_CANONICAL_SQLITE_EMBEDDED :str = 'sqlite_embedded'
SQL_BACKEND_POSTGRES_TCP :str = 'postgres_tcp'
SQL_CONCURRENCT_LIMIT_DEFAULT :int = 1


@library(scope='SUITE', version='0.1.0', doc_format='reST')
class StackQLInterfaces(OperatingSystem, Process, BuiltIn, Collections):
  ROBOT_LISTENER_API_VERSION = 2

  def __init__(self, execution_platform='native', sql_backend=SQL_BACKEND_CANONICAL_SQLITE_EMBEDDED, concurrency_limit=SQL_CONCURRENCT_LIMIT_DEFAULT):
    self._counter = 0
    self._execution_platform=execution_platform
    self._sql_backend=sql_backend
    self._concurrency_limit=concurrency_limit
    self.ROBOT_LIBRARY_LISTENER = self
    Process.__init__(self)

  def _end_suite(self, name, attrs):
    print('Suite %s (%s) ending.' % (name, attrs['id']))

  def count(self):
    self._counter += 1
    print(self._counter)

  def clear_counter(self):
    self._counter = 0


  def _run_PG_client_command(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str, *args, **cfg):
    _mod_conn =  psql_conn_str.replace("\\", "/")
    # bi = BuiltIn().get_library_instance('Builtin')
    self.log_to_console(f"curdir = '{curdir}'")
    self.log_to_console(f"psql_exe = '{psql_exe}'")
    result = super().run_process(
      psql_exe, 
      '-d', _mod_conn, 
      '-c', query,
      *args,
      **cfg,
    )
    self.log(result.stdout)
    self.log(result.stderr)
    return result
  
  @keyword
  def should_rdbms_query_return_csv_result(
    self,
    db_name :str,
    sql_client_export_connection_arg :str,
    query :str,
    expected_output :str,
    expected_stderr_output :str,
    *args,
    **kwargs):
    result = None
    if db_name == "sqlite":
      result = self._run_sqlite_command(
        SQLITE_EXE,
        sql_client_export_connection_arg,
        query,
        *("--csv",),
        **kwargs,
      )
    elif db_name == "postgres":
      result = self._run_PG_client_command(
        os.getcwd(),
        PSQL_EXE,
        sql_client_export_connection_arg,
        query,
        *("--csv",),
        **kwargs,
      )
    return self._verify_both_streams(result, expected_output, expected_stderr_output)
  

  def _verify_both_streams(self, result, expected_output, expected_stderr_output):
    stdout_ok = self.should_be_equal(result.stdout, expected_output)
    if self._execution_platform == "docker":
      # cannot silence stupid compose status logs
      stderr_ok = self.should_contain(result.stderr, expected_stderr_output)
      return stdout_ok and stderr_ok
    stderr_ok = self.should_be_equal(result.stderr, expected_stderr_output)
    return stdout_ok and stderr_ok

  def _run_sqlite_command(self, sqlite_exe :str, sqlite_db_file :str, query :str, *args, **cfg):
    self.log_to_console(f"sqlite_exe = '{sqlite_exe}'")
    result = None
    if len(args) == 0:
      result = super().run_process(
        sqlite_exe,
        sqlite_db_file, 
        query,
        **cfg,
      )
    else:
      result = super().run_process(
        sqlite_exe,
        *args,
        sqlite_db_file, 
        query,
        **cfg,
      )
    self.log(result.stdout)
    self.log(result.stderr)
    return result

  def _run_stackql_exec_command(
    self,  
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str,
    sql_backend_cfg_str :str,
    query,
    *args,
    **cfg
  ):
    if self._execution_platform == 'docker':
      return self._run_stackql_exec_command_docker(
        okta_secret_str,
        github_secret_str,
        k8s_secret_str,
        registry_cfg, 
        auth_cfg_str, 
        sql_backend_cfg_str,
        query,
        *args,
        **cfg
      )
    return self._run_stackql_exec_command_native(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      *args,
      **cfg
    )


  def _run_stackql_shell_command(
    self,  
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    queries :typing.Iterable[str],
    *args,
    **cfg
  ):
    if self._execution_platform == 'docker':
      return self._run_stackql_shell_command_docker(
        okta_secret_str,
        github_secret_str,
        k8s_secret_str,
        registry_cfg, 
        auth_cfg_str,
        sql_backend_cfg_str, 
        queries,
        *args,
        **cfg
      )
    return self._run_stackql_shell_command_native(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      queries,
      *args,
      **cfg
    )

  def _expand_docker_env_args(self, okta_secret_str :str, github_secret_str :str, k8s_secret_str :str, *args) -> typing.Iterable:
    rv = [
      "-e",
      f"OKTA_SECRET_KEY={okta_secret_str}", 
      "-e",
      f"GITHUB_SECRET_KEY={github_secret_str}",
      "-e",
      f"K8S_SECRET_KEY={k8s_secret_str}",
    ]
    for k, v in self._get_default_env().items():
      rv.append("-e")
      rv.append(f"{k}={v}")
    return rv

  def _docker_transform_args(self, *args) -> typing.Iterable:
    rv = [ f"--namespaces='{b[13:]}'" if type(b) == str and b.startswith('--namespaces=') else b for b in list(args) ]
    rv = [ f"--sqlBackend='{b[13:]}'" if type(b) == str and b.startswith('--sqlBackend=') else b for b in list(rv) ]
    rv = [ f"--export.alias='{b[15:]}'" if type(b) == str and b.startswith('--export.alias=') else b for b in list(rv) ]
    rv = [ f"--http.log.enabled='{b[19:]}'" if type(b) == str and b.startswith('--http.log.enabled=') else b for b in list(rv) ]
    return rv

  def _run_stackql_exec_command_docker(
    self,
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str,
    sql_backend_cfg_str :str,
    query,
    *args,
    **cfg
  ):
    if type(query) == bytes:
      query = query.decode("utf-8") 
    reg_location = registry_cfg.get_source_path_for_docker()
    supplied_args = []
    stackql_persist_postgres_as_needed = cfg.pop('stackql_persist_postgres_as_needed', False)
    if cfg.pop('stackql_rollback_eager', False):
      supplied_args.append("--session='{\"rollback_type\":\"eager\"}'")
    if cfg.pop('stackql_H', False):
      supplied_args.append('--output=text')
      supplied_args.append('-H')
    if cfg.pop('stackql_dataflow_permissive', False):
      supplied_args.append('--dataflow.dependency.max=50')
      supplied_args.append('--dataflow.components.max=50')
    if cfg.pop('stackql_debug_http', False):
      supplied_args.append("--http.log.enabled=true")
    if cfg.pop('stackql_dryrun', False):
      supplied_args.append('--dryrun')
    query_from_input_file_path = cfg.pop('stackql_i', False)
    if query_from_input_file_path:
      supplied_args.append(f'--infile={query_from_input_file_path}')
    query_from_input_file_data_path = cfg.pop('stackql_iqldata', False)
    if query_from_input_file_data_path:
      supplied_args.append(f'--iqldata={query_from_input_file_data_path}')
    query_var_list = cfg.pop('stackql_vars', False)
    if query_var_list:
      for q_var in query_var_list:
        supplied_args.append(f'--var={q_var}')
    registry_cfg_str = registry_cfg.get_config_str('docker')
    if registry_cfg_str != "":
      supplied_args.append(f"--registry='{registry_cfg_str}'")
    if auth_cfg_str != "":
      supplied_args.append(f"--auth='{auth_cfg_str}'")
    if sql_backend_cfg_str != "":
      supplied_args.append(f"--sqlBackend='{sql_backend_cfg_str}'")
    supplied_args.append("--tls.allowInsecure=true")
    supplied_args.append(f"--execution.concurrency.limit={self._concurrency_limit}")
    transformed_args = self._docker_transform_args(*args)
    supplied_args = supplied_args + transformed_args
    query_escaped = query.replace("'", "'\"'\"'")
    os.environ['REGISTRY_SRC']= f'./{reg_location}'
    if self._sql_backend == SQL_BACKEND_POSTGRES_TCP:
      os.environ['DB_SETUP_SRC']= f'./test/db/postgres'
    sleep_prefix = '' if self._sql_backend == SQL_BACKEND_CANONICAL_SQLITE_EMBEDDED else 'sleep 2 && '
    env_args_to_docker = self._expand_docker_env_args(okta_secret_str, github_secret_str, k8s_secret_str)
    invocation_str = f"{sleep_prefix}stackql exec {' '.join(supplied_args)} '{query_escaped}'"
    if query_from_input_file_path:
      invocation_str = f"{sleep_prefix}stackql exec {' '.join(supplied_args)}"
    res = super().run_process(
      "docker",
      "compose",
      "-p",
      "execrun",
      "run",
      "--rm",
      *env_args_to_docker,
      "stackqlsrv",
      "bash",
      "-c",
      invocation_str,
      **cfg
    )
    self.log(res.stdout)
    self.log(res.stderr)
    return res


  def _get_allowed_docker_env_keys(self):
    return [ 
        'AZURE_CLIENT_ID',
        'AZURE_CLIENT_SECRET',
        'AZURE_INTEGRATION_TESTING_SUB_ID',
        'AZURE_TENANT_ID'
    ]


  def _get_default_env(self) -> dict:
    existing = dict(os.environ)
    rv = {}
    for k, v in existing.items():
      if k in self._get_allowed_docker_env_keys():
        rv[k] = v
    rv["AZ_ACCESS_TOKEN"] = os.environ.get('AZ_ACCESS_TOKEN', "az_access_dummy_secret")
    rv["SUMO_CREDS"] = os.environ.get('SUMO_CREDS', "sumologicdummysecret")
    rv["DIGITALOCEAN_TOKEN"] = os.environ.get('DIGITALOCEAN_TOKEN', "digitaloceandummysecret")
    rv["DUMMY_DIGITALOCEAN_USERNAME"] = os.environ.get('DUMMY_DIGITALOCEAN_USERNAME', "myusername")
    rv["DUMMY_DIGITALOCEAN_PASSWORD"] = os.environ.get('DUMMY_DIGITALOCEAN_PASSWORD', "mypassword")
    if os.environ.get('YOUR_OAUTH2_CLIENT_ID_ENV_VAR') is not None:
      rv["YOUR_OAUTH2_CLIENT_ID_ENV_VAR"] = os.environ.get('YOUR_OAUTH2_CLIENT_ID_ENV_VAR')
    if os.environ.get('YOUR_OAUTH2_CLIENT_SECRET_ENV_VAR') is not None:
      rv["YOUR_OAUTH2_CLIENT_SECRET_ENV_VAR"] = os.environ.get('YOUR_OAUTH2_CLIENT_SECRET_ENV_VAR')
    return rv


  def _run_stackql_shell_command_docker(
    self,
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    queries :typing.Iterable[str],
    *args,
    **cfg
  ):
    reg_location = registry_cfg.get_source_path_for_docker()
    supplied_args = []
    if cfg.pop('stackql_rollback_eager', False):
      supplied_args.append("--session={\"rollback_type\":\"eager\"}")
    if cfg.pop('stackql_H', False):
      supplied_args.append('--output=text')
      supplied_args.append('-H')
    if cfg.pop('stackql_dataflow_permissive', False):
      supplied_args.append('--dataflow.dependency.max=50')
      supplied_args.append('--dataflow.components.max=50')
    if cfg.pop('stackql_debug_http', False):
      supplied_args.append("--http.log.enabled=true")
    registry_cfg_str = registry_cfg.get_config_str('docker')
    if registry_cfg_str != "":
      supplied_args.append(f"--registry='{registry_cfg_str}'")
    if auth_cfg_str != "":
      supplied_args.append(f"--auth='{auth_cfg_str}'")
    if sql_backend_cfg_str != "":
      supplied_args.append(f"--sqlBackend='{sql_backend_cfg_str}'")
    supplied_args.append("--tls.allowInsecure=true")
    supplied_args.append(f"--execution.concurrency.limit={self._concurrency_limit}")
    transformed_args = self._docker_transform_args(*args)
    supplied_args = supplied_args + transformed_args
    os.environ['REGISTRY_SRC']= f'./{reg_location}'
    start_cmd = [
      "docker", 
      "compose",
      "-p",
      "stackqlshell",
      "run",
      "--rm",
      "-e",
      f"OKTA_SECRET_KEY={okta_secret_str}", 
      "-e",
      f"GITHUB_SECRET_KEY={github_secret_str}",
      "-e",
      f"K8S_SECRET_KEY={k8s_secret_str}",
      "-e",
      f"AZ_ACCESS_TOKEN={self._get_default_env().get('AZ_ACCESS_TOKEN')}",
      "-e",
      f"SUMO_CREDS={self._get_default_env().get('SUMO_CREDS')}",
      "-e",
      f"DIGITALOCEAN_TOKEN={self._get_default_env().get('DIGITALOCEAN_TOKEN')}",
      "-e",
      f"DUMMY_DIGITALOCEAN_USERNAME={self._get_default_env().get('DUMMY_DIGITALOCEAN_USERNAME')}",
      "-e",
      f"DUMMY_DIGITALOCEAN_PASSWORD={self._get_default_env().get('DUMMY_DIGITALOCEAN_PASSWORD')}",
      "stackqlsrv",
      "bash",
      "-c",
      f"stackql {' '.join(supplied_args)} shell",
    ]
    stdout = cfg.get('stdout', subprocess.PIPE)
    stderr = cfg.get('stderr', subprocess.PIPE)
    shell_session = ShellSession()
    res = shell_session.run_shell_session(
      start_cmd,
      queries,
      stdout=stdout,
      stderr=stderr,
    )
    self.log(res.stdout)
    self.log(res.stderr)
    return res


  def _run_stackql_exec_command_native(
    self,  
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str,
    sql_backend_cfg_str :str,
    query,
    *args,
    **cfg
  ):
    self.set_environment_variable("OKTA_SECRET_KEY", okta_secret_str)
    self.set_environment_variable("GITHUB_SECRET_KEY", github_secret_str)
    self.set_environment_variable("K8S_SECRET_KEY", k8s_secret_str)
    self.set_environment_variable("AZ_ACCESS_TOKEN", f"{self._get_default_env().get('AZ_ACCESS_TOKEN')}")
    self.set_environment_variable("SUMO_CREDS", f"{self._get_default_env().get('SUMO_CREDS')}")
    self.set_environment_variable("DIGITALOCEAN_TOKEN", f"{self._get_default_env().get('DIGITALOCEAN_TOKEN')}")
    self.set_environment_variable("DUMMY_DIGITALOCEAN_USERNAME", f"{self._get_default_env().get('DUMMY_DIGITALOCEAN_USERNAME')}")
    self.set_environment_variable("DUMMY_DIGITALOCEAN_PASSWORD", f"{self._get_default_env().get('DUMMY_DIGITALOCEAN_PASSWORD')}")
    supplied_args = [ stackql_exe, "exec" ]
    stackql_persist_postgres_as_needed = cfg.pop('stackql_persist_postgres_as_needed', False)
    if cfg.pop('stackql_rollback_eager', False):
      supplied_args.append("--session={\"rollback_type\":\"eager\"}")
    if cfg.pop('stackql_H', False):
      supplied_args.append('--output=text')
      supplied_args.append('-H')
    if cfg.pop('stackql_dataflow_permissive', False):
      supplied_args.append('--dataflow.dependency.max=50')
      supplied_args.append('--dataflow.components.max=50')
    if cfg.pop('stackql_debug_http', False):
      supplied_args.append("--http.log.enabled=true")
    if cfg.pop('stackql_dryrun', False):
      supplied_args.append('--dryrun')
    query_from_input_file_path = cfg.pop('stackql_i', False)
    if query_from_input_file_path:
      supplied_args.append(f'--infile={query_from_input_file_path}')
    query_from_input_file_data_path = cfg.pop('stackql_iqldata', False)
    if query_from_input_file_data_path:
      supplied_args.append(f'--iqldata={query_from_input_file_data_path}')
    query_var_list = cfg.pop('stackql_vars', False)
    if query_var_list:
      for q_var in query_var_list:
        supplied_args.append(f'--var={q_var}')
    registry_cfg_str = registry_cfg.get_config_str('native')
    if registry_cfg_str != "":
      supplied_args.append(f"--registry={registry_cfg_str}")
    if auth_cfg_str != "":
      supplied_args.append(f"--auth={auth_cfg_str}")
    if sql_backend_cfg_str != "":
      supplied_args.append(f"--sqlBackend={sql_backend_cfg_str}")
    supplied_args.append("--tls.allowInsecure=true")
    supplied_args.append(f"--execution.concurrency.limit={self._concurrency_limit}")
    if not query_from_input_file_path:
      supplied_args.append(query)
    res = super().run_process(
      *supplied_args,
      *args,
      **cfg
    )
    self.log(res.stdout)
    self.log(res.stderr)
    return res


  def _run_stackql_shell_command_native(
    self,  
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    queries :typing.Iterable[str],
    *args,
    **cfg
  ):
    self.set_environment_variable("OKTA_SECRET_KEY", okta_secret_str)
    self.set_environment_variable("GITHUB_SECRET_KEY", github_secret_str)
    self.set_environment_variable("K8S_SECRET_KEY", k8s_secret_str)
    self.set_environment_variable("AZ_ACCESS_TOKEN", f"{self._get_default_env().get('AZ_ACCESS_TOKEN')}")
    self.set_environment_variable("SUMO_CREDS", f"{self._get_default_env().get('SUMO_CREDS')}")
    self.set_environment_variable("DIGITALOCEAN_TOKEN", f"{self._get_default_env().get('DIGITALOCEAN_TOKEN')}")
    self.set_environment_variable("DUMMY_DIGITALOCEAN_USERNAME", f"{self._get_default_env().get('DUMMY_DIGITALOCEAN_USERNAME')}")
    self.set_environment_variable("DUMMY_DIGITALOCEAN_PASSWORD", f"{self._get_default_env().get('DUMMY_DIGITALOCEAN_PASSWORD')}")
    supplied_args = [ stackql_exe, "shell" ]
    if cfg.pop('stackql_rollback_eager', False):
      supplied_args.append("--session={\"rollback_type\":\"eager\"}")
    if cfg.pop('stackql_H', False):
      supplied_args.append('--output=text')
      supplied_args.append('-H')
    if cfg.pop('stackql_dataflow_permissive', False):
      supplied_args.append('--dataflow.dependency.max=50')
      supplied_args.append('--dataflow.components.max=50')
    if cfg.pop('stackql_debug_http', False):
      supplied_args.append("--http.log.enabled=true")
    registry_cfg_str = registry_cfg.get_config_str('native')
    if registry_cfg_str != "":
      supplied_args.append(f"--registry={registry_cfg_str}")
    if auth_cfg_str != "":
      supplied_args.append(f"--auth={auth_cfg_str}")
    if sql_backend_cfg_str != "":
      supplied_args.append(f"--sqlBackend={sql_backend_cfg_str}")
    supplied_args.append("--tls.allowInsecure=true")
    supplied_args.append(f'--approot="{_TEST_APP_CACHE_ROOT}"')
    supplied_args.append(f"--execution.concurrency.limit={self._concurrency_limit}")
    supplied_args = supplied_args + list(args)
    stdout = cfg.get('stdout', subprocess.PIPE)
    stderr = cfg.get('stderr', subprocess.PIPE)
    shell_session = ShellSession()
    res = shell_session.run_shell_session(
      supplied_args,
      queries,
      stdout=stdout,
      stderr=stderr,
    )
    self.log(res.stdout)
    self.log(res.stderr)
    return res

  @keyword
  def should_PG_client_error_inline_contain(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str, expected_output :str):
    result =    self._run_PG_client_command(
      curdir,
      psql_exe,
      psql_conn_str,
      query
    ) #    ${CURDIR}    ${PSQL_EXE}    ${_CONN_STR}    ${_QUERY}
    return self.should_contain(result.stderr, expected_output)


  @keyword
  def should_PG_client_inline_contain(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str, expected_output :str, **cfg):
    result = self._run_PG_client_command(
      curdir,
      psql_exe,
      psql_conn_str,
      query,
      **cfg
    )
    return self.should_contain(result.stdout, expected_output)

  
  @keyword
  def should_PG_client_stderr_inline_contain(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str, expected_output :str, **cfg):
    result = self._run_PG_client_command(
      curdir,
      psql_exe,
      psql_conn_str,
      query,
      **cfg
    )
    return self.should_contain(result.stderr, expected_output)


  @keyword
  def should_PG_client_inline_equal(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str, expected_output :str):
    result =    self._run_PG_client_command(
      curdir,
      psql_exe,
      psql_conn_str,
      query
    )
    return self.should_be_equal(result.stdout, expected_output)
  

  @keyword
  def should_sqlite_inline_equal(self, curdir :str, sqlite_exe :str, sqlite_db_file :str, query :str, expected_output :str, *args, **kwargs):
    result = None
    if len(args) == 0:
      result =    self._run_sqlite_command(
        curdir,
        sqlite_exe,
        sqlite_db_file,
        query
      )
    else:
      result =    self._run_sqlite_command(
        curdir,
        sqlite_exe,
        *args,
        sqlite_db_file,
        query
      ) 
    return self.should_be_equal(result.stdout, expected_output)
  

  @keyword
  def should_PG_client_inline_equal_bench(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str, expected_output :str, max_mean_time :float = 1.7, max_time :float = 10.0, repeat_count = 10, **cfg):
    times = []
    for i in range(repeat_count):
      start_time = time.time()
      result =    self._run_PG_client_command(
        curdir,
        psql_exe,
        psql_conn_str,
        query
      )
      end_time = time.time()
      duration = end_time - start_time
      times.append(duration)
      self.should_be_equal(result.stdout, expected_output)
    mean_time = sum(times) / len(times)
    self.log(f"mean_time = {mean_time}")
    self.log(f"max_time = {max(times)}")
    self.should_be_true(mean_time < max_mean_time)
    return self.should_be_true(all([ t < max_time for t in times ]))


  @keyword
  def should_PG_client_session_inline_equal(self, conn_str :str, queries :typing.List[str], expected_output :typing.List[typing.Dict], **kwargs):
    client = PsycoPGClient(conn_str)
    result =  client.run_queries(
      queries
    )
    self.log(result)
    return self.lists_should_be_equal(result, expected_output)
  

  @keyword
  def should_PG_client_session_inline_equal_strict(self, conn_str :str, queries :typing.List[str], expected_output :typing.List[typing.Dict], **kwargs):
    client = PsycoPGClient(conn_str)
    result =  client.run_queries_strict(
      queries
    )
    self.log(result)
    return self.lists_should_be_equal(result, expected_output)

  
  @keyword
  def should_PG_client_session_inline_contain(self, conn_str :str, queries :typing.List[str], expected_output :typing.List[typing.Dict], **kwargs):
    client = PsycoPGClient(conn_str)
    result =  client.run_queries(
      queries
    )
    self.log(result)
    return self.list_should_contain_sub_list(result, expected_output)


  @keyword
  def should_sqlalchemy_raw_session_inline_equal(self, conn_str :str, queries :typing.List[str], expected_output :typing.List, **kwargs):
    client = SQLAlchemyClient(conn_str)
    result =  client.run_raw_queries(
      queries
    )
    self.log(result)
    return self.lists_should_be_equal(result, expected_output)


  @keyword
  def should_sqlalchemy_raw_session_inline_contain(self, conn_str :str, queries :typing.List[str], expected_output :typing.Tuple, **kwargs):
    client = SQLAlchemyClient(conn_str)
    result =  client.run_raw_queries(
      queries
    )
    self.log(result)
    return self.list_should_contain_value(result, expected_output)
  

  @keyword
  def should_sqlalchemy_raw_session_inline_have_length(self, conn_str :str, queries :typing.List[str], expected_length :int, **kwargs):
    client = SQLAlchemyClient(conn_str)
    result =  client.run_raw_queries(
      queries
    )
    self.log(result)
    return self.should_be_equal(len(result), expected_length)
  

  @keyword
  def should_sqlalchemy_raw_session_inline_have_length_greater_than_or_equal_to(self, conn_str :str, queries :typing.List[str], expected_length :int, **kwargs):
    client = SQLAlchemyClient(conn_str)
    result =  client.run_raw_queries(
      queries
    )
    self.log(result)
    return self.should_be_true(len(result) >= expected_length)


  @keyword
  def should_PG_client_V2_session_inline_equal(self, conn_str :str, queries :typing.List[str], expected_output :typing.List[typing.Dict], **kwargs):
    client = PsycoPG2Client(conn_str)
    result =  client.run_queries(
      queries
    )
    self.log(result)
    return self.lists_should_be_equal(result, expected_output)


  @keyword
  def should_stackql_exec_inline_equal_page_limited(
    self, 
    stackql_exe :str, 
    page_limit :int,
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    query :str,
    expected_output :str,
    *args,
    **cfg
  ):
    args = ( f"--http.response.pageLimit={page_limit}", ) + args
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str, 
      query,
      *args,
      **cfg
    )
    return self.should_be_equal(result.stdout, expected_output)


  @keyword
  def run_stackql_exec_command_no_errors(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    query :str,
    *args,
    **cfg
  ):
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      *args,
      **cfg
    )
    self.should_be_equal_as_integers(result.rc, 0)
    return result


  @keyword
  def should_stackql_exec_inline_equal(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    query :str,
    expected_output :str,
    *args,
    **cfg
  ):
    repeat_count = int(cfg.pop('repeat_count', 1))
    for _ in range(repeat_count):
      result = self._run_stackql_exec_command(
        stackql_exe, 
        okta_secret_str,
        github_secret_str,
        k8s_secret_str,
        registry_cfg, 
        auth_cfg_str, 
        sql_backend_cfg_str,
        query,
        *args,
        **cfg
      )
      self.should_be_equal(result.stdout, expected_output)
  

  @keyword
  def should_stackql_exec_inline_equal_both_streams(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    query :str,
    expected_output :str,
    expected_stderr_output :str,
    *args,
    **cfg
  ):
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      *args,
      **cfg
    )
    return self._verify_both_streams(result, expected_output, expected_stderr_output)


  @keyword
  def should_stackql_shell_inline_equal(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    queries :typing.Iterable[str],
    expected_output :str,
    *args,
    **cfg
  ):
    result = self._run_stackql_shell_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      queries,
      *args,
      **cfg
    )
    return self.should_be_equal(result.stdout, expected_output)

  
  @keyword
  def should_stackql_shell_inline_contain(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    queries :typing.Iterable[str],
    expected_output :str,
    *args,
    **cfg
  ):
    result = self._run_stackql_shell_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      queries,
      *args,
      **cfg
    )
    return self.should_contain(result.stdout, expected_output)


  @keyword
  def should_stackql_exec_inline_contain(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str,
    sql_backend_cfg_str :str,
    query :str,
    expected_output :str,
    *args,
    **cfg
  ):
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      *args,
      **cfg
    )
    return self.should_contain(result.stdout, expected_output)


  @keyword
  def should_stackql_exec_inline_equal_stderr(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    query :str,
    expected_output :str,
    *args,
    **cfg
  ):
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      *args,
      **cfg
    )
    se = result.stderr
    if self._execution_platform == 'docker':
      se_split = se.split('\n')
      if len(se_split) > 1:
        se = se_split[-1]
    return self.should_be_equal(se, expected_output, collapse_spaces=True, formatter='ascii', strip_spaces=True)

  
  @keyword
  def should_stackql_exec_inline_contain_stderr(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    sql_backend_cfg_str :str,
    query :str,
    expected_output :str,
    *args,
    **cfg
  ):
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      *args,
      **cfg
    )
    se = result.stderr
    if self._execution_platform == 'docker':
      se_split = se.split('\n')
      if len(se_split) > 1:
        se = se_split[-1]
    return self.should_contain(se, expected_output, collapse_spaces=True, strip_spaces=True)


  @keyword
  def should_horrid_query_stackql_inline_equal(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str,
    sql_backend_cfg_str :str,
    query,
    expected_output :str,
    stdout_tmp_file :str,
  ):
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      **{"stdout": stdout_tmp_file }
    )
    return self.should_be_equal(result.stdout, expected_output)


  @keyword
  def should_horrid_http_log_enabled_query_stackql_inline_equal(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str,
    sql_backend_cfg_str :str,
    query,
    expected_output :str,
    stdout_tmp_file :str,
  ):
    args = ("--http.log.enabled",)
    result = self._run_stackql_exec_command(
      stackql_exe, 
      okta_secret_str,
      github_secret_str,
      k8s_secret_str,
      registry_cfg, 
      auth_cfg_str, 
      sql_backend_cfg_str,
      query,
      *args,
      **{"stdout": stdout_tmp_file }
    )
    return self.should_be_equal(result.stdout, expected_output)

