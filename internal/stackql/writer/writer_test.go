package writer //nolint:testpackage // this violates another rule: var-naming: don't use an underscore in package name

import (
	"io"
	"os"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/presentation"
	"github.com/stretchr/testify/assert"
)

type NopWriter struct{}

func (nw NopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func TestGetOutputWriter(t *testing.T) {
	nopWriter := NopWriter{}
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want io.Writer
	}{
		{
			"stdout",
			args{"stdout"},
			os.Stdout,
		},
		{
			"stderr",
			args{"stderr"},
			os.Stderr,
		},
		{
			"file",
			args{"somefile"},
			nopWriter,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := GetOutputWriter(tt.args.filename)

			if tt.name == "file" {
				assert.Implements(t, (*io.Writer)(nil), got)
				defer os.Remove(tt.args.filename)
			} else {
				assert.Equal(t, got, tt.want)
			}
		})
	}
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
