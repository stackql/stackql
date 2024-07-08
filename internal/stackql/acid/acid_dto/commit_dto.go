package acid_dto //nolint:revive,stylecheck // meaning is clear

import (
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

var (
	_ CommitCoDomain = &standardCommitCoDomain{}
)

type CommitCoDomain interface {
	GetExecutorOutput() []internaldto.ExecutorOutput
	GetError() (error, bool)
	GetMessages() []string
	GetUndoLog() (binlog.LogEntry, bool)
}

type standardCommitCoDomain struct {
	executorOutputs      []internaldto.ExecutorOutput
	votingPhaseError     error
	completionPhaseError error
}

func NewCommitCoDomain(
	executorOutputs []internaldto.ExecutorOutput,
	votingPhaseError error,
	completionPhaseError error,
) CommitCoDomain {
	return &standardCommitCoDomain{
		executorOutputs:      executorOutputs,
		votingPhaseError:     votingPhaseError,
		completionPhaseError: completionPhaseError,
	}
}

func (c *standardCommitCoDomain) GetMessages() []string {
	var messages []string
	if c.votingPhaseError != nil {
		messages = append(messages, c.votingPhaseError.Error())
	}
	if c.completionPhaseError != nil {
		messages = append(messages, c.completionPhaseError.Error())
	}
	return messages
}

func (c *standardCommitCoDomain) GetExecutorOutput() []internaldto.ExecutorOutput {
	return c.executorOutputs
}

func (c *standardCommitCoDomain) GetError() (error, bool) { //nolint:revive // permissable deviation from norm
	if c.votingPhaseError != nil {
		return c.votingPhaseError, true
	}
	return c.completionPhaseError, c.completionPhaseError != nil
}

func (c *standardCommitCoDomain) GetUndoLog() (binlog.LogEntry, bool) {
	var undoLogs []binlog.LogEntry
	for _, executorOutput := range c.executorOutputs {
		undoLog, undoLogExists := executorOutput.GetUndoLog()
		if undoLogExists && undoLog != nil {
			undoLogs = append(undoLogs, undoLog)
		}
	}
	if len(undoLogs) == 0 {
		return nil, false
	}
	initialUndoLog := undoLogs[len(undoLogs)-1]
	rv := initialUndoLog.Clone()
	for i := len(undoLogs) - 2; i >= 0; i-- { //nolint:mnd // magic number second from last
		currentLog := undoLogs[i]
		if currentLog != nil {
			for _, s := range currentLog.GetHumanReadable() {
				rv.AppendHumanReadable(s)
			}
			rv.AppendRaw(currentLog.GetRaw())
		}
	}
	return rv, true
}
