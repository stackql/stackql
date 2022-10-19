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
	"io"
	"runtime"
	"strconv"
	"strings"

	"github.com/stackql/stackql/internal/stackql/color"
	"github.com/stackql/stackql/internal/stackql/config"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/writer"

	"github.com/spf13/cobra"

	"github.com/chzyer/readline"
)

const (
	shellLongStr string = `stackql Command Shell %s
Copyright (c) 2021, stackql studios. All rights reserved.
Welcome to the interactive shell for running stackql commands.
---`

	// Auth messages
	interactiveSuccessMsgTmpl string = `Authenticated interactively to google as %s, to change the authenticated user, use AUTH REVOKE followed by AUTH LOGIN, see https://docs.stackql.io/language-spec/auth`

	notAuthenticatedMsg string = `Not authenticated, to authenticate to a provider use the AUTH LOGIN command, see https://docs.stackql.io/language-spec/auth`

	saFileErrorMsgTmpl string = `Not authenticated, credentials referenced in %s do not exist, authenticate interactively using AUTH LOGIN, for more information see https://docs.stackql.io/language-spec/auth`

	saSuccessMsgTmpl string = `Authenticated using credentials set using the flag %s of type = '%s', for more information see https://docs.stackql.io/language-spec/auth`

	credentialProvidedMsgTmpl string = `Credentials provided using the the flag %s of type = '%s', for more information see https://docs.stackql.io/language-spec/auth`
)

func getShellIntroLong() string {
	return fmt.Sprintf(shellLongStr, SemVersion)
}

func usage(w io.Writer) {
	io.WriteString(w, getShellIntroLong()+"\r\n")
}

func getShellPRompt(authCtx *dto.AuthCtx, cd *color.ColorDriver) string {
	if runtime.GOOS == "windows" {
		return "stackql  >>"
	}
	if authCtx != nil && authCtx.Active {
		switch authCtx.Type {
		case dto.AuthInteractiveStr:
			return cd.ShellColorPrint("stackql* >>")
		case dto.AuthServiceAccountStr:
			return cd.ShellColorPrint("stackql**>>")
		}
	}
	return cd.ShellColorPrint("stackql  >>")
}

func getIntroAuthMsg(authCtx *dto.AuthCtx, prov provider.IProvider) string {
	if authCtx != nil {
		if authCtx.Active {
			switch authCtx.Type {
			case dto.AuthInteractiveStr:
				return fmt.Sprintf(interactiveSuccessMsgTmpl, authCtx.ID)
			case dto.AuthServiceAccountStr, dto.AuthApiKeyStr:
				return fmt.Sprintf(saSuccessMsgTmpl, authCtx.GetCredentialsSourceDescriptorString(), authCtx.Type)
			}
		} else if prov != nil {
			if err := prov.CheckCredentialFile(authCtx); authCtx.HasKey() && err != nil {
				return fmt.Sprintf(saFileErrorMsgTmpl, authCtx.GetCredentialsSourceDescriptorString())
			}
		}
		switch authCtx.Type {
		case dto.AuthServiceAccountStr, dto.AuthApiKeyStr:
			return fmt.Sprintf(credentialProvidedMsgTmpl, authCtx.GetCredentialsSourceDescriptorString(), authCtx.Type)
		}
	}
	return "" // notAuthenticatedMsg
}

