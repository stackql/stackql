package intrinsic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

// The host coordinates statement boundaries; provider reads are always SDK plans.
func planStatementRead(
	ctx queryContext, stmt sqlparser.SelectStatement, current string, depth int,
	inherited map[string]omnisdk.Relation,
) ([]string, derivedOpener, error) {
	if depth >= derivedRelationMaxDepth {
		return nil, nil, fmt.Errorf("derived relations exceed nesting bound")
	}
	switch node := stmt.(type) {
	case *sqlparser.ParenSelect:
		return planStatementRead(ctx, node.Select, current, depth+1, inherited)
	case *sqlparser.Union:
		return planUnionRead(ctx, node, current, depth+1, inherited)
	case *sqlparser.Select:
		prepared, stored, err := prepareStoredRelations(ctx, node)
		if err != nil {
			return nil, nil, err
		}
		if stored == nil {
			stored = map[string]omnisdk.Relation{}
		}
		for handle, relation := range inherited {
			stored[handle] = relation
		}
		expanded, err := expandDocumentViews(ctx, prepared, current)
		if err != nil {
			return nil, nil, err
		}
		prepared, children, err := resolveDerivedRelations(ctx, expanded, current, depth+1, stored)
		if err != nil {
			return nil, nil, err
		}
		dialect, err := backendDialect(ctx)
		if err != nil {
			return nil, nil, err
		}
		translator := newDocTranslator(current, dialect)
		if err = expandDerivedStars(ctx, prepared, translator, children); err != nil {
			return nil, nil, err
		}
		return planDerivedRead(ctx, prepared, translator, children)
	}
	return nil, nil, fmt.Errorf("unsupported read statement %T", stmt)
}

func planUnionRead(
	ctx queryContext, node *sqlparser.Union, current string, depth int, inherited map[string]omnisdk.Relation,
) ([]string, derivedOpener, error) {
	statements := []sqlparser.SelectStatement{node.FirstStatement}
	for _, branch := range node.UnionSelects {
		statements = append(statements, branch.Statement)
	}
	names := make([][]string, len(statements))
	opens := make([]derivedOpener, len(statements))
	for i, statement := range statements {
		var err error
		names[i], opens[i], err = planStatementRead(ctx, statement, current, depth+1, inherited)
		if err != nil {
			return nil, nil, err
		}
		if i > 0 && len(names[i]) != len(names[0]) {
			return nil, nil, fmt.Errorf("UNION branches have different column counts")
		}
	}
	open := func(runCtx context.Context, _ map[string]any) (omnisdk.Rows, error) {
		return openUnionRead(ctx, runCtx, node, names, opens)
	}
	return names[0], open, nil
}

func openUnionRead(
	ctx queryContext, runCtx context.Context, node *sqlparser.Union, names [][]string, opens []derivedOpener,
) (omnisdk.Rows, error) {
	manager, err := getStagingManager(ctx)
	if err != nil {
		return nil, err
	}
	var statements []string
	for i, open := range opens {
		id, beginErr := manager.Begin(runCtx)
		if beginErr != nil {
			return nil, beginErr
		}
		defer manager.Release(context.Background(), id) //nolint:errcheck // Preserve the query outcome.
		rows, openErr := open(runCtx, nil)
		if openErr != nil {
			return nil, openErr
		}
		source := newOmniBatchSource(rows, names[i], previewCfg.getBatchSize())
		first, readErr := source.Next(runCtx)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			rows.Close()
			return nil, readErr
		}
		source.pending = first
		_, stageErr := manager.Stage(runCtx, id, stagedColumns(names[i], first), source)
		rows.Close()
		if stageErr != nil {
			return nil, stageErr
		}
		if source.terminalErr != nil {
			return nil, source.terminalErr
		}
		if i > 0 {
			statements = append(statements, strings.ToUpper(node.UnionSelects[i-1].Type))
		}
		statements = append(statements, "SELECT * FROM "+manager.TableName(id))
	}
	formatter := sqlparser.NewTrackedBuffer(ctx.GetASTFormatter())
	formatter.Myprintf("%v%v", node.OrderBy, node.Limit)
	// Table names are generated locally and the suffix is rendered from the parsed SQL AST.
	statement := strings.Join(statements, " ") + formatter.String() // #nosec G202
	db, err := ctx.GetSQLEngine().GetDB()
	if err != nil {
		return nil, err
	}
	result, err := db.QueryContext(runCtx, statement)
	if err != nil {
		return nil, err
	}
	defer result.Close()
	return materializeSQLRows(runCtx, result, names[0], nil)
}

func statementReadFunc(ctx queryContext, stmt sqlparser.SelectStatement) func() internaldto.ExecutorOutput {
	names, open, err := planStatementRead(ctx, stmt, ctx.GetCurrentProvider(), 0, nil)
	if err != nil {
		return refuse(err)
	}
	return func() internaldto.ExecutorOutput {
		rows, openErr := open(context.Background(), nil)
		if openErr != nil {
			return internaldto.NewErroneousExecutorOutput(openErr)
		}
		stream := &rowStream{rows: rows, batchSize: previewCfg.getBatchSize(), flushInterval: previewCfg.getFlushInterval(),
			table: sqldata.NewSQLTable(0, "union"), typCfg: ctx.GetTypingConfig(), outputs: names}
		primed, readErr := newPrimedStream(stream)
		if readErr != nil {
			return internaldto.NewErroneousExecutorOutput(readErr)
		}
		return internaldto.NewExecutorOutput(primed, nil, nil, nil, nil)
	}
}
