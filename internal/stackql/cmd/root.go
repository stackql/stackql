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
	"fmt"
	"os"
	"path"

	"github.com/stackql/stackql/internal/pkg/txncounter"
	"github.com/stackql/stackql/internal/stackql/config"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"

	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	lrucache "vitess.io/vitess/go/cache"

	log "github.com/sirupsen/logrus"
)

var (
	BuildMajorVersion   string = ""
	BuildMinorVersion   string = ""
	BuildPatchVersion   string = ""
	BuildCommitSHA      string = ""
	BuildShortCommitSHA string = ""
	BuildDate           string = ""
	BuildPlatform       string = ""
)

var SemVersion string = fmt.Sprintf("%s.%s.%s", BuildMajorVersion, BuildMinorVersion, BuildPatchVersion)

var (
	runtimeCtx      dto.RuntimeCtx
	queryCache      *lrucache.LRUCache
	txnCtrMgr       *txncounter.TxnCounterManager
	replicateCtrMgr bool = false
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "stackql",
	Version: SemVersion,
	Short:   "Cloud infrastructure coding using SQL",
	Long: `
	_ __  _ ___ ___  __   __  _    
	| |  \| | __| _ \/  \ /__\| |   
	| | | ' | _|| v / /\ | \/ | |_  
	|_|_|\__|_| |_|_\_||_|\_V_\___| 

Cloud infrastructure coding using SQL. For example:

SELECT * FROM google.compute.instances;`,
	Run: func(cmd *cobra.Command, args []string) {
		// in the root command is executed with no arguments, print the help message
		usagemsg := cmd.Long + "\n\n" + cmd.UsageString()
		fmt.Println(usagemsg)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.SetVersionTemplate("stackql v{{.Version}} " + BuildPlatform + " (" + BuildShortCommitSHA + ")\nBuildDate: " + BuildDate + "\nhttps://stackql.io\n")

	rootCmd.PersistentFlags().StringVar(&runtimeCtx.CPUProfile, dto.CPUProfileKey, "", "cpuprofile file, none if empty")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().IntVar(&runtimeCtx.APIRequestTimeout, dto.APIRequestTimeoutKey, 45, "API request timeout in seconds, 0 for no timeout.")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.ColorScheme, dto.ColorSchemeKey, config.GetDefaultColorScheme(), fmt.Sprintf("Color scheme, must be one of {'%s', '%s', '%s'}", dto.DarkColorScheme, dto.LightColorScheme, dto.NullColorScheme))
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.ConfigFilePath, dto.ConfigFilePathKey, config.GetDefaultConfigFilePath(), fmt.Sprintf("Config file full path; defaults to current dir"))
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.ProviderRootPath, dto.ProviderRootPathKey, config.GetDefaultProviderCacheRoot(), fmt.Sprintf("Config and cache root path"))
	rootCmd.PersistentFlags().Uint32Var(&runtimeCtx.ProviderRootPathMode, dto.ProviderRootPathModeKey, config.GetDefaultProviderCacheDirFileMode(), fmt.Sprintf("Config and cache file mode"))
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.ViperCfgFileName, dto.ViperCfgFileNameKey, config.GetDefaultViperConfigFileName(), fmt.Sprintf("Config filename"))
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.AuthRaw, dto.AuthCtxKey, "", `auth contexts keyvals in json form, eg: '{ "google": { "keyfilepath": "/path/to/google/sevice/account/key.json",  "keyfiletype": "serviceaccount" }, "okta": { "keyenvvar": "OKTA_SECRET_KEY",  "keyfiletype": "api_key" } }'`)
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.DbEngine, dto.DbEngineKey, config.GetDefaultDbEngine(), fmt.Sprintf("DB engine id"))
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.DbFilePath, dto.DbFilePathKey, config.GetDefaultDbFilePath(), fmt.Sprintf("DB persistence filename"))
	rootCmd.PersistentFlags().IntVar(&runtimeCtx.DbGenerationId, dto.DbGenerationIdKey, txncounter.GetNextGenerationId(), fmt.Sprintf("DB generation id"))
	rootCmd.PersistentFlags().BoolVar(&runtimeCtx.HTTPLogEnabled, dto.HTTPLogEnabledKey, false, "Display http request info in terminal")
	rootCmd.PersistentFlags().IntVar(&runtimeCtx.HTTPMaxResults, dto.HTTPMaxResultsKey, -1, "max results per http request, any number <=0 results in no limitation")
	rootCmd.PersistentFlags().IntVar(&runtimeCtx.HTTPProxyPort, dto.HTTPProxyPortKey, -1, "http proxy port, any number <=0 will result in the default port for a given scheme (eg: http -> 80)")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.HTTPProxyHost, dto.HTTPProxyHostKey, "", "http proxy host, empty means no proxy")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.HTTPProxyScheme, dto.HTTPProxySchemeKey, "http", "http proxy scheme, eg 'http'")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.HTTPProxyPassword, dto.HTTPProxyPasswordKey, "", "http proxy password")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.HTTPProxyUser, dto.HTTPProxyUserKey, "", "http proxy user")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.DbInitFilePath, dto.DbInitFilePathKey, config.GetDefaultDbInitFilePath(), fmt.Sprintf("DB init file path"))
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.ProviderStr, dto.ProviderStrKey, config.GetGoogleProviderString(), fmt.Sprintf(`stackql provider`))
	rootCmd.PersistentFlags().BoolVar(&runtimeCtx.WorkOffline, dto.WorkOfflineKey, false, "Work offline, using cached data")
	rootCmd.PersistentFlags().BoolVarP(&runtimeCtx.VerboseFlag, dto.VerboseFlagKey, "v", false, "Verbose flag")
	rootCmd.PersistentFlags().BoolVar(&runtimeCtx.DryRunFlag, dto.DryRunFlagKey, false, "dryrun flag; preprocessor only will run and output returned")
	rootCmd.PersistentFlags().BoolVar(&runtimeCtx.Reinit, dto.ReinitKey, false, "reinit; will delete db file at startup and force regeneration of all dependencies")
	rootCmd.PersistentFlags().BoolVarP(&runtimeCtx.CSVHeadersDisable, dto.CSVHeadersDisableKey, "H", false, "Disable CSV headers flag")
	rootCmd.PersistentFlags().StringVarP(&runtimeCtx.OutputFormat, dto.OutputFormatKey, "o", "table", "Output format, must be (json | table | csv)")
	rootCmd.PersistentFlags().StringVarP(&runtimeCtx.OutfilePath, dto.OutfilePathKey, "f", "stdout", "Output file into which results are written")
	rootCmd.PersistentFlags().StringVarP(&runtimeCtx.InfilePath, dto.InfilePathKey, "i", "stdin", "Input file from which queries are read")
	rootCmd.PersistentFlags().StringVarP(&runtimeCtx.TemplateCtxFilePath, dto.TemplateCtxFilePathKey, "q", "", "Context file for templating")
	rootCmd.PersistentFlags().IntVar(&runtimeCtx.QueryCacheSize, dto.QueryCacheSizeKey, constants.DefaultQueryCacheSize, "Size in number of entries of LRU cache for query plans")
	rootCmd.PersistentFlags().StringVarP(&runtimeCtx.Delimiter, dto.DelimiterKey, "d", ",", "Delimiter for csv output;  single character only, ignored for all non-csv output")
	rootCmd.PersistentFlags().IntVar(&runtimeCtx.CacheKeyCount, dto.CacheKeyCountKey, 100, "Cache initial key count")
	rootCmd.PersistentFlags().IntVar(&runtimeCtx.CacheTTL, dto.CacheTTLKey, 3600, "TTL for cached metadata documents, in seconds")
	rootCmd.PersistentFlags().BoolVar(&runtimeCtx.TestWithoutApiCalls, dto.TestWithoutApiCallsKey, false, "Flag to omit api calls for testing")
	rootCmd.PersistentFlags().BoolVar(&runtimeCtx.UseNonPreferredAPIs, dto.UseNonPreferredAPIsKEy, false, "Flag to enable non-preferred APIs")
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.LogLevelStr, dto.LogLevelStrKey, config.GetDefaultLogLevelString(), fmt.Sprintf(`Log level`))
	rootCmd.PersistentFlags().StringVar(&runtimeCtx.ErrorPresentation, dto.ErrorPresentationKey, config.GetDefaultErrorPresentationString(), fmt.Sprintf(`Error presentation, options are: {"stderr", "record"}`))

	rootCmd.PersistentFlags().MarkHidden(dto.TestWithoutApiCallsKey)
	rootCmd.PersistentFlags().MarkHidden(dto.ViperCfgFileNameKey)
	rootCmd.PersistentFlags().MarkHidden(dto.ErrorPresentationKey)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	queryCache = lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize))

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(shellCmd)
	// rootCmd.AddCommand(srvCmd)

}

