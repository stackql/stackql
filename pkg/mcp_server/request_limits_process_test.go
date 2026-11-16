package mcp_server //nolint:testpackage,revive // exercise the production stdio transport in a subprocess

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

type processSlowBackend struct {
	testBackend
	active atomic.Int32
}

func (b *processSlowBackend) ListProviders(ctx context.Context) ([]map[string]any, error) {
	b.active.Add(1)
	defer b.active.Add(-1)
	<-ctx.Done()
	// Model cancellation cleanup: the permit must remain held until work exits.
	time.Sleep(200 * time.Millisecond)
	return nil, ctx.Err()
}

func (b *processSlowBackend) ServerInfo(_ context.Context, _ any) (dto.ServerInfoOutput, error) {
	return dto.ServerInfoOutput{Version: fmt.Sprintf("active=%d", b.active.Load())}, nil
}

func TestMCPRequestLimitsStdioProcessHelper(t *testing.T) {
	if os.Getenv("STACKQL_MCP_LIMITS_PROCESS_HELPER") != "1" {
		return
	}
	cfg, err := LoadFromJSON([]byte(`{"server":{"transport":"stdio","max_concurrent_requests":1,"request_timeout":"100ms","audit":{"disabled":true}}}`))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	server, err := newMCPServer(cfg, &processSlowBackend{}, nil)
	if err == nil {
		err = server.Start(context.Background())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestMCPRequestLimitsStdioProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPRequestLimitsStdioProcessHelper$")
	command.Env = append(os.Environ(), "STACKQL_MCP_LIMITS_PROCESS_HELPER=1", "GOMAXPROCS=2")
	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-limits-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("connect stdio subprocess: %v", err)
	}
	defer session.Close()
	type callResult struct {
		result *mcp.CallToolResult
		err    error
	}
	holder := make(chan callResult, 1)
	go func() {
		result, callErr := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_providers"})
		holder <- callResult{result: result, err: callErr}
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		info, callErr := session.CallTool(ctx, &mcp.CallToolParams{Name: "server_info"})
		if callErr != nil {
			t.Fatal(callErr)
		}
		if strings.Contains(firstText(t, info), "active=1") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("stdio holder never reached the backend")
		}
		time.Sleep(time.Millisecond)
	}
	queued, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_providers"})
	if err != nil || queued == nil || !queued.IsError || !strings.Contains(firstText(t, queued), "server busy") {
		t.Fatalf("stdio queued call = %#v, error=%v; want the busy tool error", queued, err)
	}
	completed := <-holder
	if completed.err != nil || completed.result == nil || !completed.result.IsError ||
		!strings.Contains(firstText(t, completed.result), "server.request_timeout of 100ms exceeded") {
		t.Fatalf("stdio deadline call = %#v, error=%v; want the propagated deadline tool error",
			completed.result, completed.err)
	}
	followup, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_services", Arguments: map[string]any{"provider": "test"},
	})
	if err != nil || followup == nil || followup.IsError {
		t.Fatalf("stdio follow-up call = %#v, error=%v; want a live server and a released permit", followup, err)
	}
}
