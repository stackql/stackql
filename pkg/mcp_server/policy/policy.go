// Package policy classifies SQL queries and decides whether the server should
// allow, refuse, or seek approval for a tool call based on the configured
// server mode.  Pure functions only - no I/O, no SDK types, no logging.
package policy

import (
	"regexp"
	"strings"
)

// Server modes.  These are the legal values for Config.Server.Mode.
const (
	ModeReadOnly   = "read_only"
	ModeSafe       = "safe"
	ModeDeleteSafe = "delete_safe"
	ModeFullAccess = "full_access"
)

// QueryClass identifies the kind of statement a query tool is being asked to run.
// The classifier is intentionally shallow: it looks at the first token only,
// except for a leading WITH, which is resolved by scanning for a
// data-modifying keyword (see ClassifyQuery).
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

// withMutationRegexp matches a data-modifying keyword appearing as a whole
// word.  It is applied only to statements that begin with WITH, to decide
// whether the common table expression heads a read-only statement.
var withMutationRegexp = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|REPLACE|MERGE|UPSERT|EXEC)\b`)

// ClassifyQuery returns the class of the SQL by inspecting only the first
// whitespace-separated token.  It does not parse the statement.  Empty or
// unrecognised inputs return QueryClassUnknown.
//
// A leading WITH is the one case that needs more than the first token: SQL
// allows a common table expression to head a read-only SELECT but also an
// INSERT, UPDATE or DELETE, so classifying every WITH as read-only would let
// a mutation through the gate.  Such statements are resolved by classifyWith.
func ClassifyQuery(sql string) QueryClass {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return QueryClassUnknown
	}
	// First whitespace-separated token.
	var verb string
	if idx := strings.IndexAny(trimmed, " \t\r\n"); idx >= 0 {
		verb = trimmed[:idx]
	} else {
		verb = trimmed
	}
	switch strings.ToUpper(verb) {
	case "SELECT", "SHOW", "DESCRIBE", "EXPLAIN":
		return QueryClassSelect
	case "WITH":
		return classifyWith(trimmed)
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

// classifyWith resolves a statement that begins with a common table
// expression.  A WITH followed only by a query body is read-only; anything
// containing a data-modifying keyword is deliberately demoted to
// QueryClassUnknown rather than to its specific mutation class, so the gate
// still applies in every mode while the shallow scan stays honest about what
// it cannot determine.  The scan cannot exclude keywords inside string
// literals or identifiers, which biases it towards demotion - the safe
// direction.
func classifyWith(sql string) QueryClass {
	if withMutationRegexp.MatchString(sql) {
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
// classifier inspects the SQL's first token.  Mode is normalised internally
// so an empty / unknown value behaves like ModeSafe.
func NewPolicy(mode, sql string, defaultClass QueryClass) Policy {
	class := defaultClass
	if strings.TrimSpace(sql) != "" {
		class = ClassifyQuery(sql)
	}
	decision, reason := GateDecision(mode, class)
	return policy{class: class, decision: decision, reason: reason}
}
