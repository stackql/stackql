#!/usr/bin/env node
/*
 * stackql-mcp - npx-able launcher for the StackQL MCP server.
 *
 * On first run, downloads the platform's signed .mcpb bundle from the GitHub
 * release pinned in platforms.json, verifies its sha256, extracts the stackql
 * binary into ~/.stackql/mcp-server-bin/<version>/, then spawns it as an MCP
 * stdio server. Subsequent runs use the cached binary.
 *
 * This file is also vendored verbatim into the ChatGPT/Codex and Claude plugin
 * packages (packaging/openai-plugin, packaging/claude-plugin) by
 * scripts/render-platforms.sh, so it must stay dependency-free: the only
 * modules it loads are Node.js built-ins, and the .mcpb (a zip archive) is
 * read with the inline extractor below rather than a package.
 *
 * Extra arguments are passed through to stackql after the standard MCP args,
 * e.g.: npx -y @stackql/mcp-server --auth='{"github":{"type":"null_auth"}}'
 *
 * Env overrides:
 *   STACKQL_MCP_BIN           path to an existing stackql binary (skips download)
 *   STACKQL_MCP_BUNDLE        path to a local .mcpb to extract from (CI/testing;
 *                             skips download and sha verification)
 *   STACKQL_PLUGIN_CONFIG_DIR StackQL configuration directory (default
 *                             ~/.stackql): the approot, plus <dir>/.env for
 *                             provider credentials and <dir>/.stackqlrc for
 *                             optional nonstandard auth configuration
 *
 * All diagnostics go to stderr - stdout belongs to the MCP protocol.
 */
"use strict";

const { spawn } = require("child_process");
const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const zlib = require("zlib");

const manifest = require("../platforms.json");

// Node.js floor: the October 2026 security releases fix TLS hostname
// verification bypasses in the fetch() path this launcher downloads through,
// a Brotli decoder RCE, and drop the compromised eMudhra intermediate CA from
// the bundled trust store. Earlier builds of 22/24/26 and every other line
// (18, 20, 23, 25 are End-of-Life) stay unpatched. package.json "engines"
// only warns at install time, so repeat the warning at run time - on stderr,
// because stdout belongs to the MCP protocol.
const PATCHED_NODE = { 22: [22, 23, 4], 24: [24, 22, 1], 26: [26, 12, 1] };
const NODE_ADVISORY =
  "https://nodejs.org/en/blog/vulnerability/october-2026-security-releases";

function compareVersions(a, b) {
  for (let i = 0; i < 3; i++) {
    if (a[i] !== b[i]) return a[i] - b[i];
  }
  return 0;
}

function warnIfUnpatchedNode() {
  const current = process.versions.node.split(".").map(Number);
  if (current[0] > 26) return; // newer lines ship the fixes
  const floor = PATCHED_NODE[current[0]];
  if (floor && compareVersions(current, floor) >= 0) return;
  console.error(
    `stackql-mcp: WARNING Node.js ${process.versions.node} predates the ` +
    "October 2026 security releases (22.23.4 / 24.22.1 / 26.12.1); " +
    `upgrade Node.js - see ${NODE_ADVISORY}`
  );
}

