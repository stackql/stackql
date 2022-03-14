package psqlwire

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"

	"github.com/jeroenrinzema/psql-wire/pkg/sqlbackend"

	wire "github.com/jeroenrinzema/psql-wire"
	"go.uber.org/zap"
)

type IWireServer interface {
	Serve() error
}

type SimpleWireServer struct {
	logger *zap.Logger
	server *wire.Server
	rtCtx  dto.RuntimeCtx
	tlsCfg dto.PgTLSCfg
}

func MakeWireServer(sbe sqlbackend.ISQLBackend, cfg dto.RuntimeCtx) (IWireServer, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	var tlsCfg dto.PgTLSCfg
	var server *wire.Server

	if cfg.PGSrvRawTLSCfg != "" {
		err = json.Unmarshal([]byte(cfg.PGSrvRawTLSCfg), &tlsCfg)
		if err != nil {
			return nil, err
		}
		cert, err := tlsCfg.GetKeyPair()
		if err != nil {
			return nil, err
		}
		certs := []tls.Certificate{cert}
		server, err = wire.NewServer(
			wire.SQLBackend(sbe),
			wire.Certificates(certs),
			wire.Logger(logger),
		)
		var cp *x509.CertPool
		if len(tlsCfg.ClientCAs) > 0 {
			cp = x509.NewCertPool()
			for _, pemStr := range tlsCfg.ClientCAs {
				b, err := base64.RawStdEncoding.DecodeString(pemStr)
				if err != nil {
					return nil, err
				}
				ok := cp.AppendCertsFromPEM([]byte(b))
				if !ok {
					logger.Error("failed loading Client CA")
				}
			}
			server.ClientCAs = cp
			server.ClientAuth = tls.RequireAndVerifyClientCert
		}
		if err != nil {
			return nil, err
		}
	} else {
		server, err = wire.NewServer(wire.SQLBackend(sbe))
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
	sws.logger.Info(fmt.Sprintf("PostgreSQL server is up and running at [%s:%d]", sws.rtCtx.PGSrvAddress, sws.rtCtx.PGSrvPort))
	return sws.server.ListenAndServe(fmt.Sprintf("%s:%d", sws.rtCtx.PGSrvAddress, sws.rtCtx.PGSrvPort))
}

func handle(ctx context.Context, query string, writer wire.DataWriter) error {
	fmt.Println(query)
	// if
	return writer.Complete("OK")
}
