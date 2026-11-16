package mcp_server //nolint:testpackage,revive // exercise internal HTTP protections

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

type shutdownBackend struct {
	testBackend
	closes   atomic.Int32
	closeErr error
}

func (b *shutdownBackend) Close() error {
	b.closes.Add(1)
	return b.closeErr
}

func TestMCPHTTPStartAndStopCloseListener(t *testing.T) {
	for _, tc := range []struct {
		useContext bool
		closeErr   error
	}{
		{false, nil},
		{true, nil},
		{true, errors.New("backend close failed")},
	} {
		t.Run(fmt.Sprintf("context=%v/close_error=%v", tc.useContext, tc.closeErr), func(t *testing.T) {
			backend := &shutdownBackend{closeErr: tc.closeErr}
			logs := &syncBuffer{}
			logger := logrus.New()
			logger.SetOutput(logs)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server, address, done := startShutdownTestServer(t, ctx, backend, logger)
			if err := server.Start(ctx); err == nil {
				t.Fatal("duplicate Start succeeded")
			}
			if tc.useContext {
				cancel()
			} else if err := server.Stop(); err != nil {
				t.Fatalf("Stop failed: %v", err)
			}
			assertShutdownTestServerClosed(t, server, address, done, backend)
			if tc.closeErr != nil {
				assertShutdownTestLog(t, logs, tc.closeErr)
			}
		})
	}
}

func startShutdownTestServer(
	t *testing.T, ctx context.Context, backend Backend, logger *logrus.Logger,
) (MCPServer, string, <-chan error) {
	t.Helper()
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.Addr().String()
	_ = reservation.Close()
	config, err := LoadFromJSON([]byte(fmt.Sprintf(
		`{"server":{"transport":"http","address":%q,"allow_unauthenticated":true,"audit":{"disabled":true}}}`,
		address)))
	if err != nil {
		t.Fatal(err)
	}
	server, err := newMCPServer(config, backend, logger)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Start(ctx) }()
	t.Cleanup(func() { _ = server.Stop() })
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for {
		response, requestErr := client.Post("http://"+address, "application/json", strings.NewReader("{}"))
		if requestErr == nil {
			_ = response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("HTTP listener never became responsive: %v", requestErr)
		}
		time.Sleep(time.Millisecond)
	}
	return server, address, done
}

func assertShutdownTestServerClosed(
	t *testing.T, server MCPServer, address string, done <-chan error, backend *shutdownBackend,
) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("normal HTTP shutdown returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after shutdown")
	}
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err == nil {
		_ = conn.Close()
		t.Fatal("HTTP listener remained open after shutdown")
	}
	if backend.closes.Load() != 1 {
		t.Fatalf("backend closes = %d, want 1", backend.closes.Load())
	}
	if err := server.Stop(); err != nil || backend.closes.Load() != 1 {
		t.Fatalf("repeated Stop was not idempotent: error=%v closes=%d", err, backend.closes.Load())
	}
}

func assertShutdownTestLog(t *testing.T, logs *syncBuffer, cause error) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !strings.Contains(strings.Join(logs.Lines(), "\n"), "MCP server shutdown failed") {
		if time.Now().After(deadline) {
			t.Fatalf("context shutdown error was not logged: %v", logs.Lines())
		}
		time.Sleep(time.Millisecond)
	}
	if !strings.Contains(strings.Join(logs.Lines(), "\n"), cause.Error()) {
		t.Fatalf("shutdown log omitted the underlying error: %v", logs.Lines())
	}
}

func TestMCPHTTPServerConfiguresSafeTimeouts(t *testing.T) {
	server := newMCPHTTPServer("127.0.0.1:0", http.NotFoundHandler(), nil)
	if server.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %s, want 10s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %s, want 30s", server.ReadTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %s, want 60s", server.IdleTimeout)
	}
	if server.MaxHeaderBytes != 1<<20 {
		t.Errorf("MaxHeaderBytes = %d, want 1 MiB", server.MaxHeaderBytes)
	}
	if server.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %s, want no write deadline for streaming responses", server.WriteTimeout)
	}
}

func TestMCPHTTPRejectsOversizedRequestBody(t *testing.T) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "http-limits-test", Version: "1"}, nil)
	config := DefaultHTTPConfig()
	config.Server.AllowUnauthenticated = true
	handler := newMCPStreamableHTTPHandler(mcpServer, config)
	handler, err := secureHTTPHandler(handler, config)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := serveMCPHTTPForTest(t, handler)

	body := bytes.NewReader(make([]byte, int(mcp.DefaultMaxRequestBodyBytes)+1))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request status = %d, want 413", response.StatusCode)
	}
}

