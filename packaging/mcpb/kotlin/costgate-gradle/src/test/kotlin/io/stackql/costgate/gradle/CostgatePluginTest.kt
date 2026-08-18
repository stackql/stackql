package io.stackql.costgate.gradle

import org.gradle.testkit.runner.GradleRunner
import org.gradle.testkit.runner.TaskOutcome
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.io.File

/**
 * Functional tests: apply the plugin in a synthetic project and drive the
 * `costgate` task with TestKit. Offline (bundled rate card), so no network or
 * credentials - the same path CI uses.
 */
class CostgatePluginTest {

    private fun project(dir: File, intent: String, costgateBlock: String = "") {
        File(dir, "settings.gradle.kts").writeText("""rootProject.name = "demo"""")
        File(dir, "build.gradle.kts").writeText(
            """
            plugins { id("io.stackql.costgate") }
            costgate {
                explain.set(true)
                $costgateBlock
            }
            """.trimIndent(),
        )
        File(dir, "costgate.yaml").writeText(intent)
    }

    private val overBudget = """
        provider: aws
        budget: 100/month
        resources:
          - type: compute
            size: m5.large
            region: us-east-1
            count: 3
    """.trimIndent()

    private val withinBudget = overBudget.replace("budget: 100/month", "budget: 500/month")

    private fun runner(dir: File) = GradleRunner.create()
        .withProjectDir(dir)
        .withPluginClasspath()
        .withArguments("costgate", "--stacktrace")

    @Test
    fun gateFailsTheBuildWhenOverBudget(@TempDir dir: File) {
        project(dir, overBudget)
        val result = runner(dir).buildAndFail()
        assertEquals(TaskOutcome.FAILED, result.task(":costgate")?.outcome)
        assertTrue(result.output.contains("over budget"), result.output)
        assertTrue(File(dir, "build/reports/costgate/costgate.xml").exists())
    }

    @Test
    fun gatePassesWithinBudget(@TempDir dir: File) {
        project(dir, withinBudget)
        val result = runner(dir).build()
        assertEquals(TaskOutcome.SUCCESS, result.task(":costgate")?.outcome)
        assertTrue(result.output.contains("gate PASSES"), result.output)
    }

    @Test
    fun softModeReportsButDoesNotFail(@TempDir dir: File) {
        project(dir, overBudget, costgateBlock = "failOnOverBudget.set(false)")
        val result = runner(dir).build()
        assertEquals(TaskOutcome.SUCCESS, result.task(":costgate")?.outcome)
        assertTrue(result.output.contains("gate FAILS"), result.output)
    }
}
