package planbuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"
)

func (p *primitiveGenerator) assembleUnarySelectionBuilder(
	handlerCtx *handler.HandlerContext,
	node sqlparser.SQLNode,
	rewrittenWhere *sqlparser.Where,
	hIds *dto.HeirarchyIdentifiers,
	schema *openapistackql.Schema,
	tbl *taxonomy.ExtendedTableMetadata,
	selectTabulation *openapistackql.Tabulation,
	insertTabulation *openapistackql.Tabulation,
	cols []parserutil.ColumnHandle,
) error {
	annotatedInsertTabulation := util.NewAnnotatedTabulation(insertTabulation, hIds, "")

	prov, err := tbl.GetProviderObject()
	if err != nil {
		return err
	}

	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	_, err = docparser.OpenapiStackQLTabulationsPersistor(method, []util.AnnotatedTabulation{annotatedInsertTabulation}, p.PrimitiveComposer.GetSQLEngine(), prov.Name)
	if err != nil {
		return err
	}
	tableDTO, err := p.PrimitiveComposer.GetDRMConfig().GetCurrentTable(hIds, handlerCtx.SQLEngine)
	if err != nil {
		return err
	}
	insPsc, err := p.PrimitiveComposer.GetDRMConfig().GenerateInsertDML(annotatedInsertTabulation, method, dto.NewTxnControlCounters(p.PrimitiveComposer.GetTxnCounterManager(), tableDTO.GetDiscoveryID()))
	if err != nil {
		return err
	}
	p.PrimitiveComposer.SetTxnCtrlCtrs(insPsc.GetGCCtrlCtrs())
	for _, col := range cols {
		foundSchema := schema.FindByPath(col.Name, nil)
		cc, ok := method.GetParameter(col.Name)
		if foundSchema == nil && col.IsColumn {
			if !(ok && cc.GetName() == col.Name) {
				return fmt.Errorf("column = '%s' is NOT present in either:  - data returned from provider, - acceptable parameters, use the DESCRIBE command to view available fields for SELECT operations", col.Name)
			}
		}
		selectTabulation.PushBackColumn(openapistackql.NewColumnDescriptor(col.Alias, col.Name, col.DecoratedColumn, col.Expr, foundSchema, col.Val))
	}

	selPsc, err := p.PrimitiveComposer.GetDRMConfig().GenerateSelectDML(util.NewAnnotatedTabulation(selectTabulation, hIds, tbl.GetAlias()), insPsc.GetGCCtrlCtrs(), astvisit.GenerateModifiedSelectSuffix(node), astvisit.GenerateModifiedWhereClause(rewrittenWhere))
	if err != nil {
		return err
	}
	p.PrimitiveComposer.SetInsertPreparedStatementCtx(insPsc)
	p.PrimitiveComposer.SetSelectPreparedStatementCtx(selPsc)
	p.PrimitiveComposer.SetColumnOrder(cols)
	return nil
}

func (p *primitiveGenerator) analyzeUnarySelection(
	handlerCtx *handler.HandlerContext,
	node sqlparser.SQLNode,
	rewrittenWhere *sqlparser.Where,
	tbl *taxonomy.ExtendedTableMetadata,
	cols []parserutil.ColumnHandle) error {
	_, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}
	schema, mediaType, err := tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return err
	}
	itemObjS, selectItemsKey, err := schema.GetSelectSchema(tbl.LookupSelectItemsKey(), mediaType)
	// rscStr, _ := tbl.GetResourceStr()
	unsuitableSchemaMsg := "analyzeUnarySelection(): schema unsuitable for select query"
	if err != nil {
		return fmt.Errorf(unsuitableSchemaMsg)
	}
	tbl.SelectItemsKey = selectItemsKey
	provStr, _ := tbl.GetProviderStr()
	svcStr, _ := tbl.GetServiceStr()
	// rscStr, _ := tbl.GetResourceStr()
	if itemObjS == nil {
		return fmt.Errorf(unsuitableSchemaMsg)
	}
	if len(cols) == 0 {
		tsa := util.NewTableSchemaAnalyzer(schema, method)
		colz := tsa.GetColumns()
		for _, v := range colz {
			cols = append(cols, parserutil.NewUnaliasedColumnHandle(v.GetName()))
		}
	}
	insertTabulation := itemObjS.Tabulate(false)

	hIds := dto.NewHeirarchyIdentifiers(provStr, svcStr, itemObjS.GetName(), "")
	selectTabulation := itemObjS.Tabulate(true)

	return p.assembleUnarySelectionBuilder(
		handlerCtx,
		node,
		rewrittenWhere,
		hIds,
		schema,
		tbl,
		selectTabulation,
		insertTabulation,
		cols,
	)
}
