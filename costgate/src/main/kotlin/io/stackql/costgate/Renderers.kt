package io.stackql.costgate

/** Renders a [CostReport] as a console report and as JUnit-style XML. */
object Renderers {

    private fun usd(v: Double): String = "$" + "%,.2f".format(v)

    /**
     * The console report. With [explain], each priced line is followed by the
     * SQL or rate-card rule that produced it, and the top cost drivers and
     * cheaper-region hints are listed.
     */
    fun console(report: CostReport, explain: Boolean): String = buildString {
        appendLine("costgate: ${report.provider} resource intent")
        appendLine("budget:   ${usd(report.budget.monthlyUsd)}/month")
        appendLine()
        appendLine("resources:")
        for (line in report.lineItems) {
            val cost = line.monthlyUsd?.let { usd(it) } ?: "unpriced"
            val countSuffix = if (line.resource.count != 1) " x${line.resource.count}" else ""
            appendLine("  - ${line.label}$countSuffix: $cost/month  [${line.source}]")
            if (explain && line.unitPrice != null) {
                appendLine("      ${line.unitPrice.derivation}")
            }
        }
        appendLine()
        appendLine("total:    ${usd(report.totalMonthlyUsd)}/month")
        appendLine("budget:   ${usd(report.budget.monthlyUsd)}/month")
        val verdict = if (report.overBudget) {
            "OVER BUDGET by ${usd(-report.remainingUsd)}/month -> gate FAILS"
        } else {
            "within budget (${usd(report.remainingUsd)}/month headroom) -> gate PASSES"
        }
        appendLine(verdict)

        if (report.unpriced.isNotEmpty()) {
            appendLine()
            appendLine("not priced (excluded from the total, review manually):")
            for (line in report.unpriced) appendLine("  - ${line.label}")
        }

        if (explain) {
            appendLine()
            appendLine("top cost drivers:")
            for (line in report.costDrivers.take(3)) {
                appendLine("  - ${line.label}: ${usd(line.monthlyUsd!!)}/month")
            }
        }
    }

    /**
     * A JUnit-style XML report: one testcase per resource plus a "budget gate"
     * case that fails (with a <failure>) when over budget, so CI UIs that
     * render JUnit XML surface the gate result inline.
     */
    fun junitXml(report: CostReport): String = buildString {
        val cases = report.lineItems.size + 1
        val failures = if (report.overBudget) 1 else 0
        appendLine("""<?xml version="1.0" encoding="UTF-8"?>""")
        append("""<testsuite name="costgate" tests="$cases" failures="$failures" """)
        appendLine("""errors="0" skipped="0">""")
        for (line in report.lineItems) {
            val cost = line.monthlyUsd?.let { usd(it) } ?: "unpriced"
            appendLine(
                """  <testcase classname="costgate.${report.provider}" """ +
                    """name="${xml(line.label)} = $cost/month" />""",
            )
        }
        val gateName = "budget gate: ${usd(report.totalMonthlyUsd)} vs ${usd(report.budget.monthlyUsd)}/month"
        if (report.overBudget) {
            appendLine("""  <testcase classname="costgate" name="${xml(gateName)}">""")
            appendLine(
                """    <failure message="over budget by ${usd(-report.remainingUsd)}/month">""" +
                    "total monthly cost ${usd(report.totalMonthlyUsd)} exceeds budget " +
                    "${usd(report.budget.monthlyUsd)}</failure>",
            )
            appendLine("""  </testcase>""")
        } else {
            appendLine("""  <testcase classname="costgate" name="${xml(gateName)}" />""")
        }
        appendLine("""</testsuite>""")
    }

    private fun xml(s: String): String = s
        .replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
        .replace("\"", "&quot;")
}
