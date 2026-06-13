package io.stackql.costgate

import io.modelcontextprotocol.kotlin.sdk.types.TextContent
import io.stackql.mcp.StackqlMcp
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * A live pricing source: prices a resource by running a read_only SELECT
 * against the provider's pricing surface through an embedded stackql MCP
 * server. Used when credentials for the provider's pricing API are present.
 *
 * The agentic part of costgate is here: cost becomes a SQL query the build
 * runs, and `--explain` surfaces the exact SQL. When a query returns nothing
 * (no rate, no credentials, an unpriceable resource) this source returns null
 * so the engine falls back to the bundled rate card.
 *
 * Pricing query shapes are provider-specific; v1 ships the AWS shape and
 * leaves others to the bundled card.
 */
class StackqlPricingSource(private val server: StackqlMcp) : PricingSource {

    override val name: String get() = "stackql live pricing"

    private val json = Json { ignoreUnknownKeys = true }

    override suspend fun priceOf(provider: String, resource: ResourceIntent): UnitPrice? {
        val sql = pricingSql(provider, resource) ?: return null
        val rows = runSelect(sql) ?: return null
        val monthly = extractMonthlyUsd(provider, resource, rows) ?: return null
        val unit = if (resource.type == "storage") "GB-month" else "instance-month"
        return UnitPrice(monthlyUsdPerUnit = monthly, unit = unit, derivation = "SQL: $sql")
    }

    /** Build the provider-specific pricing SELECT, or null if unsupported. */
    internal fun pricingSql(provider: String, resource: ResourceIntent): String? {
        if (!provider.equals("aws", ignoreCase = true)) return null
        // AWS publishes on-demand list prices through the Pricing API, exposed
        // by stackql as aws.pricing.products. The pricePerUnit is hourly for
        // compute; the engine annualizes/monthlyizes downstream.
        return when (resource.type) {
            "compute" -> """
                SELECT pricePerUnit FROM aws.pricing.products
                WHERE region = '${resource.region}'
                  AND ServiceCode = 'AmazonEC2'
                  AND instanceType = '${resource.size}'
                  AND tenancy = 'Shared' AND operatingSystem = 'Linux'
                  AND capacitystatus = 'Used' AND preInstalledSw = 'NA'
                LIMIT 1
            """.trimIndent()
            "storage" -> """
                SELECT pricePerUnit FROM aws.pricing.products
                WHERE region = '${resource.region}'
                  AND ServiceCode = 'AmazonEC2'
                  AND productFamily = 'Storage'
                  AND volumeApiName = '${resource.size}'
                LIMIT 1
            """.trimIndent()
            else -> null
        }
    }

    private suspend fun runSelect(sql: String): JsonObject? {
        val result = runCatching {
            server.client.callTool("run_select_query", mapOf("sql" to sql))
        }.getOrNull() ?: return null
        if (result.isError == true) return null
        val text = result.content.filterIsInstance<TextContent>().joinToString("\n") { it.text }
        return runCatching { json.parseToJsonElement(text).jsonObject }.getOrNull()
    }

    /**
     * Pull the unit price out of a `{"rows":[...]}` result and convert to a
     * monthly figure (compute: hourly x 730; storage: per-GB-month x gb).
     */
    internal fun extractMonthlyUsd(
        provider: String,
        resource: ResourceIntent,
        result: JsonObject,
    ): Double? {
        val rows = result["rows"]?.jsonArray ?: return null
        val first = rows.firstOrNull()?.jsonObject ?: return null
        val raw = first["pricePerUnit"]?.jsonPrimitive ?: return null
        val unitPrice = raw.doubleOrNull
            ?: raw.content.toDoubleOrNull()
            ?: return null
        return when (resource.type) {
            "compute" -> unitPrice * HOURS_PER_MONTH
            "storage" -> unitPrice * (resource.gb ?: 1)
            else -> unitPrice
        }
    }

    companion object {
        /** AWS bills compute per hour; 730 h is the conventional month. */
        const val HOURS_PER_MONTH = 730.0
    }
}
