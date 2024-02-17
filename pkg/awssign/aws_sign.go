package awssign

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
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
) (Transport, error) {
	var creds *credentials.Credentials

	if token == "" {
		creds = credentials.NewStaticCredentials(id, secret, token)
	} else {
		defaultAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
		defaultSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if defaultAccessKeyID == "" || defaultSecretAccessKey == "" {
			return nil, fmt.Errorf("AWS_SESSION_TOKEN is set, but AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must also be set")
		}
		creds = credentials.NewEnvCredentials()
	}

	signer := v4.NewSigner(creds, options...)
	return &standardAwsSignTransport{
		underlyingTransport: underlyingTransport,
		signer:              signer,
	}, nil
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
