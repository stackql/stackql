//! Ten lines to a connected StackQL MCP client.
//!
//! Uses the github provider in null_auth mode, so it runs with zero cloud
//! credentials. First run downloads and caches the server binary.

use stackql_mcp::{Mode, StackqlMcp};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let server = StackqlMcp::builder()
        .mode(Mode::ReadOnly)
        .auth(serde_json::json!({"github": {"type": "null_auth"}}))
        .start()
        .await?;
    let tools = server.list_all_tools().await?;
    println!("connected to stackql mcp server: {} tools", tools.len());
    for tool in &tools {
        println!("  {}", tool.name);
    }
    server.shutdown().await?;
    Ok(())
}
