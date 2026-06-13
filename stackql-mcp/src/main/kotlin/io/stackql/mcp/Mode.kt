package io.stackql.mcp

/**
 * The stackql MCP server safety mode. It bounds which tools the server will
 * actually execute (the server lists all tools regardless of mode but gates
 * calls server-side). The library defaults to the most restrictive; anything
 * more permissive is an explicit caller decision.
 */
enum class Mode(val value: String) {
    /** SELECT and metadata tools only. The default. */
    ReadOnly("read_only"),

    /** Reads plus non-destructive mutations. */
    Safe("safe"),

    /** Safe plus deletes. */
    DeleteSafe("delete_safe"),

    /** Everything, including lifecycle provisioning. */
    FullAccess("full_access");

    companion object {
        fun fromValue(value: String): Mode =
            entries.firstOrNull { it.value == value }
                ?: throw StackqlMcpException("unknown mode \"$value\"")
    }
}
