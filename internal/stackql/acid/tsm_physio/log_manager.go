package tsm_physio //nolint:revive,stylecheck // prefer this nomenclature
import (
	"sync"

	"github.com/stackql/stackql/internal/stackql/handler"
)

var (
	_ LogManager = (*walManager)(nil)
)

//nolint:gochecknoglobals // singleton pattern
var (
	walOnce      sync.Once
	walSingleton LogManager
)

type LogManager interface {
	//
}

type walManager struct{}

func getWalManager(_ handler.HandlerContext) (LogManager, error) {
	var err error
	walOnce.Do(func() {
		if err != nil {
			return
		}
		walSingleton = &walManager{}
	})
	return walSingleton, err
}
