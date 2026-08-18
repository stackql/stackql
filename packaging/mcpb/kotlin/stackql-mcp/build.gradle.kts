// The library: io.stackql:stackql-mcp. Acquire the stackql server binary,
// launch it as an MCP stdio server, return a connected kotlin-sdk client.
plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
    `java-library`
    alias(libs.plugins.maven.publish)
}

description = "Embedded StackQL MCP server for Kotlin/JVM agentic apps"

dependencies {
    api(libs.mcp.kotlin.sdk)
    api(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.serialization.json)

    testImplementation(libs.junit.jupiter)
    testImplementation(libs.kotlinx.coroutines.core)
    testRuntimeOnly(libs.junit.platform.launcher)
    // The kotlin-sdk routes its logs through slf4j; silence them in tests.
    testRuntimeOnly(libs.slf4j.nop)
}

// Fail early with a pointer at the renderer when the pin manifest is absent
// (it is rendered, not committed).
tasks.named("processResources") {
    doFirst {
        val manifest = layout.projectDirectory.file("src/main/resources/platforms.json").asFile
        check(manifest.isFile) {
            "missing ${manifest.path}: render it with 'make kotlin-manifest VERSION=X.Y.Z' from packaging/mcpb"
        }
    }
}

// Maven Central via the Central Portal (gradle-maven-publish-plugin):
//   ./gradlew :stackql-mcp:publishToMavenCentral
// Credentials and signing key from the environment (CI: repo secrets):
//   ORG_GRADLE_PROJECT_mavenCentralUsername / ORG_GRADLE_PROJECT_mavenCentralPassword
//   ORG_GRADLE_PROJECT_signingInMemoryKey / ORG_GRADLE_PROJECT_signingInMemoryKeyPassword
mavenPublishing {
    publishToMavenCentral(automaticRelease = true)
    // Sign only when a key is configured (release time), so local builds and
    // PR CI without secrets still assemble and publishToMavenLocal.
    if (providers.gradleProperty("signingInMemoryKey").isPresent) {
        signAllPublications()
    }
    coordinates("io.stackql", "stackql-mcp", version.toString())
    pom {
        name.set("stackql-mcp")
        description.set(project.description)
        url.set("https://github.com/stackql/stackql/tree/main/packaging/mcpb/kotlin")
        licenses {
            license {
                name.set("MIT License")
                url.set("https://github.com/stackql/stackql/blob/main/packaging/mcpb/kotlin/LICENSE")
            }
        }
        developers {
            developer {
                id.set("stackql")
                name.set("StackQL Studios")
                email.set("info@stackql.io")
            }
        }
        scm {
            url.set("https://github.com/stackql/stackql")
            connection.set("scm:git:https://github.com/stackql/stackql.git")
            developerConnection.set("scm:git:ssh://git@github.com/stackql/stackql.git")
        }
    }
}
