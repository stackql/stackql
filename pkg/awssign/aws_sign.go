package awssign

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/stackql/stackql/internal/stackql/logging"
)

var (
	_ Transport = &standardAwsSignTransport{}
)

type Transport interface {
	RoundTrip(req *http.Request) (*http.Response, error)
}

type standardAwsSignTransport struct {
	underlyingTransport http.RoundTripper
	signer              *v4.Signer
}

func NewAwsSignTransport(
	underlyingTransport http.RoundTripper,
	id, secret, token string,
	options ...func(*v4.Signer),
) Transport {
	creds := credentials.NewStaticCredentials(id, secret, token)
	// creds := credentials.NewEnvCredentials()
	signer := v4.NewSigner(creds, options...)
	return &standardAwsSignTransport{
		underlyingTransport: underlyingTransport,
		signer:              signer,
	}
}

func (t *standardAwsSignTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	svc := req.Context().Value("service")
	if svc == nil {
		return nil, fmt.Errorf("AWS service is nil")
	}
	rgn := req.Context().Value("region")
	if rgn == nil {
		return nil, fmt.Errorf("AWS region is nil")
	}
	svcStr, ok := svc.(string)
	if !ok {
		return nil, fmt.Errorf("unsupported type for AWS service: '%T'", svc)
	}
	rgnStr, ok := rgn.(string)
	if !ok {
		return nil, fmt.Errorf("unsupported type for AWS region: '%T'", rgn)
	}
	var rs io.ReadSeeker
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		rs = bytes.NewReader(body)
		req.Body = nil
	}
	header, err := t.signer.Sign(
		req,
		rs,
		svcStr,
		rgnStr,
		time.Now(),
	)
	logging.GetLogger().Infof("header = %v\n", header)
	if err != nil {
		return nil, err
	}

	return t.underlyingTransport.RoundTrip(req)
}
