// stackql-mcp (Kotlin/JVM): the JVM member of the StackQL embedded-MCP family,
// owned by packaging/mcpb in stackql/stackql.
//
//   stackql-mcp  - the library: acquire the server binary, spawn it over
//                  stdio, return a connected MCP client (io.stackql:stackql-mcp)
//   launcher     - the conformance launcher smoke-test.py drives (not published)
//
// The costgate demo (core, CLI, Gradle plugin) lives in stackql-labs.

pluginManagement {
    repositories {
        gradlePluginPortal()
        mavenCentral()
    }
}

dependencyResolutionManagement {
    repositories {
        mavenCentral()
    }
}

rootProject.name = "stackql-mcp-kotlin"

include("stackql-mcp")
include("launcher")
