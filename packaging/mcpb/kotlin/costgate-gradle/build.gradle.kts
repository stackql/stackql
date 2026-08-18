// costgate as a Gradle plugin: a `costgate` task that gates the build on cost.
// Published to the Gradle Plugin Portal (a separate, uncrowded directory).
plugins {
    alias(libs.plugins.kotlin.jvm)
    `java-gradle-plugin`
    alias(libs.plugins.gradle.plugin.publish)
}

description = "Gradle plugin: cloud cost as a build check (the costgate task)"

dependencies {
    implementation(project(":costgate"))
    implementation(libs.kotlinx.coroutines.core)
    runtimeOnly(libs.slf4j.nop)

    testImplementation(libs.junit.jupiter)
    testImplementation(gradleTestKit())
    testRuntimeOnly(libs.junit.platform.launcher)
}

@Suppress("UnstableApiUsage")
gradlePlugin {
    website.set("https://github.com/stackql/stackql-mcp-kotlin")
    vcsUrl.set("https://github.com/stackql/stackql-mcp-kotlin.git")
    plugins {
        create("costgate") {
            id = "io.stackql.costgate"
            implementationClass = "io.stackql.costgate.gradle.CostgatePlugin"
            displayName = "costgate"
            description = "Price a declared cloud resource intent and fail the build when it exceeds the budget."
            tags.set(listOf("cost", "cloud", "stackql", "finops", "ci", "mcp"))
        }
    }
}
