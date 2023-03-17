package dto

import (
	"regexp"
	"text/template"

	"gopkg.in/yaml.v2"
)

type NamespaceCfg struct {
	RegexpStr         string `json:"regex" yaml:"regex"`
	TTL               int    `json:"ttl" yaml:"ttl"`
	NamespaceTemplate string `json:"template" yaml:"template"`
}

func (nc NamespaceCfg) GetRegex() (*regexp.Regexp, error) {
	return regexp.Compile(nc.RegexpStr)
}

func (nc NamespaceCfg) GetTemplate() (*template.Template, error) {
	tmpl, err := template.New("stackqlNamespaceTmpl").Parse(nc.NamespaceTemplate)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

func GetNamespaceCfg(s string) (map[string]NamespaceCfg, error) {
	rv := make(map[string]NamespaceCfg)
	err := yaml.Unmarshal([]byte(s), &rv)
	return rv, err
}
