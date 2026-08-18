package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	stackqlmcp "github.com/stackql/stackql-mcp-go/embed"
)

const maxToolResultBytes = 64 * 1024

// agentSession is a Claude agent loop wired to the tools of an embedded
// stackql MCP server.
type agentSession struct {
	llm    anthropic.Client
	server *stackqlmcp.Client
	model  anthropic.Model
	tools  []anthropic.ToolUnionParam
}

// newAgentSession lists the server's MCP tools and converts them to
// Anthropic tool definitions.
func newAgentSession(ctx context.Context, server *stackqlmcp.Client, model string) (*agentSession, error) {
	listed, err := server.Session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("listing MCP tools: %w", err)
	}
	var tools []anthropic.ToolUnionParam
	for _, t := range listed.Tools {
		schemaJSON, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("marshalling schema for %s: %w", t.Name, err)
		}
		var schema struct {
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		}
		if err := json.Unmarshal(schemaJSON, &schema); err != nil {
			return nil, fmt.Errorf("converting schema for %s: %w", t.Name, err)
		}
		tools = append(tools, anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: schema.Properties,
				Required:   schema.Required,
			},
		}})
	}
	return &agentSession{
		llm:    anthropic.NewClient(), // reads ANTHROPIC_API_KEY
		server: server,
		model:  anthropic.Model(model),
		tools:  tools,
	}, nil
}

// run executes the agent loop until the model stops calling tools, and
// returns the model's final text. Every tool call is echoed to stdout; SQL
// statements are printed verbatim so each step is inspectable.
func (a *agentSession) run(ctx context.Context, system, user string, maxTurns int) (string, error) {
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
	}
	for turn := 0; turn < maxTurns; turn++ {
		msg, err := a.llm.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: 8192,
			System:    []anthropic.TextBlockParam{{Text: system}},
			Messages:  messages,
			Tools:     a.tools,
		})
		if err != nil {
			return "", fmt.Errorf("anthropic API: %w", err)
		}
		messages = append(messages, msg.ToParam())

		var (
			results   []anthropic.ContentBlockParamUnion
			finalText strings.Builder
		)
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				finalText.WriteString(block.Text)
			case "tool_use":
				results = append(results, a.callTool(ctx, block.ID, block.Name, block.Input))
			}
		}
		if len(results) == 0 {
			return finalText.String(), nil
		}
		if text := strings.TrimSpace(finalText.String()); text != "" {
			fmt.Printf("\n%s\n", text)
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}
	return "", fmt.Errorf("agent did not finish within %d turns", maxTurns)
}

// callTool relays one tool call to the embedded server and echoes what ran.
func (a *agentSession) callTool(ctx context.Context, id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	var args map[string]any
	if err := json.Unmarshal(input, &args); err != nil {
		return anthropic.NewToolResultBlock(id, "invalid tool input: "+err.Error(), true)
	}
	if sql, ok := args["sql"].(string); ok {
		fmt.Printf("\n  [%s]\n  %s\n", name, strings.ReplaceAll(strings.TrimSpace(sql), "\n", "\n  "))
	} else {
		compact, _ := json.Marshal(args)
		fmt.Printf("\n  [%s] %s\n", name, compact)
	}
	res, err := a.server.Session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  tool transport error: %v\n", err)
		return anthropic.NewToolResultBlock(id, "tool call failed: "+err.Error(), true)
	}
	text := toolResultText(res)
	if len(text) > maxToolResultBytes {
		text = text[:maxToolResultBytes] + "\n[truncated]"
	}
	if res.IsError {
		fmt.Printf("  -> error: %s\n", firstLine(text))
	} else {
		fmt.Printf("  -> ok (%d bytes)\n", len(text))
	}
	return anthropic.NewToolResultBlock(id, text, res.IsError)
}

func toolResultText(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
