# Behavioral divergences vs the retired C implementations

This file documents every known behavioral difference between the pure Go
implementations in this package and the retired C implementations. The
authoritative C reference is the `SQLITE_ENABLE_STACKQL` amalgamation
additions in the stackql-go-sqlite3 fork - the code that actually shipped
in the retired cgo builds. github.com/stackql/sqlite-ext-functions is the
upstream source of most of that code, but the two are NOT identical: for
`aws_policy_equal` the fork carried two modifications that were never
backported to sqlite-ext-functions (see "aws_policy_equal comparison
rules" below). Anything not listed here is intended to be byte-for-byte
compatible with the fork's amalgamation additions and is pinned by the
golden vectors in `testdata/`.

## Regexp functions (`regexp_like`, `regexp_substr`, `regexp_replace`)

`regexp(pattern, source)` is a pure addition with no C counterpart: the
retired builds never registered it (the amalgamation's `regexp()` is
ICU-gated and was not compiled), so the SQL `REGEXP` operator documented in
the stackql language spec previously failed with "no such function". It is
`regexp_like` with swapped arguments, per the SQLite REGEXP-operator
contract, and shares every divergence below.

The C implementations used tiny-regex-c (compiled with
`RE_DOT_MATCHES_NEWLINE=1`), a deliberately minimal engine. The Go
implementations use Go's `regexp` (RE2). The pattern language differences:

1. **Groups `(...)` are now real groups.** tiny-regex-c treated `(` and `)`
   as literal characters, so `abra(cad)?abra` could never match
   `abracadabra`. RE2 interprets the group, so it matches. The affected
   golden vectors (`regexp_like` `sef_10_01`, `regexp_replace` `sef_10_02`)
   are recorded with the RE2 outcome and carry the original C outcome in
   `retiredCResult`. The upstream test suite itself annotated these cases as
   engine-conditional ("If (cad)? is unsupported, test returns 0").
2. **Alternation `|` is now supported.** tiny-regex-c had its `BRANCH`
   opcode disabled; `|` was a literal character. RE2 interprets it.
3. **Inverted character classes `[^...]` now work correctly.** They are
   documented as broken in tiny-regex-c.
4. **No pattern-size ceiling.** tiny-regex-c silently failed to compile
   patterns beyond ~30 nodes. RE2 has no comparable limit (default RE2
   program-size limits apply).
5. **Lazy quantifier semantics.** tiny-regex-c implemented `?` following a
   quantifier with bespoke, partially non-standard semantics. RE2 implements
   standard non-greedy quantifiers.
6. **Rune-oriented matching.** tiny-regex-c matched bytes; RE2 matches UTF-8
   runes. For ASCII patterns and subjects (the entire golden corpus) the
   behavior is identical; multi-byte input may differ (e.g. `.` consumes one
   rune, not one byte).
7. **Rejected patterns.** RE2 rejects backreferences and lookaround with an
   error. tiny-regex-c did not support them either, but treated the
   metacharacters literally instead of erroring.

Preserved C behaviors (parity, pinned by vectors and tests):

- `.` matches newline (`RE_DOT_MATCHES_NEWLINE=1` parity): patterns are
  compiled with the `(?s)` flag.
- `regexp_replace` replacement text is **literal**: the C code copied the
  replacement verbatim, so `$1`-style capture tokens are NOT expanded
  (`ReplaceAllLiteralString`; vector `cref_literal_replacement`).
- NULL in any argument yields NULL; an unparseable pattern is an error.
- `regexp_substr` with no match yields NULL; an empty pattern matches the
  empty string at position 0 and yields `''`.
- `regexp_replace` with no match returns the source unchanged.

## JSON parsing (`json_equal`, `aws_policy_equal`)

The C implementations parsed with cJSON; the Go implementations use
`encoding/json` with a strict single-document decode:

8. **Trailing content is rejected.** cJSON's default parse entry point
   ignored trailing bytes after the first JSON value (`{}garbage` parsed).
   Go rejects trailing non-whitespace content with a parse error.
9. **Lax number forms are rejected.** cJSON tolerated some out-of-spec
   number spellings and parsed overflowing numbers to infinity via
   `strtod`. Go enforces RFC 8259 grammar and errors on numbers that
   overflow a float64.
10. **Duplicate object keys.** cJSON kept every member and its lookups
    matched the *first* occurrence; Go's `map[string]any` decoding keeps the
    *last* occurrence. Documents with duplicate keys (invalid per RFC 8259
    "SHOULD be unique") may compare differently.

Preserved C behaviors (parity):

- Numbers compare as IEEE 754 doubles using cJSON's `DBL_EPSILON` rule
  (`|a-b| <= max(|a|,|b|) * DBL_EPSILON`), so `1` equals `1.0` and one-ULP
  differences are equal (vector `cref_number_epsilon`).
- `json_equal`: NULL arguments and parse failures are **errors**, not NULL
  results. Object comparison is unordered with case-*sensitive* keys; array
  comparison is ordered.
- `aws_policy_equal`: NULL arguments and parse failures are errors, except
  that byte-identical inputs return 1 *before* parsing (the C `strcmp`
  shortcut, vector `cref_strcmp_shortcut` - even non-JSON identical strings
  compare equal).

## aws_policy_equal comparison rules

11. **Case-insensitive key lookup tie-breaking.** cJSON's
    `cJSON_GetObjectItem` scanned members in insertion order and returned
    the first case-insensitive match. Go maps do not preserve insertion
    order, so the port resolves lookups deterministically: an exact-case
    match wins, otherwise the lexicographically smallest case-folding-equal
    key is used. Policies whose objects contain multiple keys differing only
    in case (pathological input) may compare differently.

Preserved C behaviors (parity, deliberately including its quirks):

- The unordered-comparison field set is exactly `Action`, `NotAction`,
  `Resource`, `NotResource`, `Principal`, `NotPrincipal`, `AWS`, `Service`,
  `Tags`, `tags` (field name matched case-sensitively - `Tags` and `tags`
  are two distinct entries). All other arrays - including `Statement` -
  compare ordered.
- Two top-level array documents (an array at the root of both documents)
  compare unordered.
- Within an unordered field, a string beginning with `arn:` compares ASCII
  case-insensitively. The prefix test is applied to the **left** operand
  only, preserving the C code's asymmetry.
- A single-element array equals its lone scalar element in either position.
- Unordered array elements are matched left-into-right after a length
  check, without marking right-hand elements as consumed - arrays with
  duplicate elements can double-match, exactly as in the C code.

Fork-only modifications (authoritative contract):

The stackql-go-sqlite3 fork's `SQLITE_ENABLE_STACKQL` amalgamation carried
two modifications to `aws_policy_equal` that were never backported to
github.com/stackql/sqlite-ext-functions. This package implements the fork
behavior, because the fork is what shipped in the retired cgo builds:

- `Tags` and `tags` are members of the unordered-comparison field set
  (vectors `fork_tags_top_*`, `fork_tags_statement_*`,
  `fork_tags_lower_*`).
- Two top-level array documents (array vs array at the document root) are
  compared unordered (vectors `fork_root_array_*`). A root array vs a
  non-array root still compares under the ordinary rules (including the
  single-element-array-equals-scalar rule).

## Argument coercion

12. **Text coercion of non-text arguments now happens in Go.** The C
    implementations received arguments through `sqlite3_value_text()`.
    The Go functions receive typed `driver.Value`s and apply the same
    coercions themselves: integers format in base 10, REALs format in
    shortest round-trip form with a `.0` suffix for integral values
    (matching modern SQLite), blobs coerce byte-wise to text, and the
    `split_part` index applies `sqlite3_value_int` semantics (NULL is 0,
    REALs truncate toward zero, text parses as a leading integer prefix).
    Exotic REAL edge cases could format differently than a given historical
    SQLite build printed them.
