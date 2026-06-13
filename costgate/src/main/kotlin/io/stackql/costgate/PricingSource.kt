package io.stackql.costgate

/**
 * A source of unit prices for resources. The engine is agnostic to where the
 * rate comes from: a bundled snapshot (offline, no credentials) or a live
 * stackql query against a provider's pricing surface.
 */
interface PricingSource {
    /** Human-readable name of this source, shown in reports. */
    val name: String

    /**
     * The monthly unit price for one [resource], plus the SQL or rule that
     * produced it (shown by `--explain`). Returns null if this source has no
     * rate for the resource, so the engine can fall through to another source.
     */
    suspend fun priceOf(provider: String, resource: ResourceIntent): UnitPrice?
}

/**
 * A resolved unit price: monthly USD for a single unit of the resource, the
 * basis of the calculation, and the query/rule that produced it.
 */
data class UnitPrice(
    /** Monthly USD for one unit (one instance, or one GB-month for storage). */
    val monthlyUsdPerUnit: Double,
    /** What the unit is, e.g. "instance-month" or "GB-month". */
    val unit: String,
    /** The SQL or rate-card rule that produced this price, for --explain. */
    val derivation: String,
)
