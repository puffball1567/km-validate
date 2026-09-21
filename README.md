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
- Required, Unicode-aware length, regular-expression, email, and HTTP URL rules
- Choice, inclusive integer-range, and finite number-range rules
- Multiple accumulated issues with per-field queries
- Atomic batch addition and independent issue snapshots
- Strict portable field paths and stable machine-readable issue codes
- Explicit configuration errors without partial validator mutation
- No hidden process I/O, global state, or external runtime dependencies

Field paths use dot-separated identifiers. Each segment follows
`[A-Za-z_][A-Za-z0-9_-]*`. Issue codes follow `[a-z][a-z0-9_]*`.

## Installation

```sh
keika deps add github.com/puffball1567/km-validate@v0.1.0
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
returned error means the rule itself is invalid, such as a malformed regular
expression, reversed numeric bounds, an empty choice set, or an invalid field
path. Configuration errors never partially mutate the validator.

## Rules

- `required` rejects empty or whitespace-only text.
- `minLength`, `maxLength`, and `lengthRange` count Unicode code points.
- `matches` uses Go regular expressions and returns malformed patterns.
- `email` accepts a bare address and rejects display-name forms.
- `httpURL` accepts absolute HTTP and HTTPS URLs with a host.
- `oneOf` uses exact string matching.
- `integerRange` and `numberRange` use inclusive bounds.
- `numberRange` records `finite_number` for NaN and infinities while requiring
  finite ordered bounds.

`all` and `forField` return independent slices. `addAll` validates an entire
batch before adding anything. A `Validator` is intended for one validation
operation and is not safe for concurrent mutation.

## Development

Run source checks, static analysis, the complete rule/configuration/boundary
matrix, 100% generated-Go statement coverage, the race detector, four fuzz
targets, the executable example, and an independent short-import consumer:

```sh
KEIKA=keika sh scripts/test.sh
```

Generated Go is test output and is not maintained as source.

## License

Apache-2.0
