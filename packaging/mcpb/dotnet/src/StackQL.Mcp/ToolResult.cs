using ModelContextProtocol.Protocol;

namespace StackQL.Mcp;

/// <summary>
/// A flattened view over an MCP <see cref="CallToolResult"/>: the concatenated
/// text content plus the raw result for callers that need structured content or
/// the error flag.
/// </summary>
public sealed class ToolResult
{
    internal ToolResult(CallToolResult raw)
    {
        Raw = raw;
        Text = string.Join(
            "\n",
            raw.Content
                .OfType<TextContentBlock>()
                .Select(b => b.Text));
        IsError = raw.IsError ?? false;
    }

    /// <summary>Concatenated text blocks from the tool result.</summary>
    public string Text { get; }

    /// <summary>True if the server flagged the call as an error.</summary>
    public bool IsError { get; }

    /// <summary>The underlying MCP result for structured/advanced access.</summary>
    public CallToolResult Raw { get; }

    /// <inheritdoc />
    public override string ToString() => Text;
}
