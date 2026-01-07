package handler_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stretchr/testify/assert"
)

func TestAwsS3BucketAclsGet(t *testing.T) {

	vr := "v0.1.0"
	pb, err := os.ReadFile("./testdata/registry/src/aws/" + vr + "/provider.yaml")
	if err != nil {
		t.Fatalf("Test failed: could not read provider doc, error: %v", err)
	}
	prov, provErr := anysdk.LoadProviderDocFromBytes(pb)
	if provErr != nil {
		t.Fatalf("Test failed: could not load provider doc, error: %v", provErr)
	}
	svc, err := anysdk.LoadProviderAndServiceFromPaths(
		"./testdata/registry/src/aws/"+vr+"/provider.yaml",
		"./testdata/registry/src/aws/"+vr+"/services/s3.yaml",
	)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	rsc, rscErr := svc.GetResource("bucket_acls")
	if rscErr != nil {
		t.Fatalf("Test failed: could not locate resource bucket_acls, error: %v", rscErr)
	}
	method, methodErr := rsc.FindMethod("get_bucket_acl")
	if methodErr != nil {
		t.Fatalf("Test failed: could not locate method get_bucket_acl, error: %v", methodErr)
	}

	assert.Equal(t, svc.GetName(), "s3")

	expectedHost := "my-test-bucket.s3-ap-southeast-2.amazonaws.com"

	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		io.Copy(io.Discard, r.Body)
		assert.Equal(t, r.Host, expectedHost, "expected host does not match actual host")
		w.WriteHeader(http.StatusOK)
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

	authCtx := dto.GetAuthCtx([]string{}, "./testdata/dummy_credentials/dummy-sa-key.json", "null_auth")

	configurator := anysdk.NewAnySdkClientConfigurator(
		dto.RuntimeCtx{
			AllowInsecure: true,
		},
		"aws",
		dummyClient,
	)
	httpPreparator := anysdk.NewHTTPPreparator(
		prov,
		svc,
		method,
		map[int]map[string]interface{}{
			0: {
				"Bucket":       "my-test-bucket",
				"created_date": "2024-01-01T00:00:00Z",
				"region":       "ap-southeast-2",
			},
		},
		nil,
		nil,
		logrus.StandardLogger(),
	)
	armoury, armouryErr := httpPreparator.BuildHTTPRequestCtx(anysdk.NewHTTPPreparatorConfig(false))
	if armouryErr != nil {
		t.Fatalf("Test failed: could not build HTTP preparator armoury, error: %v", armouryErr)
	}
	reqParams := armoury.GetRequestParams()
	if len(reqParams) < 1 {
		t.Fatalf("Test failed: no request parameters found")
	}

	for _, v := range reqParams {

		argList := v.GetArgList()

		response, apiErr := anysdk.CallFromSignature(
			configurator,
			dto.RuntimeCtx{
				AllowInsecure: true,
			},
			authCtx,
			authCtx.Type,
			false,
			nil,
			prov,
			anysdk.NewAnySdkOpStoreDesignation(method),
			argList, // TODO: abstract
		)
		if apiErr != nil {
			t.Fatalf("Test failed: API call error: %v", apiErr)
		}
		if response.IsErroneous() {
			t.Fatalf("Test failed: API call returned erroneous response")
		}
		t.Logf("Test passed: received response: %+v", response)
	}

}
