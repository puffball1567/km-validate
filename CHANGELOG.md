# Changelog

## 0.3.1

- Preserve configuration errors for missing lookup callbacks, array rules and
  validators with explicit nullable types under Kinmokusei v0.4.12.
- Name callback types `Lookup` and `ElementRule<T>` without changing the Go API.
- Test the minimum supported v0.4.7 compiler alongside v0.4.12, and enforce full
  library statement coverage using compiler source maps.

## 0.2.0

- Add conditional required rules: `requiredIf`, `requiredUnless`,
  `requiredWith`, and `requiredWithout`.
- Add named-field equality/difference and exact integer/finite-number comparisons.
- Add array count bounds and generic `each` validation with indexed field paths,
  nested collections, and rollback on callback errors.
- Add file presence, byte limits, and MIME checks based on file content.
- Add `unique` and `exists` lookup callbacks with backend error propagation.
- Document form validation, parameterized database lookups, and executable examples.
- Preserve the v0.1 API and structured issue format.
