from robot.api.deco import keyword, library

from robot.libraries.BuiltIn import BuiltIn

from robot.libraries.Process import Process



@library(scope='SUITE', version='0.1.0', doc_format='reST')
class CloudIntegration:
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

  @keyword
  def nop_cloud_integration_keyword(self):
    return 'PASS'
