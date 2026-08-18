// Root build: shared configuration applied to every Kotlin/JVM subproject.
// JDK 17 baseline (CLAUDE.md), JUnit5 for tests, warnings kept visible.

plugins {
    alias(libs.plugins.kotlin.jvm) apply false
    alias(libs.plugins.kotlin.serialization) apply false
}

allprojects {
    group = "io.stackql"
    version = property("stackqlMcpVersion") as String
}

subprojects {
    plugins.withId("org.jetbrains.kotlin.jvm") {
        extensions.configure<org.jetbrains.kotlin.gradle.dsl.KotlinJvmProjectExtension> {
            jvmToolchain(17)
        }

        tasks.withType<Test>().configureEach {
            useJUnitPlatform()
            // Integration/conformance tests reach the network and spawn the
            // server; gate them behind -PrunIntegration to keep the default
            // build hermetic (mirrors `go test -short`).
            if (project.findProperty("runIntegration") != "true") {
                exclude("**/*ConformanceTest.class", "**/*IntegrationTest.class")
            }
            // The conformance test caches the downloaded server binary here so a
            // per-test @TempDir teardown never races the OS file lock on the
            // extracted executable (a Windows flake).
            systemProperty("io.stackql.test.buildDir", layout.buildDirectory.get().asFile.absolutePath)
            testLogging {
                events("passed", "skipped", "failed")
                showStandardStreams = true
            }
        }
    }
}
