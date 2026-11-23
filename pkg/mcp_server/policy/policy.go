// Package policy classifies SQL queries and decides whether the server should
// allow, refuse, or seek approval for a tool call based on the configured
// server mode.  Pure functions only - no I/O, no SDK types, no logging.
package policy

import (
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/pkg/sqlsplit"
)

// Server modes.  These are the legal values for Config.Server.Mode.
const (
	ModeReadOnly   = "read_only"
	ModeSafe       = "safe"
	ModeDeleteSafe = "delete_safe"
	ModeFullAccess = "full_access"
)

// QueryClass identifies the kind of statement a query tool is being asked to run.
// SELECT effects are inspected in the parser AST. Backend-owned view
// definitions and provider document effects are not available here.
type QueryClass int

const (
	QueryClassUnknown QueryClass = iota
	QueryClassSelect
	QueryClassMutationCreate // INSERT, UPDATE, REPLACE, MERGE, UPSERT
	QueryClassMutationDelete // DELETE
	QueryClassLifecycle      // EXEC
)

// String renders a QueryClass for audit logging and error messages.
func (c QueryClass) String() string {
	switch c {
	case QueryClassSelect:
		return "select"
	case QueryClassMutationCreate:
		return "mutation_create"
	case QueryClassMutationDelete:
		return "mutation_delete"
	case QueryClassLifecycle:
		return "lifecycle"
	case QueryClassUnknown:
		return "unknown"
	}
	return "unknown"
}

// ClassifyQuery returns the class of the SQL.  Every statement in the payload
// (see sqlsplit.Statements) is classified, so a mutation cannot hide behind a
// leading read-only statement; the most privileged statement decides the class.
// Empty or unrecognised inputs return QueryClassUnknown.
func ClassifyQuery(sql string) QueryClass {
	class := QueryClassUnknown
	rank := -1
	for _, stmt := range sqlsplit.Statements(sql) {
		stmtClass := classifyStatement(stmt)
		if stmtRank := classPrivilege(stmtClass); stmtRank > rank {
			class, rank = stmtClass, stmtRank
		}
	}
	return class
}

// classPrivilege ranks classes so that GateDecision is never more permissive
// for a higher rank, in any mode.
func classPrivilege(class QueryClass) int {
	if class == QueryClassSelect {
		return 0
	}
	if class == QueryClassMutationCreate {
		return 1
	}
	return 2
}

// classifyStatement never treats an unparseable SELECT or WITH as read-only.
// Existing statement-head classes remain for other verbs, including syntax
// that StackQL's parser does not currently support.
func classifyStatement(trimmed string) QueryClass {
	if node, err := sqlparser.Parse(trimmed); err == nil {
		switch node.(type) {
		case sqlparser.SelectStatement:
			return classifyReadAST(node, nil)
		case *sqlparser.Insert, *sqlparser.Update:
			return QueryClassMutationCreate
		case *sqlparser.Delete:
			return QueryClassMutationDelete
		case *sqlparser.Exec:
			return QueryClassLifecycle
		}
	}

	var verb string
	if idx := strings.IndexAny(trimmed, " \t\r\n"); idx >= 0 {
		verb = trimmed[:idx]
	} else {
		verb = trimmed
	}
	switch strings.ToUpper(verb) {
	case "SHOW", "DESCRIBE", "DESC", "EXPLAIN":
		return QueryClassSelect
	case "SELECT", "WITH":
		return QueryClassUnknown
	case "INSERT", "UPDATE", "REPLACE", "MERGE", "UPSERT":
		return QueryClassMutationCreate
	case "DELETE":
		return QueryClassMutationDelete
	case "EXEC":
		return QueryClassLifecycle
	default:
		return QueryClassUnknown
	}
}

// classifyReadAST follows subqueries and in-payload CTE definitions. A bare
// relation outside a CTE is unresolved indirection, not evidence of a read.
// Fully qualified provider resources may still be document or persisted views:
// proving their effects requires a backend-owned semantic classification API.
func classifyReadAST(root sqlparser.SQLNode, inherited map[string]bool) QueryClass {
	ctes := make(map[string]bool, len(inherited))
	for name := range inherited {
		ctes[name] = true
	}
	class := QueryClassSelect
	if selectNode, ok := root.(*sqlparser.Select); ok && selectNode.With != nil {
		if selectNode.With.Recursive {
			return QueryClassUnknown
		}
		for _, cte := range selectNode.With.CTEs {
			ctes[cte.Name.GetRawVal()] = true
		}
	}
	merge := func(candidate QueryClass) {
		if classPrivilege(candidate) > classPrivilege(class) || candidate == QueryClassLifecycle {
			class = candidate
		}
	}
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		if node != root {
			if _, ok := node.(sqlparser.SelectStatement); ok {
				merge(classifyReadAST(node, ctes))
				return false, nil
			}
		}
		switch node := node.(type) {
		case *sqlparser.Exec:
			merge(QueryClassLifecycle)
			return false, nil
		case *sqlparser.AliasedTableExpr:
			candidate, descend := classifyAliasedTableAST(node, ctes)
			merge(candidate)
			return descend, nil
		}
		return true, nil
	}, root)
	if err != nil {
		return QueryClassUnknown
	}
	return class
}

