package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

const diagnosticRedacted = "[REDACTED]"

// ParseAll visits values without setting flags or resolving credentials.
func mcpDiagnosticArgs(args []string, flags *pflag.FlagSet) []string {
	if len(args) == 0 {
		return nil
	}
	out := []string{args[0]}
	diagnosticFlags := pflag.NewFlagSet("diagnostics", pflag.ContinueOnError)
	diagnosticFlags.SetOutput(io.Discard)
	diagnosticFlags.SetNormalizeFunc(flags.GetNormalizeFunc())
	flags.VisitAll(func(flag *pflag.Flag) {
		copyFlag := *flag
		diagnosticFlags.AddFlag(&copyFlag)
	})
	diagnosticFlags.ParseErrorsAllowlist.UnknownFlags = true
	// A parse failure omits the remaining arguments; never print the error.
	_ = diagnosticFlags.ParseAll(args[1:], func(flag *pflag.Flag, value string) error {
		out = append(out, "--"+flag.Name+"="+mcpDiagnosticValue(flag.Name, value))
		return nil
	})
	return out
}

func mcpDiagnosticValue(name, raw string) string {
	var jsonSyntax bool
	switch name {
	case "http.proxy.password":
		return diagnosticRedacted
	case "mcp.config", "preview", "pgsrv.tls":
		jsonSyntax = true
	case "auth", "registry", "sqlBackend", "otel.config":
	default:
		return raw
	}
	if raw == "" {
		return raw
	}
	var value any
	if jsonSyntax {
		if !json.Valid([]byte(raw)) {
			return diagnosticRedacted
		}
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return diagnosticRedacted
		}
	} else if err := yaml.Unmarshal([]byte(raw), &value); err != nil {
		return diagnosticRedacted
	}
	value = sanitizeDiagnosticConfig(name, value, jsonSyntax)
	if jsonSyntax {
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(value); err != nil {
			return diagnosticRedacted
		}
		return strings.TrimSuffix(out.String(), "\n")
	}
	out, err := yaml.Marshal(value)
	if err != nil {
		return diagnosticRedacted
	}
	return strings.TrimSuffix(string(out), "\n")
}

func maskDiagnosticValue(any) any { return diagnosticRedacted }

func diagnosticObject(value any, visit func(string, any) any) any {
	switch obj := value.(type) {
	case nil:
		return nil
	case map[string]any:
		for key, child := range obj {
			obj[key] = visit(key, child)
		}
	case map[any]any:
		for key, child := range obj {
			name, ok := key.(string)
			if !ok {
				return diagnosticRedacted
			}
			obj[key] = visit(name, child)
		}
	default:
		return diagnosticRedacted
	}
	return value
}

func diagnosticPath(value any, path []string, fold bool, sanitize func(any) any) any {
	if len(path) == 0 {
		return sanitize(value)
	}
	return diagnosticObject(value, func(key string, child any) any {
		if key == path[0] || path[0] == "*" || (fold && strings.EqualFold(key, path[0])) {
			return diagnosticPath(child, path[1:], fold, sanitize)
		}
		return child
	})
}

func sanitizeDiagnosticConfig(name string, value any, fold bool) any {
	var masked, urls [][]string
	switch name {
	case "auth":
		return diagnosticObject(value, func(_ string, child any) any { return diagnosticAuth(child) })
	case "sqlBackend":
		masked = [][]string{{"dsn"}}
	case "pgsrv.tls":
		masked = [][]string{{"keyContents"}}
	case "registry":
		urls = [][]string{{"url"}}
	case "preview":
		return diagnosticPath(value, []string{"endpoint"}, fold, diagnosticEndpoint)
	case "otel.config":
		masked = [][]string{{"exporter", "headers", "*"}}
		urls = [][]string{{"exporter", "endpoint"}}
	case "mcp.config":
		masked = [][]string{{"backend", "dsn"}}
		urls = [][]string{{"query_library", "base_url"}, {"query_library", "fallback_url"}}
		value = diagnosticPath(value, []string{"server", "transport_cfg"}, fold, diagnosticCredentialMap)
	}
	for _, path := range masked {
		value = diagnosticPath(value, path, fold, maskDiagnosticValue)
	}
	for _, path := range urls {
		value = diagnosticPath(value, path, fold, diagnosticURL)
	}
	return value
}