func TestMCPHTTPKeepsAuthenticatedRequestsAvailable(t *testing.T) {
	t.Setenv("STACKQL_MCP_HTTP_TEST_TOKEN", "test-token")
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "http-auth-test", Version: "1"}, nil)
	config := DefaultHTTPConfig()
	config.Server.AuthTokenEnvVar = "STACKQL_MCP_HTTP_TEST_TOKEN"
	handler, err := secureHTTPHandler(newMCPStreamableHTTPHandler(mcpServer, config), config)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := serveMCPHTTPForTest(t, handler)
	client := &http.Client{Timeout: 5 * time.Second}
	const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`

	post := func(token string) int {
		req, reqErr := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(initialize))
		if reqErr != nil {
			t.Fatal(reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response, requestErr := client.Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer response.Body.Close()
		return response.StatusCode
	}

	if got := post(""); got != http.StatusUnauthorized {
		t.Fatalf("unauthenticated initialize status = %d, want 401", got)
	}
	if got := post("test-token"); got != http.StatusOK {
		t.Fatalf("authenticated initialize status = %d, want 200", got)
	}
}

func TestMCPHTTPBoundsConcurrentBackendCalls(t *testing.T) {
	config := fullAccessConfig()
	config.Server.Transport = serverTransportHTTP
	config.Server.AllowUnauthenticated = true
	config.Server.Audit.Disabled = true
	config.Server.MaxConcurrentRequests = 1
	config.Server.RequestTimeout = Duration(100 * time.Millisecond)
	backend := newGatedBackend()
	backend.deaf = true
	limitedServer, err := newMCPServer(config, backend, nil)
	if err != nil {
		t.Fatal(err)
	}
	implementation := limitedServer.(*simpleMCPServer)
	handler := newMCPStreamableHTTPHandler(implementation.server, config)
	handler, err = secureHTTPHandler(handler, config)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := serveMCPHTTPForTest(t, handler)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connect := func() *mcp.ClientSession {
		client := mcp.NewClient(&mcp.Implementation{Name: "http-limits-test", Version: "1"}, nil)
		session, connectErr := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint}, nil)
		if connectErr != nil {
			t.Fatalf("connect HTTP MCP client: %v", connectErr)
		}
		t.Cleanup(func() { _ = session.Close() })
		return session
	}
	holder, queued := connect(), connect()

	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(backend.release) }) }
	t.Cleanup(release)
	holderResult := make(chan error, 1)
	go func() {
		_, callErr := holder.CallTool(ctx, &mcp.CallToolParams{Name: "list_providers"})
		holderResult <- callErr
	}()
	deadline := time.Now().Add(2 * time.Second)
	for backend.active.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if backend.active.Load() == 0 {
		t.Fatal("HTTP holder call never reached the backend")
	}

	result, err := queued.CallTool(ctx, &mcp.CallToolParams{Name: "list_providers"})
	if err != nil {
		t.Fatalf("queued HTTP call failed at the protocol layer: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(firstText(t, result), "server busy") {
		t.Fatalf("queued HTTP call = %#v, want the busy refusal as a tool error", result)
	}
	release()
	<-holderResult
	if got := backend.calls.Load(); got != 1 {
		t.Errorf("backend calls = %d, want the queued call not to run", got)
	}
	_ = implementation.backend.Close()
}

func TestMCPHTTPServerTimesOutSlowHeaders(t *testing.T) {
	server := newMCPHTTPServer("", http.NotFoundHandler(), nil)
	server.ReadHeaderTimeout = 50 * time.Millisecond
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: localhost\r\nX-Slow: ")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	response, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read slow-header response: %v", err)
	}
	if len(response) > 0 && !bytes.Contains(response, []byte("408 Request Timeout")) {
		t.Fatalf("slow-header response = %q, want a 408 response or a closed connection", response)
	}
}

func TestMCPHTTPPreservesLongLivedSSE(t *testing.T) {
	config := DefaultHTTPConfig()
	config.Server.AllowUnauthenticated = true
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "stream-test", Version: "1"}, nil)
	handler, err := secureHTTPHandler(newMCPStreamableHTTPHandler(mcpServer, config), config)
	if err != nil {
		t.Fatal(err)
	}
	server := newMCPHTTPServer("", handler, nil)
	server.ReadTimeout = 50 * time.Millisecond
	server.IdleTimeout = 50 * time.Millisecond
	endpoint := serveConfiguredMCPHTTPForTest(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{}
	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(initialize))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	sessionID := response.Header.Get("Mcp-Session-Id")
	if response.StatusCode != http.StatusOK || sessionID == "" {
		t.Fatalf("initialize failed: status=%d session=%q", response.StatusCode, sessionID)
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Mcp-Session-Id", sessionID)
	req.Header.Set("Accept", "text/event-stream")
	response, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("SSE setup failed: status=%d headers=%v", response.StatusCode, response.Header)
	}
	priming := make([]byte, len(": ok\n\n"))
	if _, err := io.ReadFull(response.Body, priming); err != nil {
		t.Fatalf("read SSE priming event: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	for session := range mcpServer.Sessions() {
		if err := session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
			ProgressToken: "stream-test", Progress: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	event := make([]byte, len("event: message"))
	if _, err := io.ReadFull(response.Body, event); err != nil {
		t.Fatalf("SSE response stopped after the HTTP read/idle deadline: %v", err)
	}
}

func serveMCPHTTPForTest(t *testing.T, handler http.Handler) string {
	t.Helper()
	return serveConfiguredMCPHTTPForTest(t, newMCPHTTPServer("", handler, nil))
}

func serveConfiguredMCPHTTPForTest(t *testing.T, server *http.Server) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return "http://" + listener.Addr().String()
}
