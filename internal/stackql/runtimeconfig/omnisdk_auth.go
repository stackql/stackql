package dto

import (
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
)

func OmniAuth(authCtx *AuthCtx) *omnisdk.Auth {
	if authCtx == nil {
		return nil
	}
	auth := &omnisdk.Auth{
		TenancyOCID:          authCtx.OciTenancyOCID,
		TenancyOCIDEnvVar:    authCtx.OciTenancyOCIDEnvVar,
		UserOCID:             authCtx.OciUserOCID,
		UserOCIDEnvVar:       authCtx.OciUserOCIDEnvVar,
		Fingerprint:          authCtx.OciFingerprint,
		FingerprintEnvVar:    authCtx.OciFingerprintEnvVar,
		PrivateKey:           authCtx.OciPrivateKey,
		PrivateKeyEnvVar:     authCtx.OciPrivateKeyEnvVar,
		PrivateKeyPath:       authCtx.OciPrivateKeyFilePath,
		PrivateKeyPathEnvVar: authCtx.OciPrivateKeyFilePathEnvVar,
		Passphrase:           authCtx.OciPassphrase,
		PassphraseEnvVar:     authCtx.OciPassphraseEnvVar,
		ConfigFilePath:       authCtx.OciConfigFilePath,
		Profile:              authCtx.OciProfile,

		Type:           authCtx.Type,
		ValuePrefix:    authCtx.ValuePrefix,
		Name:           authCtx.Name,
		Scopes:         authCtx.Scopes,
		TokenURL:       authCtx.GetTokenURL(),
		Username:       authCtx.Username,
		Password:       authCtx.Password,
		UsernameEnvVar: authCtx.EnvVarUsername,
		PasswordEnvVar: authCtx.EnvVarPassword,
	}
	if strings.EqualFold(authCtx.Type, "basic") && auth.ValuePrefix == "" {
		// Preserve the CLI's established Basic scheme spelling.
		auth.ValuePrefix = "BASIC "
	}
	if (strings.EqualFold(authCtx.Type, "api_key") || strings.EqualFold(authCtx.Type, "custom")) && auth.Name == "" {
		auth.Name = "Authorization"
		if auth.ValuePrefix == "" && strings.EqualFold(authCtx.Type, "api_key") {
			auth.ValuePrefix = "Bearer "
		}
	}
	if credentials, credErr := authCtx.GetCredentialsBytes(); credErr == nil {
		auth.SecretAccessKey = string(credentials)
		auth.Credentials = string(credentials)
	}
	if keyID, keyErr := authCtx.GetKeyIDString(); keyErr == nil {
		auth.AccessKeyID = keyID
	}
	if sessionToken, tokErr := authCtx.GetAwsSessionTokenString(); tokErr == nil {
		auth.SessionToken = sessionToken
	}
	if clientID, idErr := authCtx.GetClientID(); idErr == nil {
		auth.ClientID = clientID
	}
	if clientSecret, secErr := authCtx.GetClientSecret(); secErr == nil {
		auth.ClientSecret = clientSecret
	}
	return auth
}
