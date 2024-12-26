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
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"

	"github.com/stackql/stackql/internal/stackql/config"
	"github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/presentation"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/writer"

	"github.com/spf13/cobra"

	"github.com/chzyer/readline"
)

//nolint:lll // contains long string constants
const (
	shellLongStr string = `stackql Command Shell %s
Copyright (c) 2021, stackql studios. All rights reserved.
Welcome to the interactive shell for running stackql commands.
---`

	// Auth messages.
	interactiveSuccessMsgTmpl string = `Authenticated interactively to google as %s, to change the authenticated user, use AUTH REVOKE followed by AUTH LOGIN, see https://docs.stackql.io/language-spec/auth`

	saFileErrorMsgTmpl string = `Not authenticated, credentials referenced in %s do not exist, authenticate interactively using AUTH LOGIN, for more information see https://docs.stackql.io/language-spec/auth`

	saSuccessMsgTmpl string = `Authenticated using credentials set using the flag %s of type = '%s', for more information see https://docs.stackql.io/language-spec/auth`

	credentialProvidedMsgTmpl string = `Credentials provided using the the flag %s of type = '%s', for more information see https://docs.stackql.io/language-spec/auth`
)

func getShellIntroLong() string {
	return fmt.Sprintf(shellLongStr, SemVersion)
}

func usage(w io.Writer) {
	io.WriteString(w, getShellIntroLong()+"\r\n") //nolint:errcheck // TODO: investigate
}

func getShellPRompt(authCtx *dto.AuthCtx, cd presentation.Driver) string {
	if runtime.GOOS == "windows" {
		return "stackql  >>"
	}
	if authCtx != nil && authCtx.Active {
		switch authCtx.Type {
		case dto.AuthInteractiveStr:
			return cd.Sprintf("stackql* >>")
		case dto.AuthServiceAccountStr:
			return cd.Sprintf("stackql**>>")
		}
	}
	return cd.Sprintf("stackql  >>")
}

func getIntroAuthMsg(authCtx *dto.AuthCtx, prov provider.IProvider) string {
	if authCtx != nil {
		if authCtx.Active {
			switch authCtx.Type {
			case dto.AuthInteractiveStr:
				return fmt.Sprintf(interactiveSuccessMsgTmpl, authCtx.ID)
			case dto.AuthServiceAccountStr, dto.AuthAPIKeyStr:
				return fmt.Sprintf(saSuccessMsgTmpl, authCtx.GetCredentialsSourceDescriptorString(), authCtx.Type)
			}
		} else if prov != nil {
			if err := prov.CheckCredentialFile(authCtx); authCtx.HasKey() && err != nil {
				return fmt.Sprintf(saFileErrorMsgTmpl, authCtx.GetCredentialsSourceDescriptorString())
			}
		}
		switch authCtx.Type {
		case dto.AuthServiceAccountStr, dto.AuthAPIKeyStr:
			return fmt.Sprintf(credentialProvidedMsgTmpl, authCtx.GetCredentialsSourceDescriptorString(), authCtx.Type)
		}
	}
	return "" // notAuthenticatedMsg
}

