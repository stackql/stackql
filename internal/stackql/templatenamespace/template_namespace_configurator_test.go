package templatenamespace

import (
	"regexp"
	"testing"
	"text/template"

	//"golang.org/x/tools/go/expect"
)

func TestGetObjectName(t *testing.T){
	regex := regexp.MustCompile(`(?P<objectName>\w+)`)
	templ := template.Must(template.New("test").Parse("{{.objectName}}"))
	config, Error := NewTemplateNamespaceConfigurator(regex,templ)
	if Error != nil {
		t.Fatal(Error)
	}

	cases:= []struct {
		input string
		expexted string
	}{
		{"one","one"},
		{"two","two"},
		{"abc123","abc123"},
		{"",""},
	}

	for _,c:= range cases{
		actual,Error := config.RenderTemplate(c.input)
		if Error != nil {
			t.Error(Error)
			continue
		}
		if actual != c.expexted{
			t.Errorf("RenderTemp(%q) = %q, wants %q",c.input,actual,c.expexted)
		}
	}
}

func TestIsAllowed(t *testing.T){
	regex := regexp.MustCompile(`(?P<objectName>\w+)`)
	temp := template.Must(template.New("test").Parse("{{.objectName}}"))
	conf,e := NewTemplateNamespaceConfigurator(regex,temp)
	if e !=nil{
		t.Fatal(e)
	}
	cases := []struct {
		input string
		expected bool
	}{
		{"one",true},
		{"twoo",true},
		{"abc123",true},
		{"",false},
	}

	for _, c:= range cases{
		actual := conf.IsAllowed(c.input)
		if actual!= c.expected {
			t.Errorf("conf.IsAllowed(%q)= %v requires %v",c.input,actual,c.expected)
		}
	}
}