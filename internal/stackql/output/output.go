package output

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/iqlutil"

	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/sqltypes"
)

const (
	errorKey               string = "error"
	stderrPressentationStr string = "stderr"
)

type IOutputWriter interface {
	Write(*sqltypes.Result) error
	WriteError(error, string) error
}

type IColumnOrderer interface {
	GetColumnOrder() []string
}

type BasicColumnOrderer struct {
	ordering []string
}

func (bco *BasicColumnOrderer) GetColumnOrder() []string {
	return bco.ordering
}

func writeStderrError(writer io.Writer, err error) error {

	_, e := fmt.Fprintln(writer, err.Error())
	return e
}

func GetOutputWriter(writer io.Writer, errWriter io.Writer, outputCtx dto.OutputContext) (IOutputWriter, error) {
	if errWriter == nil {
		errWriter = os.Stdout
	}
	switch outputCtx.RuntimeContext.OutputFormat {
	case constants.JsonStr:
		jsonWriter := JsonWriter{
			writer:    writer,
			errWriter: errWriter,
			outputCtx: outputCtx,
		}
		return &jsonWriter, nil
	case constants.TableStr:
		tablewriter := TableWriter{
			AbstractTabularWriter{
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &tablewriter, nil
	case constants.CSVStr:
		csvwriter := CSVWriter{
			AbstractTabularWriter{
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &csvwriter, nil
	case constants.TextStr:
		rawWriter := RawWriter{
			AbstractTabularWriter{
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &rawWriter, nil
	case constants.PrettyTextStr:
		prettyWriter := PrettyWriter{
			AbstractTabularWriter{
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &prettyWriter, nil
	}
	return nil, fmt.Errorf("unable to create output writer for output format = '%s'", outputCtx.RuntimeContext.OutputFormat)
}

type JsonWriter struct {
	writer    io.Writer
	errWriter io.Writer
	outputCtx dto.OutputContext
}

type AbstractTabularWriter struct {
	outputCtx dto.OutputContext
}

type TableWriter struct {
	AbstractTabularWriter
	writer    io.Writer
	errWriter io.Writer
}

type CSVWriter struct {
	AbstractTabularWriter
	writer    io.Writer
	errWriter io.Writer
}

type RawWriter struct {
	AbstractTabularWriter
	writer    io.Writer
	errWriter io.Writer
}

type PrettyWriter struct {
	AbstractTabularWriter
	writer    io.Writer
	errWriter io.Writer
}

func (jw *JsonWriter) writeRowsFromResult(res *sqltypes.Result) error {
	rows := make([]map[string]interface{}, len(res.Rows))
	for i, row := range res.Rows {
		rowMap := make(map[string]interface{})
		for j, s := range res.Fields {
			rowMap[s.Name] = row[j].ToString()
		}
		rows[i] = rowMap
	}
	return jw.writeRows(rows)
}

func (jw *JsonWriter) writeRows(rows []map[string]interface{}) error {
	var retVal error
	jsonBytes, jsonErr := json.Marshal(rows)
	bytesWritten, writeErr := jw.writer.Write(jsonBytes)
	if jsonErr != nil {
		retVal = jsonErr
	} else if writeErr != nil {
		retVal = writeErr
	} else if bytesWritten != len(jsonBytes) {
		retVal = errors.New("incorrect number of bytes written")
	}
	return retVal
}

func (jw *JsonWriter) Write(res *sqltypes.Result) error {
	return jw.writeRowsFromResult(res)
}

func (jw *JsonWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(jw.errWriter, err)
	}
	rows := make([]map[string]interface{}, 0, 1)
	rows = append(rows,
		map[string]interface{}{
			errorKey: err.Error(),
		},
	)
	return jw.writeRows(rows)
}

func (tw *AbstractTabularWriter) getHeader(res *sqltypes.Result) []string {
	headers := make([]string, len(res.Fields))
	for i, s := range res.Fields {
		headers[i] = s.Name
	}
	return headers
}

func (tw *AbstractTabularWriter) processRow(rowDict map[string]interface{}, header []string) []string {
	row := make([]string, 0, len(header))
	for idx := range header {
		switch rowDict[header[idx]].(type) {
		case string:
			row = append(row, rowDict[header[idx]].(string))
		case interface{}:
			jsonBytes, _ := json.Marshal(rowDict[header[idx]])
			row = append(row, string(jsonBytes))
		}
	}
	return row
}

func (tw *TableWriter) configureTable(table *tablewriter.Table) {
	table.SetCenterSeparator("|")
	table.SetRowLine(true)
	table.SetAutoFormatHeaders(false)
}

func (tw *TableWriter) Write(res *sqltypes.Result) error {
	var err error = nil
	header := tw.getHeader(res)
	table := tablewriter.NewWriter(tw.writer)
	table.SetHeader(header)

	for _, v := range res.Rows {
		log.Debugln(fmt.Sprintf(`tableWriter row: %v`, v))
		rowSlice := make([]string, len(v))
		for i, c := range v {
			rowSlice[i] = c.ToString()
		}
		table.Append(rowSlice)
	}
	tw.configureTable(table)
	table.Render()

	return err
}

func (csvw *CSVWriter) Write(res *sqltypes.Result) error {
	header := csvw.getHeader(res)
	w := csv.NewWriter(csvw.writer)
	w.Comma = rune(csvw.outputCtx.RuntimeContext.Delimiter[0])
	if !csvw.outputCtx.RuntimeContext.CSVHeadersDisable {
		w.Write(header)
	}

	for _, v := range res.Rows {
		log.Debugln(fmt.Sprintf(`tableWriter row: %v`, v))
		rowSlice := make([]string, len(v))
		for i, c := range v {
			rowSlice[i] = c.ToString()
		}
		w.Write(rowSlice)
	}
	w.Flush()
	return w.Error()
}

func (rw *RawWriter) Write(res *sqltypes.Result) error {
	header := rw.getHeader(res)
	w := rw.writer
	if !rw.outputCtx.RuntimeContext.CSVHeadersDisable {
		w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(header, ","), fmt.Sprintln(""))))
	}

	for _, v := range res.Rows {
		log.Debugln(fmt.Sprintf(`tableWriter row: %v`, v))
		rowSlice := make([]string, len(v))
		for i, c := range v {
			rowSlice[i] = c.ToString()
		}
		w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(rowSlice, ","), fmt.Sprintln(""))))
	}
	return nil
}

func (rw *PrettyWriter) Write(res *sqltypes.Result) error {
	header := rw.getHeader(res)
	w := rw.writer
	if !rw.outputCtx.RuntimeContext.CSVHeadersDisable {
		w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(header, ","), fmt.Sprintln(""))))
	}

	for _, v := range res.Rows {
		log.Debugln(fmt.Sprintf(`tableWriter row: %v`, v))
		rowSlice := make([]string, len(v))
		for i, c := range v {
			s := c.ToString()
			b, err := iqlutil.PrettyPrintSomeJson([]byte(s))
			if err != nil {
				rowSlice[i] = s
				continue
			}
			rowSlice[i] = string(b)
		}
		w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(rowSlice, ","), fmt.Sprintln(""))))
	}
	return nil
}

func (csvw *CSVWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(csvw.errWriter, err)
	}
	w := csv.NewWriter(csvw.writer)
	w.Write(
		[]string{
			errorKey,
		},
	)
	w.Write(
		[]string{
			err.Error(),
		},
	)
	w.Flush()
	return w.Error()
}

func (rw *RawWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(rw.errWriter, err)
	}
	w := rw.writer
	w.Write([]byte(fmt.Sprintf("%s%s", errorKey, fmt.Sprintln(""))))
	w.Write([]byte(fmt.Sprintf("%s%s", err.Error(), fmt.Sprintln(""))))
	return nil
}

func (rw *PrettyWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(rw.errWriter, err)
	}
	w := rw.writer
	w.Write([]byte(fmt.Sprintf("%s%s", errorKey, fmt.Sprintln(""))))
	w.Write([]byte(fmt.Sprintf("%s%s", err.Error(), fmt.Sprintln(""))))
	return nil
}

func (tw *TableWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(tw.errWriter, err)
	}
	table := tablewriter.NewWriter(tw.writer)
	table.SetHeader([]string{errorKey})
	table.Append(
		[]string{
			err.Error(),
		},
	)
	tw.configureTable(table)
	table.Render()
	return nil
}
