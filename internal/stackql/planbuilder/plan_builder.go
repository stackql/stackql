package planbuilder

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/astanalysis/routeanalysis"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/plan"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivegenerator"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	// only string "false" will disable
	PlanCacheEnabled string = "true"
)

func isPlanCacheEnabled() bool {
	return strings.ToLower(PlanCacheEnabled) != "false"
}

type planGraphBuilder struct {
	planGraph              primitivegraph.PrimitiveGraph
	rootPrimitiveGenerator primitivegenerator.PrimitiveGenerator
}

func newPlanGraphBuilder(concurrencyLimit int) *planGraphBuilder {
	return &planGraphBuilder{
		planGraph: primitivegraph.NewPrimitiveGraph(concurrencyLimit),
	}
}

func (pgb *planGraphBuilder) createInstructionFor(pbi planbuilderinput.PlanBuilderInput) error {
	stmt := pbi.GetStatement()
	switch stmt := stmt.(type) {
	case *sqlparser.Auth:
		return pgb.handleAuth(pbi)
	case *sqlparser.AuthRevoke:
		return pgb.handleAuthRevoke(pbi)
	case *sqlparser.Begin:
		return pgb.nop(pbi)
	case *sqlparser.Commit:
		return pgb.nop(pbi)
	case *sqlparser.DBDDL:
		return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("unsupported: Database DDL %v", sqlparser.String(stmt)))
	case *sqlparser.DDL:
		return pgb.handleDDL(pbi)
	case *sqlparser.Delete:
		return pgb.handleDelete(pbi)
	case *sqlparser.DescribeTable:
		return pgb.handleDescribe(pbi)
	case *sqlparser.Exec:
		return pgb.handleExec(pbi)
	case *sqlparser.Explain:
		return iqlerror.GetStatementNotSupportedError("EXPLAIN")
	case *sqlparser.Insert:
		return pgb.handleInsert(pbi)
	case *sqlparser.NativeQuery:
		return pgb.handleNativeQuery(pbi)
	case *sqlparser.OtherRead, *sqlparser.OtherAdmin:
		return iqlerror.GetStatementNotSupportedError("OTHER")
	case *sqlparser.Purge:
		return pgb.handlePurge(pbi)
	case *sqlparser.Registry:
		return pgb.handleRegistry(pbi)
	case *sqlparser.Rollback:
		return pgb.nop(pbi)
	case *sqlparser.Savepoint:
		return pgb.nop(pbi)
	case *sqlparser.Select:
		_, _, err := pgb.handleSelect(pbi)
		return err
	case *sqlparser.Set:
		return pgb.nop(pbi)
	case *sqlparser.SetTransaction:
		return pgb.nop(pbi)
	case *sqlparser.Show:
		return pgb.handleShow(pbi)
	case *sqlparser.Sleep:
		return pgb.handleSleep(pbi)
	case *sqlparser.SRollback:
		return pgb.nop(pbi)
	case *sqlparser.Release:
		return pgb.nop(pbi)
	case *sqlparser.Union:
		_, _, err := pgb.handleUnion(pbi)
		return err
	case *sqlparser.Update:
		return pgb.handleUpdate(pbi)
	case *sqlparser.Use:
		return pgb.handleUse(pbi)
	}
	return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("BUG: unexpected statement type: %T", stmt))
}

func (pgb *planGraphBuilder) nop(pbi planbuilderinput.PlanBuilderInput) error {
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeNop(pbi)
	if err != nil {
		return err
	}
	builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
	if builder == nil {
		return fmt.Errorf("nil nop builder")
	}
	err = builder.Build()
	return err
}

func (pgb *planGraphBuilder) pgInternal(pbi planbuilderinput.PlanBuilderInput) error {
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzePGInternal(pbi)
	if err != nil {
		return err
	}
	builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
	if builder == nil {
		return fmt.Errorf("nil pg internal builder")
	}
	err = builder.Build()
	return err
}

