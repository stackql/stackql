package io.stackql.costgate

import kotlinx.coroutines.runBlocking
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class CostEngineTest {

    private val intentYaml = """
        provider: aws
        budget: 500/month
        resources:
          - name: api servers
            type: compute
            size: m5.large
            region: us-east-1
            count: 3
          - name: data volumes
            type: storage
            size: gp3
            region: us-east-1
            count: 1
            gb: 500
    """.trimIndent()

    @Test
    fun pricesIntentWithBundledRateCard() = runBlocking {
        val report = CostEngine.offline("aws").price(Intent.parse(intentYaml))
        // 3 x m5.large @ 70.08 = 210.24 ; 500 GB gp3 @ 0.08 = 40.00
        assertEquals(210.24, report.lineItems[0].monthlyUsd!!, 0.001)
        assertEquals(40.0, report.lineItems[1].monthlyUsd!!, 0.001)
        assertEquals(250.24, report.totalMonthlyUsd, 0.001)
        assertFalse(report.overBudget)
        assertEquals(249.76, report.remainingUsd, 0.001)
    }

    @Test
    fun gateFailsWhenOverBudget() = runBlocking {
        val tight = intentYaml.replace("budget: 500/month", "budget: 100/month")
        val report = CostEngine.offline("aws").price(Intent.parse(tight))
        assertTrue(report.overBudget)
        assertTrue(report.remainingUsd < 0)
    }

    @Test
    fun regionFactorRaisesCost() = runBlocking {
        val syd = intentYaml.replace("region: us-east-1", "region: ap-southeast-2")
        val report = CostEngine.offline("aws").price(Intent.parse(syd))
        // 70.08 x 1.15 region factor x 3 = 241.776
        assertEquals(241.776, report.lineItems[0].monthlyUsd!!, 0.01)
    }

    @Test
    fun unknownResourceIsReportedButNotPriced() = runBlocking {
        val withUnknown = """
            provider: aws
            budget: 500/month
            resources:
              - name: api servers
                type: compute
                size: m5.large
                region: us-east-1
                count: 3
              - name: data volumes
                type: storage
                size: gp3
                region: us-east-1
                count: 1
                gb: 500
              - name: mystery box
                type: compute
                size: zz.nonexistent
                region: us-east-1
        """.trimIndent()
        val report = CostEngine.offline("aws").price(Intent.parse(withUnknown))
        assertEquals(1, report.unpriced.size)
        assertEquals("mystery box", report.unpriced.first().label)
        // The unpriced line does not change the total or the gate.
        assertEquals(250.24, report.totalMonthlyUsd, 0.001)
    }

    @Test
    fun costDriversAreSortedBiggestFirst() = runBlocking {
        val report = CostEngine.offline("aws").price(Intent.parse(intentYaml))
        val drivers = report.costDrivers
        assertEquals("api servers", drivers.first().label)
    }
}
