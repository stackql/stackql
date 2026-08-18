package io.stackql.mcp

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import java.nio.file.Path
import java.nio.file.Paths

class LaunchArgsTest {

    private val approot: Path = Paths.get(if (isWindows()) "C:\\home\\u\\.stackql" else "/home/u/.stackql")

    private fun isWindows() = System.getProperty("os.name").lowercase().contains("windows")

    @Test
    fun canonicalArgsMatchTheContract() {
        val args = LaunchArgs.build(Mode.ReadOnly, approot)
        assertEquals("mcp", args[0])
        assertEquals("--mcp.server.type=stdio", args[1])
        assertEquals("--approot", args[2])
        assertEquals(approot.toAbsolutePath().toString(), args[3])
        assertEquals("--mcp.config", args[4])
        // args[5] is the json config, asserted below.
        assertEquals(6, args.size)
    }

    @Test
    fun mcpConfigHasModeAndAuditDisabled() {
        for (mode in Mode.entries) {
            val args = LaunchArgs.build(mode, approot)
            val config = Json.parseToJsonElement(args[5]).jsonObject
            val server = config["server"]!!.jsonObject
            assertEquals(mode.value, server["mode"]!!.jsonPrimitive.content)
            assertEquals("true", server["audit"]!!.jsonObject["disabled"]!!.jsonPrimitive.content)
        }
    }

    @Test
    fun authIsAppendedAsASingleFlag() {
        val auth = LaunchArgs.authFor("github", "null_auth")
        val args = LaunchArgs.build(Mode.ReadOnly, approot, auth)
        val last = args.last()
        assertTrue(last.startsWith("--auth="), last)
        val parsed = Json.parseToJsonElement(last.removePrefix("--auth=")).jsonObject
        assertEquals("null_auth", parsed["github"]!!.jsonObject["type"]!!.jsonPrimitive.content)
    }

    @Test
    fun defaultModeIsReadOnly() {
        val args = LaunchArgs.build(appRoot = approot)
        val config = Json.parseToJsonElement(args[5]).jsonObject
        assertEquals("read_only", config["server"]!!.jsonObject["mode"]!!.jsonPrimitive.content)
    }

    @Test
    fun relativeApprootIsRejected() {
        assertThrows(IllegalArgumentException::class.java) {
            LaunchArgs.build(appRoot = Paths.get("relative", ".stackql"))
        }
    }
}
