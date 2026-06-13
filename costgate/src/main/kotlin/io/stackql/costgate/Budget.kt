package io.stackql.costgate

/** A monthly spend ceiling, parsed from strings like "500/month" or "500". */
data class Budget(val monthlyUsd: Double) {
    companion object {
        /**
         * Parse a budget string. Accepts a bare number ("500"), an explicit
         * monthly form ("500/month", "500/mo", "500 per month"), and an
         * optional leading currency symbol ("$500/month"). Only monthly
         * budgets are supported in v1; any other period is an error.
         */
        fun parse(raw: String): Budget {
            val s = raw.trim().removePrefix("$").trim()
            val parts = s.split("/", " per ", limit = 2)
            val amount = parts[0].trim().toDoubleOrNull()
                ?: throw CostgateException("unparseable budget amount in \"$raw\"")
            if (parts.size == 2) {
                val period = parts[1].trim().lowercase()
                if (period !in setOf("month", "mo", "monthly")) {
                    throw CostgateException("only monthly budgets are supported, got period \"$period\"")
                }
            }
            return Budget(amount)
        }
    }
}

/** Base type for costgate errors. */
open class CostgateException(message: String, cause: Throwable? = null) :
    Exception("costgate: $message", cause)
