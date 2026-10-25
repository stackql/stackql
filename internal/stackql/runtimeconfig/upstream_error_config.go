package dto

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// UpstreamErrorCfg is the opt in policy for upstream errors that a read
// tolerates, keyed by protocol.  The zero value tolerates nothing, which is
// the default behaviour.
type UpstreamErrorCfg struct {
	HTTP HTTPUpstreamErrorCfg `json:"http" yaml:"http"`
}

// HTTPUpstreamErrorCfg lists the HTTP response status codes that a read
// ignores, either exactly (eg "503") or by class (eg "5xx").
type HTTPUpstreamErrorCfg struct {
	IgnoreStatusCodes []string `json:"ignoreStatusCodes" yaml:"ignoreStatusCodes"`
}

func GetUpstreamErrorCfg(s string) (UpstreamErrorCfg, error) {
	rv := UpstreamErrorCfg{}
	err := yaml.Unmarshal([]byte(s), &rv)
	return rv, err
}

func (c HTTPUpstreamErrorCfg) IsIgnored(statusCode int) bool {
	code := strconv.Itoa(statusCode)
	for _, pattern := range c.IgnoreStatusCodes {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == code {
			return true
		}
		if len(pattern) == len(code) && strings.HasSuffix(pattern, "xx") && pattern[0] == code[0] {
			return true
		}
	}
	return false
}
