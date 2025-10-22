package mcp_server //nolint:revive // fine for now

// create an http client that can talk to the mcp server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
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

//nolint:nestif,gocognit,gocyclo,cyclop,funlen // complex but acceptable for now
func getHTTPClient(logger *logrus.Logger, clientCfgMap map[string]any) (*http.Client, error) {
	if clientCfgMap != nil && clientCfgMap["ca_file"] != nil {
		logger.Infof("Configuring HTTP client with custom CA certificate")
		caFile, isString := clientCfgMap["ca_file"].(string)
		if !isString {
			return nil, fmt.Errorf("ca_file must be a string")
		}
		caBytes, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file '%s': %w", caFile, err)
		}
		logger.Infof("Read CA file '%s' (%d bytes)", caFile, len(caBytes))

		// Start from system pool when possible
		var caCertPool *x509.CertPool
		if sysPool, sysErr := x509.SystemCertPool(); sysErr == nil && sysPool != nil {
			caCertPool = sysPool
			logger.Debug("Using system cert pool as base")
		} else {
			caCertPool = x509.NewCertPool()
			logger.Debug("System cert pool unavailable; using new pool")
		}

		// Capture the first certificate (candidate leaf) for promote_leaf_to_ca.
		var firstCertRaw []byte
		{
			tmp := caBytes
			for {
				var blk *pem.Block
				blk, tmp = pem.Decode(tmp)
				if blk == nil {
					break
				}
				if blk.Type == "CERTIFICATE" {
					firstCertRaw = blk.Bytes
					break
				}
			}
		}

		if ok := caCertPool.AppendCertsFromPEM(caBytes); !ok {
			// Fallback: manual decode to provide diagnostics
			logger.Warn("AppendCertsFromPEM returned false; attempting manual PEM decode for diagnostics")
			blockCount := 0
			validCerts := 0
			rest := caBytes
			for {
				var b *pem.Block
				b, rest = pem.Decode(rest)
				if b == nil {
					break
				}
				blockCount++
				if b.Type == "CERTIFICATE" {
					if _, perr := x509.ParseCertificate(b.Bytes); perr == nil {
						validCerts++
					} else {
						logger.Errorf("Failed to parse certificate PEM block %d: %v", blockCount, perr)
					}
				} else {
					logger.Debugf("Ignoring non-certificate PEM block type=%s", b.Type)
				}
			}
			return nil, fmt.Errorf("failed to append CA certificate '%s' into trust store: no valid CERTIFICATE PEM blocks found (blocks=%d, valid=%d)", caFile, blockCount, validCerts)
		} else {
			logger.Infof("Successfully appended custom CA(s) from '%s'", caFile)
			// Added: inspect for CA certificates
			rest := caBytes
			certBlockIdx := 0
			caCount := 0
			for {
				var b *pem.Block
				b, rest = pem.Decode(rest)
				if b == nil {
					break
				}
				if b.Type != "CERTIFICATE" {
					continue
				}
				certBlockIdx++
				parsed, perr := x509.ParseCertificate(b.Bytes)
				if perr != nil {
					logger.Debugf("Skipping unparsable certificate block %d: %v", certBlockIdx, perr)
					continue
				}
				if parsed.IsCA {
					caCount++
				}
			}
			if caCount == 0 {
				logger.Warnf("No CA certificates (IsCA=true) found in '%s'. If this file contains only the server leaf certificate it cannot establish standard trust. Supply the issuing CA (or chain) or enable 'promote_leaf_to_ca'.", caFile)
			} else {
				logger.Debugf("Detected %d CA certificate(s) in '%s'", caCount, caFile)
			}
		}

		// Read optional flags
		promoteLeaf := false
		if v, ok := clientCfgMap["promote_leaf_to_ca"]; ok {
			b, okb := v.(bool)
			if !okb {
				return nil, fmt.Errorf("promote_leaf_to_ca must be a boolean")
			}
			promoteLeaf = b
		}

		insecureSkipVerify := false
		if v, ok := clientCfgMap["insecure_skip_verify"]; ok {
			suppliedSkipVerify, isBool := v.(bool)
			if !isBool {
				return nil, fmt.Errorf("insecure_skip_verify must be a boolean")
			}
			insecureSkipVerify = suppliedSkipVerify
		}

		var serverName string
		if v, ok := clientCfgMap["server_name"]; ok {
			if s, ok2 := v.(string); ok2 {
				serverName = s
			} else {
				return nil, fmt.Errorf("server_name must be a string")
			}
		}

		// If server_name not supplied and base URL host differs from cert common name/SAN (common in IP usage),
		// user should supply server_name explicitly; we just log hint.
		if serverName == "" {
			if rawURL, ok := clientCfgMap["base_url"].(string); ok {
				if parsed, perr := url.Parse(rawURL); perr == nil && parsed.Hostname() != "" {
					// SNI will default to this hostname; log for clarity.
					logger.Debugf("Using implicit SNI server name '%s'", parsed.Hostname())
				}
			}
		} else {
			logger.Infof("Using explicit TLS server_name (SNI): %s", serverName)
		}

		//nolint:gosec // testing client only
		tlsConfig := &tls.Config{
			RootCAs:            caCertPool,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: insecureSkipVerify, // may be overridden below if promoting leaf
			ServerName:         serverName,
		}

		// If no CA certs and user wants to promote the leaf, install custom verifier.
		if promoteLeaf {
			if firstCertRaw == nil {
				return nil, fmt.Errorf("promote_leaf_to_ca enabled but no certificate PEM blocks found in '%s'", caFile)
			}
			// Re-parse to log fingerprint
			if leafCert, perr := x509.ParseCertificate(firstCertRaw); perr == nil {
				fp := sha256.Sum256(leafCert.Raw)
				logger.Warnf("Promoting leaf certificate (CN=%s, SHA256=%X) to trust anchor (non-CA). NOT recommended for production.", leafCert.Subject.CommonName, fp[:8])
			} else {
				logger.Warnf("Promoting leaf certificate (parse error for fingerprint: %v)", perr)
			}
			tlsConfig.InsecureSkipVerify = true // we will verify manually
			expected := make([]byte, len(firstCertRaw))
			copy(expected, firstCertRaw)

			tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
				if len(rawCerts) == 0 {
					return fmt.Errorf("no server certificates presented")
				}
				if !bytes.Equal(rawCerts[0], expected) {
					return fmt.Errorf("server leaf certificate mismatch with promoted leaf")
				}
				// Optionally parse for additional sanity
				if cert, certParseErr := x509.ParseCertificate(rawCerts[0]); certParseErr == nil {
					if serverName != "" && serverName != cert.Subject.CommonName {
						// Do hostname verification if a serverName was forced.
						if verr := cert.VerifyHostname(serverName); verr != nil {
							return fmt.Errorf("hostname verification failed for promoted leaf: %w", verr)
						}
					}
				}
				return nil
			}
		}

		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
		}

		// Optionally apply TLS config globally so libraries using http.DefaultClient inherit it.
		if v, ok := clientCfgMap["apply_tls_globally"]; ok {
			if b, okb := v.(bool); !okb {
				return nil, fmt.Errorf("apply_tls_globally must be a boolean")
			} else if b {
				if defTr, okd := http.DefaultTransport.(*http.Transport); okd {
					// Shallow clone to avoid races; copy keeps other fields (proxy, dialer, etc.)
					cloned := defTr.Clone()
					cloned.TLSClientConfig = tlsConfig
					http.DefaultTransport = cloned
					logger.Warn("Applied custom TLS config globally (http.DefaultTransport). This affects all outbound HTTP requests in this process.")
				} else {
					logger.Warn("apply_tls_globally requested but http.DefaultTransport is not *http.Transport; skipped")
				}
			}
		}

		return &http.Client{Transport: tr}, nil
	}
	return http.DefaultClient, nil
}

func newHTTPMCPClient(baseURL string, clientCfgMap map[string]any, logger *logrus.Logger) (MCPClient, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}
	httpClient, httpClientErr := getHTTPClient(logger, clientCfgMap)
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
	return client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url}, nil)
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

func (c *httpMCPClient) CallToolText(toolName string, args map[string]any) (string, error) {
	toolCall, toolCallErr := c.callTool(toolName, args)
	if toolCallErr != nil {
		return "", toolCallErr
	}
	var result string
	for _, content := range toolCall.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			result += textContent.Text + "\n"
		}
	}
	return result, nil
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
