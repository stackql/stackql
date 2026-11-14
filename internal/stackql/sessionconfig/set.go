package sessionconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// Apply is the shared SET/UNSET value decoder; the host applies authorization.
//
//nolint:exhaustive // Unsupported literal kinds are explicitly rejected by the default branch.
func Apply(host interface {
	SetConfigAtPath(string, interface{}, string) error
}, setExpr *sqlparser.SetExpr) error {
	lhsTrimmed := setExpr.Name.GetRawVal()
	config := strings.HasPrefix(lhsTrimmed, "$")
	if config {
		lhsTrimmed = strings.TrimPrefix(strings.TrimPrefix(lhsTrimmed, "$"), ".")
		root, _, _ := strings.Cut(lhsTrimmed, ".")
		switch root {
		case "auth", "providers", "secret_store", "secrets", "document_merge", "transaction":
		default:
			return fmt.Errorf("unknown intrinsic session configuration %q", lhsTrimmed)
		}
	} else if !strings.HasPrefix(lhsTrimmed, "variables.") {
		lhsTrimmed = "variables." + lhsTrimmed
	}
	if _, reset := setExpr.Expr.(*sqlparser.Default); reset {
		return host.SetConfigAtPath(lhsTrimmed, nil, setExpr.Scope)
	}
	if !config {
		raw, err := scalarVariable(setExpr.Expr)
		if err != nil {
			return err
		}
		return host.SetConfigAtPath(lhsTrimmed, raw, setExpr.Scope)
	}
	value, ok := setExpr.Expr.(*sqlparser.SQLVal)
	if !ok || value.Type != sqlparser.StrVal {
		return fmt.Errorf("session configuration requires a string literal or DEFAULT")
	}
	exprStr := string(value.Val)
	if strings.HasPrefix(lhsTrimmed, "secrets.") {
		return host.SetConfigAtPath(lhsTrimmed, exprStr, setExpr.Scope)
	}
	var exprObj map[string]interface{}
	if err := json.Unmarshal([]byte(exprStr), &exprObj); err == nil && exprObj != nil {
		return host.SetConfigAtPath(lhsTrimmed, exprObj, setExpr.Scope)
	}
	return host.SetConfigAtPath(lhsTrimmed, exprStr, setExpr.Scope)
}

var ErrSessionConfiguration = errors.New("session configuration failed")

func scalarVariable(expr sqlparser.Expr) (json.RawMessage, error) {
	switch value := expr.(type) {
	case *sqlparser.NullVal:
		return json.RawMessage("null"), nil
	case sqlparser.BoolVal:
		return json.RawMessage(fmt.Sprint(bool(value))), nil
	case *sqlparser.SQLVal:
		return scalarSQLValue(value)
	default:
		return nil, fmt.Errorf("variables accepts scalar literals only")
	}
}

//nolint:exhaustive // Non-scalar literal kinds are rejected.
func scalarSQLValue(value *sqlparser.SQLVal) (json.RawMessage, error) {
	switch value.Type {
	case sqlparser.StrVal:
		return json.Marshal(string(value.Val))
	case sqlparser.IntVal, sqlparser.FloatVal:
		return json.RawMessage(value.Val), nil
	default:
		return nil, fmt.Errorf("variables accepts scalar literals only")
	}
}
