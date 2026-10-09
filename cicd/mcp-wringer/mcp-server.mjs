#!/usr/bin/env node
// Helper for .github/workflows/mcp-wringer.yml.  It owns the stackql MCP
// server invocation used by the fuzzing gate so that every mode (stdio or
// Streamable HTTP, each protocol revision) is started the same hermetic way:
// a throwaway approot, the repository's file:// provider registry, audit off,
// and no credentials.
//
//   node cicd/mcp-wringer/mcp-server.mjs prepare
//       Writes `binary` and `stdio_args_<revision>` outputs (GITHUB_OUTPUT)
//       for the Action's `command` and `args` inputs.
//   node cicd/mcp-wringer/mcp-server.mjs start --name NAME --port PORT [--protocol-version REV]
//       Starts a detached Streamable HTTP server, waits until it answers, and
//       records its pid under RUNNER_TEMP.
//   node cicd/mcp-wringer/mcp-server.mjs stop --name NAME
//       Stops the server started under NAME and prints its log tail.
//
// Spawning uses an argument array and no shell, so JSON arguments survive
// every runner OS.  The script uses only the Node standard library.

import { spawn } from "node:child_process";
import { appendFileSync, closeSync, existsSync, mkdirSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import process from "node:process";

const workspace = resolve(process.env.GITHUB_WORKSPACE ?? process.cwd());
const scratch = join(process.env.RUNNER_TEMP ?? tmpdir(), "stackql-mcp-wringer");
const binary = join(workspace, "build", process.platform === "win32" ? "stackql.exe" : "stackql");
const registryRoot = toForwardSlashes(join(workspace, "test", "registry", "src"));
const readinessTimeoutMs = 60_000;

function toForwardSlashes(path) {
  return path.replaceAll("\\", "/");
}

function registryConfig() {
  return JSON.stringify({
    url: `file://${registryRoot}`,
    localDocRoot: registryRoot,
    verifyConfig: { nopVerify: true },
  });
}

function serverArgs({ transport, approot, protocolVersion, address }) {
  const serverConfig = { audit: { disabled: true } };
  if (transport === "http") {
    serverConfig.allow_unauthenticated = true;
    serverConfig.address = address;
  }
  const args = [
    "mcp",
    `--mcp.server.type=${transport}`,
    "--approot",
    toForwardSlashes(approot),
    "--registry",
    registryConfig(),
    "--mcp.config",
    JSON.stringify({ server: serverConfig }),
  ];
  if (protocolVersion !== undefined) {
    args.push(`--mcp.protocol.version=${protocolVersion}`);
  }
  return args;
}

function parseFlags(argv) {
  const flags = {};
  for (let index = 0; index < argv.length; index += 1) {
    const current = argv[index];
    if (!current.startsWith("--")) {
      throw new Error(`unexpected argument: ${current}`);
    }
    const value = argv[index + 1];
    if (value === undefined || value.startsWith("--")) {
      throw new Error(`flag ${current} needs a value`);
    }
    flags[current.slice(2)] = value;
    index += 1;
  }
  return flags;
}

function requireFlag(flags, name) {
  const value = flags[name];
  if (value === undefined) {
    throw new Error(`--${name} is required`);
  }
  return value;
}

function setOutput(name, value) {
  const outputPath = process.env.GITHUB_OUTPUT;
  if (outputPath === undefined || outputPath.length === 0) {
    process.stdout.write(`${name}=${value}\n`);
    return;
  }
  // Values hold JSON, so use the heredoc form rather than name=value.
  const delimiter = `EOF_${name}_${process.pid}`;
  appendFileSync(outputPath, `${name}<<${delimiter}\n${value}\n${delimiter}\n`, "utf8");
}

function pidFile(name) {
  return join(scratch, `${name}.pid`);
}

function logFile(name) {
  return join(scratch, `${name}.log`);
}

function printLogTail(name, lines = 40) {
  const path = logFile(name);
  if (!existsSync(path)) {
    return;
  }
  const tail = readFileSync(path, "utf8").split(/\r?\n/u).filter((line) => line.length > 0).slice(-lines);
  process.stderr.write(`--- ${name} log (last ${tail.length} lines) ---\n${tail.join("\n")}\n`);
}

function sleep(ms) {
  return new Promise((resolvePromise) => setTimeout(resolvePromise, ms));
}

async function waitUntilListening(url, child) {
  const deadline = Date.now() + readinessTimeoutMs;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) {
      throw new Error(`server exited with code ${child.exitCode} before listening`);
    }
    try {
      // Any HTTP response means the listener is up.  A bare GET without the
      // Streamable HTTP Accept header is answered with 400, which is fine.
      await fetch(url, { method: "GET", signal: AbortSignal.timeout(2_000) });
      return;
    } catch {
      await sleep(250);
    }
  }
  throw new Error(`server did not answer on ${url} within ${readinessTimeoutMs} ms`);
}

