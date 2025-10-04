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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stackql/stackql/pkg/mcp_server"
)

var (
	clientType = "http"
	url        = "127.0.0.1:9191"
)

//nolint:revive,gochecknoglobals // explicit preferred
var (
	BuildMajorVersion   string = ""
	BuildMinorVersion   string = ""
	BuildPatchVersion   string = ""
	BuildCommitSHA      string = ""
	BuildShortCommitSHA string = ""
	BuildDate           string = ""
	BuildPlatform       string = ""
)

// rootCmd represents the base command when called without any subcommands.
//
//nolint:gochecknoglobals // global vars are a pattern for this lib
var rootCmd = &cobra.Command{
	Use:     "stackql_mcp_client",
	Version: "0.1.0",
	Short:   "Cloud asset management and automation using SQL",
	Long:    `stackql mcp client`,
	//nolint:revive // acceptable for now
	Run: func(cmd *cobra.Command, args []string) {
		// in the root command is executed with no arguments, print the help message
		usagemsg := cmd.Long + "\n\n" + cmd.UsageString()
		fmt.Println(usagemsg) //nolint:forbidigo // legacy
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

//nolint:lll,funlen,gochecknoinits,mnd // init is a pattern for this lib
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.SetVersionTemplate("stackql v{{.Version}} " + BuildPlatform + " (" + BuildShortCommitSHA + ")\nBuildDate: " + BuildDate + "\nhttps://stackql.io\n")

	rootCmd.PersistentFlags().StringVar(&clientType, "client-type", mcp_server.MCPClientTypeSTDIO, "MCP client type (http or stdio for now)")
	rootCmd.PersistentFlags().StringVar(&url, "url", "http://127.0.0.1:9876", "MCP server URL.  Relevant for http and sse client types.")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// var dummyString string

	// rootCmd.PersistentFlags().StringVar(&runtimeCtx.DBInternalCfgRaw, dto.DBInternalCfgRawKey, "{}", "JSON / YAML string to configure DBMS housekeeping query handling")

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(execCmd)
	execCmd.PersistentFlags().StringVar(&actionName, "exec.action", "list_tools", "MCP server action name")
	execCmd.PersistentFlags().StringVar(&actionArgs, "exec.args", "{}", "MCP server action arguments as JSON string")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// mergeConfigFromFile(&runtimeCtx, *rootCmd.PersistentFlags())

	// logging.SetLogger(runtimeCtx.LogLevelStr)
	// config.CreateDirIfNotExists(runtimeCtx.ApplicationFilesRootPath, os.FileMode(runtimeCtx.ApplicationFilesRootPathMode))                                    //nolint:errcheck,lll // TODO: investigate
	// config.CreateDirIfNotExists(path.Join(runtimeCtx.ApplicationFilesRootPath, runtimeCtx.ProviderStr), os.FileMode(runtimeCtx.ApplicationFilesRootPathMode)) //nolint:errcheck,lll // TODO: investigate
	// config.CreateDirIfNotExists(config.GetReadlineDirPath(runtimeCtx), os.FileMode(runtimeCtx.ApplicationFilesRootPathMode))                                  //nolint:errcheck,lll // TODO: investigate
	// viper.SetConfigFile(path.Join(runtimeCtx.ApplicationFilesRootPath, runtimeCtx.ViperCfgFileName))
	// viper.AddConfigPath(runtimeCtx.ApplicationFilesRootPath)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed()) //nolint:forbidigo // legacy
	}
}
