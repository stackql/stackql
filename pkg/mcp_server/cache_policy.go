package mcp_server //nolint:revive // package name is retained for compatibility

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listCacheTTL and listCacheScope are the house list policy, shared with the
// skills/list and skills/get results (skills.go).
const (
	listCacheTTL   = 30 * time.Second
	listCacheScope = "private"
)

func setCachePolicy(_ context.Context, req mcp.Request, cache *mcp.Cacheable) {
	cache.CacheScope = listCacheScope
	cache.TTLMs = 0
	switch req.(type) {
	case *mcp.ListToolsRequest, *mcp.ListPromptsRequest, *mcp.ListResourcesRequest, *mcp.ListResourceTemplatesRequest:
		cache.TTLMs = int(listCacheTTL.Milliseconds())
	}
}
