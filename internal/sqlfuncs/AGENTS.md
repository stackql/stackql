# internal/sqlfuncs

Pure Go implementations of the StackQL SQLite extension functions, previously
implemented in C in github.com/stackql/sqlite-ext-functions (now retired for
the stackql binary). Registered into the embedded modernc.org/sqlite backend.

## Layout

```
internal/sqlfuncs/
  splitpart.go        - split_part(str, sep, n)
  regexp.go           - regexp_like, regexp_substr, regexp_replace
  jsonequal.go        - json_equal(a, b)
  awspolicyequal.go   - aws_policy_equal(a, b)
  register.go         - Register(drv *sqlite.Driver) - the ONLY driver-facing file
  testdata/           - golden vectors copied from sqlite-ext-functions/test
  DIVERGENCES.md      - documented behavioral differences vs the C implementations
```

## Invariants

- All six functions are deterministic scalars. Register them with the
  deterministic flag so SQLite can use them in indexes and generated columns.
- Registration happens ONLY via `Register(drv *sqlite.Driver)` on a
  caller-constructed driver (requires modernc.org/sqlite >= v1.57.0).
  Never use the package-global `MustRegister*` functions - no global state.
- Logic files must not import modernc.org/sqlite or database/sql/driver
  types beyond what the adapter passes through. Pure logic stays testable
  without a database.
- Arguments arrive as driver.Value: int64, float64, string, []byte, or nil.
  Every function must handle all five. NULL handling must match the golden
  vectors exactly (SQLite convention is NULL in -> NULL out, but verify per
  function against testdata, do not assume).
- Regexp functions use Go's regexp (RE2). RE2 rejects backreferences and
  lookaround. Any pattern the C engine accepted that RE2 rejects is a
  documented divergence in DIVERGENCES.md, never a silent behavior change.
  Replacement syntax: Go uses $1, verify what the C implementation used and
  translate if needed - the golden vectors are the contract.
- json_equal: number semantics (1 vs 1.0, large integers, float precision)
  and invalid-JSON behavior must match the golden vectors.
- aws_policy_equal: normalization rules (scalar vs array forms of Action /
  Resource / Principal, statement ordering insensitivity, IAM case
  sensitivity rules) must match the golden vectors.

## Testing

- testdata/ golden vectors are the compatibility contract with the retired C
  implementations. Changing a vector is a deliberate, documented decision,
  never a test "fix".
- Every function has a native Go fuzz target (FuzzSplitPart, FuzzRegexpLike,
  etc.). Fuzz targets must stay green; new crash corpus entries get committed
  as regression cases.
- Unit tests run without a database. One integration test in the backend
  package proves registration end to end via
  `SELECT split_part('a,b', ',', 2)` and one call per function.

## Adding a function

1. Pure logic in its own file, table-driven unit tests, fuzz target.
2. Golden vectors in testdata/ before wiring anything up.
3. One line in register.go.
4. Document it in the stackql function reference docs.
