package dto

import "crypto/tls"

type PgTLSCfg struct {
	KeyFilePath  string   `json:"keyFilePath" yaml:"keyFilePath"`
	CertFilePath string   `json:"certFilePath" yaml:"certFilePath"`
	KeyContents  string   `json:"keyContents" yaml:"keyContents"`
	CertContents string   `json:"certContents" yaml:"certContents"`
	ClientCAs    []string `json:"clientCAs" yaml:"clientCAs"`
}

func (pc PgTLSCfg) GetKeyPair() (tls.Certificate, error) {
	if len(pc.KeyContents) > 0 && len(pc.CertContents) > 0 {
		return tls.X509KeyPair([]byte(pc.CertContents), []byte(pc.KeyContents))
	}
	return tls.LoadX509KeyPair(pc.CertFilePath, pc.KeyFilePath)
}
