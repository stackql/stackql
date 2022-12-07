package astindirect

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/parse"

	"github.com/stackql/stackql/internal/stackql/internaldto"
	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	_ Indirect = &view{}
)

type IndirectType int

const (
	ViewType IndirectType = iota
	SubqueryType
	CTEType
)

func NewViewIndirect(viewDTO internaldto.ViewDTO) (Indirect, error) {
	rv := &view{
		viewDTO: viewDTO,
	}
	return rv, nil
}

type Indirect interface {
	GetSelectAST() (sqlparser.SelectStatement, error)
	GetType() IndirectType
}

type view struct {
	viewDTO internaldto.ViewDTO
}

func (v *view) GetType() IndirectType {
	return ViewType
}

func (v *view) getAST() (sqlparser.Statement, error) {
	return parse.ParseQuery(v.viewDTO.GetRawQuery())
}

func (v *view) GetSelectAST() (sqlparser.SelectStatement, error) {
	parseResult, err := v.getAST()
	if err != nil {
		return nil, err
	}
	switch pr := parseResult.(type) {
	case sqlparser.SelectStatement:
		return pr, nil
	default:
		return nil, fmt.Errorf("view of type '%T' not yet supported", pr)
	}
}
