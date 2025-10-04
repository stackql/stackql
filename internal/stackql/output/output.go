package output

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/iqlutil"
	"github.com/stackql/stackql/internal/stackql/psqlwire"

	"github.com/olekukonko/tablewriter"
	"github.com/stackql/psql-wire/pkg/sqldata"
)

const (
	errorKey               string = "error"
	stderrPressentationStr string = "stderr"
)

type IOutputWriter interface {
	Write(sqldata.ISQLResultStream) error
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

func GetOutputWriter(
	writer io.Writer,
	errWriter io.Writer,
	outputCtx internaldto.OutputContext,
) (IOutputWriter, error) {
	if errWriter == nil {
		errWriter = os.Stdout
	}
	ci := pgtype.NewConnInfo()
	switch outputCtx.RuntimeContext.OutputFormat {
	case constants.JSONStr:
		jsonWriter := NewJSONWriter(writer, errWriter)
		return jsonWriter, nil
	case constants.TableStr:
		tablewriter := TableWriter{
			AbstractTabularWriter{
				ci:        ci,
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &tablewriter, nil
	case constants.CSVStr:
		csvwriter := CSVWriter{
			AbstractTabularWriter{
				ci:        ci,
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &csvwriter, nil
	case constants.TextStr:
		rawWriter := RawWriter{
			AbstractTabularWriter{
				ci:        ci,
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &rawWriter, nil
	case constants.PrettyTextStr:
		prettyWriter := PrettyWriter{
			AbstractTabularWriter{
				ci:        ci,
				outputCtx: outputCtx,
			},
			writer,
			errWriter,
		}
		return &prettyWriter, nil
	}
	return nil, fmt.Errorf(
		"unable to create output writer for output format = '%s'",
		outputCtx.RuntimeContext.OutputFormat)
}

type JSONWriter struct {
	writer    io.Writer
	errWriter io.Writer
}

func NewJSONWriter(writer io.Writer, errWriter io.Writer) IOutputWriter {
	return &JSONWriter{
		writer:    writer,
		errWriter: errWriter,
	}
}

type AbstractTabularWriter struct {
	ci        *pgtype.ConnInfo
	outputCtx internaldto.OutputContext
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

func (jw *JSONWriter) writeRowsFromResult(res sqldata.ISQLResultStream) error {
	for {
		r, err := res.Read()
		logging.GetLogger().Debugln(fmt.Sprintf("result from stream: %v", r))
		if err != nil {
			if errors.Is(err, io.EOF) {
				rowsArr := r.ToArr()
				jw.writeRows(rowsArr) //nolint:errcheck // output stream is not critical
				return nil
			}
			return err
		}
		rowsArr := r.ToArr()
		jw.writeRows(rowsArr) //nolint:errcheck // output stream is not critical
	}
}

func (jw *JSONWriter) writeRows(rows []map[string]interface{}) error {
	var retVal error
	jsonBytes, jsonErr := json.Marshal(rows)
	bytesWritten, writeErr := jw.writer.Write(jsonBytes)
	//nolint:gocritic // acceptable
	if jsonErr != nil {
		retVal = jsonErr
	} else if writeErr != nil {
		retVal = writeErr
	} else if bytesWritten != len(jsonBytes) {
		retVal = errors.New("incorrect number of bytes written")
	}
	return retVal
}

func (jw *JSONWriter) Write(res sqldata.ISQLResultStream) error {
	return jw.writeRowsFromResult(res)
}

func (jw *JSONWriter) WriteError(err error, errorPresentation string) error {
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

func (tw *AbstractTabularWriter) getHeader(res sqldata.ISQLResult) []string {
	var headers []string
	for _, col := range res.GetColumns() {
		headers = append(headers, col.GetName())
	}
	return headers
}

func (tw *TableWriter) configureTable(table *tablewriter.Table) {
	table.SetCenterSeparator("|")
	table.SetRowLine(true)
	table.SetAutoFormatHeaders(false)
}

//nolint:gocognit // acceptable
func (tw *TableWriter) Write(res sqldata.ISQLResultStream) error {
	var isHeaderRead bool
	var table *tablewriter.Table
	for {
		var rowsArr [][]string
		r, err := res.Read()
		logging.GetLogger().Debugln(fmt.Sprintf("result from stream: %v", r))
		if err != nil { //nolint:nestif // acceptable
			if errors.Is(err, io.EOF) {
				if !isHeaderRead {
					header := tw.getHeader(r)
					table = tablewriter.NewWriter(tw.writer)
					table.SetHeader(header)
					isHeaderRead = true //nolint:ineffassign,wastedassign // looks false positive
				}
				rowsArr, err = tabulateResults(r, tw.ci)
				if err != nil {
					return err
				}
				for _, rs := range rowsArr {
					table.Append(rs)
				}
				tw.configureTable(table)
				table.Render()
				return nil
			}
			return err
		}
		if !isHeaderRead {
			header := tw.getHeader(r)
			table = tablewriter.NewWriter(tw.writer)
			table.SetHeader(header)
			isHeaderRead = true
		}
		rowsArr, err = tabulateResults(r, tw.ci)
		for _, rs := range rowsArr {
			table.Append(rs)
		}
		if err != nil {
			return err
		}
	}
}

func decodeRow(colz []sqldata.ISQLColumn, row sqldata.ISQLRow, ci *pgtype.ConnInfo) ([][]byte, error) {
	var retVal [][]byte
	rawRow := row.GetRowDataNaive()
	if len(rawRow) != len(colz) {
		if len(rawRow) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("row length != column count (%d != %d)", len(rawRow), len(colz))
	}
	for i, col := range colz {
		b, err := psqlwire.ExtractRowElement(col, rawRow[i], ci)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, b)
	}
	return retVal, nil
}

func tabulateResults(r sqldata.ISQLResult, ci *pgtype.ConnInfo) ([][]string, error) {
	var retVal [][]string
	colz := r.GetColumns()
	for _, v := range r.GetRows() {
		rd, err := decodeRow(colz, v, ci)
		if err != nil {
			return nil, err
		}
		if rd == nil {
			continue
		}
		var rs []string
		for _, b := range rd {
			rs = append(rs, string(b))
		}
		retVal = append(retVal, rs)
	}
	return retVal, nil
}

//nolint:gocognit // acceptable
func (csvw *CSVWriter) Write(res sqldata.ISQLResultStream) error {
	var isHeaderRead bool
	var w *csv.Writer
	for {
		var rowsArr [][]string
		r, err := res.Read()
		logging.GetLogger().Debugln(fmt.Sprintf("result from stream: %v", r))
		if err != nil { //nolint:nestif // acceptable
			if errors.Is(err, io.EOF) {
				if !isHeaderRead {
					header := csvw.getHeader(r)
					w = csv.NewWriter(csvw.writer)
					w.Comma = rune(csvw.outputCtx.RuntimeContext.Delimiter[0])
					if !csvw.outputCtx.RuntimeContext.CSVHeadersDisable {
						w.Write(header) //nolint:errcheck // output stream is not critical
					}
					isHeaderRead = true //nolint:ineffassign,wastedassign // looks false +ve
				}
				rowsArr, err = tabulateResults(r, csvw.ci)
				if err != nil {
					return err
				}
				for _, rs := range rowsArr {
					w.Write(rs) //nolint:errcheck // output stream is not critical
				}
				w.Flush()
				return w.Error()
			}
			return err
		}
		if !isHeaderRead {
			header := csvw.getHeader(r)
			w = csv.NewWriter(csvw.writer)
			w.Comma = rune(csvw.outputCtx.RuntimeContext.Delimiter[0])
			if !csvw.outputCtx.RuntimeContext.CSVHeadersDisable {
				w.Write(header) //nolint:errcheck // output stream is not critical
			}
			isHeaderRead = true
		}
		rowsArr, err = tabulateResults(r, csvw.ci)
		if err != nil {
			return err
		}
		for _, rs := range rowsArr {
			w.Write(rs) //nolint:errcheck // output stream is not critical
		}
	}
}

//nolint:gocognit // acceptable
func (rw *RawWriter) Write(res sqldata.ISQLResultStream) error {
	var isHeaderRead bool
	var w io.Writer
	for {
		var rowsArr [][]string
		r, err := res.Read()
		logging.GetLogger().Debugln(fmt.Sprintf("result from stream: %v", r))
		if err != nil { //nolint:nestif // acceptable
			if errors.Is(err, io.EOF) {
				if !isHeaderRead {
					header := rw.getHeader(r)
					w = rw.writer
					if !rw.outputCtx.RuntimeContext.CSVHeadersDisable {
						//nolint:errcheck // output stream is not critical
						w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(header, ","), fmt.Sprintln(""))))
					}
					isHeaderRead = true //nolint:ineffassign,wastedassign // looks false +ve
				}
				rowsArr, err = tabulateResults(r, rw.ci)
				if err != nil {
					return err
				}
				for _, rs := range rowsArr {
					//nolint:errcheck // output stream is not critical
					w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(rs, ","), fmt.Sprintln(""))))
				}
				return nil
			}
			return err
		}
		if !isHeaderRead {
			header := rw.getHeader(r)
			w = rw.writer
			if !rw.outputCtx.RuntimeContext.CSVHeadersDisable {
				//nolint:errcheck // output stream is not critical
				w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(header, ","), fmt.Sprintln(""))))
			}
			isHeaderRead = true
		}
		rowsArr, err = tabulateResults(r, rw.ci)
		if err != nil {
			return err
		}
		for _, rs := range rowsArr {
			//nolint:errcheck // output stream is not critical
			w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(rs, ","), fmt.Sprintln(""))))
		}
	}
}

