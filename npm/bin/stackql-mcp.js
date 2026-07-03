#!/usr/bin/env node
/*
 * stackql-mcp - npx-able launcher for the StackQL MCP server.
 *
 * On first run, downloads the platform's signed .mcpb bundle from the GitHub
 * release pinned in platforms.json, verifies its sha256, extracts the stackql
 * binary into ~/.stackql/mcp-server-bin/<version>/, then spawns it as an MCP
 * stdio server. Subsequent runs use the cached binary.
 *
 * Extra arguments are passed through to stackql after the standard MCP args,
 * e.g.: npx -y @stackql/mcp-server --auth='{"github":{"type":"null_auth"}}'
 *
 * Env overrides:
 *   STACKQL_MCP_BIN     path to an existing stackql binary (skips download)
 *   STACKQL_MCP_BUNDLE  path to a local .mcpb to extract from (CI/testing;
 *                       skips download and sha verification)
 *
 * All diagnostics go to stderr - stdout belongs to the MCP protocol.
 */
"use strict";

const { spawn } = require("child_process");
const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");

const manifest = require("../platforms.json");

function platformKey() {
  const { platform, arch } = process;
  if (platform === "linux" && arch === "x64") return "linux-x64";
  if (platform === "linux" && arch === "arm64") return "linux-arm64";
  if (platform === "win32" && arch === "x64") return "windows-x64";
  if (platform === "darwin") return "darwin-universal"; // universal binary covers x64 + arm64
  return null;
}

// Distinct per-vector UA so the download proxy can attribute traffic to the
// npm wrapper (vs the PyPI one) and the version that fetched.
const USER_AGENT = `stackql-mcp-server-npm/${manifest.version}`;

async function download(url) {
  const res = await fetch(url, {
    redirect: "follow",
    headers: { "User-Agent": USER_AGENT },
  });
  if (!res.ok) {
    throw new Error(`download failed: HTTP ${res.status} for ${url}`);
  }
  return Buffer.from(await res.arrayBuffer());
}

function extractBinary(bundleBuf, entryName, destPath) {
  const AdmZip = require("adm-zip");
  const zip = new AdmZip(bundleBuf);
  const entry = zip.getEntry(entryName);
  if (!entry) {
    throw new Error(`${entryName} not found in bundle`);
  }
  fs.mkdirSync(path.dirname(destPath), { recursive: true });
  // write-then-rename so a concurrent first run cannot see a half-written binary
  const tmp = `${destPath}.tmp-${process.pid}`;
  fs.writeFileSync(tmp, entry.getData(), { mode: 0o755 });
  fs.renameSync(tmp, destPath);
}

async function ensureBinary() {
  if (process.env.STACKQL_MCP_BIN) {
    return process.env.STACKQL_MCP_BIN;
  }

  const key = platformKey();
  if (!key) {
    throw new Error(`unsupported platform: ${process.platform}/${process.arch}`);
  }
  const info = manifest.platforms[key];
  const binName = key === "windows-x64" ? "stackql.exe" : "stackql";
  const binPath = path.join(
    os.homedir(), ".stackql", "mcp-server-bin", manifest.version, key, binName
  );
  if (fs.existsSync(binPath)) {
    return binPath;
  }

  let bundleBuf;
  if (process.env.STACKQL_MCP_BUNDLE) {
    bundleBuf = fs.readFileSync(process.env.STACKQL_MCP_BUNDLE);
  } else {
    const url = `${manifest.baseUrl}/${info.bundle}`;
    console.error(`stackql-mcp: downloading ${info.bundle} (first run only) ...`);
    bundleBuf = await download(url);
    const digest = crypto.createHash("sha256").update(bundleBuf).digest("hex");
    if (digest !== info.sha256) {
      throw new Error(
        `sha256 mismatch for ${info.bundle}\n  expected ${info.sha256}\n  got      ${digest}`
      );
    }
  }
  extractBinary(bundleBuf, `server/${binName}`, binPath);
  console.error(`stackql-mcp: installed ${binPath}`);
  return binPath;
}

async function main() {
  const bin = await ensureBinary();
  // approot and the audit sink must not depend on the cwd: MCP clients may
  // launch this with cwd '/' (read-only on macOS). Later duplicate flags win,
  // so user-passed overrides still take effect.
  const args = [
    "mcp",
    "--mcp.server.type=stdio",
    "--approot", path.join(os.homedir(), ".stackql"),
    "--mcp.config", JSON.stringify({ server: { audit: { disabled: true } } }),
    ...process.argv.slice(2),
  ];
  const child = spawn(bin, args, { stdio: "inherit", windowsHide: true });
  for (const sig of ["SIGINT", "SIGTERM"]) {
    process.on(sig, () => {
      try { child.kill(sig); } catch {}
    });
  }
  child.on("error", (err) => {
    console.error(`stackql-mcp: failed to start server: ${err.message}`);
    process.exit(1);
  });
  child.on("exit", (code, signal) => {
    process.exit(signal ? 1 : code === null ? 1 : code);
  });
}

main().catch((err) => {
  console.error(`stackql-mcp: ${err.message}`);
  process.exit(1);
});
