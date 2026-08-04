"use strict";

const { spawn } = require("child_process");
const os = require("os");
const path = require("path");

const stackqlRoot =
  process.env.STACKQL_PLUGIN_CONFIG_DIR || path.join(os.homedir(), ".stackql");
const args = [
  "-y",
  "@stackql/mcp-server@0.10.582",
  "--approot",
  stackqlRoot,
  "--env.file",
  path.join(stackqlRoot, ".env"),
  "--configfile",
  path.join(stackqlRoot, ".stackqlrc"),
];
const child = spawn("npx", args, {
  shell: process.platform === "win32",
  stdio: "inherit",
  windowsHide: true,
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => {
    try {
      child.kill(signal);
    } catch {}
  });
}

child.on("error", (err) => {
  console.error(`stackql-plugin: ${err.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  process.exit(signal ? 1 : code === null ? 1 : code);
});
