package io.stackql.costgate.cli

import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.core.main
import com.github.ajalt.clikt.core.subcommands
import com.github.ajalt.clikt.parameters.options.default
import com.github.ajalt.clikt.parameters.options.flag
import com.github.ajalt.clikt.parameters.options.option
import com.github.ajalt.clikt.parameters.types.path
import io.stackql.costgate.Costgate
import io.stackql.costgate.Renderers
import kotlinx.coroutines.runBlocking
import java.nio.file.Files
import java.nio.file.Path
import kotlin.system.exitProcess

/**
 * `costgate check` - read a resource intent, price it, and exit non-zero when
 * the total exceeds the declared budget. The gate.
 */
class Check : CliktCommand(name = "check") {
    override fun help(context: com.github.ajalt.clikt.core.Context) =
        "Price a resource intent and gate on its budget (exit 1 when over)."

    private val intent: Path by option("--intent", "-i", help = "Path to costgate.yaml")
        .path(mustExist = true, canBeDir = false)
        .default(Path.of("costgate.yaml"))

    private val live: Boolean by option(
        "--live",
        help = "Price against live provider data via an embedded stackql server (needs credentials). Default: bundled rate card, offline.",
    ).flag(default = false)

    private val explain: Boolean by option(
        "--explain",
        help = "Show the SQL/rate-card rule behind each price and the top cost drivers.",
    ).flag(default = false)

    private val budgetOverride: String? by option(
        "--budget",
        help = "Override the budget in the intent file, e.g. 500/month.",
    )

    private val junitXml: Path? by option(
        "--junit-xml",
        help = "Also write a JUnit-style XML report here (for CI UIs).",
    ).path(canBeDir = false)

    override fun run() = runBlocking {
        val report = Costgate.run(intent, live = live).let { base ->
            if (budgetOverride == null) base
            else base.copy(budget = io.stackql.costgate.Budget.parse(budgetOverride!!))
        }

        echo(Renderers.console(report, explain = explain))

        junitXml?.let { path ->
            path.parent?.let { Files.createDirectories(it) }
            Files.writeString(path, Renderers.junitXml(report))
            System.err.println("costgate: wrote JUnit report to $path")
        }

        if (report.overBudget) exitProcess(1)
    }
}

/** The `costgate` root command. Subcommands do the work. */
class CostgateCli : CliktCommand(name = "costgate") {
    override fun help(context: com.github.ajalt.clikt.core.Context) =
        "Make cloud cost a build check. costgate prices a declared resource intent before it exists."

    override fun run() = Unit
}

fun main(args: Array<String>) = CostgateCli().subcommands(Check()).main(args)
