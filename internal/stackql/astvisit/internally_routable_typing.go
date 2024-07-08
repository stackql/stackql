package astvisit

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"

	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/internal/stackql/util"
)

var (
	_ InternallyRoutableTypingAstVisitor = &standardInternallyRoutableTypingAstVisitor{}
)

type InternallyRoutableTypingAstVisitor interface {
	sqlparser.SQLAstVisitor
	GetSelectContext() (drm.PreparedStatementCtx, bool)
	WithFormatter(formatter sqlparser.NodeFormatter) InternallyRoutableTypingAstVisitor
}

type standardInternallyRoutableTypingAstVisitor struct {
	handlerCtx           handler.HandlerContext
	rawQuery             string
	dc                   drm.Config
	tables               taxonomy.TblMap
	annotations          taxonomy.AnnotationCtxMap
	discoGenIDs          map[sqlparser.SQLNode]int
	annotatedTabulations taxonomy.AnnotatedTabulationMap
	columnNames          []parserutil.ColumnHandle
	columnDescriptors    []anysdk.ColumnDescriptor
	relationalColumns    []typing.RelationalColumn
	namespaceCollection  tablenamespace.Collection
	formatter            sqlparser.NodeFormatter
	annotatedAST         annotatedast.AnnotatedAst
	//
	// single threaded, so no mutex protection
	anonColCounter int
	valuesCtx      drm.PreparedStatementCtx
}

func NewInternallyRoutableTypingAstVisitor(
	rawQuery string,
	annotatedAST annotatedast.AnnotatedAst,
	handlerCtx handler.HandlerContext,
	tables taxonomy.TblMap,
	dc drm.Config,
	namespaceCollection tablenamespace.Collection,
) InternallyRoutableTypingAstVisitor {
	rv := &standardInternallyRoutableTypingAstVisitor{
		rawQuery:             rawQuery,
		annotatedAST:         annotatedAST,
		handlerCtx:           handlerCtx,
		tables:               tables,
		annotatedTabulations: make(taxonomy.AnnotatedTabulationMap),
		dc:                   dc,
		namespaceCollection:  namespaceCollection,
	}
	return rv
}

// TODO: introduce dependency on RDBMS
func (v *standardInternallyRoutableTypingAstVisitor) getTypeFromParserType(t sqlparser.ValType) string {
	//nolint:exhaustive // acceptable
	switch t {
	case sqlparser.StrVal:
		return "TEXT"
	case sqlparser.IntVal:
		return "INT"
	case sqlparser.FloatVal:
		return "NUMERIC"
	default:
		return "TEXT"
	}
}

func (v *standardInternallyRoutableTypingAstVisitor) getNextAlias() string {
	v.anonColCounter++
	i := v.anonColCounter
	return fmt.Sprintf("col_%d", i)
}

//nolint:lll // TODO: fix this
func (v *standardInternallyRoutableTypingAstVisitor) getStarColumns(
	tbl tablemetadata.ExtendedTableMetadata,
) ([]typing.RelationalColumn, error) {
	if indirect, isIndirect := tbl.GetIndirect(); isIndirect {
		rv := indirect.GetRelationalColumns()
		if len(rv) > 0 {
			return rv, nil
		}
		rv = v.dc.ColumnsToRelationalColumns(indirect.GetColumns())
		return rv, nil
	}
	schema, _, err := tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return nil, err
	}
	itemObjS, selectItemsKey, err := tbl.GetSelectSchemaAndObjectPath()
	tbl.SetSelectItemsKey(selectItemsKey)
	unsuitableSchemaMsg := "standardInternallyRoutableTypingAstVisitor.getStarColumns(): schema unsuitable for select query"
	if err != nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	if itemObjS == nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	var cols []parserutil.ColumnHandle
	colNames := itemObjS.GetAllColumns(util.TrimSelectItemsKey(selectItemsKey))
	for _, v := range colNames {
		cols = append(cols, parserutil.NewUnaliasedColumnHandle(v))
	}
	var columnDescriptors []anysdk.ColumnDescriptor
	for _, col := range cols {
		columnDescriptors = append(
			columnDescriptors,
			anysdk.NewColumnDescriptor(
				col.Alias,
				col.Name,
				col.Qualifier,
				col.DecoratedColumn,
				nil,
				schema,
				col.Val,
			),
		)
	}
	relationalColumns := v.dc.OpenapiColumnsToRelationalColumns(columnDescriptors)
	return relationalColumns, nil
}

