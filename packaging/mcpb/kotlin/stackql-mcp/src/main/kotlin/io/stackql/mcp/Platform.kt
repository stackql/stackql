package io.stackql.mcp

/**
 * Packaging platform keys. These match the stackql-mcpb-packaging release
 * asset names and the shared binary cache layout used by the npm and pypi
 * wrappers, so every StackQL distribution on a machine agrees on where a
 * given platform's binary lives.
 */
enum class Platform(val key: String) {
    LinuxX64("linux-x64"),
    LinuxArm64("linux-arm64"),
    WindowsX64("windows-x64"),
    DarwinUniversal("darwin-universal");

    /** The bundle asset name on the stackql release for this platform. */
    val bundleName: String get() = "stackql-mcp-$key.mcpb"

    /** Name the extracted server binary takes on this platform. */
    val exeName: String get() = if (this == WindowsX64) "stackql.exe" else "stackql"

    companion object {
        /**
         * Resolve the platform for the running JVM from `os.name`/`os.arch`.
         * Both macOS architectures map to the universal binary.
         */
        fun current(): Platform =
            detect(System.getProperty("os.name"), System.getProperty("os.arch"))

        /** Pure mapping, broken out so it is unit-testable without the JVM env. */
        fun detect(osNameRaw: String, osArch: String): Platform {
            val osName = osNameRaw.lowercase()
            val arch = normalizeArch(osArch.lowercase())
            return when {
                osName.contains("linux") && arch == "x64" -> LinuxX64
                osName.contains("linux") && arch == "arm64" -> LinuxArm64
                (osName.contains("windows")) && arch == "x64" -> WindowsX64
                (osName.contains("mac") || osName.contains("darwin")) &&
                    (arch == "x64" || arch == "arm64") -> DarwinUniversal
                else -> throw UnsupportedPlatformException(osName, osArch)
            }
        }

        private fun normalizeArch(osArch: String): String = when (osArch) {
            "amd64", "x86_64", "x64" -> "x64"
            "aarch64", "arm64" -> "arm64"
            else -> osArch
        }
    }
}

/** Thrown when no published stackql binary exists for the running platform. */
class UnsupportedPlatformException(osName: String, osArch: String) :
    StackqlMcpException("no published stackql binary for $osName/$osArch")
