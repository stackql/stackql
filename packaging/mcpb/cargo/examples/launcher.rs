//! Conformance launcher: resolve the server binary, then run it with the
//! canonical launch args and inherited stdio. This is the command the
//! packaging repo's scripts/smoke-test.py drives.
//!
//! Extra argv (e.g. `--auth={...}`) is forwarded to the server verbatim:
//!
//! ```text
//! cargo run --example launcher -- '--auth={"github": {"type": "null_auth"}}'
//! ```

use stackql_mcp::{Mode, StackqlMcp};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut cmd = StackqlMcp::builder().mode(Mode::ReadOnly).command()?;
    cmd.args(std::env::args().skip(1));
    let status = cmd.status()?;
    std::process::exit(status.code().unwrap_or(1));
}
