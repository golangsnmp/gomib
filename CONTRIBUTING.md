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

### Test helper prefixes

Test helpers follow a naming convention based on what they do and whether they fail the test:

Assertive (call `t.Fatalf` on failure):

- `require*` - lookup/assertion that fails the test if not found or invalid
- `load*` - fixture/corpus loader that fails on I/O or parse error

Non-assertive (no `t.Fatal`):

- `new*` - simple constructor, returns a ready-to-use value
- `build*` - multi-step fixture builder, assembles a complex test structure

Generic assertions live in `internal/testutil/` (`Equal`, `NoError`, `NotNil`, etc.).

### Diagnostic codes

All diagnostic codes are string constants in `internal/types/diagcode.go`. Never use string literals for diagnostic codes.

Unknown codes cause a panic at call time. This is intentional - it catches typos and missing registrations immediately during development rather than silently producing broken diagnostics. All codes are validated at init.

