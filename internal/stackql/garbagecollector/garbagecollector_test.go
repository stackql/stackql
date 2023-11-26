package garbagecollector //nolint:testpackage // to test unexported methods

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type GarbageCollectorExecutorMock struct {
	mock.Mock
}

func (m *GarbageCollectorExecutorMock) Purge() error {
	args := m.Called()
	return args.Error(0)
}

func (m *GarbageCollectorExecutorMock) PurgeCache() error {
	args := m.Called()
	return args.Error(0)
}

func (m *GarbageCollectorExecutorMock) PurgeControlTables() error {
	args := m.Called()
	return args.Error(0)
}

func (m *GarbageCollectorExecutorMock) PurgeEphemeral() error {
	args := m.Called()
	return args.Error(0)
}

func (m *GarbageCollectorExecutorMock) Update(tableName string, parentTcc, tcc internaldto.TxnControlCounters) error {
	args := m.Called(tableName, parentTcc, tcc)
	return args.Error(0)
}

func (m *GarbageCollectorExecutorMock) Collect() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewGarbageCollector(t *testing.T) {
	t.Run("NewGarbageCollector", func(t *testing.T) {
		gcExecutorMock := new(GarbageCollectorExecutorMock)

		gc := NewGarbageCollector(gcExecutorMock, dto.GCCfg{}, nil)

		assert.NotNil(t, gc)
	})
}

func TestNewStandardGarbageCollector(t *testing.T) {
	t.Run("NewStandardGarbageCollector", func(t *testing.T) {
		gcExecutorMock := new(GarbageCollectorExecutorMock)

		gc := newStandardGarbageCollector(gcExecutorMock, dto.GCCfg{}, nil)

		assert.NotNil(t, gc)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("Update", func(t *testing.T) {
		gcExecutorMock := new(GarbageCollectorExecutorMock)
		gcExecutorMock.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecutorMock,
		}

		err := gc.Update("tableName", nil, nil)

		assert.NoError(t, err)
		gcExecutorMock.AssertExpectations(t)
	})
}

func TestClose(t *testing.T) {
	t.Run("Close with isEager true", func(t *testing.T) {
		gcExecutorMock := new(GarbageCollectorExecutorMock)
		gcExecutorMock.On("Collect").Return(nil)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecutorMock,
			isEager:    true,
		}

		err := gc.Close()

		assert.NoError(t, err)
		gcExecutorMock.AssertExpectations(t)
	})

	t.Run("Close with isEager false", func(t *testing.T) {
		gcExecutorMock := new(GarbageCollectorExecutorMock)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecutorMock,
			isEager:    false,
		}

		err := gc.Close()

		assert.NoError(t, err)
		gcExecutorMock.AssertExpectations(t)
	})
}

func TestCollect(t *testing.T) {
	t.Run("Collect", func(t *testing.T) {
		gcExecutorMock := new(GarbageCollectorExecutorMock)
		gcExecutorMock.On("Collect").Return(nil)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecutorMock,
		}

		err := gc.Collect()

		assert.NoError(t, err)
		gcExecutorMock.AssertExpectations(t)
	})
}

func TestPurge(t *testing.T) {
	t.Run("Purge", func(t *testing.T) {
		gcExecuterMock := new(GarbageCollectorExecutorMock)
		gcExecuterMock.On("Purge").Return(nil)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecuterMock,
		}

		err := gc.Purge()

		assert.NoError(t, err)
		gcExecuterMock.AssertExpectations(t)
	})
}

func TestPurgeEphemeral(t *testing.T) {
	t.Run("PurgeEphemeral", func(t *testing.T) {
		gcExecuterMock := new(GarbageCollectorExecutorMock)
		gcExecuterMock.On("PurgeEphemeral").Return(nil)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecuterMock,
		}

		err := gc.PurgeEphemeral()

		assert.NoError(t, err)
		gcExecuterMock.AssertExpectations(t)
	})
}

func TestPurgeCache(t *testing.T) {
	t.Run("PurgeCache", func(t *testing.T) {
		gcExecuterMock := new(GarbageCollectorExecutorMock)
		gcExecuterMock.On("PurgeCache").Return(nil)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecuterMock,
		}

		err := gc.PurgeCache()

		assert.NoError(t, err)
		gcExecuterMock.AssertExpectations(t)
	})
}

func TestPurgeControlTables(t *testing.T) {
	t.Run("PurgeControlTables", func(t *testing.T) {
		gcExecuterMock := new(GarbageCollectorExecutorMock)
		gcExecuterMock.On("PurgeControlTables").Return(nil)

		gc := &standardGarbageCollector{
			gcExecutor: gcExecuterMock,
		}

		err := gc.PurgeControlTables()

		assert.NoError(t, err)
		gcExecuterMock.AssertExpectations(t)
	})
}
