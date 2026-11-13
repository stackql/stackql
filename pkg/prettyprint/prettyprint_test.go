package prettyprint

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/sirupsen/logrus"
)

func TestRenderTemplateVarAndDelimitUsesRootName(t *testing.T) {
	printer := NewPrettyPrinter(NewPrettyPrintContext(false, 2, 0, "'", logrus.New()))
	tmpl, err := template.New("value").Parse(printer.RenderTemplateVarAndDelimit("resourceGroupName"))
	if err != nil {
		t.Fatal(err)
	}

	var got bytes.Buffer
	if err := tmpl.Execute(&got, map[string]string{"resourceGroupName": "rg-1"}); err != nil {
		t.Fatal(err)
	}
	if got.String() != "'rg-1'" {
		t.Fatalf("rendered value = %q, want %q", got.String(), "'rg-1'")
	}
}

func TestPrintTemplatedJSONUsesRootNameForNestedValues(t *testing.T) {
	printer := NewPrettyPrinter(NewPrettyPrintContext(false, 2, 0, "'", logrus.New()))
	rendered, err := printer.PrintTemplatedJSON(map[string]interface{}{
		"name": "\"{{ .data__settings.name }}\"",
	})
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.New("body").Parse(rendered)
	if err != nil {
		t.Fatal(err)
	}

	var got bytes.Buffer
	data := map[string]interface{}{
		"data__settings": map[string]string{"name": "setting-1"},
	}
	if err := tmpl.Execute(&got, data); err != nil {
		t.Fatal(err)
	}
	if got.String() != `'{ "name": "setting-1" }'` {
		t.Fatalf("rendered body = %q, want %q", got.String(), `'{ "name": "setting-1" }'`)
	}
}