func (pgb *planGraphBuilder) handleAuth(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetAuth()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Auth", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	prov, err := handlerCtx.GetProvider(node.Provider)
	if err != nil {
		return err
	}
	err = primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		logging.GetLogger().Debugln(fmt.Sprintf("err = %s", err.Error()))
		return err
	}
	authCtx, authErr := handlerCtx.GetAuthContext(node.Provider)
	if authErr != nil {
		return authErr
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			authType := strings.ToLower(node.Type)
			if node.KeyFilePath != "" {
				authCtx.KeyFilePath = node.KeyFilePath
			}
			if node.KeyEnvVar != "" {
				authCtx.KeyEnvVar = node.KeyEnvVar
			}
			_, err := prov.Auth(authCtx, authType, true)
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleAuthRevoke(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	stmt := pbi.GetStatement()
	node, ok := stmt.(*sqlparser.AuthRevoke)
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required AuthRevoke", stmt)
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	prov, err := handlerCtx.GetProvider(node.Provider)
	if err != nil {
		return err
	}
	authCtx, authErr := handlerCtx.GetAuthContext(node.Provider)
	if authErr != nil {
		return authErr
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, prov.AuthRevoke(authCtx))
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleDescribe(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDescribeTable()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required DescribeTable", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	md, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
	if err != nil {
		return err
	}
	prov, err := md.GetProvider()
	if err != nil {
		return err
	}
	var extended bool = strings.TrimSpace(strings.ToUpper(node.Extended)) == "EXTENDED"
	var full bool = strings.TrimSpace(strings.ToUpper(node.Full)) == "FULL"
	_, isView := md.GetHeirarchyObjects().GetHeirarchyIds().GetView()
	if isView {
		stmtCtx, ok := primitiveGenerator.GetPrimitiveComposer().GetIndirectDescribeSelectCtx()
		if !ok || stmtCtx == nil {
			return fmt.Errorf("cannot describe view without context")
		}
		nonControlColummns := stmtCtx.GetNonControlColumns()
		if len(nonControlColummns) < 1 {
			return fmt.Errorf("cannot describe view lacking columns")
		}
		pr := primitive.NewMetaDataPrimitive(
			prov,
			func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
				return primitivebuilder.NewDescribeViewInstructionExecutor(handlerCtx, md, nonControlColummns, extended, full)
			})
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return primitivebuilder.NewDescribeTableInstructionExecutor(handlerCtx, md, extended, full)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleDDL(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDDL()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required DDL", pbi.GetStatement())
	}
	bldr := primitivebuilder.NewDDL(
		pgb.planGraph,
		handlerCtx,
		node,
	)
	err := bldr.Build()
	if err != nil {
		return err
	}
	return nil
}

func (pgb *planGraphBuilder) handleSelect(pbi planbuilderinput.PlanBuilderInput) (primitivegraph.PrimitiveNode, primitivegraph.PrimitiveNode, error) {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSelect()
	if !ok {
		return nil, nil, fmt.Errorf("could not cast statement of type '%T' to required Select", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeStatement(pbi)
		if err != nil {
			logging.GetLogger().Infoln(fmt.Sprintf("select statement analysis error = '%s'", err.Error()))
			return nil, nil, err
		}
		builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
		_, isNativeSelect := builder.(*primitivebuilder.NativeSelect)
		_, isRawNativeSelect := builder.(*primitivebuilder.RawNativeSelect)
		_, isRawNativeExec := builder.(*primitivebuilder.RawNativeExec)
		isLocallyExecutable := !isNativeSelect && !isRawNativeSelect && !isRawNativeExec
		// check tables only if not native
		if isLocallyExecutable {
			for _, val := range primitiveGenerator.GetPrimitiveComposer().GetTables() {
				isLocallyExecutable = isLocallyExecutable && val.IsLocallyExecutable()
			}
		}
		if isLocallyExecutable {
			var colz []map[string]interface{}
			for idx := range primitiveGenerator.GetPrimitiveComposer().GetValOnlyColKeys() {
				col := primitiveGenerator.GetPrimitiveComposer().GetValOnlyCol(idx)
				colz = append(colz, col)
			}
			pr, err := primitivebuilder.NewLocalSelectExecutor(handlerCtx, node, util.DefaultRowSort, colz)
			if err != nil {
				return nil, nil, err
			}
			rv := pgb.planGraph.CreatePrimitiveNode(pr)
			return rv, rv, nil
		}
		if primitiveGenerator.GetPrimitiveComposer().GetBuilder() == nil {
			return nil, nil, fmt.Errorf("builder not created for select, cannot proceed")
		}
		err = builder.Build()
		if err != nil {
			return nil, nil, err
		}
		root := builder.GetRoot()
		tail := builder.GetTail()
		return root, tail, nil
	}
	pr := primitive.NewLocalPrimitive(nil)
	rv := pgb.planGraph.CreatePrimitiveNode(pr)
	return rv, rv, nil
}

func (pgb *planGraphBuilder) handleUnion(pbi planbuilderinput.PlanBuilderInput) (primitivegraph.PrimitiveNode, primitivegraph.PrimitiveNode, error) {
	// handlerCtx := pbi.GetHandlerCtx()
	_, ok := pbi.GetUnion()
	if !ok {
		return nil, nil, fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		logging.GetLogger().Infoln(fmt.Sprintf("select statement analysis error = '%s'", err.Error()))
		return nil, nil, err
	}
	isLocallyExecutable := true
	for _, val := range primitiveGenerator.GetPrimitiveComposer().GetTables() {
		isLocallyExecutable = isLocallyExecutable && val.IsLocallyExecutable()
	}
	if primitiveGenerator.GetPrimitiveComposer().GetBuilder() == nil {
		return nil, nil, fmt.Errorf("builder not created for union, cannot proceed")
	}
	builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
	err = builder.Build()
	if err != nil {
		return nil, nil, err
	}
	root := builder.GetRoot()
	tail := builder.GetTail()
	return root, tail, nil
}

func (pgb *planGraphBuilder) handleDelete(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDelete()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeStatement(pbi)
		if err != nil {
			return err
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		bldr := primitivebuilder.NewDelete(
			pgb.planGraph,
			handlerCtx,
			node,
			tbl,
			nil,
			primitiveGenerator.GetPrimitiveComposer().IsAwait(),
		)
		err = bldr.Build()
		if err != nil {
			return err
		}
		return nil
	} else {
		pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	}
}

func (pgb *planGraphBuilder) handleRegistry(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetRegistry()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Registry", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeRegistry(pbi)
	if err != nil {
		return err
	}
	reg, err := handler.GetRegistry(handlerCtx.GetRuntimeContext())
	if err != nil {
		return err
	}
	pr := primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			switch at := strings.ToLower(node.ActionType); at {
			case "pull":
				var err error
				providerVersion := node.ProviderVersion
				if providerVersion == "" {
					providerVersion, err = reg.GetLatestPublishedVersion(node.ProviderId)
				}
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				err = reg.PullAndPersistProviderArchive(node.ProviderId, providerVersion)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				return util.PrepareResultSet(internaldto.NewPrepareResultSetPlusRawDTO(nil, nil, nil, nil, nil, &internaldto.BackendMessages{WorkingMessages: []string{fmt.Sprintf("%s provider, version '%s' successfully installed", node.ProviderId, providerVersion)}}, nil))
			case "list":
				var colz []string
				var provz map[string]openapistackql.ProviderDescription
				keys := make(map[string]map[string]interface{})
				if node.ProviderId == "" {
					provz, err = reg.ListAllAvailableProviders()
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					colz = []string{"provider", "version"}
					var dks []string
					for k, _ := range provz {
						dks = append(dks, k)
					}
					sort.Strings(dks)
					for i, k := range dks {
						v := provz[k]
						for _, ver := range v.Versions {
							keys[strconv.Itoa(i)] = map[string]interface{}{
								"provider": k,
								"version":  ver,
							}
						}
					}
				} else {
					provz, err = reg.ListAllProviderVersions(node.ProviderId)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					colz = []string{"provider", "versions"}
					i := 0
					for k, v := range provz {
						keys[strconv.Itoa(i)] = map[string]interface{}{
							"provider": k,
							"versions": strings.Join(v.Versions, ", "),
						}
						i++
					}
				}
				return util.PrepareResultSet(internaldto.NewPrepareResultSetPlusRawDTO(nil, keys, colz, nil, nil, nil, nil))
			default:
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("registry action '%s' no supported", at))
			}
		},
	)
	pgb.planGraph.CreatePrimitiveNode(pr)

	return nil
}

