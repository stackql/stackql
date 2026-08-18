package io.stackql.costgate

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.Test

class BudgetTest {
    @Test
    fun parsesCommonForms() {
        assertEquals(500.0, Budget.parse("500").monthlyUsd)
        assertEquals(500.0, Budget.parse("500/month").monthlyUsd)
        assertEquals(500.0, Budget.parse("500/mo").monthlyUsd)
        assertEquals(500.0, Budget.parse("$500/month").monthlyUsd)
        assertEquals(1500.5, Budget.parse("1500.50 per month").monthlyUsd)
    }

    @Test
    fun rejectsNonMonthlyPeriods() {
        assertThrows(CostgateException::class.java) { Budget.parse("500/year") }
        assertThrows(CostgateException::class.java) { Budget.parse("500/hour") }
    }

    @Test
    fun rejectsUnparseableAmount() {
        assertThrows(CostgateException::class.java) { Budget.parse("lots/month") }
    }
}
