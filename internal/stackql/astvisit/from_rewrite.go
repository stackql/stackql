package astvisit

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"

	"github.com/stackql/stackql-parser/go/sqltypes"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ FromRewriteAstVisitor = &standardFromRewriteAstVisitor{}
)

type FromRewriteAstVisitor interface {
	sqlparser.SQLAstVisitor
	GetIndirectContexts() []drm.PreparedStatementCtx
	GetRewrittenQuery() string
	SetAvoidSQLSourceNaming(bool)
	GetHoistedOnClauseTables() []sqlparser.SQLNode
}

// Need not be view-aware.
type standardFromRewriteAstVisitor struct {
	iDColumnName           string
	rewrittenQuery         string
	shouldCollectTables    bool
	namespaceCollection    tablenamespace.Collection
	sqlSystem              sql_system.SQLSystem
	formatter              sqlparser.NodeFormatter
	annotations            taxonomy.AnnotationCtxMap
	dc                     drm.Config
	annotatedAST           annotatedast.AnnotatedAst
	indirectContexts       []drm.PreparedStatementCtx
	isAvoidSQLSourceNaming bool
	hoistedOnClauseTables  []sqlparser.SQLNode
}

func NewFromRewriteAstVisitor(
	annotatedAST annotatedast.AnnotatedAst,
	iDColumnName string,
	shouldCollectTables bool,
	sqlSystem sql_system.SQLSystem,
	formatter sqlparser.NodeFormatter,
	namespaceCollection tablenamespace.Collection,
	annotations taxonomy.AnnotationCtxMap,
	dc drm.Config,
) FromRewriteAstVisitor {
	return &standardFromRewriteAstVisitor{
		annotatedAST:        annotatedAST,
		iDColumnName:        iDColumnName,
		shouldCollectTables: shouldCollectTables,
		namespaceCollection: namespaceCollection,
		sqlSystem:           sqlSystem,
		formatter:           formatter,
		annotations:         annotations,
		dc:                  dc,
	}
}

func (v *standardFromRewriteAstVisitor) GetHoistedOnClauseTables() []sqlparser.SQLNode {
	return v.hoistedOnClauseTables
}

func (v *standardFromRewriteAstVisitor) SetAvoidSQLSourceNaming(isAvoidSQLSourceNaming bool) {
	v.isAvoidSQLSourceNaming = isAvoidSQLSourceNaming
}

func (v *standardFromRewriteAstVisitor) GetRewrittenQuery() string {
	return v.rewrittenQuery
}

func (v *standardFromRewriteAstVisitor) GetIndirectContexts() []drm.PreparedStatementCtx {
	return v.indirectContexts
}

