package driver_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stretchr/testify/assert"

	"github.com/stackql/stackql/internal/stackql/handler"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

func getRuntimeCtx(providerStr string, outputFmtStr string, testName string) (*dto.RuntimeCtx, error) {
	saKeyPath, err := util.GetFilePathFromRepositoryRoot("internal/stackql/driver/testdata/dummy_credentials/dummy-sa-key.json")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", saKeyPath, err)
	}
	oktaSaKeyPath, err := util.GetFilePathFromRepositoryRoot("internal/stackql/driver/testdata/dummy_credentials/okta-api-key.txt")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", oktaSaKeyPath, err)
	}
	appRoot, err := util.GetFilePathFromRepositoryRoot("internal/stackql/driver/testdata/.stackql")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", appRoot, err)
	}
	dbInitFilePath, err := util.GetFilePathFromRepositoryRoot("test/db/sqlite/setup.sql")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", dbInitFilePath, err)
	}
	registryRoot, err := util.GetFilePathFromRepositoryRoot("internal/stackql/driver/testdata/registry")
	if err != nil {
		return nil, fmt.Errorf("test failed on %s: %v", registryRoot, err)
	}
	return &dto.RuntimeCtx{
		Delimiter:                 ",",
		ProviderStr:               providerStr,
		LogLevelStr:               "warn",
		ApplicationFilesRootPath:  appRoot,
		AuthRaw:                   fmt.Sprintf(`{ "google": { "credentialsfilepath": "%s" }, "okta": { "credentialsfilepath": "%s", "type": "api_key" } }`, saKeyPath, oktaSaKeyPath),
		RegistryRaw:               fmt.Sprintf(`{ "url": "file://%s", "localDocRoot": "%s", "useEmbedded": false, "verifyConfig": { "nopVerify": true } }`, registryRoot, registryRoot),
		OutputFormat:              outputFmtStr,
		SQLBackendCfgRaw:          fmt.Sprintf(`{ "dbInitFilepath": "%s", "dsn": "file:%s?mode=memory&cache=shared" }`, dbInitFilePath, testName),
		ExecutionConcurrencyLimit: 1,
		VarList:                   []string{"test_var=test_value"},
		IndirectDepthMax:          5, //nolint:mnd // test config value
	}, nil
}

//nolint:lll // legacy test
func TestDefaultedHttpClientIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}
	t.Setenv("AWS_SECRET_ACCESS_KEY", "some-junk")
	t.Setenv("AWS_ACCESS_KEY_ID", "some-other-junk")
	runtimeCtx, err := getRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectGoogleComputeInstanceDriver")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	runtimeCtx.AllowInsecure = true

	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	expectedHost := "my-test-bucket.s3-ap-southeast-2.amazonaws.com"

	reponseCounter := 0

	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		io.Copy(io.Discard, r.Body)
		assert.Equal(t, r.Host, expectedHost, "expected host does not match actual host")
		w.WriteHeader(http.StatusOK)
		reponseCounter++
	}))
	t.Cleanup(tlsServer.Close)

	baseTransport := tlsServer.Client().Transport.(*http.Transport)
	dummyClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   baseTransport.TLSClientConfig,
			DisableKeepAlives: true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("tcp", tlsServer.Listener.Addr().String())
			},
		},
	}

	testingQuery := `SELECT * FROM aws.s3.bucket_acls WHERE Bucket = 'my-test-bucket' AND created_date = '2024-01-01T00:00:00Z' AND region = 'ap-southeast-2';`

	handlerCtx, err := handler.NewHandlerCtx(
		testingQuery,
		*runtimeCtx,
		lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)),
		inputBundle,
		"v0.1.0",
	)

	handlerCtx.SetDefaultHTTPClient(dummyClient)

	dr, _ := NewStackQLDriver(handlerCtx)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	dr.ProcessQuery(handlerCtx.GetRawQuery())

	if reponseCounter != 1 {
		t.Fatalf("Test failed: expected 1 request to test server, got %d", reponseCounter)
	}

	t.Logf("simple select driver integration test passed")
}
