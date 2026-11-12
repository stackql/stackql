package intrinsic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/util"
)

const maxSchemaPathDepth = 32

// Presentation uses the already verified signature, never a second unverified registry read.
//
//nolint:gocognit,nestif // Bounded schema selection and legacy SQL row presentation share one path.
func describeSchema(
	ctx queryContext, signature omnisdk.MethodSignature,
	options omnisdk.MethodSchemaOptions, extended bool,
) internaldto.ExecutorOutput {
	if options.Depth == 0 {
		options.Depth = 1
	}
	if err := validateSchemaOptions(options); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	rows, budget := map[string]map[string]interface{}{}, 4096
	appendField := func(name, kind string, field omnisdk.FieldSchema, typ string) error {
		if options.ParamType != "" && options.ParamType != kind && !(options.ParamType == "input" &&
			strings.HasPrefix(kind, "input_")) {
			return nil
		}
		path := strings.Split(options.Path, ".")
		if options.Path != "" {
			if path[0] != name && !(field != nil && strings.HasPrefix(name, "data__") && path[0] == "data__"+field.Name()) {
				return nil
			}
			selected, err := schemaFieldAtPath(field, path[1:], options.Path)
			if err != nil {
				return err
			}
			field = selected
			if len(path) > 1 {
				name = field.Name()
			}
		}
		shape := map[string]any{"type": typ}
		description := ""
		if field != nil {
			var err error
			shape, err = schemaShape(field, options.Depth, options.Descriptions, &budget)
			if err != nil {
				return err
			}
			typ = field.Type()
			if options.Descriptions {
				description = field.Description()
			}
		} else {
			budget--
			if budget < 0 {
				return fmt.Errorf("schema view exceeds 4096 nodes")
			}
		}
		data, err := json.Marshal(shape)
		if err != nil {
			return err
		}
		row := map[string]interface{}{"name": name, "type": typ, "param_type": kind, "shape": string(data)}
		if extended {
			row["description"] = description
		}
		rows[fmt.Sprintf("%06d", len(rows))] = row
		return nil
	}
	if err := appendMethodInputs(signature, options.IncludeGlobal, appendField); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	wantOutput := options.ParamType == "" || options.ParamType == "output"
	// Input drill-down remains usable even if the response's source schema is malformed.
	if options.Path != "" && len(rows) > 0 {
		wantOutput = false
	}
	if wantOutput {
		if err := appendMethodOutputs(signature, appendField); err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
	}
	if options.Path != "" && len(rows) == 0 {
		return internaldto.NewErroneousExecutorOutput(fmt.Errorf("schema path %q does not exist", options.Path))
	}
	order := []string{"name", "type", "param_type", "shape"}
	if extended {
		order = append(order, "description")
	}
	return prepare(ctx, order, rows, util.DefaultRowSort)
}

// schemaShape adapts the SDK read-only graph to the existing JSON shape column.
//
//nolint:gocognit,nestif // Recursive JSON-schema presentation has explicit depth and node bounds.
func schemaShape(field omnisdk.FieldSchema, depth int, descriptions bool, budget *int) (map[string]any, error) {
	*budget--
	if *budget < 0 {
		return nil, fmt.Errorf("schema view exceeds 4096 nodes; narrow path or depth")
	}
	shape := map[string]any{"type": field.Type()}
	if field.Format() != "" {
		shape["format"] = field.Format()
	}
	if values := field.Enum(); len(values) > 0 {
		shape["enum"] = values
	}
	if descriptions && field.Description() != "" {
		shape["description"] = field.Description()
	}
	if wire := field.WireName(); wire != "" && wire != field.Name() {
		shape["wire_name"] = wire
	}
	children := field.Properties()
	shape["child_count"] = len(children)
	if len(children) > 0 {
		if depth == 1 {
			shape["truncated"] = true
		} else {
			properties := map[string]any{}
			var required []string
			for _, child := range children {
				view, err := schemaShape(child, depth-1, descriptions, budget)
				if err != nil {
					return nil, err
				}
				properties[child.Name()] = view
				if child.Required() {
					required = append(required, child.Name())
				}
			}
			shape["properties"] = properties
			if len(required) > 0 {
				shape["required"] = required
			}
		}
	}
	if item, ok := field.Items(); ok {
		if depth == 1 {
			*budget--
			if *budget < 0 {
				return nil, fmt.Errorf("schema view exceeds 4096 nodes")
			}
			shape["items"] = map[string]any{
				"type": item.Type(), "child_count": len(item.Properties()),
				"truncated": len(item.Properties()) > 0,
			}
			shape["truncated"] = true
		} else {
			view, err := schemaShape(item, depth-1, descriptions, budget)
			if err != nil {
				return nil, err
			}
			shape["items"] = view
		}
	}
	return shape, nil
}

