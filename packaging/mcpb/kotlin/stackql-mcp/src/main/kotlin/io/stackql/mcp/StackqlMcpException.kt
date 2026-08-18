package io.stackql.mcp

/** Base type for every error this library raises. */
open class StackqlMcpException(message: String, cause: Throwable? = null) :
    Exception("stackql mcp: $message", cause)
