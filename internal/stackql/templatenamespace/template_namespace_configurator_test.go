package templatenamespace

import (
	"regexp"
	"testing"
	"text/template"

)

func TestGetObjectName(t *testing.T){
	regex := regexp.MustCompile(`(?P<objectName>\w+)`)
	templ := template.Must(template.New("test").Parse("{{.objectName}}"))
	config, err := NewTemplateNamespaceConfigurator(regex,templ)
	if err != nil {
		t.Fatal(err)
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
		actual := config.GetObjectName(c.input)
		if actual != c.expexted{
			t.Errorf("GetObjectNAme(%q) = %q, wants %q",c.input,actual,c.expexted)
		}
	}
}

func TestRenderTemplate(t *testing.T){
	regex := regexp.MustCompile(`(?P<objectName>\w+)`)
	templete := template.Must(template.New("test").Parse("TestRenderTemplate's {{.objectName}}"))
	config,err := NewTemplateNamespaceConfigurator(regex,templete)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		input string
		expected string
	}{
		{"one","TestRenderTemplate's one"},
		{"two","TestRenderTemplate's two"},
		{"abc12","TestRenderTemplate's abc12"},
	}

	for _,c := range cases{
		actual,err := config.RenderTemplate(c.input)
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != c.expected{
			t.Errorf("RenderTemplate(%q)= %q, and want %q.",c.input,actual,c.expected)
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