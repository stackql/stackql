package logging

import (
	"io"

	"github.com/sirupsen/logrus"
)

var (
	Logger *logrus.Logger //nolint:gochecknoglobals // This is convenient as a global variable
)

func SetLogger(logLevelStr string) {
	Logger = logrus.StandardLogger()
	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		Logger.Fatal(err)
	}
	Logger.SetLevel(logLevel)
}

func GetLogger() *logrus.Logger {
	if Logger != nil {
		return Logger
	}
	tmpLogger := logrus.New()
	tmpLogger.SetLevel(logrus.WarnLevel)
	tmpLogger.SetOutput(io.Discard)
	return tmpLogger
}
