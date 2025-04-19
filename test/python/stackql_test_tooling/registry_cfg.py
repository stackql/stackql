
import json
import os

from typing import Optional


class RegistryCfg:
  
  _DEFAULT_DOCKER_REG_PATH :str = '/opt/stackql/registry' 

  def __init__(
    self,
    cwd: str,
    local_path :str,
    remote_url :str = '', 
    nop_verify :bool = True,
    src_prefix :str = '',
    is_null_registry :bool = False,
    docker_reg_path :Optional[str] = None
  ) -> None:
    self._cwd :str = cwd
    self.local_path :str = local_path
    self.remote_url :str = remote_url
    self.nop_verify :bool = nop_verify
    self.src_prefix :str = src_prefix
    self.is_null_registry :bool = is_null_registry
    self._docker_reg_path :str = docker_reg_path if docker_reg_path is not None else self._DEFAULT_DOCKER_REG_PATH

  def _get_local_path(self, execution_environment :str) -> str:
    if self.local_path == '':
      return ''
    if execution_environment == "docker":
      return self._docker_reg_path
    return os.path.join(self._cwd, self.local_path)
  
  def _get_url(self, execution_environment :str) -> str:
    if self.remote_url != '':
      return self.remote_url
    if execution_environment == "docker":
      return f'file://{self._docker_reg_path}'
    return f'file://{os.path.join(self._cwd, self.local_path)}'
  
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
