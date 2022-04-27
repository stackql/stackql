package preprocessor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"text/template"
	"unicode"

	jsonnet "github.com/google/go-jsonnet"
)

const (
	TripleLessThanToken               string = "<<<"
	TripleGreaterThanToken            string = ">>>"
	DefaultDeclarationBlockStartToken string = TripleLessThanToken
	DefaultDeclarationBlockEndToken   string = TripleGreaterThanToken
	JsonBlockType                     string = "json"
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

func NewDeclarationBlock(blockType string, contents []byte, filename string) (*DeclarationBlock, error) {
	ct := make(map[string]interface{})
	var err error
	switch blockType {
	case JsonBlockType:

		err = json.Unmarshal(bytes.TrimSpace(contents), &ct)
		if err != nil {
			return nil, err
		}
	case JsonnetBlockType:
		vm := jsonnet.MakeVM()
		jsonStr, err := vm.EvaluateAnonymousSnippet(filename, string(bytes.TrimSpace(contents)))
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

func (pp *Preprocessor) inferBlock(block []byte, filename string) (*DeclarationBlock, error) {
	var typeStr string
	var i int
	for j, b := range block {
		if unicode.IsSpace(rune(b)) {
			break
		}
		i = j
		typeStr += string(b)
	}
	return NewDeclarationBlock(typeStr, block[i+1:len(block)], filename)
}

func (pp *Preprocessor) mergeContents(declarationBlocks []DeclarationBlock) error {
	contents := make(map[string]interface{})
	var err error
	for _, block := range declarationBlocks {
		for k, v := range block.Contents {
			contents[k] = v
		}
	}
	pp.contents = printableMapFromMap(contents)
	return err
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
	inContents, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("iqlTmpl").Parse(string(inContents))
	if err != nil {
		return nil, err
	}
	var tplWr bytes.Buffer
	if err := tmpl.Execute(&tplWr, pp.contents); err != nil {
		return nil, err
	}
	return bytes.NewReader(tplWr.Bytes()), nil
}

func (pp *Preprocessor) Prepare(infile io.Reader, infileName string) (io.Reader, error) {
	var outContents []byte
	var declarationBlocks []DeclarationBlock
	inContents, err := ioutil.ReadAll(infile)
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
		db, err := pp.inferBlock(inContents[startIdx:blockTermIdx], infileName)
		if err != nil {
			return nil, err
		}
		declarationBlocks = append(declarationBlocks, *db)
		blockIdx = bytes.Index(inContents[i:len(inContents)], pp.declarationBlockStartToken)
	}
	outContents = append(outContents, inContents[i:len(inContents)]...)
	err = pp.mergeContents(declarationBlocks)
	return bytes.NewReader(outContents), err
}

func (pp *Preprocessor) PrepareExternal(infileType string, infile io.Reader, infileName string) error {
	inContents, err := ioutil.ReadAll(infile)
	if err != nil {
		return err
	}
	db, err := NewDeclarationBlock(infileType, inContents, infileName)
	if err != nil {
		return err
	}
	pp.contents = printableMapFromMap(db.Contents)
	return err
}
