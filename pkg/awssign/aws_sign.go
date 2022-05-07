package awssign

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
)

const (
	locationHeader string = "header"
	locationQuery  string = "query"
	authTypeBasic  string = "BASIC"
	authTypeBearer string = "Bearer"
	authTypeSSWS   string = "SSWS"
)

type AwsSignTransport struct {
	underlyingTransport http.RoundTripper
	signer              *v4.Signer
}

func NewAwsSignTransport(underlyingTransport http.RoundTripper, id, secret, token string, options ...func(*v4.Signer)) *AwsSignTransport {
	creds := credentials.NewStaticCredentials(id, secret, token)
	//creds := credentials.NewEnvCredentials()
	signer := v4.NewSigner(creds, options...)
	return &AwsSignTransport{
		underlyingTransport: underlyingTransport,
		signer:              signer,
	}
}

func (t *AwsSignTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		rs = bytes.NewReader(body)
	}
	_, err := t.signer.Sign(
		req,
		rs,
		svcStr,
		rgnStr,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}
	return t.underlyingTransport.RoundTrip(req)
}
