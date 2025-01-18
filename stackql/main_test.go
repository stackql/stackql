package main

import (
	"io"
	"os"
	"testing"

	"github.com/stackql/any-sdk/pkg/logging"
)

func TestMain(m *testing.M) {
	logging.GetLogger().SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestRunSimple(t *testing.T) {
	os.Args = []string{os.Args[0], "--help"}
	main()
	t.Logf("completed")
}

func TestExitCodeZero(t *testing.T) {
	os.Args = []string{os.Args[0], "--help"}
	err := execute()
	if err == nil {
		t.Logf("Exit status 0 on legitimate command as expected")
		return
	}
	t.Fatalf("process ran with err %v, want exit status 0", err)
}

func TestExitCodeOne(t *testing.T) {
	os.Args = []string{os.Args[0], "exc"}
	err := execute()
	if err != nil {
		t.Logf("Exit status 1 on improper command as expected")
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
