package astindirect

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/drm"
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
	Parse() error
	GetColumnByName(name string) (drm.ColumnMetadata, bool)
	GetColumns() []drm.ColumnMetadata
	GetName() string
	GetOptionalParameters() map[string]openapistackql.Addressable
	GetRequiredParameters() map[string]openapistackql.Addressable
	GetSelectAST() sqlparser.SelectStatement
	GetSelectContext() drm.PreparedStatementCtx
	GetType() IndirectType
	SetSelectContext(drm.PreparedStatementCtx)
}

type view struct {
	viewDTO    internaldto.ViewDTO
	selectStmt sqlparser.SelectStatement
	selCtx     drm.PreparedStatementCtx
}

func (v *view) GetType() IndirectType {
	return ViewType
}

func (v *view) GetName() string {
	return v.viewDTO.GetName()
}

func (v *view) GetColumns() []drm.ColumnMetadata {
	return v.selCtx.GetNonControlColumns()
}

func (v *view) GetOptionalParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *view) GetRequiredParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *view) GetColumnByName(name string) (drm.ColumnMetadata, bool) {
	for _, col := range v.selCtx.GetNonControlColumns() {
		if col.GetIdentifier() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *view) SetSelectContext(selCtx drm.PreparedStatementCtx) {
	v.selCtx = selCtx
}

func (v *view) GetSelectContext() drm.PreparedStatementCtx {
	return v.selCtx
}

func (v *view) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *view) getAST() (sqlparser.Statement, error) {
	return parse.ParseQuery(v.viewDTO.GetRawQuery())
}

func (v *view) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *view) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return v.selCtx, nil
}

func (v *view) Analyze() error {

	return nil
}

func (v *view) Parse() error {
	parseResult, err := v.getAST()
	if err != nil {
		return err
	}
	switch pr := parseResult.(type) {
	case sqlparser.SelectStatement:
		v.selectStmt = pr
		return nil
	default:
		return fmt.Errorf("view of type '%T' not yet supported", pr)
	}
}
