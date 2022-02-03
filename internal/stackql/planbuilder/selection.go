package planbuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	log "github.com/sirupsen/logrus"
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

	svc, err := tbl.GetService()
	if err != nil {
		return err
	}

	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	err = docparser.OpenapiStackQLTabulationsPersistor(prov, svc, []util.AnnotatedTabulation{annotatedInsertTabulation}, p.PrimitiveBuilder.GetSQLEngine(), prov.Name)
	if err != nil {
		return err
	}
	tableDTO, err := p.PrimitiveBuilder.GetDRMConfig().GetCurrentTable(hIds, handlerCtx.SQLEngine)
	if err != nil {
		return err
	}
	insPsc, err := p.PrimitiveBuilder.GetDRMConfig().GenerateInsertDML(annotatedInsertTabulation, p.PrimitiveBuilder.GetTxnCounterManager(), tableDTO.GetDiscoveryID())
	if err != nil {
		return err
	}
	p.PrimitiveBuilder.SetTxnCtrlCtrs(insPsc.TxnCtrlCtrs)
	for _, col := range cols {
		foundSchema := schema.FindByPath(col.Name, nil)
		cc, ok := method.GetParameter(col.Name)
		if ok && cc.Name == col.Name {
			// continue
		}
		if foundSchema == nil && col.IsColumn {
			return fmt.Errorf("column = '%s' is NOT present in either:  - data returned from provider, - acceptable parameters, use the DESCRIBE command to view available fields for SELECT operations", col.Name)
		}
		selectTabulation.PushBackColumn(openapistackql.NewColumnDescriptor(col.Alias, col.Name, col.DecoratedColumn, foundSchema, col.Val))
		log.Infoln(fmt.Sprintf("rsc = %T", col))
		log.Infoln(fmt.Sprintf("schema type = %T", schema))
	}

	selPsc, err := p.PrimitiveBuilder.GetDRMConfig().GenerateSelectDML(util.NewAnnotatedTabulation(selectTabulation, hIds, tbl.GetAlias()), insPsc.TxnCtrlCtrs, node, rewrittenWhere)
	if err != nil {
		return err
	}
	p.PrimitiveBuilder.SetInsertPreparedStatementCtx(&insPsc)
	p.PrimitiveBuilder.SetSelectPreparedStatementCtx(&selPsc)
	p.PrimitiveBuilder.SetColumnOrder(cols)
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
	schema, err := tbl.GetResponseSchema()
	if err != nil {
		return err
	}
	provStr, _ := tbl.GetProviderStr()
	svcStr, _ := tbl.GetServiceStr()
	// rscStr, _ := tbl.GetResourceStr()
	unsuitableSchemaMsg := "schema unsuitable for select query"
	// log.Infoln(fmt.Sprintf("schema.ID = %v", schema.ID))
	log.Infoln(fmt.Sprintf("schema.Items = %v", schema.Items))
	log.Infoln(fmt.Sprintf("schema.Properties = %v", schema.Properties))
	var itemObjS *openapistackql.Schema
	itemObjS, tbl.SelectItemsKey, err = schema.GetSelectSchema(tbl.LookupSelectItemsKey())
	if err != nil {
		return fmt.Errorf(unsuitableSchemaMsg)
	}
	if itemObjS == nil {
		return fmt.Errorf(unsuitableSchemaMsg)
	}
	if len(cols) == 0 {
		colNames := itemObjS.GetAllColumns()
		for _, v := range colNames {
			cols = append(cols, parserutil.NewUnaliasedColumnHandle(v))
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
