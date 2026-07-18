/*
Copyright © 2025 stackql info@stackql.io

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"encoding/json"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/stackql/any-sdk/pkg/db/db_util"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/acid/tsm_physio"
	"github.com/stackql/stackql/internal/stackql/buildinfo"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/mcpbackend"
	"github.com/stackql/stackql/pkg/mcp_server"

	_ "github.com/jackc/pgx/v5" //nolint:revive // canonical driver pattern
)

// sqlBackendIdentifier derives a stable identifier for the backing SQL engine
// from the runtime context's SQLBackendCfgRaw blob. Falls back to the empty
// string when nothing usable is configured.
func sqlBackendIdentifier(runtimeCtx dto.RuntimeCtx) string {
	raw := runtimeCtx.SQLBackendCfgRaw
	if raw == "" {
		return ""
	}
	cfg, err := dto.GetSQLBackendCfg(raw)
	if err != nil {
		return ""
	}
	if d := cfg.GetSQLDialect(); d != "" {
		return d
	}
	return cfg.DBEngine
}

// providerRegistryIdentifier extracts the registry URL/path from the runtime
// context's RegistryRaw YAML/JSON blob.
func providerRegistryIdentifier(runtimeCtx dto.RuntimeCtx) string {
	raw := runtimeCtx.RegistryRaw
	if raw == "" {
		return ""
	}
	var probe struct {
		URL          string `json:"url" yaml:"url"`
		LocalDocRoot string `json:"localDocRoot" yaml:"localDocRoot"`
	}
	if err := yaml.Unmarshal([]byte(raw), &probe); err != nil {
		return ""
	}
	if probe.URL != "" {
		return probe.URL
	}
	return probe.LocalDocRoot
}

//nolint:gochecknoglobals // cobra pattern
var (
	mcpServerType string // overwritten by flag
	mcpConfig     string // overwritten by flag
	envFilePath   string // overwritten by flag; sourced for every entrypoint in initConfig
)

//nolint:gochecknoglobals // cobra pattern
var mcpSrvCmd = &cobra.Command{
	Use:   "mcp",
	Short: "run mcp server",
	Long: `
	Run a MCP protocol server.
	Supports MCP client connections from all manner or libs.
  `,
	//nolint:revive // acceptable for now
	Run: func(cmd *cobra.Command, args []string) {
		flagErr := dependentFlagHandler(&runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(flagErr)
		inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(err)
		handlerCtx, err := entryutil.BuildHandlerContext(runtimeCtx, nil, queryCache, inputBundle, false)
		iqlerror.PrintErrorAndExitOneIfError(err)
		iqlerror.PrintErrorAndExitOneIfNil(handlerCtx, "handler context is unexpectedly nil")
		if mcpServerType == "" {
			mcpServerType = "http"
		}
		runMCPServer(handlerCtx)
	},
}

func runMCPServer(handlerCtx handler.HandlerContext) {
	var config mcp_server.Config
	json.Unmarshal([]byte(mcpConfig), &config) //nolint:errcheck // TODO: investigate
	if config.Server.Transport == "" {
		config.Server.Transport = mcpServerType
	}
	// MCP clients must be able to distinguish "query ran, zero rows" from
	// "query failed upstream" (issue #670), so data acquisition failures
	// surface as statement errors rather than empty result sets.  This is
	// deliberately scoped to the MCP server; CLI and pgsrv sessions keep
	// the historical empty-result-plus-stderr-notice semantics.
	handlerCtx.SetStrictUpstreamErrors(true)
	bi := buildinfo.Get()
	transport := config.Server.Transport
	runtimeContext := handlerCtx.GetRuntimeContext()
	mode := config.GetMode()
	serverInfo := mcpbackend.NewServerBuildInfo(
		bi,
		transport,
		sqlBackendIdentifier(runtimeContext),
		providerRegistryIdentifier(runtimeContext),
		mode,
	)
	var backend mcp_server.Backend
	var backendErr error
	if mcpServerType == "reverse_proxy" {
		dsn := config.GetBackendConnectionString()
		// conn
		var cfg dto.SQLBackendCfg
		cfg.DSN = dsn
		cfg.InitMaxRetries = 5
		cfg.InitRetryInitialDelay = 2
		db, err := db_util.GetDB("pgx", "postgres", cfg)
		iqlerror.PrintErrorAndExitOneIfError(err)
		backend, backendErr = mcpbackend.NewStackqlMCPReverseProxyService(
			dsn,
			db,
			handlerCtx,
			logging.GetLogger(),
			serverInfo,
		)
		iqlerror.PrintErrorAndExitOneIfError(backendErr)
	} else {
		orchestrator, orchestratorErr := tsm_physio.NewOrchestrator(handlerCtx)
		iqlerror.PrintErrorAndExitOneIfError(orchestratorErr)
		iqlerror.PrintErrorAndExitOneIfNil(orchestrator, "orchestrator is unexpectedly nil")
		backend, backendErr = mcpbackend.NewStackqlMCPBackendService(
			orchestrator,
			handlerCtx,
			logging.GetLogger(),
			serverInfo,
			envFilePath,
		)
		iqlerror.PrintErrorAndExitOneIfError(backendErr)
		iqlerror.PrintErrorAndExitOneIfNil(backend, "mcp backend is unexpectedly nil")
	}
	server, serverErr := mcp_server.NewAgnosticBackendServer(
		backend,
		&config,
		logging.GetLogger(),
	)
	iqlerror.PrintErrorAndExitOneIfError(serverErr)
	// A transport/session failure must be loud: log it and exit non-zero
	// rather than silently returning success (issue #668).
	startErr := server.Start(context.Background())
	iqlerror.PrintErrorAndExitOneIfError(startErr)
}