function prepare() {
  if (!existsSync(binary)) {
    throw new Error(`stackql binary not found at ${binary}; build it first`);
  }
  mkdirSync(scratch, { recursive: true });
  setOutput("binary", toForwardSlashes(binary));
  for (const revision of ["2025-11-25", "2026-07-28"]) {
    const approot = join(scratch, `approot-stdio-${revision}`);
    mkdirSync(approot, { recursive: true });
    // Revision 2025-11-25 is served by the default "auto" ceiling.  The
    // sessionless revision is pinned explicitly so the server runs in its
    // 2026-07-28-only configuration.
    const protocolVersion = revision === "2026-07-28" ? revision : undefined;
    setOutput(
      `stdio_args_${revision.replaceAll("-", "_")}`,
      JSON.stringify(serverArgs({ transport: "stdio", approot, protocolVersion })),
    );
  }
}

async function start(flags) {
  const name = requireFlag(flags, "name");
  const port = Number(requireFlag(flags, "port"));
  if (!Number.isInteger(port) || port <= 0 || port > 65_535) {
    throw new Error("--port must be a TCP port number");
  }
  if (!existsSync(binary)) {
    throw new Error(`stackql binary not found at ${binary}; build it first`);
  }
  mkdirSync(scratch, { recursive: true });
  const approot = join(scratch, `approot-${name}`);
  mkdirSync(approot, { recursive: true });
  const address = `127.0.0.1:${port}`;
  const args = serverArgs({
    transport: "http",
    approot,
    address,
    protocolVersion: flags["protocol-version"],
  });
  // The log is a file descriptor rather than a pipe so that this process
  // holds nothing open on the child and can exit once the server answers.
  const log = openSync(logFile(name), "w");
  const child = spawn(binary, args, {
    cwd: workspace,
    detached: true,
    stdio: ["ignore", log, log],
    windowsHide: true,
  });
  closeSync(log);
  child.unref();
  writeFileSync(pidFile(name), String(child.pid), "utf8");
  process.stderr.write(`started ${name} (pid ${child.pid}) on http://${address}/\n`);
  try {
    await waitUntilListening(`http://${address}/`, child);
  } catch (error) {
    printLogTail(name);
    throw error;
  }
  process.stderr.write(`${name} is listening\n`);
  setOutput("url", `http://${address}/`);
}

function stop(flags) {
  const name = requireFlag(flags, "name");
  const path = pidFile(name);
  if (!existsSync(path)) {
    process.stderr.write(`no pid file for ${name}; nothing to stop\n`);
    return;
  }
  const pid = Number(readFileSync(path, "utf8").trim());
  try {
    process.kill(pid);
    process.stderr.write(`stopped ${name} (pid ${pid})\n`);
  } catch (error) {
    process.stderr.write(`could not stop ${name} (pid ${pid}): ${error instanceof Error ? error.message : String(error)}\n`);
  }
  printLogTail(name);
}

async function main() {
  const [command, ...rest] = process.argv.slice(2);
  const flags = parseFlags(rest);
  switch (command) {
    case "prepare":
      prepare();
      return;
    case "start":
      await start(flags);
      return;
    case "stop":
      stop(flags);
      return;
    default:
      throw new Error("usage: mcp-server.mjs prepare | start --name NAME --port PORT [--protocol-version REV] | stop --name NAME");
  }
}

main().catch((error) => {
  process.stderr.write(`mcp-server.mjs: ${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
});
