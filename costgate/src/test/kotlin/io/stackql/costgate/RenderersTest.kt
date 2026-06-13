package io.stackql.costgate

import kotlinx.coroutines.runBlocking
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class RenderersTest {

    private fun report(budget: String) = runBlocking {
        CostEngine.offline("aws").price(
            Intent.parse(
                """
                provider: aws
                budget: $budget
                resources:
                  - name: api servers
                    type: compute
                    size: m5.large
                    region: us-east-1
                    count: 3
                """.trimIndent(),
            ),
        )
    }

    @Test
    fun consoleShowsGatePassWithinBudget() {
        val text = Renderers.console(report("500/month"), explain = false)
        assertTrue(text.contains("gate PASSES"), text)
        assertTrue(text.contains("$210.24"), text)
    }

    @Test
    fun consoleShowsGateFailOverBudget() {
        val text = Renderers.console(report("100/month"), explain = false)
        assertTrue(text.contains("gate FAILS"), text)
        assertTrue(text.contains("OVER BUDGET"), text)
    }

    @Test
    fun explainAddsDerivationAndDrivers() {
        val plain = Renderers.console(report("500/month"), explain = false)
        val explained = Renderers.console(report("500/month"), explain = true)
        assertFalse(plain.contains("top cost drivers"))
        assertTrue(explained.contains("top cost drivers"), explained)
        assertTrue(explained.contains("rate-card compute/m5.large"), explained)
    }

    @Test
    fun junitXmlHasFailureWhenOverBudget() {
        val xml = Renderers.junitXml(report("100/month"))
        assertTrue(xml.contains("""failures="1""""), xml)
        assertTrue(xml.contains("<failure"), xml)
        assertTrue(xml.contains("over budget"), xml)
    }

    @Test
    fun junitXmlHasNoFailureWithinBudget() {
        val xml = Renderers.junitXml(report("500/month"))
        assertTrue(xml.contains("""failures="0""""), xml)
        assertFalse(xml.contains("<failure"))
    }
}
