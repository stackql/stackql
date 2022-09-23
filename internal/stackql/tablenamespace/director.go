package tablenamespace

import (
	"regexp"
	"text/template"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/pkg/textutil"
)

var (
	defaultAnalyticsCacheRegexp = regexp.MustCompile(constants.DefaultAnalyticsRegexpString)
	defaultViewsRegexp          = regexp.MustCompile(constants.DefaultViewsRegexpString)
	defaultAnalyticsTemplate    = templateParseOrPanic("defaultAnalyticsTmpl", constants.DefaultAnalyticsTemplateString)
	defaultViewsTemplate        = templateParseOrPanic("defaultViewsTmpl", constants.DefaultViewsTemplateString)
)

func templateParseOrPanic(tmplName, tmplBody string) *template.Template {
	rv, err := template.New(tmplName).Parse(tmplBody)
	if err != nil {
		panic(err)
	}
	return rv
}

type TableNamespaceConfiguratorBuilderDirector interface {
	Construct() error
	GetResult() TableNamespaceConfigurator
}

func getViewsTableNamespaceConfiguratorBuilderDirector(cfg dto.NamespaceCfg, sqlEngine sqlengine.SQLEngine) TableNamespaceConfiguratorBuilderDirector {
	return &configuratorBuilderDirector{
		sqlEngine:         sqlEngine,
		cfg:               cfg,
		defaultRegexp:     defaultViewsRegexp,
		defaultTemplate:   defaultViewsTemplate,
		defaultLikeString: textutil.GetTemplateLikeString(constants.DefaultViewsTemplateString),
	}
}

func getAnalyticsCacheTableNamespaceConfiguratorBuilderDirector(cfg dto.NamespaceCfg, sqlEngine sqlengine.SQLEngine) TableNamespaceConfiguratorBuilderDirector {
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
	configurator      TableNamespaceConfigurator
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
	bldr := newTableNamespaceConfiguratorBuilder().WithRegexp(cfgRegexp).WithLikeString(likeString).WithTTL(dr.cfg.TTL).WithTemplate(cfgTemplate).WithSQLEngine(dr.sqlEngine)
	configurator, err := bldr.Build()
	if err != nil {
		return err
	}
	dr.configurator = configurator
	return nil
}

func (dr *configuratorBuilderDirector) GetResult() TableNamespaceConfigurator {
	return dr.configurator
}
