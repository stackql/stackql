// stackql-mcp-kotlin: the JVM member of the StackQL embedded-MCP family.
//
//   stackql-mcp        - the library: acquire the server binary, spawn it over
//                        stdio, return a connected MCP client
//   costgate           - the demo core: price a declared resource intent with
//                        read_only stackql tools and gate a build on a budget
//   costgate-cli       - costgate as a standalone command
//   costgate-gradle    - costgate as a Gradle plugin (the `costgate` task)

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
include("costgate")
include("costgate-cli")
include("costgate-gradle")
