// The library: io.stackql:stackql-mcp. Acquire the stackql server binary,
// launch it as an MCP stdio server, return a connected kotlin-sdk client.
plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
    `java-library`
    `maven-publish`
    signing
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

java {
    withSourcesJar()
    withJavadocJar()
}

publishing {
    publications {
        create<MavenPublication>("maven") {
            from(components["java"])
            artifactId = "stackql-mcp"
            pom {
                name.set("stackql-mcp")
                description.set(project.description)
                url.set("https://github.com/stackql/stackql-mcp-kotlin")
                licenses {
                    license {
                        name.set("MIT License")
                        url.set("https://github.com/stackql/stackql-mcp-kotlin/blob/main/LICENSE")
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
                    url.set("https://github.com/stackql/stackql-mcp-kotlin")
                    connection.set("scm:git:https://github.com/stackql/stackql-mcp-kotlin.git")
                    developerConnection.set("scm:git:ssh://git@github.com/stackql/stackql-mcp-kotlin.git")
                }
            }
        }
    }
    repositories {
        // Maven Central via the Central Portal OSSRH staging API. Publishing
        // is manual (portal 2FA), consistent with the npm/PyPI stance.
        maven {
            name = "centralPortal"
            url = uri("https://ossrh-staging-api.central.sonatype.com/service/local/staging/deploy/maven2/")
            credentials {
                username = System.getenv("CENTRAL_USERNAME")
                password = System.getenv("CENTRAL_PASSWORD")
            }
        }
    }
}

signing {
    // Only sign when a key is configured (release time), so local and CI
    // builds without secrets still work.
    val signingKey = System.getenv("SIGNING_KEY")
    val signingPassword = System.getenv("SIGNING_PASSWORD")
    isRequired = signingKey != null
    if (signingKey != null) {
        useInMemoryPgpKeys(signingKey, signingPassword)
        sign(publishing.publications["maven"])
    }
}
