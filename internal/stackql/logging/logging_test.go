package logging_test

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stackql/stackql/internal/stackql/logging"
)

func TestSetLogger(t *testing.T) {
	// Test with a valid log level
	logging.SetLogger("info")
	if logging.Logger == nil {
		t.Error("Expected Logger to be set, but it's nil.")
	}
	if logging.Logger.GetLevel() != logrus.InfoLevel {
		t.Errorf("Expected log level to be 'info', but it's %v.", logging.Logger.GetLevel())
	}
}

func TestGetLogger(t *testing.T) {
	// Test when Logger is not set (default)
	logging.Logger = nil
	tmpLogger := logging.GetLogger()
	if tmpLogger == nil {
		t.Error("Expected Logger to be set, but it's nil.")
	}
	if tmpLogger.GetLevel() != logrus.InfoLevel {
		t.Errorf("Expected log level is info, but it's %v.", tmpLogger.GetLevel())
	}

	// Test when Logger is set
	logging.Logger = logrus.New()
	logging.Logger.SetLevel(logrus.DebugLevel)
	tmpLogger = logging.GetLogger()
	if tmpLogger == nil {
		t.Error("Expected Logger to be set, but it's nil.")
	}
	if tmpLogger.GetLevel() != logrus.DebugLevel {
		t.Errorf("Expected log level is debug, but it's %v.", tmpLogger.GetLevel())
	}
}
