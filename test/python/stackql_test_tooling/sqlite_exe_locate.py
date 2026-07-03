"""Defensive resolution of the sqlite3 CLI executable for export tests.

The Windows CI runner installs sqlite with `choco install --force sqlite` and
points SQLITE_EXE at the historical layout
(C:\\ProgramData\\chocolatey\\lib\\SQLite\\tools\\sqlite3.exe). Newer versions
of the chocolatey package nest the binary one level deeper (a
sqlite-tools-win-* folder), which broke the hardcoded path with
FileNotFoundError. Resolve in order:

1. The configured value, when it is an existing file or a bare command name
   resolvable on PATH.
2. `sqlite3` on PATH.
3. A recursive search of the chocolatey SQLite lib tree (newest hit wins).

Falls back to the configured value so a genuine miss still surfaces the
original, informative error at call time.
"""

import glob
import os
import shutil

_CHOCO_SQLITE_TOOLS_GLOB = r"C:\ProgramData\chocolatey\lib\SQLite\tools\**\sqlite3.exe"


def resolve_sqlite_exe(configured: str) -> str:
    if not configured:
        configured = "sqlite3"
    if os.path.isfile(configured):
        return configured
    if os.path.sep not in configured and shutil.which(configured):
        return configured
    path_hit = shutil.which("sqlite3")
    if path_hit:
        return path_hit
    if os.name == "nt":
        hits = sorted(glob.glob(_CHOCO_SQLITE_TOOLS_GLOB, recursive=True))
        if hits:
            return hits[-1]
    return configured
