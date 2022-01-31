package plan

import (
	"sync"
	"time"

	"github.com/stackql/stackql/internal/stackql/primitive"

	"vitess.io/vitess/go/vt/sqlparser"
	// log "github.com/sirupsen/logrus"
)

type Plan struct {
	Type                   sqlparser.StatementType // The type of query we have
	Original               string                  // Original is the original query.
	Instructions           primitive.IPrimitive    // Instructions contains the instructions needed to fulfil the query.
	sqlparser.BindVarNeeds                         // Stores BindVars needed to be provided as part of expression rewriting

	mu           sync.Mutex    // Mutex to protect the fields below
	ExecCount    uint64        // Count of times this plan was executed
	ExecTime     time.Duration // Total execution time
	ShardQueries uint64        // Total number of shard queries
	Rows         uint64        // Total number of rows
	Errors       uint64        // Total number of errors
}

// Size is defined so that Plan can be given to a cache.LRUCache,
// which requires its objects to define a Size function.
func (p *Plan) Size() int {
	return 1
}