func setLogLevel() {
	logLevel, err := log.ParseLevel(runtimeCtx.LogLevelStr)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(logLevel)
}

func mergeConfigFromFile(runtimeCtx *dto.RuntimeCtx, flagSet pflag.FlagSet) {
	props, err := properties.LoadFile(runtimeCtx.ConfigFilePath, properties.UTF8)
	if err == nil {
		propertiesMap := props.Map()
		for k, v := range propertiesMap {
			if flagSet.Lookup(k) != nil && !flagSet.Lookup(k).Changed {
				runtimeCtx.Set(k, v)
			}
		}
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	mergeConfigFromFile(&runtimeCtx, *rootCmd.PersistentFlags())

	setLogLevel()
	config.CreateDirIfNotExists(runtimeCtx.ProviderRootPath, os.FileMode(runtimeCtx.ProviderRootPathMode))
	config.CreateDirIfNotExists(path.Join(runtimeCtx.ProviderRootPath, runtimeCtx.ProviderStr), os.FileMode(runtimeCtx.ProviderRootPathMode))
	config.CreateDirIfNotExists(config.GetReadlineDirPath(runtimeCtx), os.FileMode(runtimeCtx.ProviderRootPathMode))
	if runtimeCtx.Reinit {
		os.Remove(runtimeCtx.DbFilePath)
	}
	viper.SetConfigFile(path.Join(runtimeCtx.ProviderRootPath, runtimeCtx.ViperCfgFileName))
	viper.AddConfigPath(runtimeCtx.ProviderRootPath)
	log.Infof("ProviderRootPath = %s, ViperCfgFileName = %s", runtimeCtx.ProviderRootPath, runtimeCtx.ViperCfgFileName)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