//nolint:gocognit // Validates and binds the transport-neutral metadata query.
func methodSchemaSelect(
	ctx queryContext, node *sqlparser.Select, resource string,
) (func() internaldto.ExecutorOutput, bool) {
	if resource != "method_fields" {
		return refuse(fmt.Errorf("metadata declares only method_fields")), true
	}
	params, refusal := previewPredicates(node, relationAddress("metadata", resource))
	if refusal != nil {
		return refusal, true
	}
	provider, service := popPredicate(params, "provider"), popPredicate(params, "service")
	target, method := popPredicate(params, "resource"), popPredicate(params, "method")
	if provider == "" || service == "" || target == "" || method == "" {
		return refuse(fmt.Errorf("method_fields requires provider, service, resource and method")), true
	}
	options := omnisdk.MethodSchemaOptions{
		Path: popPredicate(params, "path"), ParamType: popPredicate(params, "param_type"),
	}
	if value := popPredicate(params, "depth"); value != "" {
		depth, err := strconv.Atoi(value)
		if err != nil {
			return refuse(fmt.Errorf("invalid schema depth: %w", err)), true
		}
		options.Depth = depth
	}
	for key, destination := range map[string]*bool{
		"descriptions": &options.Descriptions, "include_global": &options.IncludeGlobal,
	} {
		if value := popPredicate(params, key); value != "" {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return refuse(fmt.Errorf("invalid %s: %w", key, err)), true
			}
			*destination = parsed
		}
	}
	if len(params) > 0 {
		return refuse(fmt.Errorf("unknown method_fields predicates")), true
	}
	return func() internaldto.ExecutorOutput {
		if len(node.SelectExprs) != 1 {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("method_fields requires SELECT *"))
		}
		if _, ok := node.SelectExprs[0].(*sqlparser.StarExpr); !ok {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("method_fields requires SELECT *"))
		}
		statement := &sqlparser.DescribeMethod{
			Provider: sqlparser.NewTableIdent(provider), Service: sqlparser.NewTableIdent(service),
			Resource: sqlparser.NewTableIdent(target), Method: sqlparser.NewTableIdent(method),
		}
		if err := checkSQLNodes(ctx, statement); err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		bundle, ok := docProvider(provider)
		if !ok {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("method_fields requires a document-backed provider"))
		}
		opts, err := documentOptions(ctx)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		address := providerAddress(ctx, UnstablePrefix+bundle+"."+service+"."+target)
		signature, err := omnisdk.DescribeMethod(registryRoot(ctx), address, method, opts...)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		return describeSchema(ctx, signature, options, true)
	}, true
}

// flatSchemaField presents scalar parameter metadata through the same formatter.
type flatSchemaField struct {
	parameter omnisdk.ParamSignature
	name      string
}

func (f flatSchemaField) Name() string     { return f.name }
func (f flatSchemaField) WireName() string { return f.parameter.Name() }
func (f flatSchemaField) Required() bool   { return f.parameter.Required() }
func (f flatSchemaField) Type() string {
	if v, ok := f.parameter.(interface{ Type() string }); ok {
		return v.Type()
	}
	return ""
}
func (f flatSchemaField) Format() string {
	if v, ok := f.parameter.(interface{ Format() string }); ok {
		return v.Format()
	}
	return ""
}
func (f flatSchemaField) Description() string {
	if v, ok := f.parameter.(interface{ Description() string }); ok {
		return v.Description()
	}
	return ""
}
func (f flatSchemaField) Enum() []string {
	if v, ok := f.parameter.(interface{ Enum() []string }); ok {
		return v.Enum()
	}
	return nil
}
func (f flatSchemaField) Properties() []omnisdk.FieldSchema  { return nil }
func (f flatSchemaField) Items() (omnisdk.FieldSchema, bool) { return nil, false }

