package dto

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
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
	omnisdk.AuthConfig `yaml:",inline"`
	SQLCfg             *SQLBackendCfg `json:"sqlDataSource" yaml:"sqlDataSource"`
	Successor          *AuthCtx       `json:"successor" yaml:"successor"`
}

func (ac *AuthCtx) GetSQLCfg() (SQLBackendCfg, bool) {
	var retVal SQLBackendCfg
	if ac.SQLCfg != nil {
		return *ac.SQLCfg, true
	}
	return retVal, false
}

func (ac *AuthCtx) Clone() *AuthCtx {
	if ac == nil {
		return nil
	}
	cloned := *ac
	cloned.Scopes = append([]string(nil), ac.Scopes...)
	cloned.Values = make(url.Values, len(ac.Values))
	for key, values := range ac.Values {
		cloned.Values[key] = append([]string(nil), values...)
	}
	cloned.Successor = ac.Successor.Clone()
	if ac.SQLCfg != nil {
		cfg := *ac.SQLCfg
		cloned.SQLCfg = &cfg
	}
	return &cloned
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

// GetAwsRoleArn resolves the ARN of the role to assume, preferring the
// environment variable indirection when supplied. The role ARN is mandatory
// for the aws_assume_role auth type.
func (ac *AuthCtx) GetAwsRoleArn() (string, error) {
	if ac.AwsRoleArnEnvVar != "" {
		rv := os.Getenv(ac.AwsRoleArnEnvVar)
		if rv == "" {
			return "", fmt.Errorf("aws_role_arn_env_var references empty string")
		}
		return rv, nil
	}
	if ac.AwsRoleArn == "" {
		return "", fmt.Errorf("aws_role_arn is empty")
	}
	return ac.AwsRoleArn, nil
}

// GetAwsRoleSessionName returns the configured STS session name, falling back to
// a deterministic default when none is supplied. AWS requires a session name on
// every AssumeRole call.
func (ac *AuthCtx) GetAwsRoleSessionName() string {
	if ac.AwsRoleSessionName != "" {
		return ac.AwsRoleSessionName
	}
	return "stackql-assume-role-session"
}

// GetAwsRoleExternalID resolves the optional STS external ID, preferring the
// environment variable indirection when supplied. An empty result is valid.
func (ac *AuthCtx) GetAwsRoleExternalID() string {
	if ac.AwsRoleExternalIDEnvVar != "" {
		return os.Getenv(ac.AwsRoleExternalIDEnvVar)
	}
	return ac.AwsRoleExternalID
}

// GetAwsStsRegion returns the region used to reach the STS endpoint when
// assuming a role, defaulting to us-east-1. This is independent of the region
// used to sign the eventual service request.
func (ac *AuthCtx) GetAwsStsRegion() string {
	if ac.AwsStsRegion != "" {
		return ac.AwsStsRegion
	}
	return "us-east-1"
}

const (
	defaultOciConfigFilePath string = "~/.oci/config"
	defaultOciProfile        string = "DEFAULT"
)

// GetOciTenancyOCID resolves the OCI tenancy OCID, preferring the environment
// variable indirection when supplied. An empty result means "not configured"
// and triggers config-file fallback.
func (ac *AuthCtx) GetOciTenancyOCID() string {
	if ac.OciTenancyOCIDEnvVar != "" {
		return os.Getenv(ac.OciTenancyOCIDEnvVar)
	}
	return ac.OciTenancyOCID
}

// GetOciUserOCID resolves the OCI user OCID, preferring the environment
// variable indirection when supplied.
func (ac *AuthCtx) GetOciUserOCID() string {
	if ac.OciUserOCIDEnvVar != "" {
		return os.Getenv(ac.OciUserOCIDEnvVar)
	}
	return ac.OciUserOCID
}

// GetOciFingerprint resolves the fingerprint of the API signing key, preferring
// the environment variable indirection when supplied.
func (ac *AuthCtx) GetOciFingerprint() string {
	if ac.OciFingerprintEnvVar != "" {
		return os.Getenv(ac.OciFingerprintEnvVar)
	}
	return ac.OciFingerprint
}

// GetOciPrivateKey resolves the RSA private key PEM: an env var carrying the
// PEM itself wins, then an inline literal, then a key file path (literal or via
// env var). An empty result with nil error means "not configured" and triggers
// config-file fallback.
func (ac *AuthCtx) GetOciPrivateKey() (string, error) {
	if ac.OciPrivateKeyEnvVar != "" {
		rv := os.Getenv(ac.OciPrivateKeyEnvVar)
		if rv == "" {
			return "", fmt.Errorf("private_key_env_var references empty string")
		}
		return rv, nil
	}
	if ac.OciPrivateKey != "" {
		return ac.OciPrivateKey, nil
	}
	keyFilePath := ac.OciPrivateKeyFilePath
	if ac.OciPrivateKeyFilePathEnvVar != "" {
		keyFilePath = os.Getenv(ac.OciPrivateKeyFilePathEnvVar)
		if keyFilePath == "" {
			return "", fmt.Errorf("private_key_path_env_var references empty string")
		}
	}
	if keyFilePath == "" {
		return "", nil
	}
	b, err := os.ReadFile(keyFilePath) // #nosec G703 -- credential file is explicitly configured by the user.
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetOciPassphrase resolves the optional private key passphrase, preferring the
// environment variable indirection when supplied. An empty result is valid.
func (ac *AuthCtx) GetOciPassphrase() string {
	if ac.OciPassphraseEnvVar != "" {
		return os.Getenv(ac.OciPassphraseEnvVar)
	}
	return ac.OciPassphrase
}

// GetOciRegion resolves the region, preferring the environment variable
// indirection when supplied. The region does not participate in request
// signing; it is a provider-config convenience.
func (ac *AuthCtx) GetOciRegion() string {
	if ac.OciRegionEnvVar != "" {
		return os.Getenv(ac.OciRegionEnvVar)
	}
	return ac.OciRegion
}

// GetOciConfigFilePath returns the OCI config file path, defaulting to the
// ~/.oci/config convention every OCI tool shares.
func (ac *AuthCtx) GetOciConfigFilePath() string {
	if ac.OciConfigFilePath != "" {
		return ac.OciConfigFilePath
	}
	return defaultOciConfigFilePath
}

// GetOciProfile returns the OCI config file profile, defaulting to DEFAULT.
func (ac *AuthCtx) GetOciProfile() string {
	if ac.OciProfile != "" {
		return ac.OciProfile
	}
	return defaultOciProfile
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
		return os.ReadFile(credentialFile) // #nosec G703 -- credential file is explicitly configured by the user.
	}
	credentialFile := ac.KeyFilePath
	if credentialFile != "" {
		return os.ReadFile(credentialFile) // #nosec G703 -- credential file is explicitly configured by the user.
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
	return &AuthCtx{AuthConfig: omnisdk.AuthConfig{Scopes: scopes,
		Type:        authType,
		KeyFilePath: keyFilePath,
		Active:      false},
	}
}
