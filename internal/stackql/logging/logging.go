package logging

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger //nolint:gochecknoglobals // This is convenient as a global variable
)

func SetLogger(logLevelStr string) {
	logger = logrus.StandardLogger()
	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		logger.Fatal(err)
	}
	logger.SetLevel(logLevel)
}

func GetLogger() *logrus.Logger {
	if logger != nil {
		return logger
	}
	tmpLogger := logrus.New()
	tmpLogger.SetOutput(ioutil.Discard)
	return tmpLogger
}
