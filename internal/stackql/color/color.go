package color

import (
	fc "github.com/fatih/color"
	"github.com/stackql/stackql/internal/stackql/dto"
)

type Attribute fc.Attribute

type Driver struct {
	stack      *fc.ColorStack
	runtimeCtx dto.RuntimeCtx
}

func NewColorDriver(runtimeCtx dto.RuntimeCtx) *Driver {
	cd := &Driver{
		stack:      fc.InitColorStack(),
		runtimeCtx: runtimeCtx,
	}
	cd.setupColor()
	return cd
}

func (cd *Driver) New(value ...Attribute) {
	attr := make([]fc.Attribute, len(value))
	for i := range value {
		attr[i] = fc.Attribute(value[i])
	}
	cd.stack.NewColor(attr...)
}

func (cd *Driver) Peek() *fc.Color {
	return cd.stack.Peek()
}

func (cd *Driver) PeekBelow() *fc.Color {
	return cd.stack.SeekBelow(1)
}

func (cd *Driver) Pop() (*fc.Color, error) {
	return cd.stack.Pop()
}

func (cd *Driver) SprintFunc() func(a ...interface{}) string {
	return cd.stack.Peek().SprintFunc()
}

func (cd *Driver) setupDarkColorScheme() {
	cd.New(Attribute(fc.FgWhite), Attribute(fc.BgBlack))
}

func (cd *Driver) setupLightColorScheme() {
	cd.New(Attribute(fc.FgBlack), 48, 5, 231) //nolint:gomnd // color functionality is mothballed
}

func (cd *Driver) setupDarkPromptColorScheme() {
	cd.New(Attribute(fc.FgYellow))
}

func (cd *Driver) setupLightPromptColorScheme() {
	cd.New(Attribute(fc.FgMagenta))
}

func (cd *Driver) getDarkErrorColorAttributes() []Attribute {
	return []Attribute{
		Attribute(fc.FgMagenta),
		Attribute(fc.BgBlack),
	}
}

func (cd *Driver) getLightErrorColorAttributes() []Attribute {
	return []Attribute{
		Attribute(fc.FgRed), 48, 5, 231,
	}
}

func (cd *Driver) GetErrorColorAttributes(runtimeCtx dto.RuntimeCtx) []Attribute {
	var retVal []Attribute
	switch cd.runtimeCtx.ColorScheme {
	case dto.LightColorScheme:
		retVal = cd.getLightErrorColorAttributes()
	case dto.DarkColorScheme:
		retVal = cd.getDarkErrorColorAttributes()
	case dto.NullColorScheme:
	default:
		retVal = cd.getDarkErrorColorAttributes()
	}
	return retVal
}

func (cd *Driver) setupColor() {
	switch cd.runtimeCtx.ColorScheme {
	case dto.LightColorScheme:
		cd.setupLightColorScheme()
	case dto.DarkColorScheme:
		cd.setupDarkColorScheme()
	case dto.NullColorScheme:
	default:
		cd.setupDarkColorScheme()
	}
}

func (cd *Driver) setupPromptColor() {
	switch cd.runtimeCtx.ColorScheme {
	case dto.LightColorScheme:
		cd.setupLightPromptColorScheme()
	case dto.DarkColorScheme:
		cd.setupDarkPromptColorScheme()
	case dto.NullColorScheme:
	default:
		cd.setupDarkPromptColorScheme()
	}
}

func (cd *Driver) ResetColorScheme() {
	cd.stack = fc.InitColorStack()
	cd.New(Attribute(fc.Reset))
}

func (cd *Driver) ShellColorPrint(s string) string {
	if cd.stack.Peek() == nil {
		return s
	}
	cd.setupPromptColor() // comes with implicit Push()
	rv := cd.SprintFunc()(s)
	cd.Pop() //nolint:errcheck // we don't care about the error
	return rv
}
