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
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
)

const (
	forbiddenRegistryCharacters string = ` ;\`
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Interaction with the stackql provider registry, as configured at initialisation time.  Usage: stackql registry {subcommand} [{arg}]",
	Long: `
	Interaction with the provider registry, as configured at initialisation time. Usage: stackql registry {subcommand}
	Currently supported subcommands:
	  - pull {provider} {version}
	  - list
	`,
	Run: func(cmd *cobra.Command, args []string) {

		var rdr io.Reader

		usagemsg := cmd.Long + "\n\n" + cmd.UsageString()
		if len(args) < 1 {
			iqlerror.PrintErrorAndExitOneWithMessage(usagemsg)
		}
		if len(args) == 0 {
			iqlerror.PrintErrorAndExitOneWithMessage(usagemsg)
		}
		subCommand := strings.ToLower(args[0])
		switch subCommand {
		case "pull":
			if len(args) != 3 {
				iqlerror.PrintErrorAndExitOneWithMessage(usagemsg)
			}
			providerName := args[1]
			providerVersion := args[2]

			if strings.ContainsAny(providerName, forbiddenRegistryCharacters) || strings.ContainsAny(providerVersion, forbiddenRegistryCharacters) {
				iqlerror.PrintErrorAndExitOneWithMessage("forbidden characters detected")
			}
			rdr = bytes.NewReader([]byte(fmt.Sprintf("registry pull %s %s;", providerName, providerVersion)))
		case "list":
			switch len(args) {
			case 1:
				rdr = bytes.NewReader([]byte("registry list;"))
			case 2:
				rdr = bytes.NewReader([]byte(fmt.Sprintf("registry list %s;", args[1])))
			default:
				iqlerror.PrintErrorAndExitOneWithMessage(fmt.Sprintf("invalid arg count = %d for registry list commmand", len(args)))
			}
		}

		inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(err)
		handlerCtx, err := entryutil.BuildHandlerContext(runtimeCtx, rdr, queryCache, inputBundle)
		iqlerror.PrintErrorAndExitOneIfError(err)
		iqlerror.PrintErrorAndExitOneIfNil(&handlerCtx, "Handler context error")
		RunCommand(&handlerCtx, nil, nil)

	},
}
