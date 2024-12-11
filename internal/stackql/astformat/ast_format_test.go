package astformat

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"

	"github.com/stretchr/testify/assert"
)

// sqlNodes are used for both
var selectNode, dropNode sqlparser.Statement
func TestMain(m *testing.M){
  p, _ := parser.NewParser()
  selectNode, _ = p.ParseQuery("select 1+2*3 from a WHERE a.v1 = 3")
  dropNode, _ = p.ParseQuery("DROP TABLE tab");
  m.Run()
}


func TestPostgresSelectExprsFormatter (t *testing.T){
  // only covers SELECT and DROP Statements
  // Doesn't cover most actual SQL Nodes 
  // no constructor for ColName, ColIdent, ...
  tests := []struct {
    name string
    node sqlparser.SQLNode
    expected string
  }{
    {
      "PostgresSelectExprsFormatter: select",
      selectNode.(sqlparser.SQLNode),
      "select 1 + 2 * 3 from \"a\" where \"a\".v1 = 3",
    },
    {
      "PostgresSelectExprsFormatter: drop",
      dropNode.(sqlparser.SQLNode),
      "drop table \"tab\"",
    },
  }
  for _, tt := range tests{
    t.Run(tt.name, func(t *testing.T){
      buf := sqlparser.NewTrackedBuffer(nil)

      PostgresSelectExprsFormatter(buf, tt.node)
      
      //assert.Equal(t, fmt.Sprintf("TYPE: %T\n",tt.node), "")
      switch tt.node.(type){
      case *sqlparser.GroupConcatExpr:
          assert.Equal(t,true,false)
          break
      }
      assert.Equal(t, tt.expected, buf.String())
    })
  }
}



func TestString(t *testing.T){

  tests := []struct {
    name  string
    node sqlparser.SQLNode
    expected string
  }{
    {
      "string: select query",
      selectNode.(sqlparser.SQLNode),
      "select 1 + 2 * 3 from \"a\" where \"a\".\"v1\" = 3",
    },
    {
      "string: drop query",
      dropNode.(sqlparser.SQLNode),
      "drop table \"tab\"",
    },
  }
  var formatter sqlparser.NodeFormatter = PostgresSelectExprsFormatter
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T){
      got := String(tt.node, formatter)
      assert.Equal(t, tt.expected, got) 
    })
  }
}


