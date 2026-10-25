package intrinsic

import (
	"fmt"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/metadatavisitors"
	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stackql/stackql/pkg/prettyprint"
)

func showDocInsert(ctx queryContext, node *sqlparser.Show, bundle string) internaldto.ExecutorOutput {
	opts, err := documentOptions(ctx)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	address := UnstablePrefix + bundle + "." + node.OnTable.Qualifier.GetRawVal() + "." + node.OnTable.Name.GetRawVal()
	table, err := omnisdk.DescribeMutation(registryRoot(ctx), address, "insert", opts...)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	if len(table.Methods()) == 0 {
		return internaldto.NewErroneousExecutorOutput(fmt.Errorf("%s declares no INSERT method", address))
	}
	context := prettyprint.NewPrettyPrintContext(
		ctx.GetRuntimeContext().OutputFormat == constants.PrettyTextStr,
		constants.DefaultPrettyPrintIndent, constants.DefaultPrettyPrintBaseIndent, "'", logging.GetLogger())
	statement, err := metadatavisitors.ToOmniInsertStatement(node.Columns, table.Methods()[0], isExtended(node.Extended),
		sqlparser.ExtractCommentDirectives(node.Comments).IsSet("REQUIRED"),
		prettyprint.NewPrettyPrinter(context), prettyprint.NewPrettyPrinter(context))
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	return prepare(ctx, []string{"insert_statement"}, map[string]map[string]interface{}{
		"1": {"insert_statement": fmt.Sprintf(statement, qualifiedName(node.OnTable))},
	}, util.DefaultRowSort)
}
