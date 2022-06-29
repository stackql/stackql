package astvisit

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/parserutil"
	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/vt/sqlparser"
)

// Extracts "parameters" with null values for subsequent analyses.
type PlaceholderParamAstVisitor struct {
	params parserutil.ParameterMap
}

func NewPlaceholderParamAstVisitor(iDColumnName string, shouldCollectTables bool) *PlaceholderParamAstVisitor {
	return &PlaceholderParamAstVisitor{
		params: parserutil.NewParameterMap(),
	}
}

func (v *PlaceholderParamAstVisitor) GetParameters() parserutil.ParameterMap {
	return v.params
}

func (v *PlaceholderParamAstVisitor) GetStringifiedParameters() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range v.params.GetMap() {
		rv[k.String()] = v
	}
	return rv
}

func (v *PlaceholderParamAstVisitor) Visit(node sqlparser.SQLNode) error {
	buf := sqlparser.NewTrackedBuffer(nil)
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
			node.From.Accept(v)
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
		if len(node.Columns) > 0 {
			err := node.Columns.Accept(v)
			if err != nil {
				return err
			}
		}
		buf.AstPrintf(node, "%s %v%sinto %v%v%v %v%v",
			node.Action,
			node.Comments, node.Ignore,
			node.Table, node.Partitions, node.Columns, node.Rows, node.OnDup)

	case *sqlparser.Update:
		buf.AstPrintf(node, "update %v%s%v set %v%v%v%v",
			node.Comments, node.Ignore, node.TableExprs,
			node.Exprs, node.Where, node.OrderBy, node.Limit)

	case *sqlparser.Delete:
		buf.AstPrintf(node, "delete %v", node.Comments)
		if node.Targets != nil {
			buf.AstPrintf(node, "%v ", node.Targets)
		}
		buf.AstPrintf(node, "from %v%v%v%v%v", node.TableExprs, node.Partitions, node.Where, node.OrderBy, node.Limit)

	case *sqlparser.Set:
		buf.AstPrintf(node, "set %v%v", node.Comments, node.Exprs)

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

	case *sqlparser.OptLike:
		buf.AstPrintf(node, "like %v", node.LikeTable)

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

	case *sqlparser.PartitionDefinition:
		if !node.Maxvalue {
			buf.AstPrintf(node, "partition %v values less than (%v)", node.Name, node.Limit)
		} else {
			buf.AstPrintf(node, "partition %v values less than (maxvalue)", node.Name)
		}

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

	case *sqlparser.ColumnDefinition:
		col := node
		buf.AstPrintf(col, "%v %v", col.Name, &col.Type)

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
			buf.AstPrintf(ct, " %s", strings.Join(opts, " "))
		}

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

	case *sqlparser.AutoIncSpec:
		buf.AstPrintf(node, "%v ", node.Column)
		buf.AstPrintf(node, "using %v", node.Sequence)

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

	case sqlparser.VindexParam:
		buf.AstPrintf(node, "%s=%s", node.Key.String(), node.Val)

	case *sqlparser.ConstraintDefinition:
		c := node
		if c.Name != "" {
			buf.AstPrintf(c, "constraint %s ", c.Name)
		}
		c.Details.Format(buf)

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

	case *sqlparser.ForeignKeyDefinition:
		f := node
		buf.AstPrintf(f, "foreign key %v references %v %v", f.Source, f.ReferencedTable, f.ReferencedColumns)
		if f.OnDelete != sqlparser.DefaultAction {
			buf.AstPrintf(f, " on delete %v", f.OnDelete)
		}
		if f.OnUpdate != sqlparser.DefaultAction {
			buf.AstPrintf(f, " on update %v", f.OnUpdate)
		}

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

	case *sqlparser.ShowFilter:
		if node == nil {
			return nil
		}
		if node.Like != "" {
			buf.AstPrintf(node, " like '%s'", node.Like)
		} else {
			buf.AstPrintf(node, " where %v", node.Filter)
		}

	case *sqlparser.Use:
		if node.DBName.GetRawVal() != "" {
			buf.AstPrintf(node, "use %v", node.DBName)
		} else {
			buf.AstPrintf(node, "use")
		}

	case *sqlparser.Commit:
		buf.WriteString("commit")

	case *sqlparser.Begin:
		buf.WriteString("begin")

	case *sqlparser.Rollback:
		buf.WriteString("rollback")

	case *sqlparser.SRollback:
		buf.AstPrintf(node, "rollback to %v", node.Name)

	case *sqlparser.Savepoint:
		buf.AstPrintf(node, "savepoint %v", node.Name)

	case *sqlparser.Release:
		buf.AstPrintf(node, "release savepoint %v", node.Name)

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

	case *sqlparser.OtherRead:
		buf.WriteString("otherread")

	case *sqlparser.DescribeTable:
		buf.WriteString("describetable")

	case *sqlparser.OtherAdmin:
		buf.WriteString("otheradmin")

	case sqlparser.Comments:
		for _, c := range node {
			buf.AstPrintf(node, "%s ", c)
		}

	case sqlparser.SelectExprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}

	case *sqlparser.StarExpr:
		if !node.TableName.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.TableName)
		}
		buf.AstPrintf(node, "*")

	case *sqlparser.AliasedExpr:
		buf.AstPrintf(node, "%v", node.Expr)
		if !node.As.IsEmpty() {
			buf.AstPrintf(node, " as %v", node.As)
		}

	case sqlparser.Nextval:
		buf.AstPrintf(node, "next %v values", node.Expr)

	case sqlparser.Columns:
		if node == nil {
			return nil
		}
		prefix := "("
		for _, n := range node {
			err := n.Accept(v)
			if err != nil {
				return err
			}
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}
		buf.WriteString(")")

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

	case sqlparser.TableExprs:
		for _, n := range node {
			n.Accept(v)
		}

	case *sqlparser.AliasedTableExpr:
		if node.Expr != nil {
			node.Expr.Accept(v)
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
		var prefix string
		for _, n := range node {
			n.Accept(v)
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}

	case sqlparser.TableName:
		if node.IsEmpty() {
			return nil
		}
		buf.AstPrintf(node, `"`)
		if !node.QualifierThird.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.QualifierThird)
		}
		if !node.QualifierSecond.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.QualifierSecond)
		}
		if !node.Qualifier.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.Qualifier)
		}
		buf.AstPrintf(node, "%v", node.Name)
		buf.AstPrintf(node, `"`)

	case *sqlparser.ParenTableExpr:
		buf.AstPrintf(node, "(%v)", node.Exprs)

	case sqlparser.JoinCondition:
		if node.On != nil {
			buf.AstPrintf(node, " on %v", node.On)
		}
		if node.Using != nil {
			buf.AstPrintf(node, " using %v", node.Using)
		}

	case *sqlparser.JoinTableExpr:
		node.LeftExpr.Accept(v)
		node.LeftExpr.Accept(v)
		buf.AstPrintf(node, "%v %s %v%v", node.LeftExpr, node.Join, node.RightExpr, node.Condition)

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

	case *sqlparser.Where:
		if node == nil || node.Expr == nil {
			return nil
		}
		buf.AstPrintf(node, " %s %v", node.Type, node.Expr)
		return node.Expr.Accept(v)

	case sqlparser.Exprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
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
		switch lt := node.Left.(type) {
		case *sqlparser.ColName:
			switch rt := node.Right.(type) {
			case *sqlparser.SQLVal:
				k, err := parserutil.NewUnknownTypeColumnarReference(lt)
				if err != nil {
					return err
				}
				v.params.Set(k, parserutil.NewComparisonParameterMetadata(
					node,
					rt,
				))
			default:
			}
		default:
			switch rt := node.Right.(type) {
			case *sqlparser.SQLVal:
			case *sqlparser.ColName:
				k, err := parserutil.NewUnknownTypeColumnarReference(rt)
				if err != nil {
					return err
				}
				v.params.Set(k, parserutil.NewComparisonParameterMetadata(
					node,
					lt,
				))
			default:
			}
		}
		return nil

	case *sqlparser.RangeCond:
		buf.AstPrintf(node, "%v %s %v and %v", node.Left, node.Operator, node.From, node.To)

	case *sqlparser.IsExpr:
		return node.Expr.Accept(v)

	case *sqlparser.ExistsExpr:
		return nil

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

	case *sqlparser.NullVal:
		buf.AstPrintf(node, "null")

	case sqlparser.BoolVal:
		if node {
			buf.AstPrintf(node, "true")
		} else {
			buf.AstPrintf(node, "false")
		}

	case *sqlparser.ColName:
		if !node.Qualifier.IsEmpty() {
			buf.AstPrintf(node, "%v.", node.Qualifier)
		}
		buf.AstPrintf(node, "%v", node.Name)

	case sqlparser.ValTuple:
		buf.AstPrintf(node, "(%v)", sqlparser.Exprs(node))

	case *sqlparser.Subquery:
		buf.AstPrintf(node, "(%v)", node.Select)

	case sqlparser.ListArg:
		buf.WriteArg(string(node))

	case *sqlparser.BinaryExpr:
		buf.AstPrintf(node, "%v %s %v", node.Left, node.Operator, node.Right)

	case *sqlparser.UnaryExpr:
		if _, unary := node.Expr.(*sqlparser.UnaryExpr); unary {
			// They have same precedence so parenthesis is not required.
			buf.AstPrintf(node, "%s %v", node.Operator, node.Expr)
			return nil
		}
		buf.AstPrintf(node, "%s%v", node.Operator, node.Expr)

	case *sqlparser.IntervalExpr:
		buf.AstPrintf(node, "interval %v %s", node.Expr, node.Unit)

	case *sqlparser.TimestampFuncExpr:
		buf.AstPrintf(node, "%s(%s, %v, %v)", node.Name, node.Unit, node.Expr1, node.Expr2)

	case *sqlparser.CurTimeFuncExpr:
		buf.AstPrintf(node, "%s(%v)", node.Name.String(), node.Fsp)

	case *sqlparser.CollateExpr:
		buf.AstPrintf(node, "%v collate %s", node.Expr, node.Charset)

	case *sqlparser.FuncExpr:
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

	case *sqlparser.GroupConcatExpr:
		buf.AstPrintf(node, "group_concat(%s%v%v%s%v)", node.Distinct, node.Exprs, node.OrderBy, node.Separator, node.Limit)

	case *sqlparser.ValuesFuncExpr:
		buf.AstPrintf(node, "values(%v)", node.Name)

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

	case *sqlparser.ConvertExpr:
		buf.AstPrintf(node, "convert(%v, %v)", node.Expr, node.Type)

	case *sqlparser.ConvertUsingExpr:
		buf.AstPrintf(node, "convert(%v using %s)", node.Expr, node.Type)

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

	case *sqlparser.MatchExpr:
		buf.AstPrintf(node, "match(%v) against (%v%s)", node.Columns, node.Expr, node.Option)

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

	case *sqlparser.Default:
		buf.AstPrintf(node, "default")
		if node.ColName != "" {
			buf.WriteString("(")
			sqlparser.FormatID(buf, node.ColName, strings.ToLower(node.ColName), sqlparser.NoAt)
			buf.WriteString(")")
		}

	case *sqlparser.When:
		buf.AstPrintf(node, "when %v then %v", node.Cond, node.Val)

	case sqlparser.GroupBy:
		prefix := " group by "
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}

	case sqlparser.OrderBy:
		prefix := " order by "
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
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

	case *sqlparser.Limit:
		if node == nil {
			return nil
		}
		buf.AstPrintf(node, " limit ")
		if node.Offset != nil {
			buf.AstPrintf(node, "%v, ", node.Offset)
		}
		buf.AstPrintf(node, "%v", node.Rowcount)

	case sqlparser.Values:
		prefix := "values "
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}

	case sqlparser.UpdateExprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}

	case *sqlparser.UpdateExpr:
		buf.AstPrintf(node, "%v = %v", node.Name, node.Expr)

	case sqlparser.SetExprs:
		var prefix string
		for _, n := range node {
			buf.AstPrintf(node, "%s%v", prefix, n)
			prefix = ", "
		}

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

	case sqlparser.OnDup:
		if node == nil {
			return nil
		}
		buf.AstPrintf(node, " on duplicate key update %v", sqlparser.UpdateExprs(node))

	case sqlparser.ColIdent:
		for i := sqlparser.NoAt; i < node.GetAtCount(); i++ {
			buf.WriteByte('@')
		}
		k, err := parserutil.NewUnknownTypeColumnarReference(node)
		if err != nil {
			return err
		}
		err = v.params.Set(k, parserutil.NewPlaceholderParameterMetadata())
		if err != nil {
			return err
		}
		sqlparser.FormatID(buf, node.GetRawVal(), node.Lowered(), node.GetAtCount())

	case sqlparser.TableIdent:
		sqlparser.FormatID(buf, node.GetRawVal(), strings.ToLower(node.GetRawVal()), sqlparser.NoAt)

	case *sqlparser.IsolationLevel:
		buf.WriteString("isolation level " + node.Level)

	case *sqlparser.AccessMode:
		buf.WriteString(node.Mode)
	}
	return nil
}
