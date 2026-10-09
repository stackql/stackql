package diagnostics //nolint:testpackage // diagnostic helpers are private

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestMCPDiagnosticValues(t *testing.T) {
	tests := []struct {
		name, raw string
		visible   []string
	}{
		{"auth", `{"google":{"type":"service_account","credentialsenvvar":"MY_CLIENT_SECRET","credentialsfilepathenvvar":"PASSWORD_ENV","credentialsfilepath":"/keys/google.json","api_key_var":"API_KEY_VAR","api_secret_var":"API_SECRET_VAR","password_var":"PASSWORD_VAR","client_secret_env_var":"CLIENT_SECRET_VAR","private_key_env_var":"PRIVATE_KEY_VAR","passphrase_env_var":"PASSPHRASE_VAR","username":"alice","api_key":"HIDE_api_key","api_secret":"HIDE_api_secret","password":"HIDE_password","client_secret":"HIDE_client_secret","private_key":"HIDE_private_key","passphrase":"HIDE_passphrase","successor":{"type":"basic","password":"HIDE_successor","successor":{"api_key":"HIDE_nested"}},"sqlDataSource":{"dsn":"HIDE_auth_dsn","dsnEnvVar":"DSN_ENV"},"values":{"client-secret":["HIDE_oauth"],"nested":[{"Access_Token":"HIDE_nested_token","client_secret_env_var":"REF","tokenizer":"visible"}],"audience":["public"]},"token_url":"https://user:HIDE_pass@auth.example/token?client_id=visible&token=HIDE_query","aws_sts_endpoint":"https://sts.example/?sig=HIDE_sig"}}`,
			[]string{"google", "service_account", "MY_CLIENT_SECRET", "PASSWORD_ENV", "/keys/google.json", "API_KEY_VAR", "API_SECRET_VAR", "PASSWORD_VAR", "CLIENT_SECRET_VAR", "PRIVATE_KEY_VAR", "PASSPHRASE_VAR", "alice", "basic", "DSN_ENV", "REF", "tokenizer", "visible", "audience", "public", "auth.example/token", "sts.example"}},
		{"auth", "provider:\n  password: HIDE_yaml\n  password_var: PASSWORD_ENV\n  successor:\n    private_key: HIDE_key\n    private_key_path: /key.pem\n", []string{"provider", "PASSWORD_ENV", "/key.pem"}},
		{"auth", `{"p":{"password":"HIDE_first","password":"HIDE_last","PASSWORD":"ignored-case","name":"visible"}}`, []string{"ignored-case", "visible"}},
		{"sqlBackend", `{"dsn":"HIDE_dsn","dsnEnvVar":"DSN_ENV","dbEngine":"sqlite","schemata":{"tableSchema":"inventory"},"initMaxRetries":3}`, []string{"DSN_ENV", "sqlite", "inventory", "3"}},
		{"sqlBackend", "dsn: HIDE_yaml_dsn\ndsnEnvVar: DSN_ENV\ndbEngine: postgres", []string{"DSN_ENV", "postgres"}},
		{"mcp.config", `{"SERVER":{"transport":"stdio","mode":"safe","auth_token_env_var":"MCP_TOKEN","tls_key_file":"/key.pem","transport_cfg":{"Password":"HIDE_transport","nested":[{"proxy-authorization":"HIDE_proxy","password_env_var":"REF"}],"passwordish":"visible"},"audit":{"disabled":true}},"backend":{"DSN":"HIDE_mcp_dsn","type":"tcp"},"query_library":{"base_url":"https://user:HIDE_library@library.example/?sig=HIDE_sig&page=2","fallback_url":"https://fallback.example/"}}`, []string{"stdio", "safe", "MCP_TOKEN", "/key.pem", "REF", "visible", "disabled", "tcp", "library.example", "page=2", "https://fallback.example/"}},
		{"mcp.config", `{"backend":{"dsn":"HIDE_first","DSN":"HIDE_case","dsn":"HIDE_last"},"enabled_tools":["server_info"]}`, []string{"server_info"}},
		{"otel.config", `{"exporter":{"endpoint":"https://user:HIDE_pass@collector.example/v1/logs?token=HIDE_token&tenant=public","headers":{"X-Custom":"HIDE_custom","authorization":"HIDE_authz"},"timeout_ms":2500,"batch_size":12}}`, []string{"collector.example/v1/logs", "tenant=public", "X-Custom", "authorization", "2500", "12"}},
		{"otel.config", "exporter:\n  endpoint: https://collector.example/v1/logs\n  headers:\n    custom: HIDE_yaml_header\n  batch_size: 10", []string{"https://collector.example/v1/logs", "custom", "10"}},
		{"pgsrv.tls", `{"KeyContents":"HIDE_key","keyContents":"HIDE_duplicate","certContents":"CERTIFICATE","clientCAs":["CA"],"keyFilePath":"/key.pem","certFilePath":"/cert.pem"}`, []string{"CERTIFICATE", "CA", "/key.pem", "/cert.pem"}},
		{"registry", `{"url":"https://registry.example/providers?version=v1","localDocRoot":"/providers","verifyConfig":{"nopVerify":true}}`, []string{"https://registry.example/providers?version=v1", "/providers", "nopVerify"}},
		{"registry", "url: https://user:HIDE_password@registry.example/providers?sig=HIDE_sig&version=v1\nlocalDocRoot: /providers", []string{"user:", "registry.example/providers", "version=v1", "/providers"}},
		{"preview", `{"endpoint":{"aws.s3":"https://user:HIDE_pass@mock.example/s3?token=HIDE_token","aws.ec2":{"host":"localhost","port":"8085","path":"/ec2"}},"batchSize":5,"unstable":true}`, []string{"mock.example/s3", "localhost", "8085", "/ec2", "batchSize", "unstable"}},
		{"preview", `{"endpoint":"{\"aws.s3\":\"https://user:HIDE_pass@mock.example/s3\"}","batchSize":2}`, []string{"mock.example/s3", "batchSize"}},
		{"registry", `{"url":"https://user:HIDE_pass@bad%zz/","localDocRoot":"/keep"}`, []string{"/keep"}},
	}
	for _, tt := range tests {
		t.Run(tt.name+tt.raw[:min(12, len(tt.raw))], func(t *testing.T) {
			got := mcpDiagnosticValue(tt.name, tt.raw)
			if strings.Contains(got, "HIDE_") {
				t.Errorf("credential disclosed: %s", got)
			}
			for _, want := range tt.visible {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in %s", want, got)
				}
			}
		})
	}
}

