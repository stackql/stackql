package mcp_server //nolint:revive // fine for now

// create an http client that can talk to the mcp server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

const (
	MCPClientTypeHTTP  = "http"
	MCPClientTypeSTDIO = "stdio"
)

type MCPClient interface {
	InspectTools() ([]map[string]any, error)
	CallToolText(toolName string, args map[string]any) (string, error)
}

func NewMCPClient(clientType string, baseURL string, clientCfgMap map[string]any, logger *logrus.Logger) (MCPClient, error) {
	switch clientType {
	case MCPClientTypeHTTP:
		return newHTTPMCPClient(baseURL, clientCfgMap, logger)
	case MCPClientTypeSTDIO:
		return newStdioMCPClient(logger)
	default:
		return nil, fmt.Errorf("unknown client type: %s", clientType)
	}
}

func getHTTPClient(clientCfgMap map[string]any) (*http.Client, error) {
	if clientCfgMap == nil || clientCfgMap["ca_file"] == nil {
		return http.DefaultClient, nil
	}
	caFile, ok := clientCfgMap["ca_file"].(string)
	if !ok {
		return nil, fmt.Errorf("ca_file must be a string")
	}
	serverName := ""
	if value, exists := clientCfgMap["server_name"]; exists {
		serverName, ok = value.(string)
		if !ok {
			return nil, fmt.Errorf("server_name must be a string")
		}
	}
	caBytes, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file %q: %w", caFile, err)
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("no valid certificates in CA file %q", caFile)
	}
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default HTTP transport must be *http.Transport")
	}
	transport := defaultTransport.Clone()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
	}
	return &http.Client{Transport: transport}, nil
}

func newHTTPMCPClient(baseURL string, clientCfgMap map[string]any, logger *logrus.Logger) (MCPClient, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}
	httpClient, httpClientErr := getHTTPClient(clientCfgMap)
	if httpClientErr != nil {
		return nil, fmt.Errorf("error creating HTTP client: %w", httpClientErr)
	}
	return &httpMCPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
		clientCfg:  clientCfgMap,
	}, nil
}

type httpMCPClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
	clientCfg  map[string]any
}

func (c *httpMCPClient) connect() (*mcp.ClientSession, error) {
	url := c.baseURL
	ctx := context.Background()

	// Create the URL for the server.
	c.logger.Infof("Connecting to MCP server at %s", url)

	// Create an MCP client.
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "stackql-client",
		Version: "1.0.0",
	}, nil)

	// Connect to the server.
	return client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url, HTTPClient: c.httpClient}, nil)
}

func (c *httpMCPClient) connectOrDie() *mcp.ClientSession {
	session, err := c.connect()
	if err != nil {
		c.logger.Fatalf("Failed to connect: %v", err)
	}
	return session
}

func (c *httpMCPClient) InspectTools() ([]map[string]any, error) {
	session := c.connectOrDie()
	defer session.Close()

	c.logger.Infof("Connected to server (session ID: %s)", session.ID())

	// First, list available tools.
	c.logger.Infof("Listing available tools...")
	toolsResult, err := session.ListTools(context.Background(), nil)
	if err != nil {
		c.logger.Fatalf("Failed to list tools: %v", err)
	}
	var rv []map[string]any
	for _, tool := range toolsResult.Tools {
		c.logger.Infof("  - %s: %s\n", tool.Name, tool.Description)
		toolInfo := map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
		}
		rv = append(rv, toolInfo)
	}

	c.logger.Infof("Client completed successfully")
	return rv, nil
}

func (c *httpMCPClient) callTool(toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	session := c.connectOrDie()
	defer session.Close()

	c.logger.Infof("Connected to server (session ID: %s)", session.ID())

	c.logger.Infof("Calling tool %s...", toolName)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		c.logger.Errorf("Failed to call tool %s: %v\n", toolName, err)
		return result, err
	}

	c.logger.Infof("Client completed successfully")
	return result, nil
}

// CallToolText returns the tool's output formatted for a scripting client.
// See formatToolResult for the contract.
func (c *httpMCPClient) CallToolText(toolName string, args map[string]any) (string, error) {
	toolCall, toolCallErr := c.callTool(toolName, args)
	if toolCallErr != nil {
		return "", toolCallErr
	}
	return formatToolResult(toolName, toolCall, c.prefersText())
}

// prefersText reports whether the client config requests the rendered text
// content blocks instead of the structured payload (`"prefer_text": true`).
// Useful for exercising / consuming the server's text renderings, eg the
// JSON render option of issue #669.
func (c *httpMCPClient) prefersText() bool {
	if c.clientCfg == nil {
		return false
	}
	preferText, isBool := c.clientCfg["prefer_text"].(bool)
	return isBool && preferText
}

// formatToolResult shapes a CallToolResult for a scripting client.
// Order of preference:
//  1. The concatenated TextContent blocks, when preferText is set.
//  2. StructuredContent re-marshalled as compact JSON (the typed DTO).
//  3. The concatenated TextContent blocks, if no structured payload is present.
//
// Tool-level errors (IsError == true) are returned as a Go error containing the
// text payload, so a CLI caller exits non-zero and the message ends up on
// stderr - matching the existing transport-error convention.
func formatToolResult(toolName string, toolCall *mcp.CallToolResult, preferText bool) (string, error) {
	if toolCall == nil {
		return "", fmt.Errorf("tool %s returned no result", toolName)
	}
	if toolCall.IsError {
		return "", fmt.Errorf("tool %s: %s", toolName, extractText(toolCall))
	}
	if preferText {
		return extractText(toolCall), nil
	}
	if toolCall.StructuredContent != nil {
		raw, err := json.Marshal(toolCall.StructuredContent)
		if err != nil {
			return "", fmt.Errorf("marshal structured content for %s: %w", toolName, err)
		}
		return string(raw), nil
	}
	return extractText(toolCall), nil
}

func extractText(toolCall *mcp.CallToolResult) string {
	var sb bytes.Buffer
	for i, content := range toolCall.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(textContent.Text)
		}
	}
	return sb.String()
}

type stdioMCPClient struct {
	logger *logrus.Logger
}

func newStdioMCPClient(logger *logrus.Logger) (MCPClient, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}
	return &stdioMCPClient{
		logger: logger,
	}, nil
}

func (c *stdioMCPClient) InspectTools() ([]map[string]any, error) {
	c.logger.Infof("stdio MCP client not implemented yet")
	return nil, nil
}

func (c *stdioMCPClient) CallToolText(toolName string, args map[string]any) (string, error) {
	c.logger.Infof("stdio MCP client not implemented yet")
	return "", nil
}
