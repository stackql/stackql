package io.stackql.costgate

import io.stackql.mcp.Mode
import io.stackql.mcp.StackqlMcp
import java.nio.file.Path

/**
 * The costgate facade shared by the CLI and the Gradle plugin. It loads an
 * intent, prices it (live stackql when [live] is set, always with the bundled
 * rate card as the fallback), and returns a [CostReport].
 */
object Costgate {

    /**
     * Price the intent at [intentPath]. When [live] is true, an embedded
     * read_only stackql server is started and its live pricing source is
     * tried before the bundled rate card; the server is shut down before
     * returning. When false, only the offline rate card is used - no network,
     * no credentials.
     */
    suspend fun run(intentPath: Path, live: Boolean = false): CostReport {
        val intent = Intent.load(intentPath)
        return if (live) {
            val server = StackqlMcp.builder().mode(Mode.ReadOnly).start()
            server.use {
                val engine = CostEngine(
                    listOf(
                        StackqlPricingSource(server),
                        BundledRateCard.forProvider(intent.provider),
                    ),
                )
                engine.price(intent)
            }
        } else {
            CostEngine.offline(intent.provider).price(intent)
        }
    }
}