func (v *standardInternallyRoutableTypingAstVisitor) WithFormatter(
	formatter sqlparser.NodeFormatter) InternallyRoutableTypingAstVisitor {
	v.formatter = formatter
	return v
}

func (v *standardInternallyRoutableTypingAstVisitor) GetSelectContext() (drm.PreparedStatementCtx, bool) {
	if v.valuesCtx != nil {
		return v.valuesCtx, true
	}
	if len(v.relationalColumns) > 0 {
		var columns []typing.ColumnMetadata
		for _, col := range v.relationalColumns {
			relationalColumn := col
			columns = append(
				columns,
				typing.NewRelayedColDescriptor(
					relationalColumn, relationalColumn.GetType()))
		}
		rv := drm.NewQueryOnlyPreparedStatementCtx(v.rawQuery, columns)
		return rv, true
	}
	return nil, false
}

//nolint:dupl,funlen,gocognit,gocyclo,cyclop,errcheck,staticcheck,gocritic,lll,govet,nestif,exhaustive,gomnd,revive // defer uplifts on analysers
func (v *standardInternallyRoutableTypingAstVisitor) Visit(node sqlparser.SQLNode) error {
	var err error

	switch node := node.(type) {
	case *sqlparser.Select:
		if node.SelectExprs != nil {
			err = node.SelectExprs.Accept(v)
			if err != nil {
				return err
			}
		}
		if node.From != nil {
			err := node.From.Accept(v)
			if err != nil {
				return err
			}
		}
		if node.Where != nil {
			node.Where.Accept(v)
		}
		if node.GroupBy != nil {
			node.GroupBy.Accept(v)
		}
		if node.Having != nil {
			node.Having.Accept(v)
		}
		if node.OrderBy != nil {
			node.OrderBy.Accept(v)
		}
		if node.Limit != nil {
			node.Limit.Accept(v)
		}
		return nil

	case *sqlparser.ParenSelect:
		node.Accept(v)

	case *sqlparser.Auth:
		return nil

	case *sqlparser.AuthRevoke:
		return nil
	case *sqlparser.Sleep:
		return nil

	case *sqlparser.Union:
		err = node.FirstStatement.Accept(v)
		if err != nil {
			return err
		}
		for _, us := range node.UnionSelects {
			err = us.Accept(v)
			if err != nil {
				return err
			}
		}
		return nil

	case *sqlparser.UnionSelect:
		return node.Statement.Accept(v)

	case *sqlparser.Stream:
		err = node.Comments.Accept(v)
		if err != nil {
			return err
		}
		err = node.SelectExpr.Accept(v)
		if err != nil {
			return err
		}
		return node.Table.Accept(v)

	case *sqlparser.Insert:
		err = node.Rows.Accept(v)
		if err != nil {
			return err
		}

	case *sqlparser.Update:

	case *sqlparser.Delete:

	case *sqlparser.Set:

	case *sqlparser.SetTransaction:

	case *sqlparser.DBDDL:

	case *sqlparser.DDL:
		switch node.Action {
		case sqlparser.CreateStr:
		case sqlparser.DropStr:
		case sqlparser.RenameStr:
		case sqlparser.AlterStr:
		case sqlparser.FlushStr:
		case sqlparser.CreateVindexStr:
		case sqlparser.DropVindexStr:
		case sqlparser.AddVschemaTableStr:
		case sqlparser.DropVschemaTableStr:
		case sqlparser.AddColVindexStr:
		case sqlparser.DropColVindexStr:
		case sqlparser.AddSequenceStr:
		case sqlparser.AddAutoIncStr:
		default:
		}

	case *sqlparser.OptLike:

	case *sqlparser.PartitionSpec:
		switch node.Action {
		case sqlparser.ReorganizeStr:
		default:
		}

	case *sqlparser.PartitionDefinition:

	case *sqlparser.TableSpec:

	case *sqlparser.ColumnDefinition:

	// Format returns a canonical string representation of the type and all relevant options
	case *sqlparser.ColumnType:
		ct := node

		if ct.Length != nil && ct.Scale != nil {

		} else if ct.Length != nil {
		}

		if ct.EnumValues != nil {
		}

		opts := make([]string, 0, 16) //nolint:mnd // TODO: tech debt sweep mnd hacks
		if ct.Unsigned {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.UNSIGNED])
		}
		if ct.Zerofill {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.ZEROFILL])
		}
		if ct.Charset != "" {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.CHARACTER], sqlparser.KeywordStrings[sqlparser.SET], ct.Charset)
		}
		if ct.Collate != "" {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.COLLATE], ct.Collate)
		}
		if ct.NotNull {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.NOT], sqlparser.KeywordStrings[sqlparser.NULL])
		}
		if ct.Default != nil {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.DEFAULT], sqlparser.String(ct.Default))
		}
		if ct.OnUpdate != nil {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.ON], sqlparser.KeywordStrings[sqlparser.UPDATE], sqlparser.String(ct.OnUpdate))
		}
		if ct.Autoincrement {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.AUTO_INCREMENT])
		}
		if ct.Comment != nil {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.COMMENT_KEYWORD], sqlparser.String(ct.Comment))
		}
		if ct.KeyOpt == sqlparser.ColKeyPrimary {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.PRIMARY], sqlparser.KeywordStrings[sqlparser.KEY])
		}
		if ct.KeyOpt == sqlparser.ColKeyUnique {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.UNIQUE])
		}
		if ct.KeyOpt == sqlparser.ColKeyUniqueKey {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.UNIQUE], sqlparser.KeywordStrings[sqlparser.KEY])
		}
		if ct.KeyOpt == sqlparser.ColKeySpatialKey {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.SPATIAL], sqlparser.KeywordStrings[sqlparser.KEY])
		}
		if ct.KeyOpt == sqlparser.ColKey {
			opts = append(opts, sqlparser.KeywordStrings[sqlparser.KEY])
		}

		if len(opts) != 0 {
		}

	case *sqlparser.IndexDefinition:
		idx := node
		for i, col := range idx.Columns {
			if i != 0 {
			} else {
			}
			if col.Length != nil {
			}
		}

		for _, opt := range idx.Options {
			if opt.Using != "" {
			} else {
			}
		}

	case *sqlparser.IndexInfo:
		ii := node
		if ii.Primary {
		} else {
			if !ii.Name.IsEmpty() {
			}
		}

	case *sqlparser.AutoIncSpec:

	case *sqlparser.ExecSubquery:
		t, ok := v.annotations[node]
		if !ok {
			return fmt.Errorf("exec: could not infer annotated table")
		}
		dID, ok := v.discoGenIDs[node]
		if !ok {
			return fmt.Errorf("exec: could not infer discovery generation ID")
		}
		replacementExpr := v.dc.GetParserTableName(t.GetHIDs(), dID)
		node.Exec.MethodName = replacementExpr

	case *sqlparser.VindexSpec:

		numParams := len(node.Params)
		if numParams != 0 {
			for i, p := range node.Params {
				logging.GetLogger().Debugf("%v\n", p)
				if i != 0 {
				}
			}
		}

	case sqlparser.VindexParam:

	case *sqlparser.ConstraintDefinition:
		c := node
		if c.Name != "" {
		}

	case sqlparser.ReferenceAction:
		a := node
		switch a {
		case sqlparser.Restrict:
		case sqlparser.Cascade:
		case sqlparser.NoAction:
		case sqlparser.SetNull:
		case sqlparser.SetDefault:
		}

	case *sqlparser.ForeignKeyDefinition:
		f := node
		if f.OnDelete != sqlparser.DefaultAction {
		}
		if f.OnUpdate != sqlparser.DefaultAction {
		}

	case *sqlparser.Show:
		nodeType := strings.ToLower(node.Type)
		if (nodeType == "tables" || nodeType == "columns" || nodeType == "fields" || nodeType == "index" || nodeType == "keys" || nodeType == "indexes") && node.ShowTablesOpt != nil {
			opt := node.ShowTablesOpt
			if node.Extended != "" {
			} else {
			}
			if (nodeType == "columns" || nodeType == "fields") && node.HasOnTable() {
			}
			if (nodeType == "index" || nodeType == "keys" || nodeType == "indexes") && node.HasOnTable() {
			}
			if opt.DbName != "" {
			}
			return nil
		}
		if node.Scope == "" {
		} else {
		}
		if node.HasOnTable() {
		}
		if nodeType == "collation" && node.ShowCollationFilterOpt != nil {
		}
		if nodeType == "charset" && node.ShowTablesOpt != nil {
		}
		if node.HasTable() {
		}

	case *sqlparser.ShowFilter:
		if node == nil {
			return nil
		}
		if node.Like != "" {
		} else {
		}

	case *sqlparser.Use:
		if node.DBName.GetRawVal() != "" {
		} else {
		}

	case *sqlparser.Commit:

	case *sqlparser.Begin:

	case *sqlparser.Rollback:

	case *sqlparser.SRollback:

	case *sqlparser.Savepoint:

	case *sqlparser.Release:

	case *sqlparser.Explain:
		switch node.Type {
		case "": // do nothing
		case sqlparser.AnalyzeStr:
		default:
		}

	case *sqlparser.OtherRead:

	case *sqlparser.DescribeTable:

	case *sqlparser.OtherAdmin:

	case sqlparser.Comments:

	case sqlparser.SelectExprs:
		for _, n := range node {
			err = v.Visit(n)
			if err != nil {
				return err
			}
		}

	case *sqlparser.StarExpr:
		var tbl tablemetadata.ExtendedTableMetadata
		if node.TableName.IsEmpty() {
			if len(v.tables) != 1 {
				return fmt.Errorf("unaliased star expr not permitted for table count = %d", len(v.tables))
			}
			for _, v := range v.tables {
				tbl = v
				break
			}
		} else {
			var ok bool
			tbl, ok = v.tables[node.TableName]
			if !ok {
				return fmt.Errorf("could not locate table for expr '%v'", node.TableName)
			}
		}
		cols, err := v.getStarColumns(tbl)
		if err != nil {
			return err
		}
		v.relationalColumns = append(v.relationalColumns, cols...)

	case *sqlparser.AliasedExpr:
		tbl, tblErr := v.tables.GetTableLoose(node)
		err := v.Visit(node.Expr)
		if err != nil {
			return err
		}
		if tblErr != nil {
			col, err := parserutil.InferColNameFromExpr(node, v.formatter)
			if err != nil {
				return err
			}
			if col.Alias == "" && (col.Name == "" || strings.Contains(col.Name, " ")) {
				col.Alias = v.getNextAlias()
			}
			v.columnNames = append(v.columnNames, col)
			// broadcastType := "TEXT"
			// switch expr := node.Expr.(type) {
			// case *sqlparser.SQLVal:
			// 	broadcastType = v.getTypeFromParserType(expr.Type)
			// default:
			// }
			// cd := anysdk.NewColumnDescriptor(col.Alias, col.Name, col.Qualifier, col.DecoratedColumn, node, nil, col.Val)
			// v.columnDescriptors = append(v.columnDescriptors, cd)
			// relCol := v.dc.OpenapiColumnsToRelationalColumn(cd)
			rv := typing.NewRelationalColumn(
				col.Name,
				v.getTypeFromParserType(col.Type),
			).WithDecorated(
				col.DecoratedColumn,
			).WithAlias(
				col.Alias,
			).WithUnquote(true)
			v.relationalColumns = append(v.relationalColumns, rv)
			return nil
		}
		if indirect, isIndirect := tbl.GetIndirect(); isIndirect {
			col, err := parserutil.InferColNameFromExpr(node, v.formatter)
			if err != nil {
				return err
			}
			if col.IsAggregateExpr {
				rv := typing.NewRelationalColumn(
					col.Name,
					v.getTypeFromParserType(col.Type),
				).WithDecorated(
					col.DecoratedColumn,
				).WithAlias(
					col.Alias,
				).WithUnquote(true)
				v.relationalColumns = append(v.relationalColumns, rv)
				return nil
			}

			relationalCol, ok := indirect.GetRelationalColumnByIdentifier(col.Name)
			if !ok {
				r, ok := indirect.GetColumnByName(col.Name)
				if !ok {
					return fmt.Errorf("internally routable typing: cannot find col = '%s'", col.Name)
				}
				relationalCol = typing.NewRelationalColumn(col.Name, r.GetType()).WithDecorated(col.DecoratedColumn)
			}
			v.relationalColumns = append(v.relationalColumns, relationalCol)
			return nil
		}
		// TODO: accomodate SQL data source
		sqlDataSource, isSQLDataSource := tbl.GetSQLDataSource()
		if isSQLDataSource {
			//
			col, err := parserutil.InferColNameFromExpr(node, v.formatter)
			if err != nil {
				return err
			}
			relationalColumn, err := v.dc.GetSQLSystem().ObtainRelationalColumnFromExternalSQLtable(
				tbl.GetHeirarchyObjects().GetHeirarchyIDs(),
				col.Name,
			)
			if err != nil {
				return err
			}
			relationalColumn = relationalColumn.WithAlias(col.Alias)
			relationalColumn = relationalColumn.WithQualifier(col.Qualifier)
			v.relationalColumns = append(v.relationalColumns, relationalColumn)
			logging.GetLogger().Debugf("sqlDataSource = '%v'\n", sqlDataSource)
			return nil
		}
		schema, err := tbl.GetSelectableObjectSchema()
		if err != nil {
			return err
		}
		col, err := parserutil.InferColNameFromExpr(node, v.formatter)
		if err != nil {
			return err
		}
		v.columnNames = append(v.columnNames, col)
		ss, _ := schema.GetProperty(col.Name)
		cd := anysdk.NewColumnDescriptor(col.Alias, col.Name, col.Qualifier, col.DecoratedColumn, node, ss, col.Val)
		v.columnDescriptors = append(v.columnDescriptors, cd)
		v.relationalColumns = append(v.relationalColumns, v.dc.OpenapiColumnsToRelationalColumn(cd))
		if !node.As.IsEmpty() {
		}

	case sqlparser.Nextval:

	case sqlparser.Columns:
		for _, n := range node {
			logging.GetLogger().Debugf("%v\n", n)
		}

	case sqlparser.Partitions:
		if node == nil {
			return nil
		}
		for _, n := range node {
			logging.GetLogger().Debugf("%v\n", n)
		}

	case sqlparser.TableExprs:
		for _, n := range node {
			err := n.Accept(v)
			if err != nil {
				return err
			}
		}

	case *sqlparser.AliasedTableExpr:
		if node.Expr != nil && !parserutil.IsNullTable(node) {
			switch node.Expr.(type) {
			case sqlparser.TableName:
				t, ok := v.annotations[node]
				if !ok {
					return fmt.Errorf("could not infer annotated table")
				}
				_, isIndirect := t.GetTableMeta().GetIndirect()
				_, isSQLDataSource := t.GetTableMeta().GetSQLDataSource()
				if isIndirect || isSQLDataSource {
					// do nothing
				} else {
					dID, ok := v.discoGenIDs[node]
					if !ok {
						return fmt.Errorf("could not infer discovery generation ID")
					}
					replacementExpr := v.dc.GetParserTableName(t.GetHIDs(), dID)
					node.Expr = replacementExpr
				}
			}
		}
		if node.Partitions != nil {
			node.Partitions.Accept(v)
		}
		if !node.As.IsEmpty() {
			node.As.Accept(v)
		}
		if node.Hints != nil {
			node.Hints.Accept(v)
		}

	case sqlparser.TableNames:
		for _, n := range node {
			n.Accept(v)
		}

	case sqlparser.TableName:
		if node.IsEmpty() {
			return nil
		}
		if !node.QualifierThird.IsEmpty() {
		}
		if !node.QualifierSecond.IsEmpty() {
		}
		if !node.Qualifier.IsEmpty() {
		}

	case *sqlparser.ParenTableExpr:

	case sqlparser.JoinCondition:
		if node.On != nil {
		}
		if node.Using != nil {
		}

	case *sqlparser.JoinTableExpr:
		err = node.LeftExpr.Accept(v)
		if err != nil {
			return err
		}
		err = node.RightExpr.Accept(v)
		if err != nil {
			return err
		}

	case *sqlparser.IndexHints:
		if len(node.Indexes) == 0 {
		} else {
			for _, n := range node.Indexes {
				logging.GetLogger().Debugf("%v\n", n)
			}
		}

	case *sqlparser.Where:
		if node == nil || node.Expr == nil {
			return nil
		}
		return node.Expr.Accept(v)

	case sqlparser.Exprs:
		for _, n := range node {
			logging.GetLogger().Debugf("%v\n", n)
		}

	case *sqlparser.AndExpr:
		err = node.Left.Accept(v)
		if err != nil {
			return err
		}
		return node.Right.Accept(v)

	case *sqlparser.OrExpr:
		err = node.Left.Accept(v)
		if err != nil {
			return err
		}
		return node.Right.Accept(v)

	case *sqlparser.XorExpr:
		err = node.Left.Accept(v)
		if err != nil {
			return err
		}
		return node.Right.Accept(v)

	case *sqlparser.NotExpr:
		return node.Expr.Accept(v)

	case *sqlparser.ComparisonExpr:
		switch left := node.Left.(type) {
		case *sqlparser.ColName:
			if b, ok := left.Metadata.(bool); ok && b {
				buf := sqlparser.NewTrackedBuffer(nil)
				node.Format(buf)
			}
		}

	case *sqlparser.RangeCond:

	case *sqlparser.IsExpr:
		return node.Expr.Accept(v)

	case *sqlparser.ExistsExpr:
		return nil

	case *sqlparser.SQLVal:
		switch node.Type {
		case sqlparser.StrVal:
		case sqlparser.IntVal, sqlparser.FloatVal, sqlparser.HexNum:
		case sqlparser.HexVal:
		case sqlparser.BitVal:
		case sqlparser.ValArg:
		default:
		}

	case *sqlparser.NullVal:

	case sqlparser.BoolVal:
		if node {
		} else {
		}

	case *sqlparser.ColName:
		if !node.Qualifier.IsEmpty() {
		}

	case sqlparser.ValTuple:
		v.valuesCtx = drm.NewQueryOnlyPreparedStatementCtx(v.rawQuery, nil)
		for _, n := range node {
			err = n.Accept(v)
			if err != nil {
				return err
			}
		}

	case *sqlparser.Subquery:

	case sqlparser.ListArg:

	case *sqlparser.BinaryExpr:

	case *sqlparser.UnaryExpr:
		if _, unary := node.Expr.(*sqlparser.UnaryExpr); unary {
			// They have same precedence so parenthesis is not required.
			return nil
		}

	case *sqlparser.IntervalExpr:

	case *sqlparser.TimestampFuncExpr:

	case *sqlparser.CurTimeFuncExpr:

	case *sqlparser.CollateExpr:

	case *sqlparser.FuncExpr:
		newNode, err := v.dc.GetSQLSystem().GetASTFuncRewriter().RewriteFunc(node)
		if err != nil {
			return err
		}
		node.Distinct = newNode.Distinct
		node.Exprs = newNode.Exprs
		node.Name = newNode.Name
		if node.Distinct {
		}
		if !node.Qualifier.IsEmpty() {
		}
		// Function names should not be back-quoted even
		// if they match a reserved word, only if they contain illegal characters
		funcName := node.Name.String()

		if sqlparser.ContainEscapableChars(funcName, sqlparser.NoAt) {
		} else {
		}

	case *sqlparser.GroupConcatExpr:
		// v.handlerCtx.GetSQLSystem().GetASTFuncRewriter().RewriteFunc(node)

	case *sqlparser.ValuesFuncExpr:

	case *sqlparser.SubstrExpr:
		if node.Name != nil {
		} else {
		}

		if node.To == nil {
		} else {
		}

	case *sqlparser.ConvertExpr:

	case *sqlparser.ConvertUsingExpr:

	case *sqlparser.ConvertType:
		if node.Length != nil {
			if node.Scale != nil {
			}
		}
		if node.Charset != "" {
		}

	case *sqlparser.MatchExpr:

	case *sqlparser.CaseExpr:
		if node.Expr != nil {
		}
		for _, when := range node.Whens {
			logging.GetLogger().Debugf("%v\n", when)
		}
		if node.Else != nil {
		}

	case *sqlparser.Default:
		if node.ColName != "" {
		}

	case *sqlparser.When:

	case sqlparser.GroupBy:
		for _, n := range node {
			logging.GetLogger().Debugf("%v\n", n)
		}

	case sqlparser.OrderBy:
		for _, n := range node {
			logging.GetLogger().Debugf("%v\n", n)
		}

	case *sqlparser.Order:
		if node, ok := node.Expr.(*sqlparser.NullVal); ok {
			logging.GetLogger().Debugf("%v\n", node)
			return nil
		}
		if node, ok := node.Expr.(*sqlparser.FuncExpr); ok {
			if node.Name.Lowered() == "rand" {
				return nil
			}
		}

	case *sqlparser.Limit:
		if node == nil {
			return nil
		}
		if node.Offset != nil {
		}

	case sqlparser.Values:
		for _, n := range node {
			err = n.Accept(v)
			if err != nil {
				return err
			}
			return nil // first row defines typing
		}

	case sqlparser.UpdateExprs:
		for _, n := range node {
			logging.GetLogger().Debugf("%v\n", n)
		}

	case *sqlparser.UpdateExpr:

	case sqlparser.SetExprs:
		for _, n := range node {
			logging.GetLogger().Debugf("%v\n", n)
		}

	case *sqlparser.SetExpr:
		if node.Scope != "" {
		}
		// We don't have to backtick set variable names.
		switch {
		case node.Name.EqualString("charset") || node.Name.EqualString("names"):
		case node.Name.EqualString(sqlparser.TransactionStr):
		default:
		}

	case sqlparser.OnDup:
		if node == nil {
			return nil
		}

	case sqlparser.ColIdent:
		for i := sqlparser.NoAt; i < node.GetAtCount(); i++ {
		}

	case sqlparser.TableIdent:

	case *sqlparser.IsolationLevel:

	case *sqlparser.AccessMode:
	}
	return nil
}
