package astvisit

import (
	"fmt"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"

	log "github.com/sirupsen/logrus"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

type TableRouteAstVisitor struct {
	handlerCtx      *handler.HandlerContext
	router          *parserutil.ParameterRouter
	tableMetaSlice  []*taxonomy.ExtendedTableMetadata
	tables          taxonomy.TblMap
	annotations     taxonomy.AnnotationCtxMap
	annotationSlice []taxonomy.AnnotationCtx
}

func NewTableRouteAstVisitor(handlerCtx *handler.HandlerContext, router *parserutil.ParameterRouter) *TableRouteAstVisitor {
	return &TableRouteAstVisitor{
		handlerCtx:  handlerCtx,
		router:      router,
		tables:      make(taxonomy.TblMap),
		annotations: make(taxonomy.AnnotationCtxMap),
	}
}

func obtainAnnotationCtx(
	sqlEngine sqlengine.SQLEngine,
	tbl *taxonomy.ExtendedTableMetadata,
	parameters map[string]interface{},
) (taxonomy.AnnotationCtx, error) {
	schema, mediaType, err := tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return taxonomy.AnnotationCtx{}, err
	}
	itemObjS, selectItemsKey, err := schema.GetSelectSchema(tbl.LookupSelectItemsKey(), mediaType)
	unsuitableSchemaMsg := "schema unsuitable for select query"
	if err != nil {
		return taxonomy.AnnotationCtx{}, fmt.Errorf(unsuitableSchemaMsg)
	}
	tbl.SelectItemsKey = selectItemsKey
	provStr, _ := tbl.GetProviderStr()
	svcStr, _ := tbl.GetServiceStr()
	if itemObjS == nil {
		return taxonomy.AnnotationCtx{}, fmt.Errorf(unsuitableSchemaMsg)
	}
	hIds := dto.NewHeirarchyIdentifiers(provStr, svcStr, itemObjS.GetName(), "")
	return taxonomy.AnnotationCtx{Schema: itemObjS, HIDs: hIds, TableMeta: tbl, Parameters: parameters}, nil
}

func (v *TableRouteAstVisitor) addAnnotationCtx(
	node sqlparser.SQLNode,
	tbl *taxonomy.ExtendedTableMetadata,
	parameters map[string]interface{},
) error {
	ac, err := obtainAnnotationCtx(v.handlerCtx.SQLEngine, tbl, parameters)
	if err != nil {
		return err
	}
	v.annotations[node] = ac
	v.annotationSlice = append(v.annotationSlice, ac)
	return nil
}

func (v *TableRouteAstVisitor) analyzeAliasedTable(tb *sqlparser.AliasedTableExpr) (*taxonomy.ExtendedTableMetadata, map[string]interface{}, error) {
	switch ex := tb.Expr.(type) {
	case sqlparser.TableName:
		err := v.router.Route(tb)
		if err != nil {
			return nil, nil, err
		}
		tpc := v.router.GetAvailableParameters(tb)
		hr, remainingParams, err := taxonomy.GetHeirarchyFromStatement(v.handlerCtx, tb, tpc.GetStringified())
		if err != nil {
			return nil, nil, err
		}
		reconstitutedConsumedParams, err := tpc.ReconstituteConsumedParams(remainingParams)
		if err != nil {
			return nil, nil, err
		}
		err = v.router.InvalidateParams(reconstitutedConsumedParams)
		if err != nil {
			return nil, nil, err
		}
		abbreviatedConsumedMap, err := tpc.AbbreviateMap(reconstitutedConsumedParams)
		if err != nil {
			return nil, nil, err
		}
		m := taxonomy.NewExtendedTableMetadata(hr, taxonomy.GetAliasFromStatement(tb))
		// v.tables[ex] = m
		return m, abbreviatedConsumedMap, nil
	default:
		return nil, nil, fmt.Errorf("table of type '%T' not curently supported", ex)
	}
}

func (v *TableRouteAstVisitor) GetTableMetaArray() []*taxonomy.ExtendedTableMetadata {
	return v.tableMetaSlice
}

func (v *TableRouteAstVisitor) GetTableMap() taxonomy.TblMap {
	return v.tables
}

