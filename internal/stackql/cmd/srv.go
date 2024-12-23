/*
Copyright © 2019 stackql info@stackql.io

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
	"github.com/spf13/cobra"

	"github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/psqlwire"
)

const MIN = 1
const MAX = 100

const DEFAULT_PORT_NO = 3406 //nolint:revive,stylecheck // legacy

//nolint:gochecknoglobals // cobra pattern
var srvCmd = &cobra.Command{
	Use:   "srv",
	Short: "run postgres wire server",
	Long: `
	Run a PostgreSQL wire protocol server.
	Supports client connections from psql and all manner or libs.
  `,
	//nolint:revive // acceptable for now
	Run: func(cmd *cobra.Command, args []string) {
		flagErr := dependentFlagHandler(&runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(flagErr)
		inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(err)
		handlerCtx, err := entryutil.BuildHandlerContextNoPreProcess(runtimeCtx, queryCache, inputBundle)
		iqlerror.PrintErrorAndExitOneIfError(err)
		sbe := driver.NewStackQLDriverFactory(handlerCtx)
		server, err := psqlwire.MakeWireServer(sbe, runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(err)
		server.Serve() //nolint:errcheck // TODO: investigate
	},
}