func TestMCPDiagnosticMalformed(t *testing.T) {
	for _, flag := range []string{"auth", "sqlBackend", "registry", "otel.config", "mcp.config", "pgsrv.tls", "preview"} {
		for _, raw := range []string{`{"dsn":"HIDE_bad"`, "[HIDE_bad]", "HIDE_scalar"} {
			got := mcpDiagnosticValue(flag, raw)
			if strings.Contains(got, "HIDE_") || !strings.Contains(got, diagnosticRedacted) {
				t.Errorf("%s malformed value: %s", flag, got)
			}
		}
	}
	for _, raw := range []string{
		`{"backend":{"dsn":{"nested":"HIDE_dsn"}},"server":{"transport_cfg":{"token":["HIDE_token"]},"mode":"safe"}}`,
		`{"backend":"HIDE_bad","server":{"mode":"safe"}}`,
		`{"server":{"transport_cfg":"HIDE_bad","mode":"safe"}}`,
		`{"server":{"transport_cfg":["HIDE_bad"],"mode":"safe"}}`,
	} {
		got := mcpDiagnosticValue("mcp.config", raw)
		if strings.Contains(got, "HIDE_") || !strings.Contains(got, "safe") {
			t.Errorf("subfield fallback: %s", got)
		}
	}
	for _, raw := range []string{`{"p":{"values":"HIDE_bad","type":"oauth2"}}`, `{"p":{"values":["HIDE_bad"],"type":"oauth2"}}`} {
		got := mcpDiagnosticValue("auth", raw)
		if strings.Contains(got, "HIDE_") || !strings.Contains(got, "oauth2") {
			t.Errorf("auth map fallback: %s", got)
		}
	}
}

func TestMCPDiagnosticURLs(t *testing.T) {
	for _, raw := range []string{
		"https://registry.example/a%2Fb?x=a+b&x=a%20b#part", "file:///C:/providers", "https://user@registry.example/",
		"https://example/?client_secret_env_var=MY_SECRET&tokenizer=visible", "",
	} {
		if got := diagnosticURL(raw); got != raw {
			t.Errorf("ordinary URL changed: %v, want %s", got, raw)
		}
	}
	keys := strings.Fields("PASSWORD Pass-w_d pwd secret client_secret api_key api_secret access_token refresh_token id_token token authorization proxy_authorization private_key passphrase signature sig X-Amz-Signature X-Amz-Security-Token X-Goog-Signature %74oken")
	for _, key := range keys {
		raw := "https://u%40ser:HIDE_password@example/a@b?" + key + "=HIDE_first&" + key + "=HIDE_repeat&visible=a%20b#fragment"
		want := "https://u%40ser:[REDACTED]@example/a@b?" + key + "=[REDACTED]&" + key + "=[REDACTED]&visible=a%20b#fragment"
		if got := diagnosticURL(raw); got != want {
			t.Errorf("URL masking: %v, want %s", got, want)
		}
	}
	for _, raw := range []string{"https://user:HIDE_bad@%zz/", "https://example/?token=%zz", "https://example/?token=HIDE_bad;other=x"} {
		if got := diagnosticURL(raw); got != diagnosticRedacted {
			t.Errorf("malformed URL retained: %v", got)
		}
	}
}

