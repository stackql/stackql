package templatenamespace

import (
	"bytes"
	"regexp"
	"text/template"
)

var (
	_ Configurator = &standardTemplateNamespaceConfigurator{}
)

type Configurator interface {
	GetObjectName(string) string
	IsAllowed(string) bool
	RenderTemplate(string) (string, error)
}

func NewTemplateNamespaceConfigurator(
	regex *regexp.Regexp,
	tmpl *template.Template,
) (Configurator, error) {
	return &standardTemplateNamespaceConfigurator{
		regex:    regex,
		template: tmpl,
	}, nil
}

type standardTemplateNamespaceConfigurator struct {
	template *template.Template
	regex    *regexp.Regexp
}

func (dc *standardTemplateNamespaceConfigurator) RenderTemplate(input string) (string, error) {
	return dc.renderTemplate(input)
}

func (dc *standardTemplateNamespaceConfigurator) IsAllowed(tableString string) bool {
	return dc.regex.MatchString(tableString)
}

func (dc *standardTemplateNamespaceConfigurator) GetObjectName(inputString string) string {
	return dc.getObjectName(inputString)
}

func (dc *standardTemplateNamespaceConfigurator) getObjectName(inputString string) string {
	for i, name := range dc.regex.SubexpNames() {
		if name == "objectName" {
			submatches := dc.regex.FindStringSubmatch(inputString)
			if len(submatches) > i {
				return submatches[i]
			}
		}
	}
	return ""
}

func (dc *standardTemplateNamespaceConfigurator) renderTemplate(input string) (string, error) {
	objName := dc.getObjectName(input)
	inputMap := map[string]interface{}{
		"objectName": objName,
	}
	return dc.render(inputMap)
}

func (dc *standardTemplateNamespaceConfigurator) render(input map[string]interface{}) (string, error) {
	var tplWr bytes.Buffer
	if err := dc.template.Execute(&tplWr, input); err != nil {
		return "", err
	}
	return tplWr.String(), nil
}
