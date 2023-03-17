package dto

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type AuthCtx struct {
	Scopes      []string       `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	SQLCfg      *SQLBackendCfg `json:"sqlDataSource" yaml:"sqlDataSource"`
	Type        string         `json:"type" yaml:"type"`
	ValuePrefix string         `json:"valuePrefix" yaml:"valuePrefix"`
	ID          string         `json:"-" yaml:"-"`
	KeyID       string         `json:"keyID" yaml:"keyID"`
	KeyIDEnvVar string         `json:"keyIDenvvar" yaml:"keyIDenvvar"`
	KeyFilePath string         `json:"credentialsfilepath" yaml:"credentialsfilepath"`
	KeyEnvVar   string         `json:"credentialsenvvar" yaml:"credentialsenvvar"`
	Active      bool           `json:"-" yaml:"-"`
}

func (ac *AuthCtx) GetSQLCfg() (SQLBackendCfg, bool) {
	var retVal SQLBackendCfg
	if ac.SQLCfg != nil {
		return *ac.SQLCfg, true
	}
	return retVal, false
}

func (ac *AuthCtx) Clone() *AuthCtx {
	var scopesCopy []string
	scopesCopy = append(scopesCopy, ac.Scopes...)
	rv := &AuthCtx{
		Scopes:      scopesCopy,
		Type:        ac.Type,
		ValuePrefix: ac.ValuePrefix,
		ID:          ac.ID,
		KeyID:       ac.KeyID,
		KeyIDEnvVar: ac.KeyIDEnvVar,
		KeyFilePath: ac.KeyFilePath,
		KeyEnvVar:   ac.KeyEnvVar,
		Active:      ac.Active,
	}
	return rv
}

func (ac *AuthCtx) HasKey() bool {
	if ac.KeyFilePath != "" || ac.KeyEnvVar != "" {
		return true
	}
	return false
}

func (ac *AuthCtx) GetKeyIDString() (string, error) {
	if ac.KeyIDEnvVar != "" {
		rv := os.Getenv(ac.KeyIDEnvVar)
		if rv == "" {
			return "", fmt.Errorf("keyIDenvvar references empty string")
		}
		return rv, nil
	}
	return ac.KeyID, nil
}

func (ac *AuthCtx) InferAuthType(authTypeRequested string) string {
	ft := strings.ToLower(authTypeRequested)
	switch ft {
	case AuthAPIKeyStr:
		return AuthAPIKeyStr
	case AuthServiceAccountStr:
		return AuthServiceAccountStr
	case AuthInteractiveStr:
		return AuthInteractiveStr
	}
	if ac.KeyFilePath != "" || ac.KeyEnvVar != "" {
		return AuthServiceAccountStr
	}
	return AuthInteractiveStr
}

func (ac *AuthCtx) GetCredentialsBytes() ([]byte, error) {
	if ac.KeyEnvVar != "" {
		rv := os.Getenv(ac.KeyEnvVar)
		if rv == "" {
			return nil, fmt.Errorf("credentialsenvvar references empty string")
		}
		return []byte(rv), nil
	}
	credentialFile := ac.KeyFilePath
	return ioutil.ReadFile(credentialFile)
}

func (ac *AuthCtx) GetCredentialsSourceDescriptorString() string {
	if ac.KeyEnvVar != "" {
		return fmt.Sprintf("credentialsenvvar:%s", ac.KeyEnvVar)
	}
	return fmt.Sprintf("credentialsfilepath:%s", ac.KeyFilePath)
}

func GetAuthCtx(scopes []string, keyFilePath string, keyFileType string) *AuthCtx {
	var authType string
	if keyFilePath == "" {
		authType = AuthInteractiveStr
	} else {
		authType = inferKeyFileType(keyFileType)
	}
	return &AuthCtx{
		Scopes:      scopes,
		Type:        authType,
		KeyFilePath: keyFilePath,
		Active:      false,
	}
}
