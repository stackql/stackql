// costgate core: read a resource intent, price it, gate on a budget. The CLI
// and the Gradle plugin are thin front-ends over this module.
plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
}

description = "costgate core: price a declared cloud resource intent and gate a build on a budget"

dependencies {
    api(project(":stackql-mcp"))
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.kaml)
    implementation(libs.kotlinx.coroutines.core)

    testImplementation(libs.junit.jupiter)
    testRuntimeOnly(libs.junit.platform.launcher)
    testRuntimeOnly(libs.slf4j.nop)
}
