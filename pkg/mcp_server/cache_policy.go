package mcp_server //nolint:revive // package name is retained for compatibility

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setCachePolicy(_ context.Context, req mcp.Request, cache *mcp.Cacheable) {
	cache.CacheScope = "private"
	cache.TTLMs = 0
	switch req.(type) {
	case *mcp.ListToolsRequest, *mcp.ListPromptsRequest, *mcp.ListResourcesRequest, *mcp.ListResourceTemplatesRequest:
		cache.TTLMs = int((30 * time.Second).Milliseconds())
	}
}