func (pgb *planGraphBuilder) handlePurge(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetPurge()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Purge", pbi.GetStatement())
	}
	pr := primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			if node.IsGlobal {
				err := handlerCtx.GetGarbageCollector().Purge()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				return util.PrepareResultSet(
					internaldto.NewPrepareResultSetPlusRawDTO(
						nil,
						map[string]map[string]interface{}{"0": {"message": "purge 'GLOBAL' completed"}},
						[]string{"message"},
						nil,
						nil,
						&internaldto.BackendMessages{
							WorkingMessages: []string{fmt.Sprintf("Global PURGE successfully completed")}},
						nil,
					),
				)
			}
			targetStr := strings.ToLower(node.Target.GetRawVal())
			switch targetStr {
			case "cache":
				err := handlerCtx.GetGarbageCollector().PurgeCache()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "conservative":
				err := handlerCtx.GetGarbageCollector().Collect()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "control_tables":
				err := handlerCtx.GetGarbageCollector().PurgeControlTables()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "ephemeral":
				err := handlerCtx.GetGarbageCollector().PurgeEphemeral()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			default:
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("purge target '%s' not supported", targetStr))
			}
			// This happens in all cases, provided the ourge is successful.
			handlerCtx.GetLRUCache().Clear()
			purgeMsg := fmt.Sprintf("PURGE of type '%s' successfully completed", targetStr)
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil,
					map[string]map[string]interface{}{"0": {"message": purgeMsg}},
					[]string{"message"},
					nil,
					nil,
					nil,
					// &internaldto.BackendMessages{
					// 	WorkingMessages: []string{fmt.Sprintf("PURGE of type '%s' successfully completed", targetStr)}},
					nil,
				),
			)
		},
	)
	pgb.planGraph.CreatePrimitiveNode(pr)

	return nil
}

