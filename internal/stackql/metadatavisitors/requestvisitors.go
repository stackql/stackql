package metadatavisitors

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type TemplatedProduct interface {
	GetBody() string
	GetPlaceholder() string
}

type standardTemplatedProduct struct {
	body        string
	placeholder string
}

func NewTemplatedProduct(body, placeholder string) TemplatedProduct {
	return &standardTemplatedProduct{
		body:        body,
		placeholder: placeholder,
	}
}

func (tp *standardTemplatedProduct) GetBody() string {
	return tp.body
}

func (tp *standardTemplatedProduct) GetPlaceholder() string {
	return tp.placeholder
}

func checkAllColumnsPresent(columns sqlparser.Columns, toInclude map[string]bool) error {
	var missingColNames []string
	if columns != nil {
		for _, col := range columns {
			cName := col.GetRawVal()
			if !toInclude[cName] {
				missingColNames = append(missingColNames, cName)
			}
		}
		if len(missingColNames) > 0 {
			return fmt.Errorf("cannot find the following columns: %s", strings.Join(missingColNames, ", "))
		}
	}
	return nil
}

func getColsMap(columns sqlparser.Columns) map[string]bool {
	retVal := make(map[string]bool)
	for _, col := range columns {
		retVal[col.GetRawVal()] = true
	}
	return retVal
}
