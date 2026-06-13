package io.stackql.costgate

/** A priced line in the report: one resource intent and its monthly cost. */
data class LineItem(
    val resource: ResourceIntent,
    /** Monthly USD for this line (unit price x count). */
    val monthlyUsd: Double?,
    /** The unit price and its derivation, or null if nothing could price it. */
    val unitPrice: UnitPrice?,
    /** The pricing source that produced this line. */
    val source: String,
) {
    val priced: Boolean get() = monthlyUsd != null
    val label: String get() = resource.name ?: "${resource.type}/${resource.size} (${resource.region})"
}

/** The result of pricing an intent against a budget. */
data class CostReport(
    val provider: String,
    val budget: Budget,
    val lineItems: List<LineItem>,
) {
    /** Total monthly USD across all priced lines. */
    val totalMonthlyUsd: Double get() = lineItems.mapNotNull { it.monthlyUsd }.sum()

    /** Lines no source could price - reported, but they do not block the gate. */
    val unpriced: List<LineItem> get() = lineItems.filterNot { it.priced }

    /** True when the total exceeds the budget: the gate fails. */
    val overBudget: Boolean get() = totalMonthlyUsd > budget.monthlyUsd

    /** Budget headroom (negative when over). */
    val remainingUsd: Double get() = budget.monthlyUsd - totalMonthlyUsd

    /** Lines sorted by cost, biggest first - the cost drivers for --explain. */
    val costDrivers: List<LineItem>
        get() = lineItems.filter { it.priced }.sortedByDescending { it.monthlyUsd }
}
