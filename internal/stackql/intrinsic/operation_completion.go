package intrinsic

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/stackql-labs/omnisdk/pkg/docparse/aot"
	"github.com/stackql-labs/omnisdk/pkg/docparse/stackqldoc"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type awaitedDocQuery struct{ docQuery }

func (awaitedDocQuery) awaitsMutation() bool { return true }

func (q awaitedDocQuery) getRelations() map[string]omnisdk.Relation {
	return queryRelations(q.docQuery)
}

func mutationAwait(stmt sqlparser.Statement) bool {
	var comments sqlparser.Comments
	switch node := stmt.(type) {
	case *sqlparser.Insert:
		comments = node.Comments
	case *sqlparser.Update:
		comments = node.Comments
	case *sqlparser.Delete:
		comments = node.Comments
	}
	return sqlparser.ExtractCommentDirectives(comments).IsSet("AWAIT")
}
func queryAwaitsMutation(q docQuery) bool {
	a, ok := q.(interface{ awaitsMutation() bool })
	return ok && a.awaitsMutation()
}

const (
	operationPollInterval = 10 * time.Second
	operationWaitTimeout  = 10 * time.Minute
)

type completionRead func(context.Context, string, map[string]any) (omnisdk.Row, error)
type completionWait func(context.Context, time.Duration) error

type linkedOperationCompletion struct {
	mu           sync.Mutex
	read         completionRead
	wait         completionWait
	writer       io.Writer
	readResource bool
}

func newLinkedOperationCompletion(
	read completionRead, wait completionWait, writer io.Writer, resourceNeeded ...bool,
) omnisdk.MutationCompletion {
	return &linkedOperationCompletion{read: read, wait: wait, writer: writer,
		readResource: len(resourceNeeded) == 0 || resourceNeeded[0]}
}

func newOperationCompletion(
	ctx queryContext, address string, resourceNeeded ...bool,
) (omnisdk.MutationCompletion, error) {
	parts := strings.Split(address, ".")
	if len(parts) != 3 || strings.TrimPrefix(parts[0], UnstablePrefix) != "google" || parts[1] != "compute" {
		return nil, fmt.Errorf("AWAIT completion is not supported for %s", address)
	}
	bundle := strings.TrimPrefix(parts[0], UnstablePrefix)
	root, err := docRoot(ctx, bundle)
	if err != nil {
		return nil, err
	}
	catalogue, err := stackqldoc.Open(os.DirFS(root))
	if err != nil {
		return nil, err
	}
	var writer io.Writer = io.Discard
	if output, ok := ctx.(interface{ GetOutErrFile() io.Writer }); ok {
		writer = output.GetOutErrFile()
	}
	read := func(runCtx context.Context, link string, original map[string]any) (omnisdk.Row, error) {
		return readCompletionLink(ctx, runCtx, root, bundle, parts[1], catalogue, link, original)
	}
	wait := func(runCtx context.Context, duration time.Duration) error {
		timer := time.NewTimer(duration)
		defer timer.Stop()
		select {
		case <-runCtx.Done():
			return runCtx.Err()
		case <-timer.C:
			return nil
		}
	}
	return newLinkedOperationCompletion(read, wait, writer, resourceNeeded...), nil
}

func (c *linkedOperationCompletion) Complete(
	parent context.Context, _ string, verb string, inputs map[string]any, operation omnisdk.Row,
) (omnisdk.Row, error) {
	ctx, cancel := context.WithTimeout(parent, operationWaitTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	kind, _ := operation["kind"].(string)
	if kind != "compute#operation" {
		return nil, fmt.Errorf("AWAIT expected compute#operation, got %q", kind)
	}
	action, _ := operation["operationType"].(string)
	if action == "" {
		return nil, fmt.Errorf("AWAIT operation has no operationType")
	}
	status, _ := operation["status"].(string)
	poll, _ := operation["selfLink"].(string)
	target, _ := operation["targetLink"].(string)
	elapsed := time.Duration(0)
	for status != "DONE" {
		if status != "PENDING" && status != "RUNNING" {
			return nil, fmt.Errorf("AWAIT operation has unsupported status %q", status)
		}
		if poll == "" {
			return nil, fmt.Errorf("AWAIT operation has no selfLink")
		}
		if err := c.wait(ctx, operationPollInterval); err != nil {
			return nil, err
		}
		elapsed += operationPollInterval
		c.diagnosticf("%s: %s in progress, %d seconds elapsed\n", kind, action, int(elapsed.Seconds()))
		next, err := c.read(ctx, poll, inputs)
		if err != nil {
			return nil, err
		}
		operation = next
		status, _ = operation["status"].(string)
		if link, ok := operation["targetLink"].(string); ok && link != "" {
			target = link
		}
	}
	if failure, exists := operation["error"]; exists && failure != nil {
		return nil, fmt.Errorf("AWAIT operation failed: %v", failure)
	}
	c.diagnosticf("%s: %s complete\n", kind, action)
	return c.completedResult(ctx, verb, action, target, inputs, operation)
}

func (c *linkedOperationCompletion) completedResult(
	ctx context.Context, verb, action, target string, inputs map[string]any, operation omnisdk.Row,
) (omnisdk.Row, error) {
	if operationDeletesResource(verb, action) {
		return nil, nil //nolint:nilnil // Successful DELETE has no resource row.
	}
	if !c.readResource {
		return operation, nil
	}
	if target == "" {
		return nil, fmt.Errorf("AWAIT completed without targetLink")
	}
	return c.read(ctx, target, inputs)
}

func operationDeletesResource(verb, action string) bool {
	return verb == "delete" || (verb == "exec" && action == "delete")
}

func (c *linkedOperationCompletion) diagnosticf(format string, values ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.writer, format, values...) //nolint:errcheck // Diagnostics do not determine completion.
}

