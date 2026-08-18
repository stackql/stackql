//! Conformance test mirroring stackql-mcpb-packaging's scripts/smoke-test.py:
//! initialize -> tools/list -> pull_provider github -> list_services github,
//! using the github provider in null_auth mode (no cloud credentials).
//!
//! Ignored by default: first run downloads the ~35 MB server bundle. CI runs
//! it with `cargo test -- --include-ignored`.

use std::time::Duration;

use rmcp::model::CallToolRequestParams;
use stackql_mcp::{Mode, StackqlMcp};

const CALL_TIMEOUT: Duration = Duration::from_secs(120);

#[tokio::test]
#[ignore = "network: downloads the server bundle on first run"]
async fn handshake_tools_and_github_fixture() {
    let server = StackqlMcp::builder()
        .mode(Mode::ReadOnly)
        .auth(serde_json::json!({"github": {"type": "null_auth"}}))
        .start()
        .await
        .expect("server should start and complete the MCP handshake");

    assert!(server.pid().is_some(), "server process should be running");

    // tools/list must include the tools the smoke test requires.
    let tools = tokio::time::timeout(CALL_TIMEOUT, server.list_all_tools())
        .await
        .expect("tools/list timed out")
        .expect("tools/list failed");
    let names: Vec<&str> = tools.iter().map(|t| t.name.as_ref()).collect();
    for required in ["pull_provider", "list_services", "list_providers"] {
        assert!(
            names.contains(&required),
            "missing tool {required} in {names:?}"
        );
    }

    // pull_provider github must succeed.
    let mut pull = CallToolRequestParams::new("pull_provider");
    pull.arguments = serde_json::json!({"provider": "github"})
        .as_object()
        .cloned();
    let pulled = tokio::time::timeout(CALL_TIMEOUT, server.call_tool(pull))
        .await
        .expect("pull_provider timed out")
        .expect("pull_provider failed");
    assert_ne!(
        pulled.is_error,
        Some(true),
        "pull_provider errored: {pulled:?}"
    );

    // list_services github must return known github services.
    let mut list = CallToolRequestParams::new("list_services");
    list.arguments = serde_json::json!({"provider": "github", "row_limit": 5})
        .as_object()
        .cloned();
    let services = tokio::time::timeout(CALL_TIMEOUT, server.call_tool(list))
        .await
        .expect("list_services timed out")
        .expect("list_services failed");
    assert_ne!(
        services.is_error,
        Some(true),
        "list_services errored: {services:?}"
    );
    let rendered = serde_json::to_string(&services).unwrap();
    assert!(
        rendered.contains("actions") || rendered.contains("apps"),
        "list_services did not include expected github services: {rendered}"
    );

    server.shutdown().await.expect("shutdown should succeed");
}
