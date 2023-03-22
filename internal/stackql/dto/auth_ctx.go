package dto

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type AuthCtx struct {
	Scopes                  []string       `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	SQLCfg                  *SQLBackendCfg `json:"sqlDataSource" yaml:"sqlDataSource"`
	Type                    string         `json:"type" yaml:"type"`
	ValuePrefix             string         `json:"valuePrefix" yaml:"valuePrefix"`
	ID                      string         `json:"-" yaml:"-"`
	KeyID                   string         `json:"keyID" yaml:"keyID"`
	KeyIDEnvVar             string         `json:"keyIDenvvar" yaml:"keyIDenvvar"`
	KeyFilePath             string         `json:"credentialsfilepath" yaml:"credentialsfilepath"`
	KeyEnvVar               string         `json:"credentialsenvvar" yaml:"credentialsenvvar"`
	APIKeyStr               string         `json:"api_key" yaml:"api_key"`
	APISecretStr            string         `json:"api_secret" yaml:"api_secret"`
	Username                string         `json:"username" yaml:"username"`
	Password                string         `json:"password" yaml:"password"`
	EncodedBasicCredentials string         `json:"-" yaml:"-"`
	Active                  bool           `json:"-" yaml:"-"`
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
		Scopes:                  scopesCopy,
		Type:                    ac.Type,
		ValuePrefix:             ac.ValuePrefix,
		ID:                      ac.ID,
		KeyID:                   ac.KeyID,
		KeyIDEnvVar:             ac.KeyIDEnvVar,
		KeyFilePath:             ac.KeyFilePath,
		KeyEnvVar:               ac.KeyEnvVar,
		Active:                  ac.Active,
		Username:                ac.Username,
		Password:                ac.Password,
		APIKeyStr:               ac.APIKeyStr,
		APISecretStr:            ac.APISecretStr,
		EncodedBasicCredentials: ac.EncodedBasicCredentials,
	}
	return rv
}

func (ac *AuthCtx) GetInlineBasicCredentials() string {
	if ac.Username != "" && ac.Password != "" {
		plaintext := fmt.Sprintf("%s:%s", ac.Username, ac.Password)
		encoded := base64.StdEncoding.EncodeToString([]byte(plaintext))
		return encoded
	}
	if ac.APIKeyStr != "" && ac.APISecretStr != "" {
		plaintext := fmt.Sprintf("%s:%s", ac.APIKeyStr, ac.APISecretStr)
		encoded := base64.StdEncoding.EncodeToString([]byte(plaintext))
		return encoded
	}
	return ""
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
	if credentialFile != "" {
		return ioutil.ReadFile(credentialFile)
	}
	if ac.GetInlineBasicCredentials() != "" {
		return []byte(ac.GetInlineBasicCredentials()), nil
	}
	if ac.EncodedBasicCredentials != "" {
		return []byte(ac.EncodedBasicCredentials), nil
	}
	return nil, fmt.Errorf("no credentials found")
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
