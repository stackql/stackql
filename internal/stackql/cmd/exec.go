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
	"io"
	"os"
	"runtime/pprof"

	"github.com/spf13/cobra"

	"github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/writer"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Run one or more stackql commands or queries",
	Long: `Run one or more stackql commands or queries from the command line or from an
input file. For example:

stackql exec \
"select id, name from compute.instances where project = 'stackql-demo' and zone = 'australia-southeast1-a'" \
--credentialsfilepath /mnt/c/tmp/stackql-demo.json --output csv

stackql exec -i iqlscripts/listinstances.iql --credentialsfilepath /mnt/c/tmp/stackql-demo.json --output json

stackql exec -i iqlscripts/create-disk.iql --credentialsfilepath /mnt/c/tmp/stackql-demo.json
`,
	Run: func(cmd *cobra.Command, args []string) {

		var err error
		var rdr io.Reader

		if runtimeCtx.CPUProfile != "" {
			f, err := os.Create(runtimeCtx.CPUProfile)
			if err != nil {
				iqlerror.PrintErrorAndExitOneIfError(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}

		switch runtimeCtx.InfilePath {
		case "stdin":
			if len(args) == 0 || args[0] == "" {
				cmd.Help()
				os.Exit(0)
			}
			rdr = bytes.NewReader([]byte(args[0]))
		default:
			rdr, err = os.Open(runtimeCtx.InfilePath)
			iqlerror.PrintErrorAndExitOneIfError(err)
		}
		inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(err)
		handlerCtx, err := entryutil.BuildHandlerContext(runtimeCtx, rdr, queryCache, inputBundle)
		iqlerror.PrintErrorAndExitOneIfError(err)
		iqlerror.PrintErrorAndExitOneIfNil(handlerCtx, "Handler context error")
		RunCommand(handlerCtx, nil, nil)
	},
}

func getOutputFile(filename string) (*os.File, error) {
	switch filename {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		return os.Create(filename)
	}
}

func RunCommand(handlerCtx handler.HandlerContext, outfile io.Writer, outErrFile io.Writer) {
	defer iqlerror.HandlePanic(outErrFile)
	if outfile == nil {
		outfile, _ = getOutputFile(handlerCtx.GetRuntimeContext().OutfilePath)
	}
	if outErrFile == nil {
		outErrFile, _ = getOutputFile(writer.StdErrStr)
	}
	handlerCtx.SetOutfile(outfile)
	handlerCtx.SetOutErrFile(outErrFile)
	stackqlDriver, err := driver.NewStackQLDriver(handlerCtx)
	iqlerror.PrintErrorAndExitOneIfError(err)
	if handlerCtx.GetRuntimeContext().DryRunFlag {
		stackqlDriver.ProcessDryRun(handlerCtx)
		return
	}
	stackqlDriver.ProcessQuery(handlerCtx)
}
