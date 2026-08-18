package io.stackql.mcp

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class PinsTest {

    @Test
    fun everyPlatformHasAWellFormedPin() {
        for (platform in Platform.entries) {
            val pin = Pins.pinFor(platform)
            assertEquals(64, pin.length, "pin length for ${platform.key}")
            assertTrue(pin.all { it.isDigit() || it in 'a'..'f' }, "lowercase hex for ${platform.key}")
            assertEquals(pin, pin.lowercase())
        }
    }

    @Test
    fun bundleNameMatchesPlatformKey() {
        assertEquals("stackql-mcp-linux-x64.mcpb", Platform.LinuxX64.bundleName)
        assertEquals("stackql-mcp-darwin-universal.mcpb", Platform.DarwinUniversal.bundleName)
    }

    @Test
    fun bundleUrlIsTheProxyFrontDoorForThePinnedVersion() {
        val v = Pins.STACKQL_VERSION
        assertTrue(v.matches(Regex("""\d+\.\d+\.\d+""")), "version $v")
        assertEquals("https://releases.stackql.io/stackql/$v", Pins.BASE_URL)
        assertEquals(
            "https://releases.stackql.io/stackql/$v/stackql-mcp-linux-x64.mcpb",
            Pins.bundleUrl(Platform.LinuxX64),
        )
        assertEquals("stackql-mcp-server-kotlin/$v", Pins.USER_AGENT)
    }
}
