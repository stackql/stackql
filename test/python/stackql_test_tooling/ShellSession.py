import subprocess
import sys
import typing


class ShellSessionResult:


  def __init__(self, stdout_str :str, stderr_str :str, rc :int) -> None:
    self.stdout :str = stdout_str
    self.stderr :str = stderr_str
    self.rc = rc


class ShellSession:


  def __init__(self):
    pass


  def _start(
    self, 
    cmd_arg_list :typing.Iterable[str], 
    stdin :typing.Any = subprocess.PIPE,
    stdout :typing.Any = subprocess.PIPE,
    stderr :typing.Any = subprocess.PIPE
  ) -> subprocess.Popen:
    command = [item.encode(sys.getdefaultencoding()) for item in cmd_arg_list]
    return subprocess.Popen(
      command,
      stdin=stdin,
      stdout=stdout,
      stderr=stderr
    )


  def _write(self, process :subprocess.Popen, message :str):
    process.stdin.write(f"{message.strip()}\n".encode(sys.getdefaultencoding()))
    process.stdin.flush()


  def _format_output(self, output :bytes) -> str:
    if not output:
      return ''
    output = output.decode(sys.getdefaultencoding())
    output = output.replace('\r\n', '\n')
    if output.endswith('\n'):
        output = output[:-1]
    return output


  def _extract_stream_from_tmp_file(self, file_path :str) -> str:
    with open(file_path, 'rb') as f:
      b = f.read()
      return self._format_output(b)


  def run_shell_session(
    self, 
    cmd_arg_list :typing.Iterable[str], 
    session_commands :typing.Iterable[str],
    stdin :typing.Any = subprocess.PIPE,
    stdout :typing.Any = subprocess.PIPE,
    stderr :typing.Any = subprocess.PIPE
  ) -> ShellSessionResult:
    stdout_raw = stdout
    stderr_raw = stderr
    if type(stdout_raw) == str:
      stdout = open(stdout_raw, 'wb')
    if type(stderr_raw) == str:
      stderr = open(stderr_raw, 'wb')
    process = self._start(
      cmd_arg_list,
      stdin=stdin,
      stdout=stdout,
      stderr=stderr,
    )
    for cmd in session_commands:
      self._write(process, cmd)
    stdout_bytes, stderr_bytes = process.communicate()
    stdout_str = self._format_output(stdout_bytes)
    stderr_str = self._format_output(stderr_bytes)
    if stdout_str == '' and type(stdout_raw) == str:
      stdout_str = self._extract_stream_from_tmp_file(stdout_raw)
    if stderr_str == '' and type(stderr_raw) == str:
      stderr_str = self._extract_stream_from_tmp_file(stderr_raw)
    return ShellSessionResult(stdout_str, stderr_str, process.returncode)