function platformKey() {
  const { platform, arch } = process;
  if (platform === "linux" && arch === "x64") return "linux-x64";
  if (platform === "linux" && arch === "arm64") return "linux-arm64";
  if (platform === "win32" && arch === "x64") return "windows-x64";
  if (platform === "win32" && arch === "arm64") return "windows-arm64";
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

// Minimal zip reader: returns the bytes of one named entry from an in-memory
// archive. Walks the central directory (the authoritative sizes and offsets,
// so data descriptors never matter), honours zip64 records, and supports the
// two methods zip tools emit for a .mcpb: stored (0) and deflate (8). The
// bundle's sha256 is verified against the pin before this runs, which is why
// the per-entry CRC is not rechecked here.
const SIG_EOCD = 0x06054b50;
const SIG_EOCD64_LOCATOR = 0x07064b50;
const SIG_EOCD64 = 0x06064b50;
const SIG_CENTRAL = 0x02014b50;
const SIG_LOCAL = 0x04034b50;
const U32_MAX = 0xffffffff;

function readZipEntry(buf, entryName) {
  const minEocd = Math.max(0, buf.length - 22 - 0xffff);
  let eocd = -1;
  for (let i = buf.length - 22; i >= minEocd; i--) {
    if (buf.readUInt32LE(i) === SIG_EOCD) {
      eocd = i;
      break;
    }
  }
  if (eocd < 0) {
    throw new Error("bundle is not a zip archive (no end-of-central-directory record)");
  }
  let count = buf.readUInt16LE(eocd + 10);
  let cdOffset = buf.readUInt32LE(eocd + 16);
  if (count === 0xffff || cdOffset === U32_MAX) {
    const locator = eocd - 20;
    if (locator < 0 || buf.readUInt32LE(locator) !== SIG_EOCD64_LOCATOR) {
      throw new Error("bundle zip64 locator missing");
    }
    const eocd64 = Number(buf.readBigUInt64LE(locator + 8));
    if (buf.readUInt32LE(eocd64) !== SIG_EOCD64) {
      throw new Error("bundle zip64 end-of-central-directory record missing");
    }
    count = Number(buf.readBigUInt64LE(eocd64 + 32));
    cdOffset = Number(buf.readBigUInt64LE(eocd64 + 48));
  }

  let p = cdOffset;
  for (let n = 0; n < count; n++) {
    if (buf.readUInt32LE(p) !== SIG_CENTRAL) {
      throw new Error("bundle central directory is corrupt");
    }
    const method = buf.readUInt16LE(p + 10);
    let compressedSize = buf.readUInt32LE(p + 20);
    let size = buf.readUInt32LE(p + 24);
    const nameLength = buf.readUInt16LE(p + 28);
    const extraLength = buf.readUInt16LE(p + 30);
    const commentLength = buf.readUInt16LE(p + 32);
    let localOffset = buf.readUInt32LE(p + 42);
    const name = buf.toString("utf8", p + 46, p + 46 + nameLength);
    if (name !== entryName) {
      p += 46 + nameLength + extraLength + commentLength;
      continue;
    }
    if (compressedSize === U32_MAX || size === U32_MAX || localOffset === U32_MAX) {
      // zip64 extended information: only the fields saturated above are
      // present, in this fixed order.
      let e = p + 46 + nameLength;
      const end = e + extraLength;
      while (e + 4 <= end) {
        const id = buf.readUInt16LE(e);
        const length = buf.readUInt16LE(e + 2);
        if (id === 0x0001) {
          let q = e + 4;
          if (size === U32_MAX) { size = Number(buf.readBigUInt64LE(q)); q += 8; }
          if (compressedSize === U32_MAX) { compressedSize = Number(buf.readBigUInt64LE(q)); q += 8; }
          if (localOffset === U32_MAX) { localOffset = Number(buf.readBigUInt64LE(q)); q += 8; }
          break;
        }
        e += 4 + length;
      }
    }
    if (buf.readUInt32LE(localOffset) !== SIG_LOCAL) {
      throw new Error(`bundle local header for ${entryName} is corrupt`);
    }
    const dataStart =
      localOffset + 30 + buf.readUInt16LE(localOffset + 26) + buf.readUInt16LE(localOffset + 28);
    const data = buf.subarray(dataStart, dataStart + compressedSize);
    let out;
    if (method === 0) {
      out = data;
    } else if (method === 8) {
      out = zlib.inflateRawSync(data);
    } else {
      throw new Error(`unsupported zip compression method ${method} for ${entryName}`);
    }
    if (out.length !== size) {
      throw new Error(`${entryName} extracted to ${out.length} bytes, expected ${size}`);
    }
    return out;
  }
  throw new Error(`${entryName} not found in bundle`);
}

function extractBinary(bundleBuf, entryName, destPath) {
  const data = readZipEntry(bundleBuf, entryName);
  fs.mkdirSync(path.dirname(destPath), { recursive: true });
  // write-then-rename so a concurrent first run cannot see a half-written binary
  const tmp = `${destPath}.tmp-${process.pid}`;
  fs.writeFileSync(tmp, data, { mode: 0o755 });
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
  const binName = key.startsWith("windows-") ? "stackql.exe" : "stackql";
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
  warnIfUnpatchedNode();
  const bin = await ensureBinary();
  // The configuration directory holds the approot (provider cache, audit
  // sink) and the two user-owned files the server reads credentials from.
  // None of it may depend on the cwd: MCP clients may launch this with cwd
  // '/' (read-only on macOS). Later duplicate flags win, so user-passed
  // overrides still take effect.
  const configDir =
    process.env.STACKQL_PLUGIN_CONFIG_DIR || path.join(os.homedir(), ".stackql");
  const args = [
    "mcp",
    "--mcp.server.type=stdio",
    "--approot", configDir,
    "--env.file", path.join(configDir, ".env"),
    "--configfile", path.join(configDir, ".stackqlrc"),
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
