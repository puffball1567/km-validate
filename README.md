# km-validate

`km-validate` is a validation library written for Kinmokusei. It accumulates
structured issues while keeping invalid rule configuration separate from
ordinary user-input failures.

The `km` prefix comes from Kinmokusei's `.km` source extension and identifies
libraries designed around Kinmokusei APIs.

## Requirements

- Kinmokusei v0.4.3 or later
- Go 1.23 or later

## Features

- Structured `field`, `code`, and `message` issues
- Required, Unicode-aware length, comparison, containment, and regular-expression rules
- Email, HTTP URL, IP address, CIDR, UUID, and RFC3339 timestamp rules
- Choice, inclusive integer-range, and finite number-range rules
- Conditional required rules and comparisons between named fields
- Typed array-element validation with indexed issues and atomic accumulation
- File byte limits and MIME type checks based on the actual content
- Database uniqueness and existence checks through caller-provided lookups
- Multiple accumulated issues with per-field queries
- Atomic batch addition and independent issue snapshots
- Strict portable field paths and stable machine-readable issue codes
- Explicit configuration errors without partial validator mutation
- No hidden process I/O, global state, or external runtime dependencies

Field paths start with an identifier and use dot-separated identifiers or
canonical non-negative array indexes: `contacts.0.email`, `matrix.1.2`.
Identifiers follow `[A-Za-z_][A-Za-z0-9_-]*`; indexes contain only digits and
have no leading zeros except `0`. Wildcards are not issue paths. Issue codes
follow `[a-z][a-z0-9_]*`.

## Installation

```sh
keika deps add github.com/puffball1567/km-validate@v0.2.0
```

```ts
import { newValidator } from "km-validate";
```

## Example

```ts
import { newValidator } from "km-validate";
import go fmt from "fmt";

function main(): void {
  const validator = newValidator();
  const [nameError] = validator.required("user.name", "");
  const [emailError] = validator.email("user.email", "not-an-email");
  const [ageError] = validator.integerRange("user.age", 15, 18, 120);
  if (nameError !== nil || emailError !== nil || ageError !== nil) {
    fmt.Println("validator configuration error");
    return;
  }

  for (const issue of validator.all()) {
    fmt.Printf("%s: %s\n", issue.field, issue.message);
  }
}
```

Ordinary input mismatches add an `Issue`; they do not return an error. A
returned error means the rule itself is invalid or a caller-provided lookup failed.
Invalid rules include a malformed regular
expression, reversed numeric bounds, an empty choice set, or an invalid field
path. Configuration and lookup errors never partially mutate the validator.

## Rules

- `required` rejects empty or whitespace-only text.
- `minLength`, `maxLength`, and `lengthRange` count Unicode code points.
- `exactLength` checks an exact Unicode code-point count.
- `equal`, `notEqual`, `contains`, `startsWith`, and `endsWith` use exact,
  case-sensitive string comparisons.
- `matches` uses Go regular expressions and returns malformed patterns.
- `email` accepts a bare address and rejects display-name forms.
- `httpURL` accepts absolute HTTP and HTTPS URLs with a host.
- `oneOf` uses exact string matching.
- `integerRange` and `numberRange` use inclusive bounds.
- `numberRange` records `finite_number` for NaN and infinities while requiring
  finite ordered bounds.
- `ipAddress` accepts IPv4 and IPv6 addresses, while `cidr` requires a prefix.
- `uuid` accepts the canonical 36-character hexadecimal representation.
- `timestamp` accepts RFC3339 timestamps including offsets and fractional seconds.

`all` and `forField` return independent slices. `addAll` validates an entire
batch before adding anything. A `Validator` is intended for one validation
operation and is not safe for concurrent mutation.

## Form validation

All rules retain the same `Result<void>` contract and accumulate ordinary input
failures as structured issues. Existing v0.1 APIs remain available.

