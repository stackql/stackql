// costgate as a standalone command. `costgate check --intent costgate.yaml`.
plugins {
    alias(libs.plugins.kotlin.jvm)
    application
}

description = "costgate command-line interface"

dependencies {
    implementation(project(":costgate"))
    implementation(libs.clikt)
    implementation(libs.kotlinx.coroutines.core)
    runtimeOnly(libs.slf4j.nop)

    testImplementation(libs.junit.jupiter)
    testRuntimeOnly(libs.junit.platform.launcher)
}

application {
    mainClass.set("io.stackql.costgate.cli.MainKt")
    applicationName = "costgate"
}
