package intrinsic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type completionChain struct{ first, second omnisdk.MutationCompletion }

func (c completionChain) Complete(
	ctx context.Context, address, verb string, inputs map[string]any, row omnisdk.Row,
) (omnisdk.Row, error) {
	var err error
	if c.first != nil {
		row, err = c.first.Complete(ctx, address, verb, inputs, row)
		if err != nil {
			return nil, err
		}
	}
	return c.second.Complete(ctx, address, verb, inputs, row)
}

type observedQueryContext struct {
	queryContext
	observer *inverseCapture
}

func (c observedQueryContext) mutationObserver() omnisdk.MutationCompletion { return c.observer }

type inverseCapture struct {
	mu          sync.Mutex
	rows        []map[string]string
	declaration omnisdk.DeclaredInverse
}

func (c *inverseCapture) Complete(
	_ context.Context, _ string, _ string, inputs map[string]any, row omnisdk.Row,
) (omnisdk.Row, error) {
	params := map[string]string{}
	for name, value := range inputs {
		params[name] = fmt.Sprint(value)
	}
	for name, token := range c.declaration.Tokens() {
		if token.Location != "body" || token.Algorithm != "toString" {
			return nil, fmt.Errorf("unsupported inverse token %s: %s/%s", name, token.Location, token.Algorithm)
		}
		value, ok := row[token.Key]
		if !ok || value == nil {
			return nil, fmt.Errorf("inverse token %s missing response field %s", name, token.Key)
		}
		params[name] = fmt.Sprint(value)
	}
	c.mu.Lock()
	c.rows = append(c.rows, params)
	c.mu.Unlock()
	return row, nil
}

// GenerateTransactionStreamFunc attaches only document-declared reversals to the
// host transaction graph. Accepted results supply their inverse identity.
//
//nolint:gocognit // Planning handles optional inverse metadata before constructing independent executors.
func GenerateTransactionStreamFunc(
	ctx queryContext, stmt sqlparser.Statement,
) (func() internaldto.ExecutorOutput, func() internaldto.ExecutorOutput, bool) {
	if _, ok := stmt.(*sqlparser.Insert); !ok {
		f, handled := GenerateStreamFunc(ctx, stmt)
		return f, nil, handled
	}
	dialect, err := backendDialect(ctx)
	if err != nil {
		f, handled := GenerateStreamFunc(ctx, stmt)
		return f, nil, handled
	}
	translated, err := newDocTranslator(ctx.GetCurrentProvider(), dialect).mutation(stmt)
	if err != nil {
		f, handled := GenerateStreamFunc(ctx, stmt)
		return f, nil, handled
	}
	target := translated.getQuery().Target()
	if target == nil {
		f, handled := GenerateStreamFunc(ctx, stmt)
		return f, nil, handled
	}
	address := target.Resource().Handle()
	declaration, err := omnisdk.DocInverse(registryRoot(ctx), address, "insert")
	if err != nil || declaration == nil {
		f, handled := GenerateStreamFunc(ctx, stmt)
		return f, nil, handled
	}
	capture := &inverseCapture{declaration: declaration}
	forward, handled := GenerateStreamFunc(observedQueryContext{queryContext: ctx, observer: capture}, stmt)
	if !handled {
		return nil, nil, false
	}
	run := func() internaldto.ExecutorOutput {
		result := forward()
		if result.GetError() == nil {
			message := "Undo the insert on " + strings.TrimPrefix(address, UnstablePrefix)
			result.SetUndoLog(binlog.NewSimpleLogEntry(nil, []string{message}))
		}
		return result
	}
	inverse := func() internaldto.ExecutorOutput { return executeDeclaredInverse(ctx, address, declaration, capture) }
	return run, inverse, true
}

func configureMutationObserver(ctx queryContext, mutation bool, args *omnisdk.Args) {
	if observed, ok := ctx.(interface {
		mutationObserver() omnisdk.MutationCompletion
	}); ok && mutation {
		args.MutationCompletion = completionChain{first: args.MutationCompletion, second: observed.mutationObserver()}
	}
}
func (c observedQueryContext) GetOutErrFile() io.Writer {
	if host, ok := c.queryContext.(interface{ GetOutErrFile() io.Writer }); ok {
		return host.GetOutErrFile()
	}
	return io.Discard
}

func executeDeclaredInverse(
	ctx queryContext, address string, declaration omnisdk.DeclaredInverse, capture *inverseCapture,
) internaldto.ExecutorOutput {
	capture.mu.Lock()
	rows := append([]map[string]string(nil), capture.rows...)
	capture.mu.Unlock()
	for i := len(rows) - 1; i >= 0; i-- {
		params, err := inverseParameters(ctx, address, declaration.Method(), rows[i])
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		node, e := omnisdk.NewInverseMutationNode("inverse", address, "delete", declaration.Method(), params)
		if e != nil {
			return internaldto.NewErroneousExecutorOutput(e)
		}
		graph, e := omnisdk.NewGraph([]omnisdk.Node{node}, nil)
		if e != nil {
			return internaldto.NewErroneousExecutorOutput(e)
		}
		args, e := previewArgs(ctx, address, params)
		if e != nil {
			return internaldto.NewErroneousExecutorOutput(e)
		}
		p, e := omnisdk.NewGraphQuery(registryRoot(ctx), graph, args)
		if e != nil {
			return internaldto.NewErroneousExecutorOutput(e)
		}
		cursor, e := p.Open(context.Background())
		if e != nil {
			return internaldto.NewErroneousExecutorOutput(e)
		}
		output := drainMutation(cursor)
		if e = output.GetError(); e != nil {
			return internaldto.NewErroneousExecutorOutput(mutationHTTPFailure(ctx, "undo", e))
		}
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func inverseParameters(
	ctx queryContext, address, method string, original map[string]string,
) (map[string]string, error) {
	table, err := omnisdk.DescribeMutation(registryRoot(ctx), address, "delete")
	if err != nil {
		return nil, err
	}
	for _, signature := range table.Methods() {
		if signature.Method() != method {
			continue
		}
		params := map[string]string{}
		for _, p := range signature.Params() {
			if value, ok := original[p.Name()]; ok {
				params[p.Name()] = value
			}
		}
		return params, nil
	}
	return nil, fmt.Errorf("declared inverse method %s not found", method)
}

type mutationPresentationError struct {
	cause   error
	message string
}

func (e mutationPresentationError) Error() string { return e.message }
func (e mutationPresentationError) Unwrap() error { return e.cause }
func mutationHTTPFailure(ctx queryContext, verb string, err error) error {
	var response omnisdk.HTTPStatusError
	if !errors.As(err, &response) {
		return err
	}
	if output, ok := ctx.(interface{ GetOutErrFile() io.Writer }); ok {
		diagnostic := fmt.Sprintf(
			"http response status code: %d, response body: %s", response.StatusCode(), response.ResponseBody(),
		)
		if response.ResponseBody() == "" {
			diagnostic = fmt.Sprintf("http response status code: %d, response body is nil", response.StatusCode())
		}
		//nolint:errcheck // Diagnostics do not determine the outcome.
		fmt.Fprintln(output.GetOutErrFile(), diagnostic)
	}
	message := fmt.Sprintf(
		"%s over HTTP error: %d %s", verb, response.StatusCode(), strings.ToUpper(http.StatusText(response.StatusCode())),
	)
	return mutationPresentationError{cause: err, message: message}
}
