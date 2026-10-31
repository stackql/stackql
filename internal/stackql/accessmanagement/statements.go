package accessmanagement

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/stackql-labs/omnisdk/public/authorization"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// CheckStatement runs before planning or cache lookup. SQL syntax alone never grants SDK effects.
func CheckStatement(ctx context.Context, caller authorization.Session, statement sqlparser.Statement) error {
	if caller == nil {
		return authorization.ErrDenied
	}
	switch statement.(type) {
	case *sqlparser.Auth, *sqlparser.AuthRevoke:
		return fmt.Errorf("%w: provider AUTH administration is not integrated with caller protection",
			authorization.ErrDenied)
	}
	read := false
	switch statement.(type) {
	case *sqlparser.Select, *sqlparser.Union, *sqlparser.ParenSelect, *sqlparser.DescribeTable, *sqlparser.DescribeMethod:
		read = true
	}
	if show, ok := statement.(*sqlparser.Show); ok {
		switch strings.ToUpper(show.Type) {
		case "PROVIDERS", "SERVICES", "RESOURCES", "METHODS":
			read = true
		}
	}
	if !read || !slices.Contains(caller.Principal().Groups(), "readers") {
		action := strings.ToLower(fmt.Sprintf("%T", statement))
		action = strings.TrimPrefix(action, "*sqlparser.")
		if read {
			action = "read"
		}
		if err := authorization.Check(ctx, "host/sql/"+action, "execute"); err != nil {
			return err
		}
	}
	return CheckNodes(ctx, statement)
}

// CheckNodes also applies to expanded stored definitions and opaque backend reads.
func CheckNodes(ctx context.Context, node sqlparser.SQLNode) error {
	return sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) { return true, checkNode(ctx, node) }, node)
}
func checkNode(ctx context.Context, node sqlparser.SQLNode) error {
	switch n := node.(type) {
	case *sqlparser.Exec:
		return authorization.Check(ctx, "host/sql/exec", "execute")
	case sqlparser.TableName:
		return checkTable(ctx, n)
	case *sqlparser.FuncExpr:
		return checkFunction(ctx, n)
	}
	return nil
}
func checkTable(ctx context.Context, n sqlparser.TableName) error {
	if strings.HasPrefix(strings.ToLower(n.Name.GetRawVal()), "stackql_access_") {
		return fmt.Errorf("%w: directory tables require the administration path", authorization.ErrDenied)
	}
	if strings.EqualFold(n.Qualifier.GetRawVal(), "iac") &&
		strings.EqualFold(n.QualifierSecond.GetRawVal(), "stackql_preview") {
		return authorization.Check(ctx, "host/iac/"+strings.ToLower(n.Name.GetRawVal()), "execute")
	}
	return nil
}
func checkFunction(ctx context.Context, n *sqlparser.FuncExpr) error {
	name := strings.ToLower(n.Name.GetRawVal())
	// Only known pure host functions receive the reader capability.
	const pure = "|whoami|groups|count|sum|avg|min|max|coalesce|ifnull|nullif|lower|upper|length|trim|ltrim|rtrim|" +
		"substr|substring|replace|abs|round|ceil|floor|strftime|datetime|date|time|julianday|unixepoch|json|" +
		"json_extract|json_array|json_object|json_type|json_valid|json_array_length|split_part|regexp_like|" +
		"regexp_substr|regexp_replace|json_equal|aws_policy_equal|"
	if !n.Qualifier.IsEmpty() || !strings.Contains(pure, "|"+name+"|") {
		return authorization.Check(ctx, "host/function/"+name, "execute")
	}
	return nil
}
