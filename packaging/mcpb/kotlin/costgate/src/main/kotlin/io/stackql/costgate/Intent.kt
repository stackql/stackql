package io.stackql.costgate

import com.charleskorn.kaml.Yaml
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import java.nio.file.Files
import java.nio.file.Path

/**
 * The declared resource intent: what a deploy intends to create, before it
 * exists. v1 keeps this deliberately simple - a flat list of resources with a
 * type, size, region, and count. terraform-plan ingestion is a v2 item.
 *
 * Example `costgate.yaml`:
 * ```yaml
 * provider: aws
 * budget: 500/month
 * resources:
 *   - type: compute
 *     size: m5.large
 *     region: us-east-1
 *     count: 3
 *   - type: storage
 *     size: gp3
 *     region: us-east-1
 *     count: 1
 *     gb: 500
 * ```
 */
@Serializable
data class Intent(
    /** Provider the resources belong to (rate-card namespace), e.g. "aws". */
    val provider: String,
    /** Budget the total monthly cost is gated against, e.g. "500/month". */
    val budget: String,
    /** The resources this deploy intends to create. */
    val resources: List<ResourceIntent>,
) {
    companion object {
        private val yaml = Yaml.default

        fun parse(text: String): Intent = yaml.decodeFromString(serializer(), text)

        fun load(path: Path): Intent = parse(Files.readString(path))
    }
}

/** One declared resource line. */
@Serializable
data class ResourceIntent(
    /** Resource family the rate card prices, e.g. "compute", "storage". */
    val type: String,
    /** Size/SKU within the family, e.g. an instance type or a volume class. */
    val size: String,
    /** Region the resource runs in; rate cards vary cost by region. */
    val region: String,
    /** How many of this resource. Defaults to 1. */
    val count: Int = 1,
    /** Capacity in GB for storage-like resources (ignored otherwise). */
    val gb: Int? = null,
    /** Optional free-text label for the report. */
    @SerialName("name") val name: String? = null,
)
