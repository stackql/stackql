package tablenamespace

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/templatenamespace"
)

type TableNamespaceConfigurator interface {
	GetTTL() int
	GetLikeString() string
	GetObjectName(string) string
	IsAllowed(string) bool
	Match(string, string, string, string) (*dto.TxnControlCounters, bool)
	Read(string, string, string, []string) (*sql.Rows, error)
	RenderTemplate(string) (string, error)
	WithSQLDialect(sqlDialect sqldialect.SQLDialect) (TableNamespaceConfigurator, error)
}

var (
	_ TableNamespaceConfigurator = &regexTableNamespaceConfigurator{}
)

type regexTableNamespaceConfigurator struct {
	sqlDialect                    sqldialect.SQLDialect
	sqlEngine                     sqlengine.SQLEngine
	templateNamespaceConfigurator templatenamespace.TemplateNamespaceConfigurator
	likeString                    string
	ttl                           int
}

func (stc *regexTableNamespaceConfigurator) IsAllowed(tableString string) bool {
	return stc.templateNamespaceConfigurator.IsAllowed(tableString)
}

func (stc *regexTableNamespaceConfigurator) GetTTL() int {
	return stc.ttl
}

func (stc *regexTableNamespaceConfigurator) GetLikeString() string {
	return stc.getLikeString()
}

func (stc *regexTableNamespaceConfigurator) getLikeString() string {
	return stc.likeString
}

func (stc *regexTableNamespaceConfigurator) Read(tableString string, requestEncoding string, requestEncodingColName string, nonControlColumnNames []string) (*sql.Rows, error) {
	isAllowed := stc.templateNamespaceConfigurator.IsAllowed(tableString)
	if !isAllowed {
		return nil, fmt.Errorf("disallowed tableString = '%s'", tableString)
	}
	actualTableName, err := stc.templateNamespaceConfigurator.RenderTemplate(tableString)
	if err != nil {
		return nil, fmt.Errorf("could not infer actual table name for tableString = '%s'", tableString)
	}
	isPresent := stc.sqlDialect.IsTablePresent(actualTableName, requestEncoding, requestEncodingColName)
	if !isPresent {
		return nil, fmt.Errorf("absent table name = '%s'", actualTableName)
	}
	var quotedNonControlColNames []string
	for _, c := range nonControlColumnNames {
		quotedNonControlColNames = append(quotedNonControlColNames, fmt.Sprintf(`"%s"`, c))
	}
	colzString := strings.Join(quotedNonControlColNames, ", ")
	return stc.sqlDialect.QueryNamespaced(colzString, actualTableName, requestEncodingColName, requestEncoding)
}

func (stc *regexTableNamespaceConfigurator) Match(tableString string, requestEncoding string, lastModifiedColName string, requestEncodingColName string) (*dto.TxnControlCounters, bool) {
	isAllowed := stc.templateNamespaceConfigurator.IsAllowed(tableString)
	if !isAllowed {
		return nil, false
	}
	actualTableName, err := stc.templateNamespaceConfigurator.RenderTemplate(tableString)
	if err != nil {
		return nil, false
	}
	isPresent := stc.sqlDialect.IsTablePresent(actualTableName, requestEncoding, requestEncodingColName)
	if !isPresent {
		return nil, false
	}
	oldestUpdate, tcc := stc.sqlDialect.TableOldestUpdateUTC(actualTableName, requestEncoding, lastModifiedColName, requestEncodingColName)
	diff := time.Since(oldestUpdate)
	ds := diff.Seconds()
	if stc.ttl > -1 && int(ds) > stc.ttl {
		return nil, false
	}
	return tcc, true
}

func (stc *regexTableNamespaceConfigurator) RenderTemplate(input string) (string, error) {
	return stc.templateNamespaceConfigurator.RenderTemplate(input)
}

func (stc *regexTableNamespaceConfigurator) GetObjectName(inputString string) string {
	return stc.templateNamespaceConfigurator.GetObjectName(inputString)
}

func (stc *regexTableNamespaceConfigurator) WithSQLDialect(sqlDialect sqldialect.SQLDialect) (TableNamespaceConfigurator, error) {
	stc.sqlDialect = sqlDialect
	return stc, nil
}
