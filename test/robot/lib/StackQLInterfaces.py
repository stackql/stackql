from robot.api.deco import keyword, library

from robot.libraries.BuiltIn import BuiltIn

from robot.libraries.Process import Process



@library(scope='SUITE', version='0.1.0', doc_format='reST')
class StackQLInterfaces(BuiltIn, Process):
  ROBOT_LISTENER_API_VERSION = 2

  def __init__(self):
    self._counter = 0
    self.ROBOT_LIBRARY_LISTENER = self

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
    result = self.run_process(
      psql_exe, 
      '-d', _mod_conn, 
      '-c', query
    )
    self.log(result.stdout)
    self.log(result.stderr)
    return result
  
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
    result =    self._run_PG_client_command(
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