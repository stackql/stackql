package psqlwire

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/logging"
	"gopkg.in/yaml.v2"

	"github.com/stackql/psql-wire/pkg/sqlbackend"

	wire "github.com/stackql/psql-wire"
)

type IWireServer interface {
	Serve() error
}

type SimpleWireServer struct {
	logger *logrus.Logger
	server *wire.Server
	rtCtx  dto.RuntimeCtx
	tlsCfg dto.PgTLSCfg
}

//nolint:gocognit,nestif,nolintlint
func MakeWireServer(sbe sqlbackend.SQLBackendFactory, cfg dto.RuntimeCtx) (IWireServer, error) {
	logger := logging.GetLogger()

	var tlsCfg dto.PgTLSCfg
	var server *wire.Server

	var err error
	pgSrvMiscConfig := make(map[string]interface{})
	if cfg.PGSrvRawSrvCfg != "" {
		err = yaml.Unmarshal([]byte(cfg.PGSrvRawSrvCfg), &pgSrvMiscConfig)
		if err != nil {
			return nil, err
		}
	}
	if cfg.PGSrvRawTLSCfg != "" {
		err = json.Unmarshal([]byte(cfg.PGSrvRawTLSCfg), &tlsCfg)
		if err != nil {
			return nil, err
		}
		var cert tls.Certificate
		cert, err = tlsCfg.GetKeyPair()
		if err != nil {
			return nil, err
		}
		certs := []tls.Certificate{cert}
		server, err = wire.NewServer(
			wire.SQLBackendFactory(sbe),
			wire.Certificates(certs),
			wire.Logger(logging.GetLogger()),
		)
		var cp *x509.CertPool
		if len(tlsCfg.ClientCAs) > 0 {
			cp = x509.NewCertPool()
			for _, pemStr := range tlsCfg.ClientCAs {
				var b []byte
				b, err = base64.StdEncoding.DecodeString(pemStr)
				if err != nil {
					return nil, fmt.Errorf("failed to decode Client CA PEM: %w, with string '%s'", err, pemStr)
				}
				ok := cp.AppendCertsFromPEM(b)
				if !ok {
					logger.Error("failed loading Client CA")
				}
			}
			server.ClientCAs = cp
			// The strongest assertion a server can provide, per https://smallstep.com/hello-mtls/doc/server/go
			server.ClientAuth = tls.RequireAndVerifyClientCert
		}
		if err != nil {
			return nil, err
		}
	} else {
		server, err = wire.NewServer(
			wire.SQLBackendFactory(sbe),
			wire.SundryConfig(pgSrvMiscConfig),
			wire.IsCaptureDebug(cfg.PGSrvIsDebugNoticesEnabled),
		)
		if err != nil {
			return nil, err
		}
	}
	return &SimpleWireServer{
		logger: logger,
		rtCtx:  cfg,
		server: server,
		tlsCfg: tlsCfg,
	}, nil
}

func (sws *SimpleWireServer) Serve() error {
	sws.logger.Info(
		fmt.Sprintf("PostgreSQL server is up and running at [%s:%d]",
			sws.rtCtx.PGSrvAddress,
			sws.rtCtx.PGSrvPort),
	)
	return sws.server.ListenAndServe(fmt.Sprintf("%s:%d", sws.rtCtx.PGSrvAddress, sws.rtCtx.PGSrvPort))
}
