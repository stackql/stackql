package sqlfuncs

import (
	"database/sql/driver"
	"errors"

	sqlite "modernc.org/sqlite"
)

// Register registers the StackQL extension functions as deterministic scalar
// functions on the supplied driver. It is the only driver-facing entry point
// of this package.
func Register(drv *sqlite.Driver) error {
	if drv == nil {
		return errors.New("sqlfuncs: cannot register functions on a nil driver")
	}
	registrations := []struct {
		name  string
		nArgs int32
		impl  func(args []driver.Value) (driver.Value, error)
	}{
		{"split_part", 3, func(args []driver.Value) (driver.Value, error) {
			return SplitPart(args[0], args[1], args[2])
		}},
		{"regexp_like", 2, func(args []driver.Value) (driver.Value, error) {
			return RegexpLike(args[0], args[1])
		}},
		{"regexp", 2, func(args []driver.Value) (driver.Value, error) {
			return Regexp(args[0], args[1])
		}},
		{"regexp_substr", 2, func(args []driver.Value) (driver.Value, error) {
			return RegexpSubstr(args[0], args[1])
		}},
		{"regexp_replace", 3, func(args []driver.Value) (driver.Value, error) {
			return RegexpReplace(args[0], args[1], args[2])
		}},
		{"json_equal", 2, func(args []driver.Value) (driver.Value, error) {
			return JSONEqual(args[0], args[1])
		}},
		{"aws_policy_equal", 2, func(args []driver.Value) (driver.Value, error) {
			return AWSPolicyEqual(args[0], args[1])
		}},
	}
	for _, reg := range registrations {
		impl := reg.impl
		err := drv.RegisterDeterministicScalarFunction(
			reg.name,
			reg.nArgs,
			func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				return impl(args)
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}
