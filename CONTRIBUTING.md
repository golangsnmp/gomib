# Contributing to gomib

## Development

```bash
go test ./...           # run all tests
gofumpt -w .            # format
golangci-lint run ./... # lint
```

## Code conventions

### Parser method prefixes

The parser uses four method prefixes, distinguished by what they return:

- `expect*` - consume a single token matching a condition, return `lexer.Token`
- `parse*` - consume tokens and build a typed AST fragment (returns `ast.*` types)
- `skip*` - discard tokens without producing a value
- `convert*` - transform already-consumed span text into a Go value (no token consumption)

### Resolver lookup prefixes

The resolver uses two prefixes, distinguished by search scope:

- `lookup*` - scope-bounded lookup (module's own symbols + imports)
- `resolve*` - lookup with fallback/search behavior (tries well-known modules, global search)

Scope qualifiers (`Direct`, `ByModuleName`, `Global`) are added only when deviating from the default module+imports lookup scope.

### Diagnostic codes

All diagnostic codes are string constants in `internal/types/diagcode.go`. Never use string literals for diagnostic codes.

Unknown codes cause a panic at call time. This is intentional - it catches typos and missing registrations immediately during development rather than silently producing broken diagnostics. All codes are validated at init.

