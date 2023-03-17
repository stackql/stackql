package prettyprint

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

type Context struct {
	PrettyPrint     bool
	Indentation     int
	BaseIndentation int
	Delimiter       string
	Logger          *logrus.Logger
}

type PrettyPrinter struct {
	prettyPrintContext Context
	currentIndentation int
}

func NewPrettyPrintContext(
	isPrettyPrint bool,
	indentation int,
	baseIndentation int,
	delimiter string,
	logger *logrus.Logger,
) Context {
	return Context{
		PrettyPrint:     isPrettyPrint,
		Indentation:     indentation,
		BaseIndentation: baseIndentation,
		Delimiter:       delimiter,
		Logger:          logger,
	}
}

func NewPrettyPrinter(ppCtx Context) *PrettyPrinter {
	return &PrettyPrinter{
		prettyPrintContext: ppCtx,
		currentIndentation: ppCtx.BaseIndentation,
	}
}

func (pp *PrettyPrinter) getCurrentIndentation() int {
	return pp.currentIndentation
}

func (pp *PrettyPrinter) incrementCurrentIndentation() int {
	return pp.setCurrentIndentation(pp.currentIndentation + pp.prettyPrintContext.Indentation)
}

func (pp *PrettyPrinter) setCurrentIndentation(indentation int) int {
	pp.currentIndentation = indentation
	return pp.currentIndentation
}

func (pp *PrettyPrinter) baseIndentationAndDelimit(rendition string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		strings.Repeat(" ", pp.prettyPrintContext.BaseIndentation),
		pp.prettyPrintContext.Delimiter,
		rendition,
		pp.prettyPrintContext.Delimiter,
	)
}

func (pp *PrettyPrinter) baseIndentationNoDelimit(rendition string) string {
	return fmt.Sprintf(
		"%s%s%s",
		strings.Repeat(" ", pp.prettyPrintContext.BaseIndentation),
		strings.Repeat(" ", len(pp.prettyPrintContext.Delimiter)),
		rendition,
	)
}

func (pp *PrettyPrinter) baseIndentationColumnNoDelimit(rendition string) string {
	return fmt.Sprintf(
		"%s%s",
		strings.Repeat(" ", pp.prettyPrintContext.BaseIndentation),
		rendition,
	)
}

func (pp *PrettyPrinter) RenderColumnName(cn string) string {
	return pp.baseIndentationColumnNoDelimit(cn)
}

func (pp *PrettyPrinter) RenderTemplateVarAndDelimit(tv string) string {
	return pp.baseIndentationAndDelimit(fmt.Sprintf("{{ .values.%s }}", tv))
}

func (pp *PrettyPrinter) RenderTemplateVarNoDelimit(tv string) string {
	return pp.baseIndentationNoDelimit(fmt.Sprintf("{{ .values.%s }}", tv))
}

func (pp *PrettyPrinter) RenderTemplateVarPlaceholderNoDelimit(tv string) string {
	return pp.baseIndentationNoDelimit(fmt.Sprintf("%s: << %s >>", tv, tv))
}

func (pp *PrettyPrinter) RenderTemplateVarPlaceholderKeyNoDelimit(tv string) string {
	return pp.baseIndentationNoDelimit(tv)
}

func (pp *PrettyPrinter) PrintTemplatedJSON(body interface{}) (string, error) {
	rv, err := pp.printTemplatedJSON(body)
	if err != nil {
		return "", err
	}
	switch body.(type) {
	case map[string]interface{}, []interface{}:
		return pp.baseIndentationAndDelimit(rv), err
	case string:
		trimmed := strings.TrimSuffix(strings.TrimPrefix(rv, `"`), `"`)
		if rv == trimmed {
			return pp.baseIndentationNoDelimit(rv), err
		}
		return pp.baseIndentationAndDelimit(trimmed), err
	default:
		return "", fmt.Errorf("cannot perform PrintTemplatedJSON() for type = %T", rv)
	}
}

func (pp *PrettyPrinter) PrintPlaceholderJSON(body interface{}) (string, error) {
	rv, err := pp.printPlaceholderJSON(body)
	if err != nil {
		return "", err
	}
	switch body.(type) {
	case map[string]interface{}, []interface{}:
		return pp.baseIndentationNoDelimit(rv), err
	case string:
		trimmed := strings.TrimSuffix(strings.TrimPrefix(rv, `"`), `"`)
		if rv == trimmed {
			return pp.baseIndentationNoDelimit(rv), err
		}
		return pp.baseIndentationNoDelimit(trimmed), err
	default:
		return "", fmt.Errorf("cannot perform PrintTemplatedJSON() for type = %T", rv)
	}
}

