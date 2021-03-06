

import json
import os

from robot.api.deco import keyword, library
from robot.libraries.BuiltIn import BuiltIn
from robot.libraries.Process import Process
from robot.libraries.OperatingSystem import OperatingSystem 

from stackql_context import RegistryCfg



@library(scope='SUITE', version='0.1.0', doc_format='reST')
class StackQLInterfaces(OperatingSystem, Process, BuiltIn):
  ROBOT_LISTENER_API_VERSION = 2

  def __init__(self, execution_platform='native'):
    self._counter = 0
    self._execution_platform=execution_platform
    self.ROBOT_LIBRARY_LISTENER = self
    Process.__init__(self)

  def _end_suite(self, name, attrs):
    print('Suite %s (%s) ending.' % (name, attrs['id']))

  def count(self):
    self._counter += 1
    print(self._counter)

  def clear_counter(self):
    self._counter = 0


  def _run_PG_client_command(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str):
    _mod_conn =  psql_conn_str.replace("\\", "/")
    # bi = BuiltIn().get_library_instance('Builtin')
    self.log_to_console(f"curdir = '{curdir}'")
    self.log_to_console(f"psql_exe = '{psql_exe}'")
    result = super().run_process(
      psql_exe, 
      '-d', _mod_conn, 
      '-c', query
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
      query,
      *args,
      **cfg
    )

  def _run_stackql_exec_command_docker(
    self,
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
    query,
    *args,
    **cfg
  ):
    if type(query) == bytes:
      query = query.decode("utf-8") 
    reg_location = registry_cfg.get_source_path_for_docker()
    supplied_args = []
    registry_cfg_str = registry_cfg.get_config_str('docker')
    if registry_cfg_str != "":
      supplied_args.append(f"--registry='{registry_cfg_str}'")
    if auth_cfg_str != "":
      supplied_args.append(f"--auth='{auth_cfg_str}'")
    supplied_args.append("--tls.allowInsecure=true")
    supplied_args = supplied_args + list(args)
    query_escaped = query.replace("'", "'\"'\"'")
    os.environ['REGISTRY_SRC']= f'./{reg_location}'
    res = super().run_process(
      "docker-compose",
      "-p",
      "execrun",
      "run",
      "--rm",
      "-e",
      f"OKTA_SECRET_KEY={okta_secret_str}", 
      "-e",
      f"GITHUB_SECRET_KEY={github_secret_str}",
      "-e",
      f"K8S_SECRET_KEY={k8s_secret_str}",
      "stackqlsrv",
      "bash",
      "-c",
      f"stackql exec {' '.join(supplied_args)} '{query_escaped}'",
      **cfg
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
    query,
    *args,
    **cfg
  ):
    self.set_environment_variable("OKTA_SECRET_KEY", okta_secret_str)
    self.set_environment_variable("GITHUB_SECRET_KEY", github_secret_str)
    self.set_environment_variable("K8S_SECRET_KEY", k8s_secret_str)
    supplied_args = [ stackql_exe, "exec" ]
    registry_cfg_str = registry_cfg.get_config_str('native')
    if registry_cfg_str != "":
      supplied_args.append(f"--registry={registry_cfg_str}")
    if auth_cfg_str != "":
      supplied_args.append(f"--auth={auth_cfg_str}")
    supplied_args.append("--tls.allowInsecure=true")
    res = super().run_process(
      *supplied_args,
      query,
      *args,
      **cfg
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
  def should_PG_client_inline_contain(self, curdir :str, psql_exe :str, psql_conn_str :str, query :str, expected_output :str):
    result = self._run_PG_client_command(
      curdir,
      psql_exe,
      psql_conn_str,
      query
    )
    return self.should_contain(result.stdout, expected_output)
  
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
  def should_stackql_exec_inline_equal_page_limited(
    self, 
    stackql_exe :str, 
    page_limit :int,
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
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
      query,
      *args,
      **cfg
    )
    return self.should_be_equal(result.stdout, expected_output)

  @keyword
  def should_stackql_exec_inline_equal(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
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
      query,
      *args,
      **cfg
    )
    return self.should_be_equal(result.stdout, expected_output)

  @keyword
  def should_stackql_exec_inline_contain(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str, 
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
  def should_horrid_query_stackql_inline_equal(
    self, 
    stackql_exe :str, 
    okta_secret_str :str,
    github_secret_str :str,
    k8s_secret_str :str,
    registry_cfg :RegistryCfg, 
    auth_cfg_str :str,
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
      query,
      **{"stdout": stdout_tmp_file }
    )
    return self.should_be_equal(result.stdout, expected_output)