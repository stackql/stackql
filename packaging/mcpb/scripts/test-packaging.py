"""Packaging regressions. Requires bash, node, npm dependencies and the MCPB CLI."""

import hashlib
import importlib.util
import json
import os
import platform
from pathlib import Path
import shutil
import struct
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch
import zipfile


ROOT = Path(__file__).resolve().parents[1]
TARGETS = ("linux-x64", "linux-arm64", "windows-x64", "windows-arm64", "darwin-universal")
BASH = os.environ.get("BASH") or (
    str(Path(os.environ.get("ProgramFiles", "C:/Program Files")) / "Git/bin/bash.exe")
    if os.name == "nt" else "bash"
)


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def pe(machine):
    data = bytearray(256)
    data[:2] = b"MZ"
    struct.pack_into("<I", data, 0x3C, 0x80)
    data[0x80:0x84] = b"PE\0\0"
    struct.pack_into("<H", data, 0x84, machine)
    return bytes(data)


class PackagingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory(prefix="mcpb tests ")
        cls.root = Path(cls.temp.name)
        cls.dist = cls.root / "dist"
        cls.dist.mkdir()
        cls.payloads = {
            "linux-x64": b"\x7fELF" + bytes(60),
            "linux-arm64": b"\x7fELF" + bytes(60),
            "windows-x64": pe(0x8664),
            "windows-arm64": pe(0xAA64),
            "darwin-universal": b"\xca\xfe\xba\xbe" + bytes(60),
        }
        for target, data in cls.payloads.items():
            bundle = cls.dist / f"stackql-mcp-{target}.mcpb"
            member = "server/stackql.exe" if target.startswith("windows-") else "server/stackql"
            with zipfile.ZipFile(bundle, "w") as archive:
                archive.writestr(member, data)
            bundle.with_suffix(".mcpb.sha256").write_text(
                hashlib.sha256(bundle.read_bytes()).hexdigest() + "  " + bundle.name + "\n"
            )
        cls.env = {**os.environ, "DIST_DIR": cls.dist.as_posix()}
        cls.run_script("package.sh", "--version", "0.0.1", "--combined")
        cls.combined = cls.dist / "stackql-mcp-multiplatform.mcpb"

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    @classmethod
    def run_script(cls, name, *args, env=None, check=True):
        result = subprocess.run(
            [BASH, (ROOT / "scripts" / name).as_posix(), *args],
            env=env or cls.env, capture_output=True, text=True, encoding="utf-8",
        )
        if check and result.returncode:
            raise AssertionError(result.stdout + result.stderr)
        return result

    def test_combined_preserves_binaries_and_checks_pe_architecture(self):
        verifier = ROOT / "scripts" / "verify-combined.py"
        result = subprocess.run([sys.executable, str(verifier), str(self.combined)], capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        with zipfile.ZipFile(self.combined) as bundle:
            for target, data in self.payloads.items():
                folder = "darwin" if target == "darwin-universal" else target
                name = "stackql.exe" if target.startswith("windows-") else "stackql"
                self.assertEqual(bundle.read(f"server/{folder}/{name}"), data)
            wrong = self.root / "wrong.mcpb"
            with zipfile.ZipFile(wrong, "w") as out:
                for name in bundle.namelist():
                    data = bundle.read(name)
                    if name == "server/windows-arm64/stackql.exe":
                        data = self.payloads["windows-x64"]
                    out.writestr(name, data)
        result = subprocess.run([sys.executable, str(verifier), str(wrong)], capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(b"windows ARM64 PE", result.stderr)

    def test_combined_requires_windows_arm64(self):
        incomplete = self.root / "incomplete"
        incomplete.mkdir()
        for target in TARGETS:
            if target != "windows-arm64":
                shutil.copy(self.dist / f"stackql-mcp-{target}.mcpb", incomplete)
        result = self.run_script("package.sh", "--version", "0.0.1", "--combined",
                                 env={**self.env, "DIST_DIR": incomplete.as_posix()}, check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing: windows-arm64", result.stderr)

    def test_registry_and_wrapper_pins_include_windows_arm64(self):
        out = self.root / "server.json"
        self.run_script("render-server-json.sh", "--version", "0.0.1",
                        env={**self.env, "OUT": out.as_posix()})
        packages = json.loads(out.read_text())["packages"]
        arm = next(p for p in packages if p["identifier"].endswith("windows-arm64.mcpb"))
        digest = hashlib.sha256((self.dist / "stackql-mcp-windows-arm64.mcpb").read_bytes()).hexdigest()
        self.assertEqual(arm["fileSha256"], digest)
        self.run_script("render-platforms.sh", "--version", "0.0.1", "--vector", "go",
                        env={**self.env, "CHECKSUM_DIR": self.dist.as_posix(), "GO_DIR": self.root.as_posix()})
        platforms = json.loads((self.root / "embed/platforms.json").read_text())["platforms"]
        self.assertEqual(set(platforms), set(TARGETS))
        self.assertEqual(platforms["windows-arm64"]["sha256"], digest)

    def test_pypi_arm64_extracts_exe_into_its_own_cache(self):
        wrapper = load_module("pypi_wrapper", ROOT / "pypi/src/stackql_mcp_server/__init__.py")
        for machine in ("ARM64", "aarch64"):
            with self.subTest(machine=machine), patch.object(wrapper.sys, "platform", "win32"), \
                    patch.object(wrapper._platform, "machine", return_value=machine):
                self.assertEqual(wrapper._platform_key(), "windows-arm64")
        manifest = {"version": "0.0.1", "platforms": {"windows-arm64": {}}}
        with patch.object(wrapper, "_platform_key", return_value="windows-arm64"), \
                patch.object(wrapper, "_load_manifest", return_value=manifest), \
                patch.object(Path, "home", return_value=self.root / "pypi-home"), \
                patch.dict(os.environ, {"STACKQL_MCP_BUNDLE": str(self.dist / "stackql-mcp-windows-arm64.mcpb")}, clear=True):
            binary = Path(wrapper._ensure_binary())
            self.assertEqual(binary.parent.name, "windows-arm64")
            self.assertEqual(binary.name, "stackql.exe")
            self.assertEqual(binary.read_bytes(), self.payloads["windows-arm64"])

    def test_wrapper_manifest_rejects_missing_checksum(self):
        result = self.run_script("render-platforms.sh", "--version", "0.0.1", "--vector", "go",
                                 env={**self.env, "CHECKSUM_DIR": (self.root / "missing").as_posix(),
                                      "GO_DIR": (self.root / "missing-go").as_posix()}, check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.root / "missing-go/embed/platforms.json").exists())

    @unittest.skipUnless(os.name == "nt", "Windows launcher")
    def test_windows_launcher_preserves_arguments_and_exit_code(self):
        dest = self.root / "install with spaces"
        with zipfile.ZipFile(self.combined) as archive:
            archive.extractall(dest)
        source = self.root / "probe.go"
        source.write_text('''package main
import ("encoding/json"; "os")
func main() { json.NewEncoder(os.Stdout).Encode(os.Args); os.Exit(7) }
''')
        binary = self.root / "probe.exe"
        subprocess.run(["go", "build", "-o", str(binary), str(source)], check=True)
        for target in ("windows-x64", "windows-arm64"):
            shutil.copy(binary, dest / "server" / target / "stackql.exe")
        smoke = load_module("bundle_smoke", ROOT / "scripts/smoke-test.py")
        manifest = json.loads((dest / "manifest.json").read_text())
        arguments = smoke.manifest_args(manifest, dest, self.root / "home with spaces")
        arguments += ['--auth={"token":"a b\\c"}', 'trailing\\', '', '& literal | < >']
        result = subprocess.run(["powershell.exe", *arguments], capture_output=True, text=True)
        self.assertEqual(result.returncode, 7, result.stderr)
        argv = json.loads(result.stdout)
        expected = "windows-arm64" if platform.machine().lower() == "arm64" else "windows-x64"
        self.assertEqual(Path(argv[0]).parent.name, expected)
        self.assertEqual(argv[1:], arguments[arguments.index("-File") + 2:])

    def test_npm_arm64_extracts_exe_into_its_own_cache(self):
        npm = self.root / "npm"
        (npm / "bin").mkdir(parents=True)
        shutil.copy(ROOT / "npm/bin/stackql-mcp.js", npm / "bin")
        (npm / "platforms.json").write_text(json.dumps({
            "version": "0.0.1", "platforms": {"windows-arm64": {}}
        }))
        script = r'''
Object.defineProperty(process, "platform", {value: "win32"});
Object.defineProperty(process, "arch", {value: "arm64"});
require("os").homedir = () => process.env.TEST_HOME;
require("child_process").spawn = (bin) => {
  console.log(JSON.stringify(bin));
  const child = new (require("events").EventEmitter)();
  setImmediate(() => child.emit("exit", 0));
  return child;
};
require(process.env.TEST_LAUNCHER);
'''
        env = {**os.environ, "STACKQL_MCP_BIN": "", "STACKQL_MCP_BUNDLE": str(self.dist / "stackql-mcp-windows-arm64.mcpb"),
               "NODE_PATH": str(ROOT / "npm/node_modules"), "TEST_HOME": str(self.root / "npm-home"),
               "TEST_LAUNCHER": str(npm / "bin/stackql-mcp.js")}
        result = subprocess.run(["node", "-e", script], env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        binary = Path(json.loads(result.stdout))
        self.assertEqual(binary.parent.name, "windows-arm64")
        self.assertEqual(binary.name, "stackql.exe")
        self.assertEqual(binary.read_bytes(), self.payloads["windows-arm64"])


if __name__ == "__main__":
    unittest.main()