func diagnosticAuth(value any) any {
	// Auth is decoded by yaml.v2, including when its input uses JSON syntax.
	return diagnosticObject(value, func(key string, child any) any {
		switch key {
		case "api_key", "api_secret", "password", "client_secret", "private_key", "passphrase":
			return diagnosticRedacted
		case "successor":
			return diagnosticAuth(child)
		case "sqlDataSource":
			return diagnosticPath(child, []string{"dsn"}, false, maskDiagnosticValue)
		case "values":
			return diagnosticCredentialMap(child)
		case "token_url", "aws_sts_endpoint":
			return diagnosticURL(child)
		default:
			return child
		}
	})
}

func diagnosticCredentialKey(key string) bool {
	key = strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
	switch key {
	case "password", "passwd", "pwd", "secret", "clientsecret", "apikey", "apisecret",
		"accesstoken", "refreshtoken", "idtoken", "token", "authorization", "proxyauthorization",
		"privatekey", "passphrase":
		return true
	default:
		return false
	}
}

func diagnosticCredentialMap(value any) any {
	return diagnosticObject(value, func(key string, child any) any {
		if diagnosticCredentialKey(key) {
			return diagnosticRedacted
		}
		return diagnosticCredentials(child)
	})
}

func diagnosticCredentials(value any) any {
	switch obj := value.(type) {
	case map[string]any, map[any]any:
		return diagnosticCredentialMap(value)
	case []any:
		for i, child := range obj {
			obj[i] = diagnosticCredentials(child)
		}
	}
	return value
}

func diagnosticEndpoint(value any) any {
	if raw, ok := value.(string); ok {
		if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
			return diagnosticURL(raw)
		}
		var endpoints any
		if err := json.Unmarshal([]byte(raw), &endpoints); err != nil {
			return diagnosticRedacted
		}
		return diagnosticEndpoint(endpoints)
	}
	return diagnosticObject(value, func(_ string, child any) any {
		if _, ok := child.(map[string]any); ok {
			return child // Per-service {scheme,host,port,path} overrides contain no URL field.
		}
		return diagnosticURL(child)
	})
}

func diagnosticURL(value any) any {
	raw, ok := value.(string)
	if !ok {
		return diagnosticRedacted
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return diagnosticRedacted
	}
	if _, queryErr := url.ParseQuery(parsed.RawQuery); queryErr != nil {
		return diagnosticRedacted
	}
	// Edit only credential spans so ordinary URL text stays byte-for-byte intact.
	base, fragment, hasFragment := strings.Cut(raw, "#")
	base, query, hasQuery := strings.Cut(base, "?")
	if _, hasPassword := parsed.User.Password(); hasPassword {
		start := strings.Index(base, "//") + len("//")
		if start < len("//") {
			return diagnosticRedacted
		}
		authority, _, _ := strings.Cut(base[start:], "/")
		end := start + strings.LastIndex(authority, "@")
		if end < start {
			return diagnosticRedacted
		}
		colon := strings.Index(base[start:end], ":")
		if colon < 0 {
			return diagnosticRedacted
		}
		base = base[:start+colon+1] + diagnosticRedacted + base[end:]
	}
	if hasQuery {
		parts := strings.Split(query, "&")
		for i, part := range parts {
			key, _, _ := strings.Cut(part, "=")
			decoded, decodeErr := url.QueryUnescape(key)
			if decodeErr != nil {
				return diagnosticRedacted
			}
			if diagnosticURLCredentialKey(decoded) {
				parts[i] = key + "=" + diagnosticRedacted
			}
		}
		base += "?" + strings.Join(parts, "&")
	}
	if hasFragment {
		base += "#" + fragment
	}
	return base
}

func diagnosticURLCredentialKey(key string) bool {
	if diagnosticCredentialKey(key) {
		return true
	}
	switch strings.ToLower(key) {
	case "signature", "sig", "x-amz-signature", "x-amz-security-token", "x-goog-signature":
		return true
	default:
		return false
	}
}