| API | Behavior / issue code |
| --- | --- |
| `requiredIf(field, value, condition)` | Required when `condition` is true; `required_if` |
| `requiredUnless(field, value, condition)` | Required when `condition` is false; `required_unless` |
| `requiredWith(field, value, others)` | Required when any related string is nonblank; `required_with` |
| `requiredWithout(field, value, others)` | Required when any related string is blank; `required_without` |
| `same(field, value, otherField, otherValue)` | Exact, case-sensitive equality; `same` |
| `different(field, value, otherField, otherValue)` | Exact inequality; `different` |
| `compareInteger(field, value, operator, otherField, otherValue)` | Exact int64 comparison; `compare_integer` |
| `compareNumber(field, value, operator, otherField, otherValue)` | Finite float64 comparison; `compare_number` or `finite_number` |
| `arrayLength(field, count, minimum, maximum)` | Inclusive item-count bounds; `array_length` |
| `each<T>(validator, field, values, rule)` | Runs typed rules on indexed paths and accumulates their issues |
| `file(field, content, maximumBytes, allowedTypes)` | Presence, byte limit, detected MIME type; `required_file`, `file_size`, `file_type` |
| `unique(field, lookup)` | Rejects when a matching record exists; `unique` |
| `exists(field, lookup)` | Rejects when a matching record does not exist; `exists` |

Blank means empty or Unicode whitespace only. Related-value lists must not be
empty. Express richer conditions directly, for example
`requiredIf("company", company, accountType == "business")`.
The numeric operators are `<`, `<=`, `>`, `>=`, `==`, and `!=`; invalid
operators are configuration errors. The comparison target for `compareNumber`
must be finite; a non-finite submitted value adds `finite_number`.

```ts
import { newValidator, Validator, each } from "km-validate";

function validateContacts(emails: string[]): Result<void> {
  const validator = newValidator();
  validator.arrayLength("contacts", len(emails), 1, 10)?;
  each<string>(validator, "contacts", emails,
    (items: Validator, field: string, value: string): Result<void> => {
      items.required(field, value)?;
      return items.email(field, value);
    })?;
  // Invalid entries produce contacts.0, contacts.1, etc.
  // Handle validator.all() or validator.valid() here.
  return ok();
}
```

`each` supplies a temporary validator to the callback. Use that validator and
the supplied path; nested calls can validate arrays of structs or arrays within
arrays. All element issues are committed in input order only if every callback
returns successfully. Configuration or lookup errors discard the whole batch,
retaining any pre-existing issues. Callback side effects outside the temporary
validator are the caller's responsibility. Empty arrays succeed; use
`arrayLength` to require at least one item. Optional rules are explicitly gated
by the caller; this library does not automatically skip empty strings.

### Files

Pass the complete file bytes. A nil slice means absent; a non-nil empty slice
means a present zero-byte file. A non-negative byte limit and a nonempty list of
exact MIME types (without parameters or wildcards) are required. Types are
normalized case-insensitively. Validation stops at the first failure: absent,
size, then type. MIME detection uses Go's content sniffing on at most the first
512 bytes; client filenames and declared Content-Type are not used.

The caller reads multipart uploads and applies request/read limits before
buffering. `file` performs no file I/O and does not parse, decode, or scan the
entire document. Content sniffing is a format check, not proof that a file is
safe or structurally complete. Skip the rule explicitly for optional uploads.

### Database lookups

`lookup` is `() => Result<boolean>`, returning whether a matching row exists.
Closures can capture any typed value, the request context, database connection,
tenant scope, and current record ID. Each valid rule calls it exactly once;
nil callbacks or invalid paths fail before the lookup runs. Backend failures
return the original error without adding an input issue.

For example, capture a parameterized query in a callback:

```ts
import { Validator } from "km-validate";
import go context from "context";
import go sql from "database/sql";

function validateEmail(validator: Validator, db: *sql.DB,
    requestContext: context.Context, email: string, currentID: int64): Result<void> {
  return validator.unique("email", (): Result<boolean> => {
    let count: int64 = 0;
    const lookupError = db.QueryRowContext(requestContext,
      "SELECT COUNT(*) FROM users WHERE email = ? AND id <> ?",
      email, currentID).Scan(&count);
    if (lookupError !== nil) { return fail(lookupError); }
    return ok(count > 0);
  });
}
```

Use the placeholder syntax required by your driver (the example uses `?`),
include tenant filters when applicable, and bind input values as parameters.
Create flows omit the current-ID exclusion. `exists` uses the same callback
contract. Keep database unique/foreign-key constraints: validation is a useful
form error check and cannot prevent concurrent writes between lookup and save.

See [the executable form example](examples/forms/main.km) for all five rule
families in one validation operation.

## Development

Run source checks, static analysis, the complete rule/configuration/boundary
matrix, 100% generated-Go statement coverage, the race detector, seven fuzz
targets, executable basic/form examples, and an independent short-import consumer:

```sh
KEIKA=keika sh scripts/test.sh
```

Generated Go is test output and is not maintained as source.

## License

Apache-2.0

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development, pull request,
release, and hotfix rules.
