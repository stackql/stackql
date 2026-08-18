package io.stackql.costgate.gradle

import org.gradle.api.file.RegularFileProperty
import org.gradle.api.provider.Property

/**
 * `costgate { }` configuration block. All properties have build-sensible
 * defaults so a plugin user can apply the plugin and run `gradle costgate`
 * with only a `costgate.yaml` in the project root.
 */
interface CostgateExtension {
    /** The resource intent file. Default: `costgate.yaml` in the project dir. */
    val intent: RegularFileProperty

    /**
     * Price against live provider data via an embedded stackql server (needs
     * credentials). Default false: the bundled rate card, offline.
     */
    val live: Property<Boolean>

    /** Override the budget declared in the intent file, e.g. "500/month". */
    val budget: Property<String>

    /** Show price derivations and cost drivers in the console report. */
    val explain: Property<Boolean>

    /**
     * Fail the build when over budget. Default true (the gate). Set false to
     * report cost without blocking - a soft check.
     */
    val failOnOverBudget: Property<Boolean>

    /** Where the JUnit-style XML report is written. Default: build/reports/costgate/costgate.xml. */
    val junitReport: RegularFileProperty
}
