package writer //nolint:testpackage // this violates another rule: var-naming: don't use an underscore in package name

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/presentation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOutputWriterStdStreams(t *testing.T) {
	for filename, want := range map[string]io.Writer{
		StdOutStr: os.Stdout,
		StdErrStr: os.Stderr,
	} {
		t.Run(filename, func(t *testing.T) {
			got, err := GetOutputWriter(filename)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestGetOutputWriterFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	got, err := GetOutputWriter(path)
	require.NoError(t, err)
	file, isFile := got.(*os.File)
	require.True(t, isFile, "a file name must yield an *os.File")
	_, err = io.WriteString(file, "some output")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "some output", string(content))
}

func TestGetDecoratedOutputWriter(t *testing.T) {
	type args struct {
		filename string
		cd       presentation.Driver
	}
	tests := []struct {
		name string
		args args
		want io.Writer
	}{
		{
			"stdout",
			args{"stdout", presentation.NewPresentationDriver(dto.RuntimeCtx{})},
			&StdStreamWriter{os.Stdout, presentation.NewPresentationDriver(dto.RuntimeCtx{})},
		},
		{
			"stderr",
			args{"stderr", presentation.NewPresentationDriver(dto.RuntimeCtx{})},
			&StdStreamWriter{os.Stderr, presentation.NewPresentationDriver(dto.RuntimeCtx{})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := GetDecoratedOutputWriter(tt.args.filename, tt.args.cd)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStdStreamWriter_Write(t *testing.T) {
	type fields struct {
		writer       io.Writer
		prezzoDriver presentation.Driver
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			"stdout",
			fields{os.Stdout, presentation.NewPresentationDriver(dto.RuntimeCtx{})},
			args{[]byte("test")},
			4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ssw := &StdStreamWriter{
				writer:       tt.fields.writer,
				prezzoDriver: tt.fields.prezzoDriver,
			}
			got, _ := ssw.Write(tt.args.p)
			assert.Equal(t, got, tt.want)
		})
	}
}