func TestMCPDiagnosticArgsNonMutation(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.StringP("http.proxy.password", "p", "runtime-password", "")
	flags.String("auth", "runtime-auth", "")
	flags.StringSlice("var", nil, "")
	flags.StringP("output", "o", "table", "")
	flags.BoolP("verbose", "v", false, "")
	flags.String("future", "", "")
	args := []string{"stackql", "mcp", "--http.proxy.password=HIDE_equal", "--http.proxy.password", "-HIDE_dash", "-pHIDE_alias", "-p=HIDE_alias_equal", "-p", "HIDE_alias_separate", "--http.proxy.password=", "--auth", `{"p":{"password":"HIDE_auth","credentialsenvvar":"MY_CLIENT_SECRET"}}`, "--var", "password=visible,token=also-visible", "--var=secret=visible-too", "-vojson", "--future=future-visible", "--unknown=HIDE_unknown", "--unknown", "HIDE_value", "HIDE_positional", "--", "--future=HIDE_after_dash"}
	before := slices.Clone(args)
	diagnoser := NewMCPServerDiagnoser(args, flags)
	got := strings.Join(diagnoser.Diagnose(), " ")
	if strings.Contains(got, "HIDE_") {
		t.Errorf("secret/payload disclosed: %s", got)
	}
	for _, want := range []string{"MY_CLIENT_SECRET", "password=visible,token=also-visible", "secret=visible-too", "--output=json", "--verbose=true", "future-visible"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	if strings.Count(got, "--http.proxy.password=[REDACTED]") != 6 {
		t.Errorf("missing repeated passwords: %s", got)
	}
	if !reflect.DeepEqual(args, before) || flags.Parsed() || flags.NFlag() != 0 {
		t.Fatal("argv or flags mutated")
	}
	password, _ := flags.GetString("http.proxy.password")
	if password != "runtime-password" {
		t.Fatal("runtime credential mutated")
	}
	args[1] = "--future=changed"
	if repeated := strings.Join(diagnoser.Diagnose(), " "); repeated != got {
		t.Errorf("diagnostic changed on reuse: %s", repeated)
	}
	if got := NewMCPServerDiagnoser([]string{"stackql", "--http.proxy.password"}, flags).Diagnose(); !reflect.DeepEqual(got, []string{"stackql"}) {
		t.Errorf("missing flag value: %v", got)
	}
	if got := NewMCPServerDiagnoser(nil, flags).Diagnose(); len(got) != 0 {
		t.Errorf("empty argv: %v", got)
	}
}

func TestMCPDiagnosticPreservesOrdinaryArguments(t *testing.T) {
	t.Setenv("MY_CLIENT_SECRET", "HIDE_env_contents")
	for _, flag := range []string{"var", "env.file", "approot", "http.proxy.user", "http.proxy.host", "dbInternal", "namespaces", "store.txn", "gc", "acid", "session", "pgsrv.sundry", "future"} {
		raw := "password=ordinary-value,http://user:literal@example/?token=literal"
		if got := mcpDiagnosticValue(flag, raw); got != raw {
			t.Errorf("%s changed: %s", flag, got)
		}
	}
	raw := `{"p":{"credentialsenvvar":"MY_CLIENT_SECRET","credentialsfilepath":"/does/not/exist"}}`
	got := mcpDiagnosticValue("auth", raw)
	if !strings.Contains(got, "MY_CLIENT_SECRET") || strings.Contains(got, "HIDE_env_contents") || strings.Contains(got, diagnosticRedacted) {
		t.Errorf("reference-only auth changed: %s", got)
	}
	// Large numeric siblings must survive JSON re-encoding exactly.
	got = mcpDiagnosticValue("mcp.config", `{"server":{"max_concurrent_requests":9007199254740993},"backend":{"dsn":"HIDE_dsn"}}`)
	if !json.Valid([]byte(got)) || !strings.Contains(got, "9007199254740993") {
		t.Errorf("numeric configuration changed: %s", got)
	}
}
