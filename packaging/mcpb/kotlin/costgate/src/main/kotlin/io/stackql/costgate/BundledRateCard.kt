package io.stackql.costgate

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.double
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * An offline pricing source backed by a bundled rate-card JSON resource
 * (`ratecards/<provider>.json`). Deterministic and credential-free, so the
 * costgate demo and its CI test gate a budget with no cloud account. The live
 * stackql source supersedes it when credentials are present.
 *
 * The rate card groups unit prices by resource [ResourceIntent.type], and
 * carries per-region multipliers. compute/database/loadbalancer prices are
 * per instance-month; storage is per GB-month (multiplied by the intent's gb).
 */
class BundledRateCard private constructor(
    private val provider: String,
    private val regionFactors: Map<String, Double>,
    private val families: Map<String, Map<String, Double>>,
) : PricingSource {

    override val name: String get() = "bundled rate card ($provider)"

    override suspend fun priceOf(provider: String, resource: ResourceIntent): UnitPrice? {
        if (!provider.equals(this.provider, ignoreCase = true)) return null
        val family = families[resource.type] ?: return null
        val base = family[resource.size] ?: return null
        val factor = regionFactors[resource.region] ?: 1.0

        return if (resource.type == "storage") {
            val gb = resource.gb ?: 1
            UnitPrice(
                monthlyUsdPerUnit = base * factor * gb,
                unit = "GB-month",
                derivation = "rate-card ${resource.type}/${resource.size} " +
                    "${"%.4f".format(base)} USD/GB-month x region(${resource.region}=$factor) x ${gb}GB",
            )
        } else {
            UnitPrice(
                monthlyUsdPerUnit = base * factor,
                unit = "instance-month",
                derivation = "rate-card ${resource.type}/${resource.size} " +
                    "${"%.2f".format(base)} USD/month x region(${resource.region}=$factor)",
            )
        }
    }

    companion object {
        private val json = Json { ignoreUnknownKeys = true }

        /** Load the bundled rate card for [provider], or throw if none is bundled. */
        fun forProvider(provider: String): BundledRateCard {
            val path = "/io/stackql/costgate/ratecards/${provider.lowercase()}.json"
            val stream = BundledRateCard::class.java.getResourceAsStream(path)
                ?: throw CostgateException("no bundled rate card for provider \"$provider\"")
            val doc = json.parseToJsonElement(stream.bufferedReader().readText()).jsonObject
            return fromJson(provider, doc)
        }

        internal fun fromJson(provider: String, doc: JsonObject): BundledRateCard {
            val regionFactors = (doc["regionFactors"]?.jsonObject ?: JsonObject(emptyMap()))
                .mapValues { it.value.jsonPrimitive.double }
            val reserved = setOf("provider", "note", "regionFactors")
            val families = doc.filterKeys { it !in reserved }
                .mapValues { (_, v) ->
                    v.jsonObject.mapValues { it.value.jsonPrimitive.double }
                }
            return BundledRateCard(provider, regionFactors, families)
        }
    }
}
