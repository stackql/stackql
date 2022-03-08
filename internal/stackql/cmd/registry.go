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

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
)

// execCmd represents the exec command
var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Interaction with the provider registry, as configured at initialisation time.  Usage: stackql registry {subcommand} [{arg}]",
	Long: `
	Interaction with the provider registry, as configured at initialisation time. Usage: stackql registry {subcommand}
	Currently supported subcommands:
	  - pull {provider} {version}
    
	`,
	Run: func(cmd *cobra.Command, args []string) {
		usagemsg := cmd.Long + "\n\n" + cmd.UsageString()
		if len(args) < 1 {
			iqlerror.PrintErrorAndExitOneWithMessage(usagemsg)
		}
		reg, err := handler.GetRegistry(runtimeCtx)
		if err != nil {
			iqlerror.PrintErrorAndExitOneWithMessage(err.Error())
		}
		if len(args) == 0 {
			iqlerror.PrintErrorAndExitOneWithMessage(usagemsg)
		}
		subCommand := args[0]
		switch subCommand {
		case "pull":
			if len(args) != 3 {
				iqlerror.PrintErrorAndExitOneWithMessage(usagemsg)
			}
			providerName := args[1]
			providerVersion := args[2]
			err := reg.PullAndPersistProviderArchive(providerName, providerVersion)
			if err != nil {
				iqlerror.PrintErrorAndExitOneWithMessage(err.Error())
			}
			return
		}
		iqlerror.PrintErrorAndExitOneWithMessage(usagemsg)
	},
}