func (pgb *planGraphBuilder) handleNativeQuery(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetNativeQuery()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Purge", pbi.GetStatement())
	}
	rns := primitivebuilder.NewRawNativeSelect(pgb.planGraph, handlerCtx, pbi.GetTxnCtrlCtrs(), node.QueryString)

	err := rns.Build()

	if err != nil {
		return err
	}

	return nil
}

func (pgb *planGraphBuilder) handleInsert(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetInsert()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Insert", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := primitivegenerator.NewRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
		err := primitiveGenerator.AnalyzeInsert(pbi)
		if err != nil {
			return err
		}
		insertValOnlyRows, nonValCols, err := parserutil.ExtractInsertValColumns(node)
		if err != nil {
			return err
		}
		// selectPrimitive here forms the insert data
		var selectPrimitive primitive.IPrimitive
		var selectPrimitiveNode primitivegraph.PrimitiveNode
		if nonValCols > 0 {
			switch rowsNode := node.Rows.(type) {
			case *sqlparser.Select:
				selPbi, err := planbuilderinput.NewPlanBuilderInput(pbi.GetAnnotatedAST(), pbi.GetHandlerCtx(), rowsNode, pbi.GetTableExprs(), pbi.GetAssignedAliasedColumns(), pbi.GetAliasedTables(), pbi.GetColRefs(), pbi.GetPlaceholderParams(), pbi.GetTxnCtrlCtrs())
				if err != nil {
					return err
				}
				sr := routeanalysis.NewSelectRoutePass(rowsNode, selPbi, nil)
				err = sr.RoutePass()
				if err != nil {
					return err
				}
				_, selectPrimitiveNode, err = pgb.handleSelect(selPbi)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("insert with rows of type '%T' not currently supported", rowsNode)
			}
		} else {
			selectPrimitive, err = primitivebuilder.NewInsertableValsPrimitive(handlerCtx, insertValOnlyRows)
			if err != nil {
				return err
			}
			sn := pgb.planGraph.CreatePrimitiveNode(selectPrimitive)
			selectPrimitiveNode = sn
		}
		if selectPrimitiveNode == nil {
			return fmt.Errorf("nil selection for insert -- cannot work")
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		bldr := primitivebuilder.NewInsert(
			pgb.planGraph,
			handlerCtx,
			node,
			tbl,
			selectPrimitiveNode,
			primitiveGenerator.GetPrimitiveComposer().GetCommentDirectives(),
			primitiveGenerator.GetPrimitiveComposer().IsAwait(),
		)
		err = bldr.Build()
		if err != nil {
			return err
		}
		return nil
	} else {
		pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	}
	return nil
}

