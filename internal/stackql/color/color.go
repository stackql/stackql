package color

import (
	fc "github.com/fatih/color"
	"github.com/stackql/stackql/internal/stackql/dto"
	// log "github.com/sirupsen/logrus"
)

type Attribute fc.Attribute

type ColorDriver struct {
	stack      *fc.ColorStack
	runtimeCtx dto.RuntimeCtx
}

func NewColorDriver(runtimeCtx dto.RuntimeCtx) *ColorDriver {
	cd := &ColorDriver{
		stack:      fc.InitColorStack(),
		runtimeCtx: runtimeCtx,
	}
	cd.setupColor()
	return cd
}

func (cd *ColorDriver) New(value ...Attribute) {
	attr := make([]fc.Attribute, len(value))
	for i := range value {
		attr[i] = fc.Attribute(value[i])
	}
	cd.stack.NewColor(attr...)
}

func (cd *ColorDriver) Peek() *fc.Color {
	return cd.stack.Peek()
}

func (cd *ColorDriver) PeekBelow() *fc.Color {
	return cd.stack.SeekBelow(1)
}

func (cd *ColorDriver) Pop() (*fc.Color, error) {
	return cd.stack.Pop()
}

func (cd *ColorDriver) SprintFunc() func(a ...interface{}) string {
	return cd.stack.Peek().SprintFunc()
}

func (cd *ColorDriver) setupDarkColorScheme() {
	cd.New(Attribute(fc.FgWhite), Attribute(fc.BgBlack))
}

func (cd *ColorDriver) setupLightColorScheme() {
	cd.New(Attribute(fc.FgBlack), 48, 5, 231)
}

func (cd *ColorDriver) setupDarkPromptColorScheme() {
	cd.New(Attribute(fc.FgYellow))
}

func (cd *ColorDriver) setupLightPromptColorScheme() {
	cd.New(Attribute(fc.FgMagenta))
}

func (cd *ColorDriver) getDarkErrorColorAttributes() []Attribute {
	return []Attribute{
		Attribute(fc.FgMagenta),
		Attribute(fc.BgBlack),
	}
}

func (cd *ColorDriver) getLightErrorColorAttributes() []Attribute {
	return []Attribute{
		Attribute(fc.FgRed), 48, 5, 231,
	}
}

func (cd *ColorDriver) GetErrorColorAttributes(runtimeCtx dto.RuntimeCtx) []Attribute {
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

func (cd *ColorDriver) setupColor() {
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

func (cd *ColorDriver) setupPromptColor() {
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

func (cd *ColorDriver) ResetColorScheme() {
	cd.stack = fc.InitColorStack()
	cd.New(Attribute(fc.Reset))
}

func (cd *ColorDriver) ShellColorPrint(s string) string {
	if cd.stack.Peek() == nil {
		return s
	}
	cd.setupPromptColor() // comes with implicit Push()
	rv := cd.SprintFunc()(s)
	cd.Pop()
	return rv
}
