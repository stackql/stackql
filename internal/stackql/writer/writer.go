package writer

import (
	"io"
	"os"
	"runtime"

	"github.com/stackql/stackql/internal/stackql/color"
	"github.com/stackql/stackql/internal/stackql/logging"
)

const (
	StdOutStr string = "stdout"
	StdErrStr string = "stderr"
)

func GetOutputWriter(filename string) (io.Writer, error) {
	switch filename {
	case StdOutStr:
		return os.Stdout, nil
	case StdErrStr:
		return os.Stderr, nil
	default:
		return os.Create(filename)
	}
}

func GetDecoratedOutputWriter(filename string, cd *color.ColorDriver, overrideColor ...color.Attribute) (io.Writer, error) {
	if cd.Peek() == nil || runtime.GOOS == "windows" {
		return GetOutputWriter(filename)
	}
	switch filename {
	case StdOutStr:
		return &StdStreamWriter{writer: os.Stdout, colorDriver: cd, overrideColor: overrideColor}, nil
	case StdErrStr:
		return &StdStreamWriter{writer: os.Stderr, colorDriver: cd, overrideColor: overrideColor}, nil
	default:
		return os.Create(filename)
	}
}

type BaseWriter struct {
	writer io.Writer
}

func (ssw *BaseWriter) Write(p []byte) (n int, err error) {
	return ssw.Write(p)
}

type StdStreamWriter struct {
	writer        io.Writer
	colorDriver   *color.ColorDriver
	overrideColor []color.Attribute
}

func (ssw *StdStreamWriter) render(p []byte) []byte {
	return []byte(ssw.colorDriver.Peek().Sprintf(string(p)))
}

func (ssw *StdStreamWriter) enclose(p []byte) []byte {
	if ssw.overrideColor != nil {
		ssw.colorDriver.New(ssw.overrideColor...)
		retVal := ssw.render(p)
		ssw.colorDriver.Pop()
		return retVal
	}
	return ssw.render(p)
}

func (ssw *StdStreamWriter) Write(p []byte) (n int, err error) {
	logging.GetLogger().Infoln("stylised write called")
	return ssw.writer.Write(ssw.enclose(p))
}
