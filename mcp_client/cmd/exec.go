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
package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackql/stackql/pkg/mcp_server"
)

var (
	actionName string // overwritten by flag
	actionArgs string // overwritten by flag
)

const (
	listToolsAction     = "list_tools"
	listProvidersAction = "list_providers"
)

// execCmd represents the exec command.
//
//nolint:gochecknoglobals // cobra pattern
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run mcp client queries",
	Long: `simple mcp client example
`,
	Run: func(cmd *cobra.Command, args []string) {
		clientCfgMap := make(map[string]any)
		jsonErr := json.Unmarshal([]byte(clientCfgJSON), &clientCfgMap)
		if jsonErr != nil {
			panic(fmt.Sprintf("error unmarshaling client cfg json: %v", jsonErr))
		}
		client, setupErr := mcp_server.NewMCPClient(
			clientType,
			url,
			clientCfgMap,
			nil,
		)
		if setupErr != nil {
			panic(fmt.Sprintf("error setting up mcp client: %v", setupErr))
		}
		var outputString string
		switch actionName {
		case listToolsAction:
			rv, rvErr := client.InspectTools()
			if rvErr != nil {
				panic(fmt.Sprintf("error inspecting tools: %v", rvErr))
			}
			output, outPutErr := json.MarshalIndent(rv, "", "  ")
			if outPutErr != nil {
				panic(fmt.Sprintf("error marshaling output: %v", outPutErr))
			}
			outputString = string(output)
		default:
			var args map[string]any
			jsonCfgErr := json.Unmarshal([]byte(actionArgs), &args)
			if jsonCfgErr != nil {
				panic(fmt.Sprintf("error unmarshaling action args: %v", jsonCfgErr))
			}
			rv, rvErr := client.CallToolText(actionName, args)
			if rvErr != nil {
				panic(fmt.Sprintf("error calling tool %s: %v", actionName, rvErr))
			}
			outputString = rv
		}
		//nolint:forbidigo // legacy
		fmt.Println(outputString)
	},
}
