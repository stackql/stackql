package iqlerror

import (
	"fmt"
	"io"
	"os"
)

func GetStatementNotSupportedError(stmtName string) error {
	return fmt.Errorf("statement type = '%s' not yet supported", stmtName)
}

func PrintErrorAndExitOneIfNil(subject interface{}, msg string) {
	if subject == nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintln(msg))
		os.Exit(1)
	}
}

func PrintErrorAndExitOneWithMessage(msg string) {
	fmt.Fprintln(os.Stderr, fmt.Sprintln(msg))
	os.Exit(1)
}

func PrintErrorAndExitOneIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintln(err.Error()))
		os.Exit(1)
	}
}

func HandlePanic(outFile io.Writer) {
	if r := recover(); r != nil {
		msg := fmt.Sprintln("Error: Recovered in HandlePanic():", r)
		if outFile != nil {
			outFile.Write([]byte(msg))
		} else {
			fmt.Print(msg)
		}
	}
}
