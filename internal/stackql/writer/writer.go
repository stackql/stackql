package writer

import (
	"io"
	"os"

	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/presentation"
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

func GetDecoratedOutputWriter(filename string, cd presentation.Driver) (io.Writer, error) {
	switch filename {
	case StdOutStr:
		return &StdStreamWriter{writer: os.Stdout, prezzoDriver: cd}, nil
	case StdErrStr:
		return &StdStreamWriter{writer: os.Stderr, prezzoDriver: cd}, nil
	default:
		return os.Create(filename)
	}
}

type StdStreamWriter struct {
	writer       io.Writer
	prezzoDriver presentation.Driver
}

func (ssw *StdStreamWriter) render(p []byte) []byte {
	return []byte(ssw.prezzoDriver.Print(string(p)))
}

func (ssw *StdStreamWriter) enclose(p []byte) []byte {
	return ssw.render(p)
}

func (ssw *StdStreamWriter) Write(p []byte) (int, error) {
	logging.GetLogger().Infoln("stylised write called")
	return ssw.writer.Write(ssw.enclose(p))
}
