package metadatavisitors

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/docparse/aot"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/pkg/prettyprint"
)

// ToOmniInsertStatement renders host SQL from the SDK's declared request shape.
//
//nolint:gocognit // Render parameter and body declarations in one deterministic pass.
func ToOmniInsertStatement(
	columns sqlparser.Columns, method omnisdk.MethodSignature, extended, requiredOnly bool,
	printer, placeholders *prettyprint.PrettyPrinter,
) (string, error) {
	selected := getColsMap(columns)
	included := make(map[string]bool)
	var names, values, hints []string
	appendField := func(name, value, hint string) {
		names, values, hints = append(names, printer.RenderColumnName(name)), append(values, value), append(hints, hint)
		included[name] = true
	}
	metadata, _ := method.(omnisdk.RequestMetadata)
	inputName := func(name string) string {
		if metadata != nil {
			return metadata.InputName(name)
		}
		return name
	}
	parameters := method.Params()
	slices.SortFunc(parameters, func(a, b omnisdk.ParamSignature) int { return strings.Compare(a.Name(), b.Name()) })
	for _, parameter := range parameters {
		name := inputName(parameter.Name())
		if parameter.In() == aot.InBody || (columns != nil && !selected[name]) {
			continue
		}
		if columns == nil && (!extended || requiredOnly) && !parameter.Required() {
			continue
		}
		value := printer.RenderTemplateVarAndDelimit(name)
		if typed, ok := parameter.(interface{ Type() string }); ok && typed.Type() != "" && typed.Type() != "string" {
			value = printer.RenderTemplateVarNoDelimit(name)
		}
		appendField(name, value, placeholders.RenderTemplateVarPlaceholderNoDelimit(name))
	}
	if metadata != nil && metadata.RequestBodySchema() != nil {
		body := metadata.RequestBodySchema()
		properties := slices.Clone(body.Properties())
		slices.Sort(properties)
		for _, property := range properties {
			name := "data__" + inputName(property)
			if columns != nil && !selected[name] {
				continue
			}
			shape, exists := body.Property(property)
			if !exists || !includeRequestProperty(body, shape, property, requiredOnly) {
				continue
			}
			value, err := printer.PrintTemplatedJSON(requestTemplate(shape, ".values."+name, false, 0))
			if err != nil {
				return "", err
			}
			hint, err := placeholders.PrintPlaceholderJSON(requestTemplate(shape, name, true, 0))
			if err != nil {
				return "", err
			}
			appendField(name, value, placeholders.RenderTemplateVarPlaceholderKeyNoDelimit(name)+": "+hint)
		}
	}
	if err := checkAllColumnsPresent(columns, included); err != nil {
		return "", err
	}
	return "<<<jsonnet\n{\n" + strings.Join(hints, ",\n") + "\n}\n>>>\nINSERT INTO %s(\n" +
		strings.Join(names, ",\n") + "\n)\nSELECT\n" + strings.Join(values, ",\n") + "\n;\n", nil
}

func includeRequestProperty(parent, property aot.Schema, name string, requiredOnly bool) bool {
	required := false
	if declaration, ok := parent.(aot.RequiredSchema); ok {
		required = slices.Contains(declaration.RequiredProperties(), name)
	}
	if required {
		return true
	}
	if readOnly, ok := property.(aot.ReadOnlySchema); ok && readOnly.ReadOnly() {
		return false
	}
	return !requiredOnly
}

const maxRequestTemplateDepth = 1

func requestTemplate(shape aot.Schema, name string, placeholder bool, depth int) any {
	variable := "{{ " + name + " }}"
	if placeholder {
		variable = "<< " + name + " >>"
	}
	if shape == nil {
		return variable
	}
	if item, array := shape.Items(); array {
		return []any{requestTemplate(item, name+"[0]", placeholder, depth)}
	}
	if len(shape.Properties()) > 0 {
		if depth >= maxRequestTemplateDepth {
			return variable
		}
		object := make(map[string]any)
		for _, key := range shape.Properties() {
			property, ok := shape.Property(key)
			if ok && includeRequestProperty(shape, property, key, false) {
				childName := name + "." + key
				if placeholder {
					childName = key
				}
				object[key] = requestTemplate(property, childName, placeholder, depth+1)
			}
		}
		return object
	}
	if typed, ok := shape.(aot.TypedSchema); ok && typed.Type() == "string" && !placeholder {
		return fmt.Sprintf("\"%s\"", variable)
	}
	return variable
}
