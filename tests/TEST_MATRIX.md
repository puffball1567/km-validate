# Test matrix

| Area | Normal cases | Edge cases | Failure cases |
| --- | --- | --- | --- |
| Field paths | ASCII identifiers and dotted paths | `_`, mixed case, digits and hyphens after segment start | empty, surrounding whitespace, empty segments, numeric starts, controls, separators and Unicode |
| Issue codes | lowercase machine-readable identifiers | one byte, digits and underscores after first letter | empty, uppercase, numeric/underscore starts, hyphens, whitespace and Unicode |
| Collections | reject, batch add, all, per-field lookup and presence | empty/missing lookups and snapshot independence | invalid batch members fail atomically without partial mutation |
| Required | nonblank text | empty and whitespace-only values | invalid field paths are configuration errors |
| Length | minimum, maximum and inclusive range | zero, exact boundaries and Unicode rune counts | negative/reversed configuration and out-of-range values |
| Pattern | matching and nonmatching expressions | empty values and anchored patterns | malformed regular expressions are configuration errors |
| Email | ordinary and tagged addresses | empty, surrounding whitespace and display-name forms | malformed addresses become issues without runtime errors |
| HTTP URL | HTTP/HTTPS, uppercase schemes, paths, queries, fragments, IPv4 and ports | exact host-only URLs | relative, malformed, hostless and non-HTTP URLs |
| Choices | first/later choice and missing value | duplicate or empty choice values | empty choice sets are configuration errors |
| Integers | inclusive lower/upper boundaries | full int64-compatible values | reversed bounds and outside values |
| Numbers | inclusive boundaries and fractional values | NaN and both infinities become finite-number issues | non-finite/reversed bounds are configuration errors |
| State | multiple accumulated issues and clear | existing state survives configuration errors | invalid operations and batches never partially mutate state |
| Robustness | fixed regression matrices | arbitrary fields, codes, messages, Unicode/invalid UTF-8 strings and rule arguments | four fuzz targets never panic or violate atomicity |
