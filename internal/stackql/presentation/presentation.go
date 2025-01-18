package presentation

import (
	"fmt"

	"github.com/stackql/any-sdk/pkg/dto"
)

type Driver interface {
	Sprintf(format string, args ...interface{}) string
	Print(s string) string
}

type prezzoDriver struct {
	runtimeCtx dto.RuntimeCtx
}

func NewPresentationDriver(runtimeCtx dto.RuntimeCtx) Driver {
	cd := &prezzoDriver{
		runtimeCtx: runtimeCtx,
	}
	return cd
}

func (prezzoDriver *prezzoDriver) Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func (prezzoDriver *prezzoDriver) Print(s string) string {
	return fmt.Sprint(s)
}
