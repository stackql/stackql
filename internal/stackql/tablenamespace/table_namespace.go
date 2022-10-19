package tablenamespace

import (
	"bytes"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

type TableNamespaceConfigurator interface {
	GetTTL() int
	GetLikeString() string
	GetObjectName(string) string
	IsAllowed(string) bool
	Match(string, string, string, string) (*dto.TxnControlCounters, bool)
	Read(string, string, string, []string) (*sql.Rows, error)
	RenderTemplate(string) (string, error)
}

var (
	_ TableNamespaceConfigurator = &regexTableNamespaceConfigurator{}
)

type regexTableNamespaceConfigurator struct {
	sqlEngine  sqlengine.SQLEngine
	regex      *regexp.Regexp
	template   *template.Template
	likeString string
	ttl        int
}

func (stc *regexTableNamespaceConfigurator) IsAllowed(tableString string) bool {
	return stc.isAllowed(tableString)
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

func (stc *regexTableNamespaceConfigurator) isAllowed(tableString string) bool {
	return stc.regex.MatchString(tableString)
}

func (stc *regexTableNamespaceConfigurator) Read(tableString string, requestEncoding string, requestEncodingColName string, nonControlColumnNames []string) (*sql.Rows, error) {
	isAllowed := stc.isAllowed(tableString)
	if !isAllowed {
		return nil, fmt.Errorf("disallowed tableString = '%s'", tableString)
	}
	actualTableName, err := stc.renderTemplate(tableString)
	if err != nil {
		return nil, fmt.Errorf("could not infer actual table name for tableString = '%s'", tableString)
	}
	isPresent := stc.sqlEngine.IsTablePresent(actualTableName, requestEncoding, requestEncodingColName)
	if !isPresent {
		return nil, fmt.Errorf("absent table name = '%s'", actualTableName)
	}
	var quotedNonControlColNames []string
	for _, c := range nonControlColumnNames {
		quotedNonControlColNames = append(quotedNonControlColNames, fmt.Sprintf(`"%s"`, c))
	}
	colzString := strings.Join(quotedNonControlColNames, ", ")
	return stc.sqlEngine.Query(fmt.Sprintf(`SELECT %s FROM "%s" WHERE "%s" = ?`, colzString, actualTableName, requestEncodingColName), requestEncoding)
}

func (stc *regexTableNamespaceConfigurator) Match(tableString string, requestEncoding string, lastModifiedColName string, requestEncodingColName string) (*dto.TxnControlCounters, bool) {
	isAllowed := stc.isAllowed(tableString)
	if !isAllowed {
		return nil, false
	}
	actualTableName, err := stc.renderTemplate(tableString)
	if err != nil {
		return nil, false
	}
	isPresent := stc.sqlEngine.IsTablePresent(actualTableName, requestEncoding, requestEncodingColName)
	if !isPresent {
		return nil, false
	}
	oldestUpdate, tcc := stc.sqlEngine.TableOldestUpdateUTC(actualTableName, requestEncoding, lastModifiedColName, requestEncodingColName)
	diff := time.Since(oldestUpdate)
	ds := diff.Seconds()
	if stc.ttl > -1 && int(ds) > stc.ttl {
		return nil, false
	}
	return tcc, true
}

func (stc *regexTableNamespaceConfigurator) RenderTemplate(input string) (string, error) {
	return stc.renderTemplate(input)
}

func (stc *regexTableNamespaceConfigurator) renderTemplate(input string) (string, error) {
	objName := stc.getObjectName(input)
	inputMap := map[string]interface{}{
		"objectName": objName,
	}
	return stc.render(inputMap)
}

func (stc *regexTableNamespaceConfigurator) render(input map[string]interface{}) (string, error) {
	var tplWr bytes.Buffer
	if err := stc.template.Execute(&tplWr, input); err != nil {
		return "", err
	}
	return tplWr.String(), nil
}

func (stc *regexTableNamespaceConfigurator) GetObjectName(inputString string) string {
	return stc.getObjectName(inputString)
}

func (stc *regexTableNamespaceConfigurator) getObjectName(inputString string) string {
	for i, name := range stc.regex.SubexpNames() {
		if name == "objectName" {
			submatches := stc.regex.FindStringSubmatch(inputString)
			if len(submatches) > i {
				return submatches[i]
			}
		}
	}
	return ""
}