func colorIsNull(runtimeCtx dto.RuntimeCtx) bool {
	return runtimeCtx.ColorScheme == dto.NullColorScheme || runtime.GOOS == "windows"
}

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Interactive shell for running stackql commands",
	Long:  getShellIntroLong(),
	Run: func(command *cobra.Command, args []string) {

		cd := color.NewColorDriver(runtimeCtx)

		outfile, _ := writer.GetDecoratedOutputWriter(runtimeCtx.OutfilePath, cd)

		outErrFile, _ := writer.GetDecoratedOutputWriter(writer.StdErrStr, cd, cd.GetErrorColorAttributes(runtimeCtx)...)

		var sb strings.Builder
		fmt.Fprintln(outfile, "") // necesary hack to get 'square' coloring
		fmt.Fprintln(outfile, getShellIntroLong())

		inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
		iqlerror.PrintErrorAndExitOneIfError(err)

		handlerCtx, handlerrErr := handler.GetHandlerCtx("", runtimeCtx, queryCache, inputBundle)
		if handlerrErr != nil {
			fmt.Fprintln(outErrFile, fmt.Sprintf("Error setting up handler context for provider '%s': \"%s\"", runtimeCtx.ProviderStr, handlerrErr))
		}
		var authCtx *dto.AuthCtx
		var prov provider.IProvider
		var pErr, authErr error
		if handlerCtx.RuntimeContext.ProviderStr != "" {
			prov, pErr = handlerCtx.GetProvider(handlerCtx.RuntimeContext.ProviderStr)
			authCtx, authErr = handlerCtx.GetAuthContext(prov.GetProviderString())
			if authErr != nil {
				fmt.Fprintln(outErrFile, fmt.Sprintf("Error setting up AUTH for provider '%s'", handlerCtx.RuntimeContext.ProviderStr))
			}
			if pErr == nil {
				prov.ShowAuth(authCtx)
			} else {
				fmt.Fprintln(outErrFile, fmt.Sprintf("Error setting up API for provider '%s'", handlerCtx.RuntimeContext.ProviderStr))
			}
		}

		var readlineCfg *readline.Config

		if colorIsNull(handlerCtx.RuntimeContext) {
			readlineCfg = &readline.Config{
				Prompt:               getShellPRompt(authCtx, cd),
				InterruptPrompt:      "^C",
				EOFPrompt:            "exit",
				HistoryFile:          config.GetReadlineFilePath(handlerCtx.RuntimeContext),
				HistorySearchFold:    true,
				HistoryExternalWrite: true,
			}
		} else {
			readlineCfg = &readline.Config{
				Stderr:               outErrFile,
				Stdout:               outfile,
				Prompt:               getShellPRompt(authCtx, cd),
				InterruptPrompt:      "^C",
				EOFPrompt:            "exit",
				HistoryFile:          config.GetReadlineFilePath(handlerCtx.RuntimeContext),
				HistorySearchFold:    true,
				HistoryExternalWrite: true,
			}
		}

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
			rawLine, err := l.Readline()
			if err == readline.ErrInterrupt {
				if len(rawLine) == 0 {
					break
				} else {
					continue
				}
			} else if err == io.EOF {
				break
			}

			line := strings.TrimSpace(rawLine)
			switch {
			case line == "help":
				usage(outErrFile)
			case line == "clear":
				readline.ClearScreen(l.Stdout())
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
					semiColonIdx := strings.Index(line, ";")
					sb.WriteString(" " + line[:semiColonIdx+1])
					rawQuery := sb.String()
					queryToExecute, err := entryutil.PreprocessInline(runtimeCtx, rawQuery)
					if err != nil {
						io.WriteString(outErrFile, "\r\n"+err.Error()+"\r\n")
					}
					handlerCtx.RawQuery = queryToExecute
					l.WriteToHistory(rawQuery)
					RunCommand(&handlerCtx, outfile, outErrFile)
					sb.Reset()
					sb.WriteString(line[semiColonIdx+1:])
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
		if !colorIsNull(runtimeCtx) {
			cd.ResetColorScheme()
		}
		fmt.Fprintf(outfile, "")
		fmt.Fprintf(outErrFile, "")
		outfile, _ = writer.GetOutputWriter(writer.StdOutStr)
		outErrFile, _ = writer.GetOutputWriter(writer.StdErrStr)
		l.Config.Stdout = outfile
		l.Config.Stderr = outErrFile
	},
}
