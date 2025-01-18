package tablenamespace

import (
	"regexp"
	"text/template"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/pkg/textutil"
)

var (
	defaultAnalyticsCacheRegexp = regexp.MustCompile(constants.DefaultAnalyticsRegexpString)
	defaultViewsRegexp          = regexp.MustCompile(constants.DefaultViewsRegexpString)
	defaultAnalyticsTemplate    = templateParseOrPanic("defaultAnalyticsTmpl", constants.DefaultAnalyticsTemplateString) //nolint:gochecknoglobals,lll // local visibility only
	defaultViewsTemplate        = templateParseOrPanic("defaultViewsTmpl", constants.DefaultViewsTemplateString)         //nolint:gochecknoglobals,lll // local visibility only
)

func templateParseOrPanic(tmplName, tmplBody string) *template.Template {
	rv, err := template.New(tmplName).Parse(tmplBody)
	if err != nil {
		panic(err)
	}
	return rv
}

type ConfiguratorBuilderDirector interface {
	Construct() error
	GetResult() Configurator
}

func getViewsTableNamespaceConfiguratorBuilderDirector(
	cfg dto.NamespaceCfg,
	sqlEngine sqlengine.SQLEngine,
) ConfiguratorBuilderDirector {
	return &configuratorBuilderDirector{
		sqlEngine:         sqlEngine,
		cfg:               cfg,
		defaultRegexp:     defaultViewsRegexp,
		defaultTemplate:   defaultViewsTemplate,
		defaultLikeString: textutil.GetTemplateLikeString(constants.DefaultViewsTemplateString),
	}
}

func getAnalyticsCacheTableNamespaceConfiguratorBuilderDirector(
	cfg dto.NamespaceCfg,
	sqlEngine sqlengine.SQLEngine,
) ConfiguratorBuilderDirector {
	return &configuratorBuilderDirector{
		sqlEngine:         sqlEngine,
		cfg:               cfg,
		defaultRegexp:     defaultAnalyticsCacheRegexp,
		defaultTemplate:   defaultAnalyticsTemplate,
		defaultLikeString: textutil.GetTemplateLikeString(constants.DefaultAnalyticsTemplateString),
	}
}

type configuratorBuilderDirector struct {
	sqlEngine         sqlengine.SQLEngine
	cfg               dto.NamespaceCfg
	defaultRegexp     *regexp.Regexp
	defaultTemplate   *template.Template
	defaultLikeString string
	configurator      Configurator
}

func (dr *configuratorBuilderDirector) Construct() error {
	var err error
	cfgRegexp := dr.defaultRegexp
	cfgTemplate := dr.defaultTemplate
	likeString := dr.defaultLikeString
	if dr.cfg.RegexpStr != "" {
		cfgRegexp, err = dr.cfg.GetRegex()
		if err != nil {
			return err
		}
	}
	if dr.cfg.NamespaceTemplate != "" {
		cfgTemplate, err = dr.cfg.GetTemplate()
		if err != nil {
			return err
		}
		likeString = textutil.GetTemplateLikeString(dr.cfg.NamespaceTemplate)
	}
	//nolint:lll // chaining
	bldr := newTableNamespaceConfiguratorBuilder().WithRegexp(cfgRegexp).WithLikeString(likeString).WithTTL(dr.cfg.TTL).WithTemplate(cfgTemplate).WithSQLEngine(dr.sqlEngine)
	configurator, err := bldr.Build()
	if err != nil {
		return err
	}
	dr.configurator = configurator
	return nil
}

func (dr *configuratorBuilderDirector) GetResult() Configurator {
	return dr.configurator
}