func (pp *PrettyPrinter) printTemplatedJSON(body interface{}) (string, error) {
	startPos := pp.getCurrentIndentation()
	switch bt := body.(type) {
	case map[string]interface{}:
		var keys []string
		for k := range bt {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var keyVals []string
		keyIndent := pp.incrementCurrentIndentation()
		for _, k := range keys {
			indent := ""
			val, err := pp.printTemplatedJSON(bt[k])
			if err != nil {
				pp.setCurrentIndentation(startPos)
				return "", err
			}
			if pp.prettyPrintContext.PrettyPrint {
				indent = strings.Repeat(" ", keyIndent)
			}
			keyVals = append(keyVals, fmt.Sprintf(`%s"%s": %s`, indent, k, val))
		}
		if pp.prettyPrintContext.PrettyPrint {
			terminalIndent := strings.Repeat(" ", startPos+len(pp.prettyPrintContext.Delimiter))
			pp.setCurrentIndentation(startPos)
			return fmt.Sprintf("{\n%s\n%s}", strings.Join(keyVals, ",\n"), terminalIndent), nil
		}
		return fmt.Sprintf(`{ %s }`, strings.Join(keyVals, ", ")), nil
	case []interface{}:
		var vals []string
		elemPos := pp.incrementCurrentIndentation()
		for _, v := range bt {
			val, err := pp.printTemplatedJSON(v)
			if err != nil {
				pp.prettyPrintContext.Logger.Infof("%s\n", err.Error())
				pp.setCurrentIndentation(startPos)
				return "", err
			}
			vals = append(vals, val)
		}
		if pp.prettyPrintContext.PrettyPrint {
			rv := fmt.Sprintf("[\n%s%s\n%s]",
				strings.Repeat(" ", elemPos),
				strings.Join(
					vals,
					",\n"+strings.Repeat(" ", elemPos),
				),
				strings.Repeat(" ", startPos+len(pp.prettyPrintContext.Delimiter)),
			)
			pp.setCurrentIndentation(startPos)
			return rv, nil
		}
		pp.setCurrentIndentation(startPos)
		return fmt.Sprintf("[ %s ]", strings.Join(vals, ", ")), nil
	case string:
		return bt, nil
	default:
		return "", fmt.Errorf("cannot perform template marshal for type = %T", bt)
	}
}

//nolint:gocognit
func (pp *PrettyPrinter) printPlaceholderJSON(body interface{}) (string, error) {
	startPos := pp.getCurrentIndentation()
	switch bt := body.(type) {
	case map[string]interface{}:
		var keys []string
		for k := range bt {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var keyVals []string
		keyIndent := pp.incrementCurrentIndentation()
		for _, k := range keys {
			indent := ""
			val, err := pp.printPlaceholderJSON(bt[k])
			if err != nil {
				pp.setCurrentIndentation(startPos)
				return "", err
			}
			if pp.prettyPrintContext.PrettyPrint {
				indent = strings.Repeat(" ", keyIndent)
			}
			keyVals = append(keyVals, fmt.Sprintf(`%s%s: %s`, indent, k, val))
		}
		if pp.prettyPrintContext.PrettyPrint || true {
			terminalIndent := strings.Repeat(" ", startPos+len(pp.prettyPrintContext.Delimiter))
			decrementIndent := strings.Repeat(" ", startPos+len(pp.prettyPrintContext.Delimiter))
			pp.setCurrentIndentation(startPos)
			return fmt.Sprintf(
				"{\n%s%s\n%s}",
				terminalIndent,
				strings.Join(keyVals, fmt.Sprintf(",\n%s", terminalIndent)),
				decrementIndent), nil
		}
		return fmt.Sprintf(` %s `, strings.Join(keyVals, ",\n")), nil
	case []interface{}:
		var vals []string
		elemPos := pp.incrementCurrentIndentation()
		for _, v := range bt {
			val, err := pp.printPlaceholderJSON(v)
			if err != nil {
				pp.prettyPrintContext.Logger.Infof("%s\n", err.Error())
				pp.setCurrentIndentation(startPos)
				return "", err
			}
			vals = append(vals, val)
		}
		if pp.prettyPrintContext.PrettyPrint {
			rv := fmt.Sprintf("[\n%s%s\n%s]",
				strings.Repeat(" ", elemPos),
				strings.Join(
					vals,
					",\n"+strings.Repeat(" ", elemPos),
				),
				strings.Repeat(" ", startPos+len(pp.prettyPrintContext.Delimiter)),
			)
			pp.setCurrentIndentation(startPos)
			return rv, nil
		}
		pp.setCurrentIndentation(startPos)
		return fmt.Sprintf("[ %s ]", strings.Join(vals, ", ")), nil
	case string:
		return bt, nil
	default:
		return "", fmt.Errorf("cannot perform template marshal for type = %T", bt)
	}
}
