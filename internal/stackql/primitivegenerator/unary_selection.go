package primitivegenerator

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func (pb *standardPrimitiveGenerator) assembleUnarySelectionBuilder(
	pbi planbuilderinput.PlanBuilderInput,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	rewrittenWhere *sqlparser.Where,
	hIds internaldto.HeirarchyIdentifiers,
	schema openapistackql.Schema,
	tbl tablemetadata.ExtendedTableMetadata,
	selectTabulation openapistackql.Tabulation,
	insertTabulation openapistackql.Tabulation,
	cols []parserutil.ColumnHandle,
) error {
	inputTableName, err := tbl.GetInputTableName()
	if err != nil {
		return err
	}
	annotatedInsertTabulation := util.NewAnnotatedTabulation(insertTabulation, hIds, inputTableName, "")

	prov, err := tbl.GetProviderObject()
	if err != nil {
		return err
	}

	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	_, err = docparser.OpenapiStackQLTabulationsPersistor(
		method, []util.AnnotatedTabulation{annotatedInsertTabulation},
		pb.PrimitiveComposer.GetSQLEngine(),
		prov.GetName(),
		handlerCtx.GetNamespaceCollection(),
		handlerCtx.GetControlAttributes(),
		handlerCtx.GetSQLSystem())
	if err != nil {
		return err
	}
	ctrs := pbi.GetTxnCtrlCtrs()
	insPsc, err := pb.PrimitiveComposer.GetDRMConfig().GenerateInsertDML(annotatedInsertTabulation, method, ctrs)
	if err != nil {
		return err
	}
	pb.PrimitiveComposer.SetTxnCtrlCtrs(insPsc.GetGCCtrlCtrs())
	for _, col := range cols {
		foundSchema := schema.FindByPath(col.Name, nil)
		cc, ok := method.GetParameter(col.Name)
		if foundSchema == nil && col.IsColumn {
			if !(ok && cc.GetName() == col.Name) {
				return fmt.Errorf(
					"column = '%s' is NOT present in either:  - data returned from provider, - acceptable parameters, use the DESCRIBE command to view available fields for SELECT operations", //nolint:lll // long string
					col.Name)
			}
		}
		selectTabulation.PushBackColumn(
			openapistackql.NewColumnDescriptor(
				col.Alias,
				col.Name,
				col.Qualifier,
				col.DecoratedColumn,
				col.Expr,
				foundSchema,
				col.Val,
			),
		)
	}
	selectSuffix := astvisit.GenerateModifiedSelectSuffix(
		pbi.GetAnnotatedAST(),
		node,
		handlerCtx.GetSQLSystem(),
		handlerCtx.GetASTFormatter(),
		handlerCtx.GetNamespaceCollection(),
	)
	selPsc, err := pb.PrimitiveComposer.GetDRMConfig().GenerateSelectDML(
		util.NewAnnotatedTabulation(selectTabulation, hIds, inputTableName, tbl.GetAlias()),
		insPsc.GetGCCtrlCtrs(),
		selectSuffix,
		astvisit.GenerateModifiedWhereClause(
			pbi.GetAnnotatedAST(),
			rewrittenWhere,
			handlerCtx.GetSQLSystem(),
			handlerCtx.GetASTFormatter(),
			handlerCtx.GetNamespaceCollection()),
	)
	if err != nil {
		return err
	}
	pb.PrimitiveComposer.SetInsertPreparedStatementCtx(insPsc)
	pb.PrimitiveComposer.SetSelectPreparedStatementCtx(selPsc)
	pb.PrimitiveComposer.SetColumnOrder(cols)
	return nil
}

func (pb *standardPrimitiveGenerator) analyzeUnarySelection(
	pbi planbuilderinput.PlanBuilderInput,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	rewrittenWhere *sqlparser.Where,
	tbl tablemetadata.ExtendedTableMetadata,
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
	tbl.SetSelectItemsKey(selectItemsKey)
	provStr, _ := tbl.GetProviderStr()
	svcStr, _ := tbl.GetServiceStr()
	// rscStr, _ := tbl.GetResourceStr()
	if itemObjS == nil {
		return fmt.Errorf(unsuitableSchemaMsg)
	}
	if len(cols) == 0 {
		tsa := util.NewTableSchemaAnalyzer(schema, method)
		colz, localErr := tsa.GetColumns()
		if localErr != nil {
			return localErr
		}
		for _, v := range colz {
			cols = append(cols, parserutil.NewUnaliasedColumnHandle(v.GetName()))
		}
	}
	insertTabulation := itemObjS.Tabulate(false)

	hIds := internaldto.NewHeirarchyIdentifiers(provStr, svcStr, itemObjS.GetName(), "")
	viewDTO, isView := handlerCtx.GetSQLSystem().GetViewByName(hIds.GetTableName())
	if isView {
		hIds = hIds.WithView(viewDTO)
	}
	selectTabulation := itemObjS.Tabulate(true)

	return pb.assembleUnarySelectionBuilder(
		pbi,
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
