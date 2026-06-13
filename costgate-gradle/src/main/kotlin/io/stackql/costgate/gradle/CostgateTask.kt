package io.stackql.costgate.gradle

import io.stackql.costgate.Budget
import io.stackql.costgate.Costgate
import io.stackql.costgate.Renderers
import kotlinx.coroutines.runBlocking
import org.gradle.api.DefaultTask
import org.gradle.api.GradleException
import org.gradle.api.file.RegularFileProperty
import org.gradle.api.provider.Property
import org.gradle.api.tasks.Input
import org.gradle.api.tasks.InputFile
import org.gradle.api.tasks.Optional
import org.gradle.api.tasks.OutputFile
import org.gradle.api.tasks.TaskAction
import java.nio.file.Files

/**
 * The `costgate` task: price the intent and, by default, fail the build when
 * the total exceeds the budget. The cost gate, alongside test and lint.
 */
abstract class CostgateTask : DefaultTask() {

    @get:InputFile
    abstract val intent: RegularFileProperty

    @get:Input
    abstract val live: Property<Boolean>

    @get:Input
    @get:Optional
    abstract val budget: Property<String>

    @get:Input
    abstract val explain: Property<Boolean>

    @get:Input
    abstract val failOnOverBudget: Property<Boolean>

    @get:OutputFile
    abstract val junitReport: RegularFileProperty

    init {
        group = "verification"
        description = "Price the declared cloud resource intent and gate the build on its budget."
    }

    @TaskAction
    fun check() = runBlocking {
        val intentPath = intent.get().asFile.toPath()
        val base = Costgate.run(intentPath, live = live.get())
        val report = if (budget.isPresent) base.copy(budget = Budget.parse(budget.get())) else base

        logger.lifecycle(Renderers.console(report, explain = explain.getOrElse(false)))

        val xmlPath = junitReport.get().asFile.toPath()
        Files.createDirectories(xmlPath.parent)
        Files.writeString(xmlPath, Renderers.junitXml(report))
        logger.lifecycle("costgate: JUnit report at $xmlPath")

        if (report.overBudget && failOnOverBudget.getOrElse(true)) {
            throw GradleException(
                "costgate: over budget by \$${"%,.2f".format(-report.remainingUsd)}/month " +
                    "(\$${"%,.2f".format(report.totalMonthlyUsd)} vs \$${"%,.2f".format(report.budget.monthlyUsd)}). " +
                    "Build gated.",
            )
        }
    }
}