func (pgb *planGraphBuilder) handleUpdate(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUpdate()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Insert", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeUpdate(pbi)
		if err != nil {
			return err
		}
		insertValOnlyRows, nonValCols, err := parserutil.ExtractUpdateValColumns(node)
		if err != nil {
			return err
		}
		// selectPrimitive here forms the insert data
		var selectPrimitive primitive.IPrimitive
		var selectPrimitiveNode primitivegraph.PrimitiveNode
		if len(nonValCols) > 0 {
			// TODO: support dynamic content
			return fmt.Errorf("update does not currently support dynamic content")
		} else {
			selectPrimitive, err = primitivebuilder.NewUpdateableValsPrimitive(handlerCtx, insertValOnlyRows)
			if err != nil {
				return err
			}
			sn := pgb.planGraph.CreatePrimitiveNode(selectPrimitive)
			selectPrimitiveNode = sn
		}
		if selectPrimitiveNode == nil {
			return fmt.Errorf("nil selection for insert -- cannot work")
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		bldr := primitivebuilder.NewInsert(
			pgb.planGraph,
			handlerCtx,
			node,
			tbl,
			selectPrimitiveNode,
			primitiveGenerator.GetPrimitiveComposer().GetCommentDirectives(),
			primitiveGenerator.GetPrimitiveComposer().IsAwait(),
		)
		err = bldr.Build()
		if err != nil {
			return err
		}
		return nil
	} else {
		pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	}
}

func (pgb *planGraphBuilder) handleExec(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetExec()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Exec", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeStatement(pbi)
		if err != nil {
			return err
		}
		//
		if primitiveGenerator.IsShowResults() && primitiveGenerator.GetPrimitiveComposer().GetBuilder() != nil {
			err := primitiveGenerator.GetPrimitiveComposer().GetBuilder().Build()
			if err != nil {
				return err
			}
			return nil
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		bldr := primitivebuilder.NewExec(
			primitiveGenerator.GetPrimitiveComposer().GetGraph(),
			handlerCtx,
			node,
			tbl,
			primitiveGenerator.GetPrimitiveComposer().IsAwait(),
			primitiveGenerator.IsShowResults(),
		)
		err = bldr.Build()
		if err != nil {
			return err
		}
		return nil
	}
	pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleShow(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetShow()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Show", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	nodeTypeUpper := strings.ToUpper(node.Type)
	var tbl tablemetadata.ExtendedTableMetadata
	switch nodeTypeUpper {
	case "TRANSACTION_ISOLATION_LEVEL":
		builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
		_, isNativeSelect := builder.(*primitivebuilder.NativeSelect)
		if isNativeSelect {
			err := builder.Build()
			return err
		}
		return fmt.Errorf("improper usage of 'show transaction isolation level'")
	case "INSERT":
		tbl, err = primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
	case "METHODS":
		tbl, err = primitiveGenerator.GetPrimitiveComposer().GetTable(node.OnTable)
		if err != nil {
			// TODO: fix this for readability
		}
	}
	prov := primitiveGenerator.GetPrimitiveComposer().GetProvider()
	pr := primitive.NewMetaDataPrimitive(
		prov,
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return primitivebuilder.NewShowInstructionExecutor(
				node,
				prov,
				tbl,
				handlerCtx,
				primitiveGenerator.GetPrimitiveComposer().GetCommentDirectives(),
				primitiveGenerator.GetPrimitiveComposer().GetTableFilter(),
			)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleSleep(pbi planbuilderinput.PlanBuilderInput) error {
	// handlerCtx := pbi.GetHandlerCtx()
	_, ok := pbi.GetSleep()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Sleep", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	return nil
}

func (pgb *planGraphBuilder) handleUse(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUse()
	if !ok {
		return fmt.Errorf("node type '%T' is incorrect; expected *Use", node)
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	pr := primitive.NewMetaDataPrimitive(
		primitiveGenerator.GetPrimitiveComposer().GetProvider(),
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			handlerCtx.SetCurrentProvider(node.DBName.GetRawVal())
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func createErroneousPlan(handlerCtx handler.HandlerContext, qPlan *plan.Plan, rowSort func(map[string]map[string]interface{}) []string, err error) (*plan.Plan, error) {
	qPlan.Instructions = primitive.NewLocalPrimitive(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		return util.PrepareResultSet(
			internaldto.PrepareResultSetDTO{
				OutputBody:  nil,
				Msg:         nil,
				RowMap:      nil,
				ColumnOrder: nil,
				RowSort:     rowSort,
				Err:         err,
			},
		)
	})
	return qPlan, err
}
