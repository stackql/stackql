package envfile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultContent = `# StackQL credential store, sourced at startup and on each reload_credentials call
# One KEY=VALUE per line, eg:
# AWS_ACCESS_KEY_ID=AKIA...
# AWS_SECRET_ACCESS_KEY=...
`

const (
	dirPermissions  os.FileMode = 0o700
	filePermissions os.FileMode = 0o600
)

// EnsureExists creates path (parent directories included) as an empty
// commented dotenv file when absent, so packaged installs (eg MCP bundles)
// get a working credential store on first launch; an existing file is never
// touched, since bundle updates must not wipe populated credentials.  Returns
// whether this call created the file.  An empty path is a no-op.
func EnsureExists(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, dirPermissions); err != nil {
			return false, err
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePermissions)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close() //nolint:errcheck // write error is surfaced below
	if _, writeErr := f.WriteString(defaultContent); writeErr != nil {
		return false, writeErr
	}
	return true, nil
}

// parse reads a dotenv-style file: KEY=VALUE lines; `#` comments, blanks,
// `export ` prefixes, surrounding quotes and CRLF tolerated.  Keys with
// empty values are dropped so a reload cannot blank out a working credential.
func parse(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rv := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 { //nolint:mnd // shortest quoted form is ""
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if val == "" {
			continue
		}
		rv[key] = val
	}
	return rv, nil
}

// Source injects the file's pairs into the process environment, overwriting
// existing vars, and returns the sorted key names set.  An empty path is a
// no-op; a missing file is not an error ("nothing to source yet").
func Source(path string) ([]string, bool, error) {
	if path == "" {
		return nil, false, nil
	}
	vars, err := parse(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	keys := make([]string, 0, len(vars))
	for k, v := range vars {
		if setErr := os.Setenv(k, v); setErr != nil {
			return nil, false, setErr
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, true, nil
}
