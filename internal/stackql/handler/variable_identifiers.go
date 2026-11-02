package handler

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// Identifier values are AST names, never SQL fragments. IDENTIFIER(expr) opts
// into a column name where an ordinary scalar expression would be ambiguous.
//
//nolint:gocognit,gocyclo,cyclop,funlen // One AST rewrite keeps identifier substitutions atomic and context-aware.
func resolveVariableIdentifiers(statement sqlparser.Statement, variables map[string]any) error {
	var failure error
	lookup := func(name string) (string, error) {
		key := strings.TrimPrefix(strings.TrimPrefix(name, "$"), ".")
		key = strings.TrimPrefix(key, "variables.")
		value, exists := variables[key]
		if !exists {
			return "", fmt.Errorf("identifier variable %q is not defined", name)
		}
		text, ok := value.(string)
		if !ok || text == "" {
			return "", fmt.Errorf("identifier variable %q requires a nonempty string", name)
		}
		return text, nil
	}
	sqlparser.Rewrite(statement, func(cursor *sqlparser.Cursor) bool {
		if failure != nil {
			return false
		}
		switch node := cursor.Node().(type) {
		case *sqlparser.FuncExpr:
			if !strings.EqualFold(node.Name.GetRawVal(), "identifier") || !node.Qualifier.IsEmpty() {
				return true
			}
			if len(node.Exprs) != 1 {
				failure = fmt.Errorf("IDENTIFIER requires one string value")
				return false
			}
			arg, ok := node.Exprs[0].(*sqlparser.AliasedExpr)
			if !ok {
				failure = fmt.Errorf("IDENTIFIER requires one string value")
				return false
			}
			value, ok := arg.Expr.(*sqlparser.SQLVal)
			if !ok || value.Type != sqlparser.StrVal {
				failure = fmt.Errorf("IDENTIFIER requires one string value")
				return false
			}
			parts, err := identifierParts(string(value.Val))
			if err != nil {
				failure = err
				return false
			}
			col := &sqlparser.ColName{Name: sqlparser.NewColIdent(parts[len(parts)-1])}
			if len(parts) > 1 {
				col.Qualifier, failure = tableIdentifier(strings.Join(parts[:len(parts)-1], "."))
			}
			cursor.Replace(col)
			return false
		case sqlparser.TableName:
			parts := []string{
				node.QualifierThird.GetRawVal(), node.QualifierSecond.GetRawVal(),
				node.Qualifier.GetRawVal(), node.Name.GetRawVal(),
			}
			changed := false
			for i, part := range parts {
				if !strings.HasPrefix(part, "$") {
					continue
				}
				parts[i], failure = lookup(part)
				if failure != nil {
					return false
				}
				changed = true
			}
			if changed {
				var names []string
				for _, part := range parts {
					if part != "" {
						names = append(names, part)
					}
				}
				replacement, err := tableIdentifier(strings.Join(names, "."))
				if err != nil {
					failure = err
					return false
				}
				cursor.Replace(replacement)
				return false
			}
		case *sqlparser.ColName:
			if !strings.HasPrefix(node.Name.GetRawVal(), "$") {
				return true
			}
			target, isUpdate := cursor.Parent().(*sqlparser.UpdateExpr)
			if node.Qualifier.IsEmpty() && (!isUpdate || target.Name != node) {
				return true
			}
			value, err := lookup(node.Name.GetRawVal())
			if err != nil {
				failure = err
				return false
			}
			copied := *node
			copied.Name = sqlparser.NewColIdent(value)
			cursor.Replace(&copied)
			return false
		case sqlparser.ColIdent:
			if !strings.HasPrefix(node.GetRawVal(), "$") {
				return true
			}
			switch cursor.Parent().(type) {
			case *sqlparser.ColName, *sqlparser.SetExpr, *sqlparser.FuncExpr:
				return true
			default:
				value, err := lookup(node.GetRawVal())
				if err != nil {
					failure = err
					return false
				}
				cursor.Replace(sqlparser.NewColIdent(value))
				return false
			}
		case sqlparser.TableIdent:
			if !strings.HasPrefix(node.GetRawVal(), "$") {
				return true
			}
			if _, alias := cursor.Parent().(*sqlparser.AliasedTableExpr); alias {
				value, err := lookup(node.GetRawVal())
				if err != nil {
					failure = err
					return false
				}
				cursor.Replace(sqlparser.NewTableIdent(value))
				return false
			}
		}
		return true
	}, nil)
	return failure
}

func identifierParts(value string) ([]string, error) {
	parts := strings.Split(value, ".")
	const maxIdentifierComponents = 4
	if len(parts) > maxIdentifierComponents {
		return nil, fmt.Errorf("identifier requires at most four components")
	}
	for _, part := range parts {
		if part == "" || strings.ContainsAny(part, ";\x00\r\n") {
			return nil, fmt.Errorf("invalid identifier value")
		}
	}
	return parts, nil
}

func tableIdentifier(value string) (sqlparser.TableName, error) {
	parts, err := identifierParts(value)
	if err != nil {
		return sqlparser.TableName{}, err
	}
	var name sqlparser.TableName
	names := []*sqlparser.TableIdent{&name.Name, &name.Qualifier, &name.QualifierSecond, &name.QualifierThird}
	for i, part := range parts {
		*names[len(parts)-1-i] = sqlparser.NewTableIdent(part)
	}
	return name, nil
}
