package io.stackql.mcp

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.Test

class PlatformTest {

    @Test
    fun detectsLinux() {
        assertEquals(Platform.LinuxX64, Platform.detect("linux", "amd64"))
        assertEquals(Platform.LinuxX64, Platform.detect("Linux", "x86_64"))
        assertEquals(Platform.LinuxArm64, Platform.detect("linux", "aarch64"))
    }

    @Test
    fun detectsWindows() {
        assertEquals(Platform.WindowsX64, Platform.detect("Windows 11", "amd64"))
    }

    @Test
    fun bothMacArchesMapToUniversal() {
        assertEquals(Platform.DarwinUniversal, Platform.detect("Mac OS X", "x86_64"))
        assertEquals(Platform.DarwinUniversal, Platform.detect("Mac OS X", "aarch64"))
    }

    @Test
    fun unknownPlatformThrows() {
        assertThrows(UnsupportedPlatformException::class.java) {
            Platform.detect("solaris", "sparc")
        }
        // 32-bit windows has no published binary.
        assertThrows(UnsupportedPlatformException::class.java) {
            Platform.detect("windows", "x86")
        }
    }

    @Test
    fun exeNameIsExeOnlyOnWindows() {
        assertEquals("stackql.exe", Platform.WindowsX64.exeName)
        assertEquals("stackql", Platform.LinuxX64.exeName)
        assertEquals("stackql", Platform.DarwinUniversal.exeName)
    }
}
