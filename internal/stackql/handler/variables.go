package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stackql-labs/omnisdk/public/authorization"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// ResolveVariables takes one session snapshot and replaces references with data, never SQL text.
//
//nolint:gocognit,gocyclo,cyclop,govet,mnd,nestif,funlen // Keep statement resolution atomic.
func (hc *standardHandlerContext) ResolveVariables(statement sqlparser.Statement) error {
	paths := map[string]string{}
	identifiers := false
	assignmentTargets := map[*sqlparser.ColName]bool{}
	if err := sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		switch item := node.(type) {
		case *sqlparser.UpdateExpr:
			assignmentTargets[item.Name] = true
		case sqlparser.TableName:
			identifiers = identifiers || strings.Contains(item.GetRawVal(), "$")
		case *sqlparser.FuncExpr:
			identifiers = identifiers || strings.EqualFold(item.Name.GetRawVal(), "identifier")
		case sqlparser.ColIdent:
			if _, isSet := statement.(*sqlparser.Set); !isSet {
				identifiers = identifiers || strings.HasPrefix(item.GetRawVal(), "$")
			}
		case sqlparser.TableIdent:
			identifiers = identifiers || strings.HasPrefix(item.GetRawVal(), "$")
		}
		column, ok := node.(*sqlparser.ColName)
		if !ok || assignmentTargets[column] {
			return true, nil
		}
		name := column.Name.GetRawVal()
		if !column.Qualifier.IsEmpty() {
			name = column.Qualifier.Name.GetRawVal() + "." + name
			if !column.Qualifier.Qualifier.IsEmpty() {
				name = column.Qualifier.Qualifier.GetRawVal() + "." + name
			}
		}
		if strings.HasPrefix(name, "$") {
			path := strings.TrimPrefix(strings.TrimPrefix(name, "$"), ".")
			if !strings.Contains(path, ".") {
				path = "variables." + path
			}
			paths[name] = path
		}
		return true, nil
	}, statement); err != nil {
		return err
	}

	if len(paths) == 0 && !identifiers {
		return nil
	}
	if hc.runtimeContext.AccessConfigFile != "" {
		if err := authorization.Check(hc.GetExecutionContext(), "session/variables", "read"); err != nil {
			return err
		}
	}
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, path := range paths {
		variable := strings.TrimPrefix(path, "variables.")
		_, exists := state.variables[variable]
		if !exists && secretVariablePath(path) && !hc.runtimeContext.AllowSecretDereference {
			return fmt.Errorf("secret variable dereferencing is disabled")
		}
	}
	snapshot := map[string]any{"variables": state.variables, "providers": state.bindings}
	if hc.runtimeContext.AllowSecretDereference {
		snapshot["auth"] = hc.GetAuthContexts()
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("cannot snapshot session variables")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var values map[string]any
	if err = decoder.Decode(&values); err != nil {
		return err
	}
	resolved := map[string]sqlparser.Expr{}
	secretValues := map[string][]byte{}
	for name, path := range paths {
		if value, exists := state.variables[strings.TrimPrefix(path, "variables.")]; exists {
			literal, err := scalarVariableLiteral(value)
			if err != nil {
				return err
			}
			resolved[name] = literal
			continue
		}
		parts := strings.Split(path, ".")
		var value any = values
		if parts[0] == "secrets" {
			if len(parts) < 2 || state.store == nil {
				return fmt.Errorf("secret variable is unavailable")
			}
			data, found := secretValues[parts[1]]
			if !found {
				data, err = state.store.Get(hc.GetExecutionContext(), parts[1])
				if err != nil {
					return fmt.Errorf("secret variable is unavailable")
				}
				secretValues[parts[1]] = data
			}
			value = string(data)
			if len(parts) > 2 {
				d := json.NewDecoder(bytes.NewReader(data))
				d.UseNumber()
				if err = d.Decode(&value); err != nil {
					return fmt.Errorf("secret variable does not contain an object")
				}
			}
			parts = parts[2:]
		}
		for _, part := range parts {
			object, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("variable %q is not defined", name)
			}
			value, ok = object[part]
			if !ok {
				return fmt.Errorf("variable %q is not defined", name)
			}
		}
		literal, err := scalarVariableLiteral(value)
		if err != nil {
			return err
		}
		resolved[name] = literal
	}
	sqlparser.Rewrite(statement, func(cursor *sqlparser.Cursor) bool {
		column, ok := cursor.Node().(*sqlparser.ColName)
		if !ok {
			return true
		}
		name := column.Name.GetRawVal()
		if !column.Qualifier.IsEmpty() {
			name = column.Qualifier.Name.GetRawVal() + "." + name
			if !column.Qualifier.Qualifier.IsEmpty() {
				name = column.Qualifier.Qualifier.GetRawVal() + "." + name
			}
		}
		if literal, found := resolved[name]; found {
			cursor.Replace(literal)
			return false
		}
		return true
	}, nil)
	return resolveVariableIdentifiers(statement, state.variables)
}

func secretVariablePath(path string) bool {
	parts := strings.Split(path, ".")
	if parts[0] == "secrets" || parts[0] == "auth" || parts[0] == "secret_store" {
		return true
	}
	return parts[0] == "providers" && len(parts) > 2 && (parts[2] == "auth" || parts[2] == "auth_secret")
}

func scalarVariableLiteral(value any) (sqlparser.Expr, error) {
	switch v := value.(type) {
	case nil:
		return &sqlparser.NullVal{}, nil
	case string:
		return sqlparser.NewStrVal([]byte(v)), nil
	case bool:
		return sqlparser.BoolVal(v), nil
	case json.Number:
		if strings.ContainsAny(string(v), ".eE") {
			return sqlparser.NewFloatVal([]byte(v)), nil
		}
		return sqlparser.NewIntVal([]byte(v)), nil
	default:
		return nil, fmt.Errorf("variable dereferencing supports scalars only")
	}
}

func (hc *standardHandlerContext) setSessionVariable(path string, rhs any) error {
	name := strings.TrimPrefix(path, "variables.")
	if name == "" || name == path {
		return fmt.Errorf("variables requires one explicit name")
	}
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	if rhs == nil {
		delete(state.variables, name)
		return nil
	}
	raw, ok := rhs.(json.RawMessage)
	if !ok {
		return fmt.Errorf("variables accepts scalar values only")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if _, err := scalarVariableLiteral(value); err != nil {
		return err
	}
	if state.variables == nil {
		state.variables = map[string]any{}
	}
	state.variables[name] = value
	return nil
}