func classifyAliasedTableAST(node *sqlparser.AliasedTableExpr, ctes map[string]bool) (QueryClass, bool) {
	table, ok := node.Expr.(sqlparser.TableName)
	if !ok {
		return QueryClassSelect, true
	}
	if table.Qualifier.IsEmpty() && table.QualifierSecond.IsEmpty() &&
		table.QualifierThird.IsEmpty() && ctes[table.Name.GetRawVal()] {
		return QueryClassSelect, false
	}
	return classifyRelation(table), true
}

func classifyRelation(table sqlparser.TableName) QueryClass {
	service, resource := table.Qualifier.GetRawVal(), table.Name.GetRawVal()
	provider := table.QualifierSecond.GetRawVal()
	if provider == "" {
		// The parser inserts dual for SELECTs without FROM. Bare names and
		// abbreviated provider names otherwise require backend resolution.
		if service == "" && table.QualifierThird.IsEmpty() && resource == "dual" {
			return QueryClassSelect
		}
		return QueryClassUnknown
	}
	return QueryClassSelect
}

// Decision is what GateDecision returns: what the server wants to do
// with a tool call given its mode and the class of the query.
type Decision int

const (
	DecisionAllow Decision = iota
	DecisionRefuseImmediate
	DecisionNeedsApproval
)

// String renders a Decision for audit logging.  These string forms are part
// of the audit event contract.
func (d Decision) String() string {
	switch d {
	case DecisionAllow:
		return "allow"
	case DecisionRefuseImmediate:
		return "refuse_immediate"
	case DecisionNeedsApproval:
		return "needs_approval"
	default:
		return "unknown"
	}
}

// GateDecision is pure: given the mode and the class of the query, what does
// the server want to do?  The reason is a human-readable phrase suitable for
// an error message ("server is in 'read_only' mode").
//
// An empty or unknown mode is treated as the safe default.
func GateDecision(mode string, class QueryClass) (Decision, string) {
	normalized := mode
	if normalized == "" {
		normalized = ModeSafe
	}
	switch normalized {
	case ModeFullAccess:
		return DecisionAllow, ""
	case ModeReadOnly:
		if class == QueryClassSelect {
			return DecisionAllow, ""
		}
		return DecisionRefuseImmediate, "server is in 'read_only' mode"
	case ModeDeleteSafe:
		if class == QueryClassSelect || class == QueryClassMutationCreate {
			return DecisionAllow, ""
		}
		return DecisionNeedsApproval, "server is in 'delete_safe' mode"
	case ModeSafe:
		if class == QueryClassSelect {
			return DecisionAllow, ""
		}
		return DecisionNeedsApproval, "server is in 'safe' mode"
	default:
		// Unknown mode -> behave like safe.
		if class == QueryClassSelect {
			return DecisionAllow, ""
		}
		return DecisionNeedsApproval, "server is in 'safe' mode"
	}
}

// IsLegalMode reports whether the given string is one of the four legal modes
// (or the empty string, which is treated as the default elsewhere).
func IsLegalMode(mode string) bool {
	switch mode {
	case "", ModeReadOnly, ModeSafe, ModeDeleteSafe, ModeFullAccess:
		return true
	default:
		return false
	}
}

// Policy is the composite output of classifying a SQL statement and applying
// the gate decision for the configured server mode.  Callers use NewPolicy at
// the boundary, then read Class / Decision / Reason in whichever combination
// they need.  The interface is unexported in spirit (it carries inherent
// types only and never crosses out of the policy package as a typed value
// beyond what mcp_server consumes internally), but is exposed for explicit
// type assertions in tests.
type Policy interface {
	Class() QueryClass
	Decision() Decision
	Reason() string
}

type policy struct {
	class    QueryClass
	decision Decision
	reason   string
}

func (p policy) Class() QueryClass  { return p.class }
func (p policy) Decision() Decision { return p.decision }
func (p policy) Reason() string     { return p.reason }

// NewPolicy is the factory.  When sql is empty (no SQL input on a metadata
// tool, for example), defaultClass becomes the effective class; otherwise the
// classifier inspects every statement in the SQL.  Mode is normalised internally
// so an empty / unknown value behaves like ModeSafe.
func NewPolicy(mode, sql string, defaultClass QueryClass) Policy {
	class := defaultClass
	if strings.TrimSpace(sql) != "" {
		class = ClassifyQuery(sql)
	}
	decision, reason := GateDecision(mode, class)
	return policy{class: class, decision: decision, reason: reason}
}