func appendMethodInputs(signature omnisdk.MethodSignature, includeGlobal bool,
	appendField func(string, string, omnisdk.FieldSchema, string) error,
) error {
	bodyFields := methodBodyFields(signature)
	naming, _ := signature.(omnisdk.RequestMetadata)
	for _, parameter := range signature.Params() {
		if global, ok := parameter.(interface{ Global() bool }); ok && global.Global() && !includeGlobal {
			continue
		}
		name := parameter.Name()
		if naming != nil {
			name = naming.InputName(name)
		}
		var field omnisdk.FieldSchema
		typ := ""
		if typed, ok := parameter.(interface{ Type() string }); ok {
			typ = typed.Type()
		}
		if parameter.In() == "body" {
			field = bodyFields[parameter.Name()]
			name = "data__" + name
		}
		if field == nil {
			field = flatSchemaField{parameter: parameter, name: name}
		}
		kind := "input_optional"
		if parameter.Required() {
			kind = "input_required"
		}
		if err := appendField(name, kind, field, typ); err != nil {
			return err
		}
	}

	return nil
}

func appendMethodOutputs(signature omnisdk.MethodSignature,
	appendField func(string, string, omnisdk.FieldSchema, string) error,
) error {
	rich, ok := signature.(omnisdk.SchemaMethodSignature)
	if !ok {
		for name, typ := range signature.ColumnTypes() {
			if err := appendField(name, "output", nil, typ.Type); err != nil {
				return err
			}
		}
		return nil
	}
	row, available := rich.RowSchema()
	if !available {
		if len(signature.Columns()) > 0 {
			return fmt.Errorf("method %s has an unavailable or self-referencing response row schema", signature.Method())
		}
		return nil
	}
	columns := signature.Columns()
	for index, field := range row.Properties() {
		name := field.Name()
		if index < len(columns) {
			name = columns[index]
		}
		if err := appendField(name, "output", field, field.Type()); err != nil {
			return err
		}
	}
	return nil
}

func methodBodyFields(signature omnisdk.MethodSignature) map[string]omnisdk.FieldSchema {
	fields := map[string]omnisdk.FieldSchema{}
	rich, ok := signature.(omnisdk.SchemaMethodSignature)
	if !ok {
		return fields
	}
	body, ok := rich.BodySchema()
	if !ok {
		return fields
	}
	for _, field := range body.Properties() {
		fields[field.Name()] = field
	}
	return fields
}

func schemaFieldAtPath(field omnisdk.FieldSchema, parts []string, path string) (omnisdk.FieldSchema, error) {
	for _, part := range parts {
		object, err := schemaObjectAtPath(field, path)
		if err != nil {
			return nil, err
		}
		var found omnisdk.FieldSchema
		for _, child := range object.Properties() {
			if child.Name() == part || child.WireName() == part {
				found = child
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("schema path %q does not exist", path)
		}
		field = found
	}
	return field, nil
}

func schemaObjectAtPath(field omnisdk.FieldSchema, path string) (omnisdk.FieldSchema, error) {
	if field == nil {
		return nil, fmt.Errorf("path %q has no nested schema", path)
	}
	for range maxSchemaPathDepth {
		item, ok := field.Items()
		if !ok {
			return field, nil
		}
		field = item
	}
	return nil, fmt.Errorf("recursive array path %q", path)
}

func validateSchemaOptions(options omnisdk.MethodSchemaOptions) error {
	if options.Depth < 1 || options.Depth > maxSchemaPathDepth {
		return fmt.Errorf("depth must be between 1 and 32")
	}
	switch options.ParamType {
	case "", "input", "output", "input_required", "input_optional":
		return nil
	}
	return fmt.Errorf("invalid param_type %q", options.ParamType)
}
