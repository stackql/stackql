package taxonomy

import (
	"github.com/stackql/stackql/internal/stackql/util"
	"vitess.io/vitess/go/vt/sqlparser"
)

type AnnotatedTabulationMap map[sqlparser.SQLNode]util.AnnotatedTabulation
