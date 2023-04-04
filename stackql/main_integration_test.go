package main

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testobjects"

	"net/url"
	"os"
	"strings"
	"testing"
)

func TestSimpleSelectGoogleComputeInstance(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(
		testobjects.GetGoogleProviderString(), "text",
		"TestSimpleSelectGoogleComputeInstance")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(
		nil, nil, "GET", url, "compute.googleapis.com",
		testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	exp := testhttpapi.NewExpectationStore(1)
	exp.Put("compute.googleapis.com"+path, ex)
	testhttpapi.StartServer(t, exp)
	provider.DummyAuth = true
	args := []string{
		"--loglevel=warn",
		fmt.Sprintf("--auth=%s", runtimeCtx.AuthRaw),
		fmt.Sprintf("--registry=%s", runtimeCtx.RegistryRaw),
		fmt.Sprintf("--sqlBackend=%s", runtimeCtx.SQLBackendCfgRaw),
		"-i=stdin",
		"exec",
		testobjects.SimpleSelectGoogleComputeInstance,
	}
	t.Logf("simple select integration: about to invoke main() with args:\n\t%s", strings.Join(args, ",\n\t"))
	os.Args = args
	err = execute()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	t.Logf("simple select integration test passed")
}

func TestK8STemplatedE2eSuccess(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(
		testobjects.GetGoogleProviderString(), "text", "TestK8STemplatedE2eSuccess")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	k8sthwRenderedFile, err := util.GetFilePathFromRepositoryRoot(testobjects.ExpectedK8STheHardWayRenderedFile)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	args := []string{
		"--loglevel=warn",
		fmt.Sprintf("--auth=%s", runtimeCtx.AuthRaw),
		fmt.Sprintf("--registry=%s", runtimeCtx.RegistryRaw),
		fmt.Sprintf("--sqlBackend=%s", runtimeCtx.SQLBackendCfgRaw),
		fmt.Sprintf("--provider=%s", runtimeCtx.ProviderStr),
		fmt.Sprintf("-i=%s", k8sthwRenderedFile),
		"exec",
	}
	t.Logf("k8s e2e integration: about to invoke main() with args:\n\t%s", strings.Join(args, ",\n\t"))

	stackqltestutil.SetupK8sTheHardWayE2eSuccess(t)

	os.Args = args

	stackqltestutil.RunStdOutTestAgainstFiles(t, execStuff, []string{testobjects.ExpectedK8STheHardWayAsyncFile})
}

func TestInsertAwaitExecSuccess(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(
		testobjects.GetGoogleProviderString(), "text",
		"TestInsertAwaitExecSuccess")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	args := []string{
		"--loglevel=warn",
		fmt.Sprintf("--auth=%s", runtimeCtx.AuthRaw),
		fmt.Sprintf("--registry=%s", runtimeCtx.RegistryRaw),
		fmt.Sprintf("--sqlBackend=%s", runtimeCtx.SQLBackendCfgRaw),
		"-i=stdin",
		"exec",
		testobjects.SimpleInsertExecComputeNetwork,
	}
	t.Logf("k8s e2e integration: about to invoke main() with args:\n\t%s", strings.Join(args, ",\n\t"))

	stackqltestutil.SetupSimpleInsertGoogleComputeNetworks(t)

	os.Args = args

	stackqltestutil.RunStdOutTestAgainstFiles(t, execStuff, []string{testobjects.ExpectedComputeNetworkInsertAsyncFile})
}

func TestDeleteAwaitSuccess(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(
		testobjects.GetGoogleProviderString(), "text", "TestDeleteAwaitSuccess")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	args := []string{
		"--loglevel=warn",
		fmt.Sprintf("--auth=%s", runtimeCtx.AuthRaw),
		fmt.Sprintf("--registry=%s", runtimeCtx.RegistryRaw),
		fmt.Sprintf("--sqlBackend=%s", runtimeCtx.SQLBackendCfgRaw),
		"-i=stdin",
		"exec",
		testobjects.SimpleDeleteComputeNetwork,
	}
	t.Logf("k8s e2e integration: about to invoke main() with args:\n\t%s", strings.Join(args, ",\n\t"))

	stackqltestutil.SetupSimpleDeleteGoogleComputeNetworks(t)

	os.Args = args

	stackqltestutil.RunStdOutTestAgainstFiles(t, execStuff, []string{testobjects.ExpectedComputeNetworkDeleteAsyncFile})
}

func TestDeleteAwaitExecSuccess(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(
		testobjects.GetGoogleProviderString(), "text",
		"TestDeleteAwaitExecSuccess")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	args := []string{
		"--loglevel=warn",
		fmt.Sprintf("--auth=%s", runtimeCtx.AuthRaw),
		fmt.Sprintf("--registry=%s", runtimeCtx.RegistryRaw),
		fmt.Sprintf("--sqlBackend=%s", runtimeCtx.SQLBackendCfgRaw),
		"-i=stdin",
		"exec",
		testobjects.SimpleDeleteExecComputeNetwork,
	}
	t.Logf("k8s e2e integration: about to invoke main() with args:\n\t%s", strings.Join(args, ",\n\t"))

	stackqltestutil.SetupSimpleDeleteGoogleComputeNetworks(t)

	os.Args = args

	stackqltestutil.RunStdOutTestAgainstFiles(t, execStuff, []string{testobjects.ExpectedComputeNetworkDeleteAsyncFile})
}

func execStuff(t *testing.T) {
	err := execute()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
}
