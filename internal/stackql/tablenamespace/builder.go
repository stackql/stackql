package tablenamespace

import (
	"regexp"
	"text/template"

	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/templatenamespace"
)

var (
	_ ConfiguratorBuilder = &standardTableNamespaceConfiguratorBuilder{}
)

type ConfiguratorBuilder interface {
	Build() (Configurator, error)
	WithLikeString(likeString string) ConfiguratorBuilder
	WithTTL(ttl int) ConfiguratorBuilder
	WithRegexp(regex *regexp.Regexp) ConfiguratorBuilder
	WithSQLEngine(sqlEngine sqlengine.SQLEngine) ConfiguratorBuilder
	WithTemplate(tmpl *template.Template) ConfiguratorBuilder
}

type standardTableNamespaceConfiguratorBuilder struct {
	sqlEngine  sqlengine.SQLEngine
	regex      *regexp.Regexp
	tmpl       *template.Template
	likeString string
	ttl        int
}

func newTableNamespaceConfiguratorBuilder() ConfiguratorBuilder {
	return &standardTableNamespaceConfiguratorBuilder{}
}

func (b *standardTableNamespaceConfiguratorBuilder) WithRegexp(regex *regexp.Regexp) ConfiguratorBuilder {
	b.regex = regex
	return b
}

func (b *standardTableNamespaceConfiguratorBuilder) WithLikeString(
	likeString string,
) ConfiguratorBuilder {
	b.likeString = likeString
	return b
}

func (b *standardTableNamespaceConfiguratorBuilder) WithTemplate(
	tmpl *template.Template,
) ConfiguratorBuilder {
	b.tmpl = tmpl
	return b
}

func (b *standardTableNamespaceConfiguratorBuilder) WithTTL(ttl int) ConfiguratorBuilder {
	b.ttl = ttl
	return b
}

func (b *standardTableNamespaceConfiguratorBuilder) WithSQLEngine(
	sqlEngine sqlengine.SQLEngine,
) ConfiguratorBuilder {
	b.sqlEngine = sqlEngine
	return b
}

func (b *standardTableNamespaceConfiguratorBuilder) Build() (Configurator, error) {
	tmplCfg, err := templatenamespace.NewTemplateNamespaceConfigurator(
		b.regex,
		b.tmpl,
	)
	if err != nil {
		return nil, err
	}
	return &regexTableNamespaceConfigurator{
		sqlEngine:                     b.sqlEngine,
		templateNamespaceConfigurator: tmplCfg,
		ttl:                           b.ttl,
		likeString:                    b.likeString,
	}, nil
}