//nolint:gocognit // acceptable
func (rw *PrettyWriter) Write(res sqldata.ISQLResultStream) error {
	var isHeaderRead bool
	var w io.Writer
	for {
		var rowsArr [][]string
		r, err := res.Read()
		logging.GetLogger().Debugln(fmt.Sprintf("result from stream: %v", r))
		if err != nil { //nolint:nestif // acceptable
			if errors.Is(err, io.EOF) {
				if !isHeaderRead {
					header := rw.getHeader(r)
					w = rw.writer
					if !rw.outputCtx.RuntimeContext.CSVHeadersDisable {
						//nolint:errcheck // output stream is not critical
						w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(header, ","), fmt.Sprintln(""))))
					}
					isHeaderRead = true //nolint:ineffassign,wastedassign // looks false +ve
				}
				rowsArr, err = tabulateResults(r, rw.ci)
				if err != nil {
					return err
				}
				for _, rs := range rowsArr {
					rowSlice := make([]string, len(rs))
					for i, c := range rs {
						s := c
						b, bErr := iqlutil.PrettyPrintSomeJSON([]byte(s))
						if bErr != nil {
							rowSlice[i] = s
							continue
						}
						rowSlice[i] = string(b)
					}
					//nolint:errcheck // output stream is not critical
					w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(rowSlice, ","), fmt.Sprintln(""))))
				}
				return nil
			}
			return err
		}
		if !isHeaderRead {
			header := rw.getHeader(r)
			w = rw.writer
			if !rw.outputCtx.RuntimeContext.CSVHeadersDisable {
				//nolint:errcheck // output stream is not critical
				w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(header, ","), fmt.Sprintln(""))))
			}
			isHeaderRead = true
		}
		rowsArr, err = tabulateResults(r, rw.ci)
		if err != nil {
			return err
		}
		for _, rs := range rowsArr {
			rowSlice := make([]string, len(rs))
			for i, c := range rs {
				s := c
				b, bErr := iqlutil.PrettyPrintSomeJSON([]byte(s))
				if bErr != nil {
					rowSlice[i] = s
					continue
				}
				rowSlice[i] = string(b)
			}
			//nolint:errcheck // output stream is not critical
			w.Write([]byte(fmt.Sprintf("%s%s", strings.Join(rowSlice, ","), fmt.Sprintln(""))))
		}
	}
}

func (csvw *CSVWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(csvw.errWriter, err)
	}
	w := csv.NewWriter(csvw.writer)
	//nolint:errcheck // output stream is not critical
	w.Write(
		[]string{
			errorKey,
		},
	)
	//nolint:errcheck // output stream is not critical
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
	//nolint:errcheck // output stream is not critical
	w.Write([]byte(fmt.Sprintf("%s%s", errorKey, fmt.Sprintln(""))))
	//nolint:errcheck // output stream is not critical
	w.Write([]byte(fmt.Sprintf("%s%s", err.Error(), fmt.Sprintln(""))))
	return nil
}

func (rw *PrettyWriter) WriteError(err error, errorPresentation string) error {
	if errorPresentation == stderrPressentationStr {
		return writeStderrError(rw.errWriter, err)
	}
	w := rw.writer
	//nolint:errcheck // output stream is not critical
	w.Write([]byte(fmt.Sprintf("%s%s", errorKey, fmt.Sprintln(""))))
	//nolint:errcheck // output stream is not critical
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