// shellCmd represents the shell command.
//
//nolint:gochecknoglobals // cobra command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Interactive shell for running stackql commands",
	Long:  getShellIntroLong(),
	//nolint:revive // acceptable for now
	Run: func(command *cobra.Command, args []string) {

		flagErr := dependentFlagHandler(&runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(flagErr)

		cd := presentation.NewPresentationDriver(runtimeCtx)

		outfile, _ := writer.GetDecoratedOutputWriter(runtimeCtx.OutfilePath, cd)

		outErrFile, _ := writer.GetDecoratedOutputWriter(writer.StdErrStr, cd)

		var sb strings.Builder
		fmt.Fprintln(outErrFile, getShellIntroLong())

		inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(err)

		handlerCtx, handlerrErr := handler.GetHandlerCtx("", runtimeCtx, queryCache, inputBundle)
		if handlerrErr != nil {
			fmt.Fprintln( //nolint:gosimple // legacy
				outErrFile,
				fmt.Sprintf(
					"Error setting up handler context for provider '%s': \"%s\"",
					runtimeCtx.ProviderStr, handlerrErr))
		}
		var authCtx *dto.AuthCtx
		var prov provider.IProvider
		var pErr, authErr error
		if handlerCtx.GetRuntimeContext().ProviderStr != "" {
			prov, pErr = handlerCtx.GetProvider(handlerCtx.GetRuntimeContext().ProviderStr)
			authCtx, authErr = handlerCtx.GetAuthContext(prov.GetProviderString())
			if authErr != nil {
				fmt.Fprintln( //nolint:gosimple // legacy
					outErrFile,
					fmt.Sprintf(
						"Error setting up AUTH for provider '%s'",
						handlerCtx.GetRuntimeContext().ProviderStr))
			}
			if pErr == nil {
				prov.ShowAuth(authCtx) //nolint:errcheck // TODO: investigate
			} else {
				fmt.Fprintln( //nolint:gosimple // legacy
					outErrFile,
					fmt.Sprintf(
						"Error setting up API for provider '%s'",
						handlerCtx.GetRuntimeContext().ProviderStr,
					),
				)
			}
		}

		readlineCfg := &readline.Config{
			Stderr:               outErrFile,
			Stdout:               outfile,
			Prompt:               getShellPRompt(authCtx, cd),
			InterruptPrompt:      "^C",
			EOFPrompt:            "exit",
			HistoryFile:          config.GetReadlineFilePath(handlerCtx.GetRuntimeContext()),
			HistorySearchFold:    true,
			HistoryExternalWrite: true,
		}

		sessionRunnerInstance, sessionErr := newSessionRunner(
			handlerCtx,
			outfile,
			outfile,
		)
		iqlerror.PrintErrorAndExitOneIfError(sessionErr)

		l, err := readline.NewEx(readlineCfg)
		if err != nil {
			panic(err)
		}
		defer l.Close()

		fmt.Fprintln(
			outErrFile,
			getIntroAuthMsg(authCtx, prov),
		)

		for {
			l.SetPrompt(getShellPRompt(authCtx, cd))
			var rawLine string
			rawLine, err = l.Readline()
			if errors.Is(err, readline.ErrInterrupt) {
				if len(rawLine) == 0 {
					break
				} else { //nolint:revive // TODO: investigate
					continue
				}
			} else if errors.Is(err, io.EOF) {
				break
			}

			line := strings.TrimSpace(rawLine)
			switch {
			case line == "help":
				usage(outErrFile)
			case line == "clear":
				readline.ClearScreen(l.Stdout()) //nolint:errcheck // TODO: investigate
			case line == "exit" || line == `\q` || line == "quit":
				goto exit
			case line == "":
			default:
				logging.GetLogger().Debugln("you said:", strconv.Quote(line))
				inlineCommentIdx := strings.Index(line, "--")
				if inlineCommentIdx > -1 {
					line = line[:inlineCommentIdx]
				}
				semiColonIdx := strings.Index(line, ";")
				if semiColonIdx > -1 {
					line = strings.TrimSpace(line[:semiColonIdx+1])
					subSemiColonIdx := strings.Index(line, ";")
					sb.WriteString(" " + line[:subSemiColonIdx+1])
					rawQuery := sb.String()
					queryToExecute, qErr := entryutil.PreprocessInline(runtimeCtx, rawQuery)
					if qErr != nil {
						io.WriteString(outErrFile, "\r\n"+qErr.Error()+"\r\n") //nolint:errcheck // TODO: investigate
					}
					l.WriteToHistory(rawQuery) //nolint:errcheck // TODO: investigate
					sessionRunnerInstance.RunCommand(queryToExecute)
					sb.Reset()
					sb.WriteString(line[subSemiColonIdx+1:])
				} else {
					sb.WriteString(" " + line)
				}
			}
		}
	exit:
		fmt.Fprintln(
			outErrFile,
			"goodbye",
		)
		fmt.Fprintf(outfile, "")
		fmt.Fprintf(outErrFile, "")
		outfile, _ = writer.GetOutputWriter(writer.StdOutStr)
		outErrFile, _ = writer.GetOutputWriter(writer.StdErrStr)
		l.Config.Stdout = outfile
		l.Config.Stderr = outErrFile
	},
}

type sessionRunner interface {
	RunCommand(command string)
}

func newSessionRunner(
	handlerCtx handler.HandlerContext,
	outfile io.Writer,
	outErrFile io.Writer,
) (sessionRunner, error) {
	var err error
	if outfile == nil {
		outfile, err = getOutputFile(handlerCtx.GetRuntimeContext().OutfilePath)
		if err != nil {
			return nil, err
		}
	}
	if outErrFile == nil {
		outErrFile, err = getOutputFile(writer.StdErrStr)
		if err != nil {
			return nil, err
		}
	}
	handlerCtx.SetOutfile(outfile)
	handlerCtx.SetOutErrFile(outErrFile)
	stackqlDriver, driverErr := driver.NewStackQLDriver(handlerCtx)
	if driverErr != nil {
		return nil, driverErr
	}
	return &sessionRunnerImpl{
		handlerCtx: handlerCtx,
		outfile:    outfile,
		outErrFile: outErrFile,
		drv:        stackqlDriver,
	}, nil
}

type sessionRunnerImpl struct {
	handlerCtx handler.HandlerContext
	outfile    io.Writer
	outErrFile io.Writer
	drv        driver.StackQLDriver
}

func (cr *sessionRunnerImpl) RunCommand(
	query string,
) {
	defer iqlerror.HandlePanic(cr.handlerCtx.GetOutErrFile())
	cloneCtx := cr.handlerCtx.Clone()
	cloneCtx.SetRawQuery(query)
	if cloneCtx.GetRuntimeContext().DryRunFlag {
		cr.drv.ProcessDryRun(query)
		return
	}
	cr.drv.ProcessQuery(query)
}
