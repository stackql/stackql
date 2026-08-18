// Conformance launcher (not published): resolves the StackQL MCP server
// binary via the library (STACKQL_MCP_BIN / STACKQL_MCP_BUNDLE, the shared
// cache, or a pin-verified download) and runs it with the canonical launch
// arguments and inherited stdio. Extra argv is forwarded to the server.
//
//   ./gradlew -q :launcher:installDist
//   python scripts/smoke-test.py --cmd "kotlin/launcher/build/install/stackql-mcp-launch/bin/stackql-mcp-launch"
plugins {
    alias(libs.plugins.kotlin.jvm)
    application
}

dependencies {
    implementation(project(":stackql-mcp"))
    runtimeOnly(libs.slf4j.nop)
}

application {
    mainClass.set("io.stackql.mcp.launcher.MainKt")
    applicationName = "stackql-mcp-launch"
}