func (v *TableRouteAstVisitor) GetAnnotations() taxonomy.AnnotationCtxMap {
	return v.annotations
}

func (v *TableRouteAstVisitor) GetAnnotationSlice() []taxonomy.AnnotationCtx {
	return v.annotationSlice
}

func (v *TableRouteAstVisitor) Visit(node sqlparser.SQLNode) error {
	var err error

	switch node := node.(type) {
	case *sqlparser.Select:
		var options string
		addIf := func(b bool, s string) {
			if b {
				options += s
			}
		}
		addIf(node.Distinct, sqlparser.DistinctStr)
		if node.Cache != nil {
			if *node.Cache {
				options += sqlparser.SQLCacheStr
			} else {
				options += sqlparser.SQLNoCacheStr
			}
		}
		addIf(node.StraightJoinHint, sqlparser.StraightJoinHint)
		addIf(node.SQLCalcFoundRows, sqlparser.SQLCalcFoundRowsStr)

		if node.Comments != nil {
			node.Comments.Accept(v)
		}
		if node.SelectExprs != nil {
			node.SelectExprs.Accept(v)
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

		opts := make([]string, 0, 16)
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

	case *sqlparser.VindexSpec:

		numParams := len(node.Params)
		if numParams != 0 {
			for i, p := range node.Params {
				log.Debugf("%v\n", p)
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

	case *sqlparser.StarExpr:
		if !node.TableName.IsEmpty() {
		}

	case *sqlparser.AliasedExpr:
		if !node.As.IsEmpty() {
		}

	case sqlparser.Nextval:

	case sqlparser.Columns:
		for _, n := range node {
			log.Debugf("%v\n", n)
		}

	case sqlparser.Partitions:
		if node == nil {
			return nil
		}
		for _, n := range node {
			log.Debugf("%v\n", n)
		}

	case sqlparser.TableExprs:
		for _, n := range node {
			err := n.Accept(v)
			if err != nil {
				return err
			}
		}

	case *sqlparser.AliasedTableExpr:
		if node.Expr != nil {
			t, params, err := v.analyzeAliasedTable(node)
			if err != nil {
				return err
			}
			if t == nil {
				return fmt.Errorf("nil table returned")
			}
			v.tableMetaSlice = append(v.tableMetaSlice, t)
			v.tables[node] = t
			err = v.addAnnotationCtx(node, t, params)
			if err != nil {
				return err
			}

			// node.Expr.Accept(v)
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
				log.Debugf("%v\n", n)
			}
		}

	case *sqlparser.Where:
		if node == nil || node.Expr == nil {
			return nil
		}
		return node.Expr.Accept(v)

	case sqlparser.Exprs:
		for _, n := range node {
			log.Debugf("%v\n", n)
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
		return nil

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

	case *sqlparser.Subquery:

	case sqlparser.ListArg:

	case *sqlparser.BinaryExpr:

	case *sqlparser.UnaryExpr:
		if _, unary := node.Expr.(*sqlparser.UnaryExpr); unary {
			return nil
		}

	case *sqlparser.IntervalExpr:

	case *sqlparser.TimestampFuncExpr:

	case *sqlparser.CurTimeFuncExpr:

	case *sqlparser.CollateExpr:

	case *sqlparser.FuncExpr:
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
			log.Debugf("%v\n", when)
		}
		if node.Else != nil {
		}

	case *sqlparser.Default:
		if node.ColName != "" {
		}

	case *sqlparser.When:

	case sqlparser.GroupBy:
		for _, n := range node {
			log.Debugf("%v\n", n)
		}

	case sqlparser.OrderBy:
		for _, n := range node {
			log.Debugf("%v\n", n)
		}

	case *sqlparser.Order:
		if node, ok := node.Expr.(*sqlparser.NullVal); ok {
			log.Debugf("%v\n", node)
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
			log.Debugf("%v\n", n)
		}

	case sqlparser.UpdateExprs:
		for _, n := range node {
			log.Debugf("%v\n", n)
		}

	case *sqlparser.UpdateExpr:

	case sqlparser.SetExprs:
		for _, n := range node {
			log.Debugf("%v\n", n)
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
