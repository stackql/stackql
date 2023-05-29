package preprocessor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"
	"unicode"

	jsonnet "github.com/google/go-jsonnet"
)

const (
	TripleLessThanToken               string = "<<<"
	TripleGreaterThanToken            string = ">>>"
	DefaultDeclarationBlockStartToken string = TripleLessThanToken
	DefaultDeclarationBlockEndToken   string = TripleGreaterThanToken
	JSONBlockType                     string = "json"
	JsonnetBlockType                  string = "jsonnet"
)

type printableMap map[string]interface{}

type printableSlice []interface{}

func printableMapFromMap(m map[string]interface{}) printableMap {
	rv := make(printableMap)
	for k, v := range m {
		switch vt := v.(type) {
		case map[string]interface{}:
			rv[k] = printableMapFromMap(vt)
		case []interface{}:
			rv[k] = printableSliceFromSlice(vt)
		default:
			rv[k] = vt
		}
	}
	return rv
}

func printableSliceFromSlice(sl []interface{}) printableSlice {
	var rv printableSlice
	for _, v := range sl {
		switch vt := v.(type) {
		case map[string]interface{}:
			rv = append(rv, printableMapFromMap(vt))
		case []interface{}:
			rv = append(rv, printableSliceFromSlice(vt))
		default:
			rv = append(rv, vt)
		}
	}
	return rv
}

func (m printableMap) String() string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

func (m printableSlice) String() string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

type DeclarationBlock struct {
	Type     string
	Contents map[string]interface{}
}

func parseVarList(varList []string) (map[string]string, error) {
	vars := make(map[string]string)
	for _, v := range varList {
		if v == "" {
			return nil, fmt.Errorf("invalid empty variable declaration")
		}
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid variable declaration '%s'", v)
		}
		k := parts[0]
		v := parts[1]
		vars[k] = v
	}
	return vars, nil
}

func newDeclarationBlock(
	blockType string,
	contents []byte,
	filename string,
	varList []string,
) (*DeclarationBlock, error) {
	ct := make(map[string]interface{})
	switch blockType {
	case JSONBlockType:
		err := json.Unmarshal(bytes.TrimSpace(contents), &ct)
		if err != nil {
			return nil, err
		}
	case JsonnetBlockType:
		vars, err := parseVarList(varList)
		if err != nil {
			return nil, err
		}
		vm := jsonnet.MakeVM()
		for k, v := range vars {
			vm.ExtVar(k, v)
		}
		var jsonStr string
		jsonStr, err = vm.EvaluateAnonymousSnippet(filename, string(bytes.TrimSpace(contents)))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(jsonStr), &ct)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("disallowed preprocessor block type '%s'", blockType)
	}
	return &DeclarationBlock{
		Type:     blockType,
		Contents: ct,
	}, nil
}

type Preprocessor struct {
	declarationBlockStartToken []byte
	declarationBlockEndToken   []byte
	contents                   printableMap
}

func (pp *Preprocessor) inferBlock(block []byte, filename string, varList []string) (*DeclarationBlock, error) {
	var typeStr string
	var i int
	for j, b := range block {
		if unicode.IsSpace(rune(b)) {
			break
		}
		i = j
		typeStr += string(b)
	}
	return newDeclarationBlock(typeStr, block[i+1:], filename, varList)
}

func (pp *Preprocessor) mergeContents(declarationBlocks []DeclarationBlock) {
	contents := make(map[string]interface{})
	for _, block := range declarationBlocks {
		for k, v := range block.Contents {
			contents[k] = v
		}
	}
	pp.contents = printableMapFromMap(contents)
}

func NewPreprocessor(declarationBlockStartToken, declarationBlockEndToken string) *Preprocessor {
	if declarationBlockStartToken == "" {
		declarationBlockStartToken = DefaultDeclarationBlockStartToken
	}
	if declarationBlockEndToken == "" {
		declarationBlockEndToken = DefaultDeclarationBlockEndToken
	}
	return &Preprocessor{
		declarationBlockStartToken: []byte(declarationBlockStartToken),
		declarationBlockEndToken:   []byte(declarationBlockEndToken),
	}
}

func (pp *Preprocessor) Render(input io.Reader) (io.Reader, error) {
	inContents, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("iqlTmpl").Parse(string(inContents))
	if err != nil {
		return nil, err
	}
	var tplWr bytes.Buffer
	if err = tmpl.Execute(&tplWr, pp.contents); err != nil {
		return nil, err
	}
	return bytes.NewReader(tplWr.Bytes()), nil
}

func (pp *Preprocessor) Prepare(infile io.Reader, infileName string, varList []string) (io.Reader, error) {
	var outContents []byte
	var declarationBlocks []DeclarationBlock
	inContents, err := io.ReadAll(infile)
	if err != nil {
		return nil, err
	}
	i := 0
	var blockTermIdx int
	blockIdx := bytes.Index(inContents, pp.declarationBlockStartToken)
	for blockIdx > -1 {
		outContents = append(outContents, inContents[i:blockIdx]...)
		blockTermIdx = bytes.Index(inContents, pp.declarationBlockEndToken)
		if blockTermIdx < 0 {
			return nil, fmt.Errorf("declaration block unclosed, cannot preprocess input")
		}
		i = blockTermIdx + len(pp.declarationBlockEndToken)
		startIdx := blockIdx + len(pp.declarationBlockStartToken)
		if startIdx > blockTermIdx {
			return nil, fmt.Errorf("declaration block delimiters improperly placed")
		}
		var db *DeclarationBlock
		db, err = pp.inferBlock(inContents[startIdx:blockTermIdx], infileName, varList)
		if err != nil {
			return nil, err
		}
		declarationBlocks = append(declarationBlocks, *db)
		blockIdx = bytes.Index(inContents[i:], pp.declarationBlockStartToken)
	}
	outContents = append(outContents, inContents[i:]...)
	pp.mergeContents(declarationBlocks)
	return bytes.NewReader(outContents), err
}

func (pp *Preprocessor) PrepareExternal(
	infileType string,
	infile io.Reader,
	infileName string,
	varList []string,
) error {
	inContents, err := io.ReadAll(infile)
	if err != nil {
		return err
	}
	db, err := newDeclarationBlock(infileType, inContents, infileName, varList)
	if err != nil {
		return err
	}
	pp.contents = printableMapFromMap(db.Contents)
	return err
}
