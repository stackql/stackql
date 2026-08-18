package io.stackql.costgate.gradle

import org.gradle.api.Plugin
import org.gradle.api.Project

/**
 * Registers the `costgate` task and its `costgate { }` configuration block.
 * Applying the plugin and dropping a `costgate.yaml` in the project root is
 * enough to run `gradle costgate`.
 */
class CostgatePlugin : Plugin<Project> {
    override fun apply(project: Project) {
        val ext = project.extensions.create("costgate", CostgateExtension::class.java)

        ext.intent.convention(project.layout.projectDirectory.file("costgate.yaml"))
        ext.live.convention(false)
        ext.explain.convention(false)
        ext.failOnOverBudget.convention(true)
        ext.junitReport.convention(
            project.layout.buildDirectory.file("reports/costgate/costgate.xml"),
        )

        project.tasks.register("costgate", CostgateTask::class.java) { task ->
            task.intent.set(ext.intent)
            task.live.set(ext.live)
            task.budget.set(ext.budget)
            task.explain.set(ext.explain)
            task.failOnOverBudget.set(ext.failOnOverBudget)
            task.junitReport.set(ext.junitReport)
        }
    }
}