//nolint:dupl,funlen,gocognit,gocyclo,cyclop,errcheck,gocritic,lll,exhaustive,nestif,gomnd // defer uplifts on analysers
func (v *standardFromRewriteAstVisitor) Visit(node sqlparser.SQLNode) error {
	buf := sqlparser.NewTrackedBuffer(v.formatter)

	switch node := node.(type) {
	case *sqlparser.Select:
		return fmt.Errorf("unsupported subquery pattern")

	case *sqlparser.ParenSelect:
		node.Accept(v)
		selStr := v.GetRewrittenQuery()
		rq := fmt.Sprintf("(%v)", selStr)
		v.rewrittenQuery = rq

	case *sqlparser.Auth:
		var stackqlOpt string
		if node.SessionAuth {
			stackqlOpt = "stackql "
		}
		rq := fmt.Sprintf("%sAUTH %v %s %v %v", stackqlOpt, node.Provider, node.Type, node.KeyFilePath, node.KeyEnvVar)
		v.rewrittenQuery = rq

	case *sqlparser.AuthRevoke:
		var stackqlOpt string
		if node.SessionAuth {
			stackqlOpt = "stackql "
		}
		rq := fmt.Sprintf("%sauth revoke %v", stackqlOpt, node.Provider)
		v.rewrittenQuery = rq

	case *sqlparser.Sleep:
		rq := fmt.Sprintf("sleep %v", node.Duration)
		v.rewrittenQuery = rq

	case *sqlparser.Union:
		buf.AstPrintf(node, "%v", node.FirstStatement)
		for _, us := range node.UnionSelects {
			buf.AstPrintf(node, "%v", us)
		}
		buf.AstPrintf(node, "%v%v%s", node.OrderBy, node.Limit, node.Lock)
		v.rewrittenQuery = buf.String()

	case *sqlparser.UnionSelect:
		buf.AstPrintf(node, " %s %v", node.Type, node.Statement)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Stream:
		buf.AstPrintf(node, "stream %v%v from %v",
			node.Comments, node.SelectExpr, node.Table)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Insert:
		buf.AstPrintf(node, "%s %v%sinto %v%v%v %v%v",
			node.Action,
			node.Comments, node.Ignore,
			node.Table, node.Partitions, node.Columns, node.Rows, node.OnDup)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Update:
		buf.AstPrintf(node, "update %v%s%v set %v%v%v%v",
			node.Comments, node.Ignore, node.TableExprs,
			node.Exprs, node.Where, node.OrderBy, node.Limit)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Delete:
		buf.AstPrintf(node, "delete %v", node.Comments)
		if node.Targets != nil {
			buf.AstPrintf(node, "%v ", node.Targets)
		}
		buf.AstPrintf(node, "from %v%v%v%v%v", node.TableExprs, node.Partitions, node.Where, node.OrderBy, node.Limit)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Set:
		buf.AstPrintf(node, "set %v%v", node.Comments, node.Exprs)
		v.rewrittenQuery = buf.String()

	case *sqlparser.SetTransaction:
		if node.Scope == "" {
			buf.AstPrintf(node, "set %vtransaction ", node.Comments)
		} else {
			buf.AstPrintf(node, "set %v%s transaction ", node.Comments, node.Scope)
		}

		for i, char := range node.Characteristics {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.AstPrintf(node, "%v", char)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.DBDDL:
		switch node.Action {
		case sqlparser.CreateStr, sqlparser.AlterStr:
			buf.WriteString(fmt.Sprintf("%s database %s", node.Action, node.DBName))
		case sqlparser.DropStr:
			exists := ""
			if node.IfExists {
				exists = " if exists"
			}
			buf.WriteString(fmt.Sprintf("%s database%s %v", node.Action, exists, node.DBName))
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.DDL:
		switch node.Action {
		case sqlparser.CreateStr:
			if node.OptLike != nil {
				buf.AstPrintf(node, "%s table %v %v", node.Action, node.Table, node.OptLike)
			} else if node.TableSpec != nil {
				buf.AstPrintf(node, "%s table %v %v", node.Action, node.Table, node.TableSpec)
			} else {
				buf.AstPrintf(node, "%s table %v", node.Action, node.Table)
			}
		case sqlparser.DropStr:
			exists := ""
			if node.IfExists {
				exists = " if exists"
			}
			buf.AstPrintf(node, "%s table%s %v", node.Action, exists, node.FromTables)
		case sqlparser.RenameStr:
			buf.AstPrintf(node, "%s table %v to %v", node.Action, node.FromTables[0], node.ToTables[0])
			for i := 1; i < len(node.FromTables); i++ {
				buf.AstPrintf(node, ", %v to %v", node.FromTables[i], node.ToTables[i])
			}
		case sqlparser.AlterStr:
			if node.PartitionSpec != nil {
				buf.AstPrintf(node, "%s table %v %v", node.Action, node.Table, node.PartitionSpec)
			} else {
				buf.AstPrintf(node, "%s table %v", node.Action, node.Table)
			}
		case sqlparser.FlushStr:
			buf.AstPrintf(node, "%s", node.Action)
		case sqlparser.CreateVindexStr:
			buf.AstPrintf(node, "alter vschema create vindex %v %v", node.Table, node.VindexSpec)
		case sqlparser.DropVindexStr:
			buf.AstPrintf(node, "alter vschema drop vindex %v", node.Table)
		case sqlparser.AddVschemaTableStr:
			buf.AstPrintf(node, "alter vschema add table %v", node.Table)
		case sqlparser.DropVschemaTableStr:
			buf.AstPrintf(node, "alter vschema drop table %v", node.Table)
		case sqlparser.AddColVindexStr:
			buf.AstPrintf(node, "alter vschema on %v add vindex %v (", node.Table, node.VindexSpec.Name)
			for i, col := range node.VindexCols {
				if i != 0 {
					buf.AstPrintf(node, ", %v", col)
				} else {
					buf.AstPrintf(node, "%v", col)
				}
			}
			buf.AstPrintf(node, ")")
			if node.VindexSpec.Type.String() != "" {
				buf.AstPrintf(node, " %v", node.VindexSpec)
			}
		case sqlparser.DropColVindexStr:
			buf.AstPrintf(node, "alter vschema on %v drop vindex %v", node.Table, node.VindexSpec.Name)
		case sqlparser.AddSequenceStr:
			buf.AstPrintf(node, "alter vschema add sequence %v", node.Table)
		case sqlparser.AddAutoIncStr:
			buf.AstPrintf(node, "alter vschema on %v add auto_increment %v", node.Table, node.AutoIncSpec)
		default:
			buf.AstPrintf(node, "%s table %v", node.Action, node.Table)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.OptLike:
		buf.AstPrintf(node, "like %v", node.LikeTable)
		v.rewrittenQuery = buf.String()

	case *sqlparser.PartitionSpec:
		switch node.Action {
		case sqlparser.ReorganizeStr:
			buf.AstPrintf(node, "%s %v into (", node.Action, node.Name)
			var prefix string
			for _, pd := range node.Definitions {
				buf.AstPrintf(node, "%s%v", prefix, pd)
				prefix = ", "
			}
			buf.AstPrintf(node, ")")
		default:
			panic("unimplemented")
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.PartitionDefinition:
		if !node.Maxvalue {
			buf.AstPrintf(node, "partition %v values less than (%v)", node.Name, node.Limit)
		} else {
			buf.AstPrintf(node, "partition %v values less than (maxvalue)", node.Name)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.TableSpec:
		ts := node
		buf.AstPrintf(ts, "(\n")
		for i, col := range ts.Columns {
			if i == 0 {
				buf.AstPrintf(ts, "\t%v", col)
			} else {
				buf.AstPrintf(ts, ",\n\t%v", col)
			}
		}
		for _, idx := range ts.Indexes {
			buf.AstPrintf(ts, ",\n\t%v", idx)
		}
		for _, c := range ts.Constraints {
			buf.AstPrintf(ts, ",\n\t%v", c)
		}

		buf.AstPrintf(ts, "\n)%s", strings.Replace(ts.Options, ", ", ",\n  ", -1))
		v.rewrittenQuery = buf.String()

	case *sqlparser.ColumnDefinition:
		col := node
		buf.AstPrintf(col, "%v %v", col.Name, &col.Type)
		v.rewrittenQuery = buf.String()

	// Format returns a canonical string representation of the type and all relevant options
	case *sqlparser.ColumnType:
		ct := node
		buf.AstPrintf(ct, "%s", ct.Type)

		if ct.Length != nil && ct.Scale != nil {
			buf.AstPrintf(ct, "(%v,%v)", ct.Length, ct.Scale)
		} else if ct.Length != nil {
			buf.AstPrintf(ct, "(%v)", ct.Length)
		}

		if ct.EnumValues != nil {
			buf.AstPrintf(ct, "(%s)", strings.Join(ct.EnumValues, ", "))
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
			buf.AstPrintf(ct, " %s", strings.Join(opts, " "))
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.IndexDefinition:
		idx := node
		buf.AstPrintf(idx, "%v (", idx.Info)
		for i, col := range idx.Columns {
			if i != 0 {
				buf.AstPrintf(idx, ", %v", col.Column)
			} else {
				buf.AstPrintf(idx, "%v", col.Column)
			}
			if col.Length != nil {
				buf.AstPrintf(idx, "(%v)", col.Length)
			}
		}
		buf.AstPrintf(idx, ")")

		for _, opt := range idx.Options {
			buf.AstPrintf(idx, " %s", opt.Name)
			if opt.Using != "" {
				buf.AstPrintf(idx, " %s", opt.Using)
			} else {
				buf.AstPrintf(idx, " %v", opt.Value)
			}
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.IndexInfo:
		ii := node
		if ii.Primary {
			buf.AstPrintf(ii, "%s", ii.Type)
		} else {
			buf.AstPrintf(ii, "%s", ii.Type)
			if !ii.Name.IsEmpty() {
				buf.AstPrintf(ii, " %v", ii.Name)
			}
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.AutoIncSpec:
		buf.AstPrintf(node, "%v ", node.Column)
		buf.AstPrintf(node, "using %v", node.Sequence)
		v.rewrittenQuery = buf.String()

	case *sqlparser.VindexSpec:
		buf.AstPrintf(node, "using %v", node.Type)

		numParams := len(node.Params)
		if numParams != 0 {
			buf.AstPrintf(node, " with ")
			for i, p := range node.Params {
				if i != 0 {
					buf.AstPrintf(node, ", ")
				}
				buf.AstPrintf(node, "%v", p)
			}
		}
		v.rewrittenQuery = buf.String()

	case sqlparser.VindexParam:
		buf.AstPrintf(node, "%s=%s", node.Key.String(), node.Val)
		v.rewrittenQuery = buf.String()

	case *sqlparser.ConstraintDefinition:
		c := node
		if c.Name != "" {
			buf.AstPrintf(c, "constraint %s ", c.Name)
		}
		c.Details.Format(buf)
		v.rewrittenQuery = buf.String()

	case sqlparser.ReferenceAction:
		a := node
		switch a {
		case sqlparser.Restrict:
			buf.WriteString("restrict")
		case sqlparser.Cascade:
			buf.WriteString("cascade")
		case sqlparser.NoAction:
			buf.WriteString("no action")
		case sqlparser.SetNull:
			buf.WriteString("set null")
		case sqlparser.SetDefault:
			buf.WriteString("set default")
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.ForeignKeyDefinition:
		f := node
		buf.AstPrintf(f, "foreign key %v references %v %v", f.Source, f.ReferencedTable, f.ReferencedColumns)
		if f.OnDelete != sqlparser.DefaultAction {
			buf.AstPrintf(f, " on delete %v", f.OnDelete)
		}
		if f.OnUpdate != sqlparser.DefaultAction {
			buf.AstPrintf(f, " on update %v", f.OnUpdate)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.Show:
		nodeType := strings.ToLower(node.Type)
		if (nodeType == "tables" || nodeType == "columns" || nodeType == "fields" || nodeType == "index" || nodeType == "keys" || nodeType == "indexes") && node.ShowTablesOpt != nil {
			opt := node.ShowTablesOpt
			if node.Extended != "" {
				buf.AstPrintf(node, "show %s%s", node.Extended, nodeType)
			} else {
				buf.AstPrintf(node, "show %s%s", opt.Full, nodeType)
			}
			if (nodeType == "columns" || nodeType == "fields") && node.HasOnTable() {
				buf.AstPrintf(node, " from %v", node.OnTable)
			}
			if (nodeType == "index" || nodeType == "keys" || nodeType == "indexes") && node.HasOnTable() {
				buf.AstPrintf(node, " from %v", node.OnTable)
			}
			if opt.DbName != "" {
				buf.AstPrintf(node, " from %s", opt.DbName)
			}
			buf.AstPrintf(node, "%v", opt.Filter)
			return nil
		}
		if node.Scope == "" {
			buf.AstPrintf(node, "show %s", nodeType)
		} else {
			buf.AstPrintf(node, "show %s %s", node.Scope, nodeType)
		}
		if node.HasOnTable() {
			buf.AstPrintf(node, " on %v", node.OnTable)
		}
		if nodeType == "collation" && node.ShowCollationFilterOpt != nil {
			buf.AstPrintf(node, " where %v", node.ShowCollationFilterOpt)
		}
		if nodeType == "charset" && node.ShowTablesOpt != nil {
			buf.AstPrintf(node, "%v", node.ShowTablesOpt.Filter)
		}
		if node.HasTable() {
			buf.AstPrintf(node, " %v", node.Table)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.ShowFilter:
		if node == nil {
			return nil
		}
		if node.Like != "" {
			buf.AstPrintf(node, " like '%s'", node.Like)
		} else {
			buf.AstPrintf(node, " where %v", node.Filter)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.Use:
		if node.DBName.GetRawVal() != "" {
			buf.AstPrintf(node, "use %v", node.DBName)
		} else {
			buf.AstPrintf(node, "use")
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.Commit:
		buf.WriteString("commit")
		v.rewrittenQuery = buf.String()

	case *sqlparser.Begin:
		buf.WriteString("begin")
		v.rewrittenQuery = buf.String()

	case *sqlparser.Rollback:
		buf.WriteString("rollback")
		v.rewrittenQuery = buf.String()

	case *sqlparser.SRollback:
		buf.AstPrintf(node, "rollback to %v", node.Name)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Savepoint:
		buf.AstPrintf(node, "savepoint %v", node.Name)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Release:
		buf.AstPrintf(node, "release savepoint %v", node.Name)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Explain:
		format := ""
		switch node.Type {
		case "": // do nothing
		case sqlparser.AnalyzeStr:
			format = sqlparser.AnalyzeStr + " "
		default:
			format = "format = " + node.Type + " "
		}
		buf.AstPrintf(node, "explain %s%v", format, node.Statement)
		v.rewrittenQuery = buf.String()

	case *sqlparser.OtherRead:
		buf.WriteString("otherread")
		v.rewrittenQuery = buf.String()

	case *sqlparser.DescribeTable:
		buf.WriteString("describetable")
		v.rewrittenQuery = buf.String()

	case *sqlparser.OtherAdmin:
		buf.WriteString("otheradmin")
		v.rewrittenQuery = buf.String()

	case sqlparser.Comments:
		for _, c := range node {
			buf.AstPrintf(node, "%s ", c)
		}
		v.rewrittenQuery = buf.String()

	case sqlparser.SelectExprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.StarExpr:
		if !node.TableName.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.TableName)
		}
		buf.AstPrintf(node, "*")
		v.rewrittenQuery = buf.String()

	case *sqlparser.AliasedExpr:
		buf.AstPrintf(node, "%v", node.Expr)
		if !node.As.IsEmpty() {
			buf.AstPrintf(node, " as %v", node.As)
		}
		v.rewrittenQuery = buf.String()

	case sqlparser.Nextval:
		buf.AstPrintf(node, "next %v values", node.Expr)
		v.rewrittenQuery = buf.String()

	case sqlparser.Columns:
		if node == nil {
			return nil
		}
		prefix := "("
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		buf.WriteString(")")
		v.rewrittenQuery = buf.String()

	case sqlparser.Partitions:
		if node == nil {
			return nil
		}
		prefix := " partition ("
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		buf.WriteString(")")
		v.rewrittenQuery = buf.String()

	case *sqlparser.TableValuedFuncTableExpr:
		buf.AstPrintf(node, "%v", node)
		v.rewrittenQuery = buf.String()
		// return nil

	case sqlparser.TableExprs:
		var exprs []string
		for i, n := range node {
			var s string
			// TODO: fix this hack
			//     prevents indirect expansion
			//     except on first TableExpr
			if i == 0 {
				n.Accept(v)
				s = v.GetRewrittenQuery()
			} else {
				nodeVis := NewFromRewriteAstVisitor(v.annotatedAST, "", true, v.sqlSystem, v.formatter, v.namespaceCollection, v.annotations, v.dc)
				n.Accept(nodeVis)
				s = nodeVis.GetRewrittenQuery()
			}
			exprs = append(exprs, s)
		}
		v.rewrittenQuery = strings.Join(exprs, ", ")

	case *sqlparser.AliasedTableExpr:
		var exprStr, partitionStr string
		if node.Expr != nil {
			anCtx, ok := v.annotations[node]
			if !ok {
				return fmt.Errorf("cannot find annotated tabulation for table object")
			}
			if anCtx.GetTableMeta().IsOnClauseHoistable() {
				v.hoistedOnClauseTables = append(v.hoistedOnClauseTables, node)
			}
			if indirect, isIndirect := anCtx.GetTableMeta().GetIndirect(); isIndirect {
				//
				name := indirect.GetName()
				indirectType := indirect.GetType()
				switch indirectType {
				case astindirect.ViewType:
					templateString := fmt.Sprintf(` ( %%s ) AS "%s" `, name)
					v.rewrittenQuery = templateString
					v.indirectContexts = append(v.indirectContexts, indirect.GetSelectContext())
				case astindirect.SubqueryType:
					templateString := ` ( %s ) `
					v.rewrittenQuery = templateString
					v.indirectContexts = append(v.indirectContexts, indirect.GetSelectContext())
				case astindirect.MaterializedViewType, astindirect.PhysicalTableType:
					refString := fmt.Sprintf(` %s `, name)
					isQuoted, _ := regexp.MatchString(`^".*"$`, name)
					isPostgres := v.sqlSystem.GetName() == constants.SQLDialectPostgres
					isRelationExported := v.sqlSystem.IsRelationExported(name)
					if !isQuoted && !(isPostgres && isRelationExported) {
						refString = fmt.Sprintf(` "%s" `, name)
					}
					alias := ""
					if alias != "" {
						refString = fmt.Sprintf(` %s AS "%s" `, refString, alias)
					}
					v.rewrittenQuery = refString
				default:
					return fmt.Errorf("unknown indirect type '%T'", indirectType)
				}
			} else {
				switch ex := node.Expr.(type) {
				case sqlparser.TableName:
					if ex.IsEmpty() {
						return nil
					}
					_, isSQLDataSource := anCtx.GetTableMeta().GetSQLDataSource()
					var tblStr string
					// TODO: clean this garbage up
					if isSQLDataSource && !v.isAvoidSQLSourceNaming {
						tblStr = anCtx.GetHIDs().GetStackQLTableName()
					} else {
						dbTbl, err := v.dc.GetCurrentTable(anCtx.GetHIDs())
						if err != nil {
							return err
						}
						tblStr = dbTbl.GetName()
					}
					fqtn, err := v.sqlSystem.GetFullyQualifiedTableName(tblStr)
					v.rewrittenQuery = fqtn
					if err != nil {
						return err
					}
				default:
					err := node.Expr.Accept(v)
					if err != nil {
						return err
					}
				}
			}
			exprStr = v.GetRewrittenQuery()
		}
		if node.Partitions != nil {
			node.Partitions.Accept(v)
			partitionStr = v.GetRewrittenQuery()
		}
		q := fmt.Sprintf("%s%s", exprStr, partitionStr)
		if !node.As.IsEmpty() {
			node.As.Accept(v)
			asStr := v.GetRewrittenQuery()
			q = fmt.Sprintf("%s as %v", q, asStr)
		}
		if node.Hints != nil {
			node.Hints.Accept(v)
			// Hint node provides the space padding.
			hintStr := v.GetRewrittenQuery()
			q = fmt.Sprintf("%s%v", q, hintStr)
		}
		v.rewrittenQuery = q

	case sqlparser.TableNames:
		var prefix string
		for _, n := range node {
			n.Accept(v)
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		v.rewrittenQuery = buf.String()

	case sqlparser.TableName:
		return fmt.Errorf("FromVisitor should not be able to assess an unadorned TableName")

	case *sqlparser.ParenTableExpr:
		buf.AstPrintf(node, "(%v)", node.Exprs)
		v.rewrittenQuery = buf.String()

	case *sqlparser.NativeQuery:
		buf.AstPrintf(node, "NATIVEQUERY '&s'", strings.ReplaceAll(node.QueryString, "'", "''"))

	case sqlparser.JoinCondition:
		v.Visit(node.On)
		if node.On != nil {
			buf.AstPrintf(node, " %v", node.On)
		}
		if node.Using != nil {
			buf.AstPrintf(node, " using %v", node.Using)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.JoinTableExpr:
		lVis := NewFromRewriteAstVisitor(v.annotatedAST, "", true, v.sqlSystem, v.formatter, v.namespaceCollection, v.annotations, v.dc)
		lVis.SetAvoidSQLSourceNaming(v.isAvoidSQLSourceNaming)
		node.LeftExpr.Accept(lVis)
		rVis := NewFromRewriteAstVisitor(v.annotatedAST, "", true, v.sqlSystem, v.formatter, v.namespaceCollection, v.annotations, v.dc)
		rVis.SetAvoidSQLSourceNaming(v.isAvoidSQLSourceNaming)
		node.RightExpr.Accept(rVis)
		conditionVis := NewFromRewriteAstVisitor(v.annotatedAST, "", true, v.sqlSystem, v.formatter, v.namespaceCollection, v.annotations, v.dc)
		conditionVis.SetAvoidSQLSourceNaming(v.isAvoidSQLSourceNaming)
		node.Condition.Accept(conditionVis)
		lhsHoistedIntoOn := lVis.GetHoistedOnClauseTables()
		rhsHoistedIntoOn := rVis.GetHoistedOnClauseTables()
		lhsIndirectContexts := lVis.GetIndirectContexts()
		rhsIndirectContexts := rVis.GetIndirectContexts()
		v.indirectContexts = append(v.indirectContexts, lhsIndirectContexts...)
		v.indirectContexts = append(v.indirectContexts, rhsIndirectContexts...)
		// accumulate hoisted tables
		v.hoistedOnClauseTables = append(v.hoistedOnClauseTables, lhsHoistedIntoOn...)
		v.hoistedOnClauseTables = append(v.hoistedOnClauseTables, rhsHoistedIntoOn...)
		if len(v.hoistedOnClauseTables) > 0 {
			v.rewrittenQuery = fmt.Sprintf("%s %s %s ON ( %%s ) AND %s", lVis.GetRewrittenQuery(), node.Join, rVis.GetRewrittenQuery(), conditionVis.GetRewrittenQuery())
		} else {
			v.rewrittenQuery = fmt.Sprintf("%s %s %s ON %s", lVis.GetRewrittenQuery(), node.Join, rVis.GetRewrittenQuery(), conditionVis.GetRewrittenQuery())
		}

	case *sqlparser.IndexHints:
		buf.AstPrintf(node, " %sindex ", node.Type)
		if len(node.Indexes) == 0 {
			buf.AstPrintf(node, "()")
		} else {
			prefix := "("
			for _, n := range node.Indexes {
				buf.AstPrintf(node, "%s%v", prefix, n)
				prefix = ", "
			}
			buf.AstPrintf(node, ")")
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.Where:
		if node == nil || node.Expr == nil {
			return nil
		}
		buf.AstPrintf(node, " %s %v", node.Type, node.Expr)
		v.rewrittenQuery = buf.String()

	case sqlparser.Exprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.AndExpr:
		buf.AstPrintf(node, "%v and %v", node.Left, node.Right)
		v.rewrittenQuery = buf.String()

	case *sqlparser.OrExpr:
		buf.AstPrintf(node, "%v or %v", node.Left, node.Right)
		v.rewrittenQuery = buf.String()

	case *sqlparser.XorExpr:
		buf.AstPrintf(node, "%v xor %v", node.Left, node.Right)
		v.rewrittenQuery = buf.String()

	case *sqlparser.NotExpr:
		buf.AstPrintf(node, "not %v", node.Expr)
		v.rewrittenQuery = buf.String()

	case *sqlparser.ComparisonExpr:
		lVis := NewFromRewriteAstVisitor(v.annotatedAST, "", true, v.sqlSystem, v.formatter, v.namespaceCollection, v.annotations, v.dc)
		node.Left.Accept(lVis)
		rVis := NewFromRewriteAstVisitor(v.annotatedAST, "", true, v.sqlSystem, v.formatter, v.namespaceCollection, v.annotations, v.dc)
		node.Right.Accept(rVis)
		buf.AstPrintf(node, "%s %s %s", lVis.GetRewrittenQuery(), node.Operator, rVis.GetRewrittenQuery())
		if node.Escape != nil {
			buf.AstPrintf(node, " escape %v", node.Escape)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.RangeCond:
		buf.AstPrintf(node, "%v %s %v and %v", node.Left, node.Operator, node.From, node.To)
		v.rewrittenQuery = buf.String()

	case *sqlparser.IsExpr:
		buf.AstPrintf(node, "%v %s", node.Expr, node.Operator)
		v.rewrittenQuery = buf.String()

	case *sqlparser.ExistsExpr:
		buf.AstPrintf(node, "exists %v", node.Subquery)
		v.rewrittenQuery = buf.String()

	case *sqlparser.SQLVal:
		switch node.Type {
		case sqlparser.StrVal:
			sqltypes.MakeTrusted(sqltypes.VarBinary, node.Val).EncodeSQL(buf)
		case sqlparser.IntVal, sqlparser.FloatVal, sqlparser.HexNum:
			buf.AstPrintf(node, "%s", node.Val)
		case sqlparser.HexVal:
			buf.AstPrintf(node, "X'%s'", node.Val)
		case sqlparser.BitVal:
			buf.AstPrintf(node, "B'%s'", node.Val)
		case sqlparser.ValArg:
			buf.WriteArg(string(node.Val))
		default:
			panic("unexpected")
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.NullVal:
		buf.AstPrintf(node, "null")
		v.rewrittenQuery = buf.String()

	case sqlparser.BoolVal:
		if node {
			buf.AstPrintf(node, "true")
		} else {
			buf.AstPrintf(node, "false")
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.ColName:
		if !node.Qualifier.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.Qualifier)
		}
		buf.AstPrintf(node, "%v", node.Name)
		v.rewrittenQuery = buf.String()

	case sqlparser.ValTuple:
		buf.AstPrintf(node, "(%v)", sqlparser.Exprs(node))
		v.rewrittenQuery = buf.String()

	case *sqlparser.Subquery:
		buf.AstPrintf(node, "(%v)", node.Select)
		v.rewrittenQuery = buf.String()

	case sqlparser.ListArg:
		buf.WriteArg(string(node))
		v.rewrittenQuery = buf.String()

	case *sqlparser.BinaryExpr:
		buf.AstPrintf(node, "%v %s %v", node.Left, node.Operator, node.Right)
		v.rewrittenQuery = buf.String()

	case *sqlparser.UnaryExpr:
		if _, unary := node.Expr.(*sqlparser.UnaryExpr); unary {
			// They have same precedence so parenthesis is not required.
			buf.AstPrintf(node, "%s %v", node.Operator, node.Expr)
			return nil
		}
		buf.AstPrintf(node, "%s%v", node.Operator, node.Expr)
		v.rewrittenQuery = buf.String()

	case *sqlparser.IntervalExpr:
		buf.AstPrintf(node, "interval %v %s", node.Expr, node.Unit)
		v.rewrittenQuery = buf.String()

	case *sqlparser.TimestampFuncExpr:
		buf.AstPrintf(node, "%s(%s, %v, %v)", node.Name, node.Unit, node.Expr1, node.Expr2)
		v.rewrittenQuery = buf.String()

	case *sqlparser.CurTimeFuncExpr:
		buf.AstPrintf(node, "%s(%v)", node.Name.String(), node.Fsp)
		v.rewrittenQuery = buf.String()

	case *sqlparser.CollateExpr:
		buf.AstPrintf(node, "%v collate %s", node.Expr, node.Charset)
		v.rewrittenQuery = buf.String()

	case *sqlparser.ExecSubquery:
		if node.Exec == nil {
			return fmt.Errorf("cannont accomodate nil exec table container")
		}
		// if v.shouldCollectTables {
		// 	switch te := node.Exec.MethodName.(type) {
		// 	case sqlparser.TableName:
		// 		v.tablesCited[node] = te
		// 	}
		// }
		s := astformat.String(node.Exec.MethodName, v.sqlSystem.GetASTFormatter())
		v.rewrittenQuery = s

	case *sqlparser.FuncExpr:
		newNode, err := v.sqlSystem.GetASTFuncRewriter().RewriteFunc(node)
		if err != nil {
			return err
		}
		node.Distinct = newNode.Distinct
		node.Exprs = newNode.Exprs
		node.Name = newNode.Name
		var distinct string
		if node.Distinct {
			distinct = "distinct "
		}
		if !node.Qualifier.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.Qualifier)
		}
		// Function names should not be back-quoted even
		// if they match a reserved word, only if they contain illegal characters
		funcName := node.Name.String()

		if sqlparser.ContainEscapableChars(funcName, sqlparser.NoAt) {
			sqlparser.WriteEscapedString(buf, funcName)
		} else {
			buf.WriteString(funcName)
		}
		buf.AstPrintf(node, "(%s%v)", distinct, node.Exprs)
		v.rewrittenQuery = buf.String()

	case *sqlparser.GroupConcatExpr:
		buf.AstPrintf(node, "group_concat(%s%v%v%s%v)", node.Distinct, node.Exprs, node.OrderBy, node.Separator, node.Limit)
		v.rewrittenQuery = buf.String()

	case *sqlparser.ValuesFuncExpr:
		buf.AstPrintf(node, "values(%v)", node.Name)
		v.rewrittenQuery = buf.String()

	case *sqlparser.SubstrExpr:
		var val interface{}
		if node.Name != nil {
			val = node.Name
		} else {
			val = node.StrVal
		}

		if node.To == nil {
			buf.AstPrintf(node, "substr(%v, %v)", val, node.From)
		} else {
			buf.AstPrintf(node, "substr(%v, %v, %v)", val, node.From, node.To)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.ConvertExpr:
		buf.AstPrintf(node, "convert(%v, %v)", node.Expr, node.Type)
		v.rewrittenQuery = buf.String()

	case *sqlparser.ConvertUsingExpr:
		buf.AstPrintf(node, "convert(%v using %s)", node.Expr, node.Type)
		v.rewrittenQuery = buf.String()

	case *sqlparser.ConvertType:
		buf.AstPrintf(node, "%s", node.Type)
		if node.Length != nil {
			buf.AstPrintf(node, "(%v", node.Length)
			if node.Scale != nil {
				buf.AstPrintf(node, ", %v", node.Scale)
			}
			buf.AstPrintf(node, ")")
		}
		if node.Charset != "" {
			buf.AstPrintf(node, "%s %s", node.Operator, node.Charset)
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.MatchExpr:
		buf.AstPrintf(node, "match(%v) against (%v%s)", node.Columns, node.Expr, node.Option)
		v.rewrittenQuery = buf.String()

	case *sqlparser.CaseExpr:
		buf.AstPrintf(node, "case sqlparser.")
		if node.Expr != nil {
			buf.AstPrintf(node, "%v ", node.Expr)
		}
		for _, when := range node.Whens {
			buf.AstPrintf(node, "%v ", when)
		}
		if node.Else != nil {
			buf.AstPrintf(node, "else %v ", node.Else)
		}
		buf.AstPrintf(node, "end")
		v.rewrittenQuery = buf.String()

	case *sqlparser.Default:
		buf.AstPrintf(node, "default")
		if node.ColName != "" {
			buf.WriteString("(")
			sqlparser.FormatID(buf, node.ColName, strings.ToLower(node.ColName), sqlparser.NoAt)
			buf.WriteString(")")
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.When:
		buf.AstPrintf(node, "when %v then %v", node.Cond, node.Val)
		v.rewrittenQuery = buf.String()

	case sqlparser.GroupBy:
		var colz []string
		for _, n := range node {
			switch n := n.(type) {
			case *sqlparser.ColName:
				if n.Qualifier.GetRawVal() == "" {
					colz = append(colz, v.sqlSystem.DelimitGroupByColumn(n.Name.GetRawVal()))
				} else {
					colz = append(colz, fmt.Sprintf(`%s.%s`, v.sqlSystem.DelimitGroupByColumn(n.Qualifier.GetRawVal()), v.sqlSystem.DelimitGroupByColumn(n.Name.GetRawVal())))
				}
			default:
				colz = append(colz, sqlparser.String(n))
			}
		}
		if len(colz) > 0 {
			v.rewrittenQuery = fmt.Sprintf(" group by %s", strings.Join(colz, ", "))
		} else {
			v.rewrittenQuery = ""
		}

	case sqlparser.OrderBy:
		var colz []string
		for _, orderNode := range node {
			switch n := orderNode.Expr.(type) {
			case *sqlparser.ColName:
				if n.Qualifier.GetRawVal() == "" {
					colz = append(colz, fmt.Sprintf(`%s %s`, v.sqlSystem.DelimitOrderByColumn(n.Name.GetRawVal()), orderNode.Direction))
				} else {
					colz = append(colz, fmt.Sprintf(`%s.%s %s`, v.sqlSystem.DelimitOrderByColumn(n.Qualifier.GetRawVal()), v.sqlSystem.DelimitOrderByColumn(n.Name.GetRawVal()), orderNode.Direction))
				}
			default:
				colz = append(colz, fmt.Sprintf("%s %s", sqlparser.String(n), orderNode.Direction))
			}
		}
		if len(colz) > 0 {
			v.rewrittenQuery = fmt.Sprintf(" order by %s", strings.Join(colz, ", "))
		} else {
			v.rewrittenQuery = ""
		}

	case *sqlparser.Order:
		if node, ok := node.Expr.(*sqlparser.NullVal); ok {
			buf.AstPrintf(node, "%v", node)
			return nil
		}
		if node, ok := node.Expr.(*sqlparser.FuncExpr); ok {
			if node.Name.Lowered() == "rand" {
				buf.AstPrintf(node, "%v", node)
				return nil
			}
		}

		buf.AstPrintf(node, "%v %s", node.Expr, node.Direction)
		v.rewrittenQuery = buf.String()

	case *sqlparser.Limit:
		if node == nil {
			return nil
		}
		buf.AstPrintf(node, " limit ")
		if node.Offset != nil {
			buf.AstPrintf(node, "%v, ", node.Offset)
		}
		buf.AstPrintf(node, "%v", node.Rowcount)
		v.rewrittenQuery = buf.String()

	case sqlparser.Values:
		prefix := "values "
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		v.rewrittenQuery = buf.String()

	case sqlparser.UpdateExprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.UpdateExpr:
		buf.AstPrintf(node, "%v = %v", node.Name, node.Expr)
		v.rewrittenQuery = buf.String()

	case sqlparser.SetExprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		v.rewrittenQuery = buf.String()

	case *sqlparser.SetExpr:
		if node.Scope != "" {
			buf.WriteString(node.Scope)
			buf.WriteString(" ")
		}
		// We don't have to backtick set variable names.
		switch {
		case node.Name.EqualString("charset") || node.Name.EqualString("names"):
			buf.AstPrintf(node, "%s %v", node.Name.String(), node.Expr)
		case node.Name.EqualString(sqlparser.TransactionStr):
			sqlVal := node.Expr.(*sqlparser.SQLVal)
			buf.AstPrintf(node, "%s %s", node.Name.String(), strings.ToLower(string(sqlVal.Val)))
		default:
			buf.AstPrintf(node, "%v = %v", node.Name, node.Expr)
		}
		v.rewrittenQuery = buf.String()

	case sqlparser.OnDup:
		if node == nil {
			return nil
		}
		buf.AstPrintf(node, " on duplicate key update %v", sqlparser.UpdateExprs(node))
		v.rewrittenQuery = buf.String()

	case sqlparser.ColIdent:
		for i := sqlparser.NoAt; i < node.GetAtCount(); i++ {
			buf.WriteByte('@')
		}
		sqlparser.FormatID(buf, node.GetRawVal(), node.Lowered(), node.GetAtCount())
		v.rewrittenQuery = buf.String()

	case sqlparser.TableIdent:
		tn := node.GetRawVal()
		v.rewrittenQuery = tn

	case *sqlparser.IsolationLevel:
		buf.WriteString("isolation level " + node.Level)
		v.rewrittenQuery = buf.String()

	case *sqlparser.AccessMode:
		buf.WriteString(node.Mode)
		v.rewrittenQuery = buf.String()
	}
	return nil
}
