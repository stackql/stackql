package dto

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type AuthContexts map[string]*AuthCtx

func (as AuthContexts) Clone() AuthContexts {
	rv := make(AuthContexts)
	for k, v := range as {
		rv[k] = v.Clone()
	}
	return rv
}

type AuthCtx struct {
	Scopes                  []string       `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	SQLCfg                  *SQLBackendCfg `json:"sqlDataSource" yaml:"sqlDataSource"`
	Type                    string         `json:"type" yaml:"type"`
	ValuePrefix             string         `json:"valuePrefix" yaml:"valuePrefix"`
	ID                      string         `json:"-" yaml:"-"`
	KeyID                   string         `json:"keyID" yaml:"keyID"`
	KeyIDEnvVar             string         `json:"keyIDenvvar" yaml:"keyIDenvvar"`
	KeyFilePath             string         `json:"credentialsfilepath" yaml:"credentialsfilepath"`
	KeyFilePathEnvVar       string         `json:"credentialsfilepathenvvar" yaml:"credentialsfilepathenvvar"`
	KeyEnvVar               string         `json:"credentialsenvvar" yaml:"credentialsenvvar"`
	APIKeyStr               string         `json:"api_key" yaml:"api_key"`
	APISecretStr            string         `json:"api_secret" yaml:"api_secret"`
	Username                string         `json:"username" yaml:"username"`
	Password                string         `json:"password" yaml:"password"`
	EnvVarAPIKeyStr         string         `json:"api_key_var" yaml:"api_key_var"`
	EnvVarAPISecretStr      string         `json:"api_secret_var" yaml:"api_secret_var"`
	EnvVarUsername          string         `json:"username_var" yaml:"username_var"`
	EnvVarPassword          string         `json:"password_var" yaml:"password_var"`
	EncodedBasicCredentials string         `json:"-" yaml:"-"`
	Successor               *AuthCtx       `json:"successor" yaml:"successor"`
	Subject                 string         `json:"sub" yaml:"sub"`
	Active                  bool           `json:"-" yaml:"-"`
	Location                string         `json:"location" yaml:"location"`
	Name                    string         `json:"name" yaml:"name"`
	TokenURL                string         `json:"token_url" yaml:"token_url"`
	GrantType               string         `json:"grant_type" yaml:"grant_type"`
	ClientID                string         `json:"client_id" yaml:"client_id"`
	ClientSecret            string         `json:"client_secret" yaml:"client_secret"`
	ClientIDEnvVar          string         `json:"client_id_env_var" yaml:"client_id_env_var"`
	ClientSecretEnvVar      string         `json:"client_secret_env_var" yaml:"client_secret_env_var"`
	Values                  url.Values     `json:"values" yaml:"values"`
	AuthStyle               int            `json:"auth_style" yaml:"auth_style"`
	AccountID               string         `json:"account_id" yaml:"account_id"`
	AccoountIDEnvVar        string         `json:"account_id_env_var" yaml:"account_id_var"`
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
		KeyFilePathEnvVar:       ac.KeyFilePathEnvVar,
		KeyEnvVar:               ac.KeyEnvVar,
		Active:                  ac.Active,
		Username:                ac.Username,
		Password:                ac.Password,
		APIKeyStr:               ac.APIKeyStr,
		APISecretStr:            ac.APISecretStr,
		EnvVarAPIKeyStr:         ac.EnvVarAPIKeyStr,
		EnvVarAPISecretStr:      ac.EnvVarAPISecretStr,
		EnvVarUsername:          ac.EnvVarUsername,
		EnvVarPassword:          ac.EnvVarPassword,
		Successor:               ac.Successor,
		EncodedBasicCredentials: ac.EncodedBasicCredentials,
		Location:                ac.Location,
		Name:                    ac.Name,
		Subject:                 ac.Subject,
		TokenURL:                ac.TokenURL,
		GrantType:               ac.GrantType,
		ClientID:                ac.ClientID,
		ClientSecret:            ac.ClientSecret,
		ClientIDEnvVar:          ac.ClientIDEnvVar,
		ClientSecretEnvVar:      ac.ClientSecretEnvVar,
		Values:                  ac.Values,
		AuthStyle:               ac.AuthStyle,
		AccountID:               ac.AccountID,
		AccoountIDEnvVar:        ac.AccoountIDEnvVar,
	}
	return rv
}

func (ac *AuthCtx) GetValues() url.Values {
	if ac.Values == nil {
		return url.Values{}
	}
	return ac.Values
}

func (ac *AuthCtx) GetSuccessor() (*AuthCtx, bool) {
	if ac.Successor != nil {
		return ac.Successor, true
	}
	return nil, false
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

func (ac *AuthCtx) getEnvVarBasicCredentials() string {
	if ac.EnvVarUsername != "" && ac.EnvVarPassword != "" {
		userName := os.Getenv(ac.EnvVarUsername)
		passWord := os.Getenv(ac.EnvVarPassword)
		plaintext := fmt.Sprintf("%s:%s", userName, passWord)
		encoded := base64.StdEncoding.EncodeToString([]byte(plaintext))
		return encoded
	}
	if ac.EnvVarAPIKeyStr != "" && ac.EnvVarAPISecretStr != "" {
		userName := os.Getenv(ac.EnvVarAPIKeyStr)
		passWord := os.Getenv(ac.EnvVarAPISecretStr)
		plaintext := fmt.Sprintf("%s:%s", userName, passWord)
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

func (ac *AuthCtx) GetAwsSessionTokenString() (string, error) {
	token := os.Getenv("AWS_SESSION_TOKEN")
	return token, nil // Session token is optional, so an empty token isn't considered an error.
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
	if ac.KeyFilePath != "" || ac.KeyEnvVar != "" || ac.KeyFilePathEnvVar != "" {
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
	if ac.KeyFilePathEnvVar != "" {
		credentialFile := os.Getenv(ac.KeyFilePathEnvVar)
		return os.ReadFile(credentialFile)
	}
	credentialFile := ac.KeyFilePath
	if credentialFile != "" {
		return os.ReadFile(credentialFile)
	}
	if ac.getEnvVarBasicCredentials() != "" {
		return []byte(ac.getEnvVarBasicCredentials()), nil
	}
	if ac.GetInlineBasicCredentials() != "" {
		return []byte(ac.GetInlineBasicCredentials()), nil
	}
	if ac.EncodedBasicCredentials != "" {
		return []byte(ac.EncodedBasicCredentials), nil
	}
	return nil, fmt.Errorf("no credentials found")
}

func (ac *AuthCtx) GetClientID() (string, error) {
	if ac.ClientIDEnvVar != "" {
		rv := os.Getenv(ac.ClientIDEnvVar)
		if rv == "" {
			return "", fmt.Errorf("client_id_env_var references empty string")
		}
		return rv, nil
	}
	if ac.ClientID == "" {
		return "", fmt.Errorf("client_id is empty")
	}
	return ac.ClientID, nil
}

func (ac *AuthCtx) GetClientSecret() (string, error) {
	if ac.ClientSecretEnvVar != "" {
		rv := os.Getenv(ac.ClientSecretEnvVar)
		if rv == "" {
			return "", fmt.Errorf("client_secret_env_var references empty string")
		}
		return rv, nil
	}
	if ac.ClientSecret == "" {
		return "", fmt.Errorf("client_secret is empty")
	}
	return ac.ClientSecret, nil
}

func (ac *AuthCtx) GetGrantType() string {
	return ac.GrantType
}

func (ac *AuthCtx) GetTokenURL() string {
	return ac.TokenURL
}

func (ac *AuthCtx) GetAuthStyle() int {
	return ac.AuthStyle
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