// Links select only declared GET operations within the original provider/service.
// The returned host is never used for dispatch; SDK catalog routing and auth remain authoritative.
func readCompletionLink(
	ctx queryContext, runCtx context.Context, root, bundle, service string,
	catalogue aot.Catalog, link string, original map[string]any,
) (omnisdk.Row, error) {
	address, params, err := completionAddress(catalogue, service, link)
	if err != nil {
		return nil, err
	}
	if err = checkCompletionScope(params, original); err != nil {
		return nil, err
	}
	args, err := previewArgs(ctx, bundle, params)
	if err != nil {
		return nil, err
	}
	args.HTTPReadErrors = nil // Completion absence remains an unknown outcome.
	plan, err := omnisdk.NewSelectFromCatalog(root, address, args)
	if err != nil {
		return nil, err
	}
	rows, err := plan.Open(runCtx)
	if err != nil {
		return nil, err
	}
	return singleCompletionRow(rows)
}

func checkCompletionScope(params map[string]string, original map[string]any) error {
	for name, value := range params {
		if supplied, present := original[name]; present && fmt.Sprint(supplied) != value {
			return fmt.Errorf("AWAIT link changes explicit %s scope", name)
		}
	}
	return nil
}

func completionAddress(catalogue aot.Catalog, service, link string) (string, map[string]string, error) {
	resources, err := catalogue.Resources(service)
	if err != nil {
		return "", nil, err
	}
	for _, resource := range resources {
		address, ok := catalogue.Address(service, resource)
		if !ok {
			continue
		}
		operations, opErr := catalogue.Operations(address, "select")
		if opErr != nil {
			continue
		}
		for _, operation := range operations {
			if operation.Request().Method() != "GET" {
				continue
			}
			if params, matched := completionLinkParams(operation.Request().URL(), link); matched {
				return address, params, nil
			}
		}
	}
	return "", nil, fmt.Errorf("AWAIT link does not match a declared GET operation in %s", service)
}

func configureMutationCompletion(ctx queryContext, translated docQuery, args *omnisdk.Args) error {
	if !queryAwaitsMutation(translated) {
		return nil
	}
	completion, err := newOperationCompletion(
		ctx, translated.getQuery().Target().Resource().Handle(), len(translated.getOutputs()) > 0)
	if err != nil {
		return err
	}
	args.MutationCompletion = completion
	return nil
}

func singleCompletionRow(rows omnisdk.Rows) (omnisdk.Row, error) {
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("AWAIT read returned no row")
	}
	row := maps.Clone(rows.Row())
	if rows.Next() {
		return nil, fmt.Errorf("AWAIT read returned multiple rows")
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return row, nil
}

func completionLinkParams(template, link string) (map[string]string, bool) {
	declared, err := url.Parse(template)
	if err != nil {
		return nil, false
	}
	actual, err := url.Parse(link)
	if err != nil || actual.Scheme == "" || actual.Host == "" || actual.RawQuery != "" {
		return nil, false
	}
	expected := strings.Split(strings.Trim(declared.Path, "/"), "/")
	received := strings.Split(strings.Trim(actual.Path, "/"), "/")
	if len(expected) != len(received) {
		return nil, false
	}
	params := map[string]string{}
	for i, part := range expected {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			if received[i] == "" {
				return nil, false
			}
			if prior, exists := params[name]; exists && prior != received[i] {
				return nil, false
			}
			params[name] = received[i]
		} else if part != received[i] {
			return nil, false
		}
	}
	return params, true
}
