//! Fetch the pinned platform .mcpb into the shared cache and print its path.
//! The producer step for vendored builds:
//!
//! ```text
//! BUNDLE=$(cargo run --example fetch_bundle)
//! STACKQL_MCP_BUNDLE_FILE=$BUNDLE cargo build -p auditron --features vendored --release
//! ```

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let path = stackql_mcp::fetch_bundle()?;
    println!("{}", path.display());
    Ok(())
}
