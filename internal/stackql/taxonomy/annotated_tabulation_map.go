package taxonomy

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/util"
)

type AnnotatedTabulationMap map[sqlparser.SQLNode]util.AnnotatedTabulation
