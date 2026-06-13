package io.stackql.costgate

/**
 * Prices an [Intent] against a budget by asking each [PricingSource] in order
 * for a unit price per resource, first hit wins. The conventional ordering is
 * live stackql first (authoritative when credentials are present), bundled
 * rate card second (the offline fallback), so a costgate run is deterministic
 * with no credentials and exact with them.
 */
class CostEngine(private val sources: List<PricingSource>) {

    init {
        require(sources.isNotEmpty()) { "costgate: at least one pricing source is required" }
    }

    suspend fun price(intent: Intent): CostReport {
        val budget = Budget.parse(intent.budget)
        val lines = intent.resources.map { resource ->
            priceResource(intent.provider, resource)
        }
        return CostReport(intent.provider, budget, lines)
    }

    private suspend fun priceResource(provider: String, resource: ResourceIntent): LineItem {
        for (source in sources) {
            val unit = source.priceOf(provider, resource) ?: continue
            return LineItem(
                resource = resource,
                monthlyUsd = unit.monthlyUsdPerUnit * resource.count,
                unitPrice = unit,
                source = source.name,
            )
        }
        return LineItem(resource = resource, monthlyUsd = null, unitPrice = null, source = "none")
    }

    companion object {
        /** The offline engine: bundled rate card only. No credentials, deterministic. */
        fun offline(provider: String): CostEngine =
            CostEngine(listOf(BundledRateCard.forProvider(provider)))
    }
}
