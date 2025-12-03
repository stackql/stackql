//nolint:govet,errorlint,errcheck,lll // test boilerplate
package stackqltestutil

import (
	"fmt"
	"io"
	"os"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/bundle"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/util"
)

func GetRuntimeCtx(providerStr string, outputFmtStr string, testName string) (*dto.RuntimeCtx, error) {
	saKeyPath, err := util.GetFilePathFromRepositoryRoot("test/assets/credentials/dummy/google/dummy-sa-key.json")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", saKeyPath, err)
	}
	oktaSaKeyPath, err := util.GetFilePathFromRepositoryRoot("test/assets/credentials/dummy/okta/api-key.txt")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", saKeyPath, err)
	}
	appRoot, err := util.GetFilePathFromRepositoryRoot("test/.stackql")
	if err != nil {
		return nil, fmt.Errorf("test failed: %v", err)
	}
	dbInitFilePath, err := util.GetFilePathFromRepositoryRoot("test/db/sqlite/setup.sql")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", dbInitFilePath, err)
	}
	registryRoot, err := util.GetForwardSlashFilePathFromRepositoryRoot("test/registry")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", dbInitFilePath, err)
	}
	return &dto.RuntimeCtx{
		Delimiter:                 ",",
		ProviderStr:               providerStr,
		LogLevelStr:               "warn",
		ApplicationFilesRootPath:  appRoot,
		AuthRaw:                   fmt.Sprintf(`{ "google": { "credentialsfilepath": "%s" }, "okta": { "credentialsfilepath": "%s", "type": "api_key" } }`, saKeyPath, oktaSaKeyPath),
		RegistryRaw:               fmt.Sprintf(`{ "url": "file://%s",  "useEmbedded": false, useEmbedded, "verifyConfig": { "nopVerify": true } }`, registryRoot),
		OutputFormat:              outputFmtStr,
		SQLBackendCfgRaw:          fmt.Sprintf(`{ "dbInitFilepath": "%s", "dsn": "file:%s?mode=memory&cache=shared" }`, dbInitFilePath, testName),
		ExecutionConcurrencyLimit: 1,
		VarList:                   []string{"test_var=test_value"},
		IndirectDepthMax:          5,
	}, nil
}

func getBytesFromLocalPath(path string) ([]byte, error) {
	fullPath, err := util.GetFilePathFromRepositoryRoot(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}

func BuildInputBundle(runtimeCtx dto.RuntimeCtx) (bundle.Bundle, error) {
	inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
	if err != nil {
		return nil, err
	}
	googleRootDiscoveryBytes, err := getBytesFromLocalPath("test/db/google._root_.json")
	if err != nil {
		return nil, err
	}
	googleComputeDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.compute.json")
	if err != nil {
		return nil, err
	}
	googleContainerDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.container.json")
	if err != nil {
		return nil, err
	}
	googleCloudResourceManagerDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.cloudresourcemanager.json")
	if err != nil {
		return nil, err
	}
	googleBQDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.bigquery.json")
	if err != nil {
		return nil, err
	}
	sqlEng := inputBundle.GetSQLEngine()
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://www.googleapis.com/discovery/v1/apis", googleRootDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://www.googleapis.com/discovery/v1/apis/compute/v1/rest", googleComputeDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://container.googleapis.com/$discovery/rest?version=v1", googleContainerDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://cloudresourcemanager.googleapis.com/$discovery/rest?version=v3", googleCloudResourceManagerDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://bigquery.googleapis.com/$discovery/rest?version=v2", googleBQDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	return inputBundle, nil
}

func BuildBenchInputBundle(runtimeCtx dto.RuntimeCtx) (bundle.Bundle, error) {
	inputBundle, err := entryutil.BuildInputBundle(runtimeCtx)
	if err != nil {
		return nil, err
	}
	googleRootDiscoveryBytes, err := getBytesFromLocalPath("test/db/google._root_.json")
	if err != nil {
		return nil, err
	}
	googleComputeDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.compute.json")
	if err != nil {
		return nil, err
	}
	googleContainerDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.container.json")
	if err != nil {
		return nil, err
	}
	googleCloudResourceManagerDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.cloudresourcemanager.json")
	if err != nil {
		return nil, err
	}
	googleBQDiscoveryBytes, err := getBytesFromLocalPath("test/db/google.bigquery.json")
	if err != nil {
		return nil, err
	}
	sqlEng := inputBundle.GetSQLEngine()
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://www.googleapis.com/discovery/v1/apis", googleRootDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://www.googleapis.com/discovery/v1/apis/compute/v1/rest", googleComputeDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://container.googleapis.com/$discovery/rest?version=v1", googleContainerDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://cloudresourcemanager.googleapis.com/$discovery/rest?version=v3", googleCloudResourceManagerDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	sqlEng.Exec(`INSERT INTO "__iql__.cache.key_val"(k, v) VALUES(?, ?)`, "https://bigquery.googleapis.com/$discovery/rest?version=v2", googleBQDiscoveryBytes)
	if err != nil {
		return nil, err
	}
	inputBundle = inputBundle.WithStdErr(io.Discard).WithStdOut(io.Discard)
	return inputBundle, nil
}
