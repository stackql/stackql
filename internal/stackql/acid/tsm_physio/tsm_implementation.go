package tsm_physio //nolint:stylecheck,revive // prefer this nomenclature

import (
	"github.com/stackql/stackql/internal/stackql/acid/tsm"
	"github.com/stackql/stackql/internal/stackql/handler"
)

//nolint:unused // TODO: finalise pattern
type tsmImplementation struct {
	accessMethods AccessMethods
	bufferPool    BufferPool
	lockManager   LockManager
	logManager    LogManager
}

func GetTSM(handlerCtx handler.HandlerContext) (tsm.TSM, error) {
	walManager, walErr := getWalManager(handlerCtx)
	if walErr != nil {
		return nil, walErr
	}
	return &tsmImplementation{
		logManager: walManager,
	}, nil
}
