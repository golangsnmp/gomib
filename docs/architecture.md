# gomib Architecture Overview

A guide for developers working on or integrating with gomib. Assumes
familiarity with Go and a general understanding of what SNMP MIBs are, but not
deep knowledge of the SMI RFCs or the quirks of real-world MIB files.


## Why Does This Need So Many Steps?

MIB files look deceptively simple - they're text files with type definitions
and OID assignments. In practice, parsing and interpreting them correctly is
harder than it appears:

- **Two incompatible language versions.** SMIv1 (RFC 1155/1212, circa 1990)
  and SMIv2 (RFC 2578/2579/2580, circa 1999) use different keywords, different
  access levels, different notification mechanisms, and different conformance
  models. Both are still in wide use. A MIB parser has to accept both and unify
  them into one model.

- **Cross-module dependencies everywhere.** A typical MIB imports types and
  OID roots from other modules. Those modules import from others. An OBJECT-TYPE
  definition might reference a type defined in one module, an OID parent from
  another, and an index object from a third. Nothing is self-contained.

- **Type inheritance chains.** SMI types form parent chains - DisplayString is
  a refinement of OCTET STRING with a size constraint and display hint.
  SnmpAdminString refines DisplayString with a different size. To know the
  effective constraints on an object, you have to walk the full chain.

- **OID tree assembly.** OIDs are defined symbolically (`{ ifEntry 1 }` means
  "child 1 of ifEntry"), so building the numeric OID tree requires resolving
  symbolic references in dependency order across all loaded modules.

- **Real-world MIBs break the rules.** Vendor MIBs routinely have missing
  imports, wrong module names, deprecated syntax, and undeclared dependencies.
  A useful parser can't just reject these - it needs to recover gracefully and
  report what's wrong while still producing usable output.

These problems layer on top of each other: you can't resolve types until you've
resolved imports, you can't build OIDs until you've resolved types (SEQUENCE OF
references determine table structure), and you can't classify objects until you
have the OID tree. The multi-stage pipeline exists because each stage handles a
genuinely different kind of complexity.


## SMI/MIB Background

If you're already comfortable with SMI, skip this section. This is the minimum
context needed to understand why the code does what it does.

**MIB files** define the management information that SNMP agents expose. Each
file declares a module (e.g., IF-MIB) containing definitions for managed
objects, types, notifications (traps), and conformance groups.

**OIDs** (Object Identifiers) are hierarchical numeric addresses arranged in a
global tree. `1.3.6.1.2.1.2.2.1.1` is ifIndex - each number is an arc in the
tree. MIB files define OIDs symbolically (`{ ifEntry 1 }` rather than
numerically), so the parser must assemble the full numeric path by resolving
references.

**OBJECT-TYPE** is the core definition. It assigns an OID, declares a syntax
(data type), access level (read-only, read-write, etc.), and description.
Tables are defined through a specific pattern: a table object contains a row
object (with SEQUENCE type), which contains column objects (the actual data).

**Types** in SMI are based on ASN.1 but heavily restricted. The base types are
INTEGER, OCTET STRING, OBJECT IDENTIFIER, and a few application types
(Counter32, Gauge32, TimeTicks, IpAddress). Textual conventions (like
DisplayString or DateAndTime) layer constraints and display hints on top of
base types, forming inheritance chains.

**Imports** are explicit - each module declares which symbols it uses from
which other modules. This creates a dependency graph that must be resolved
before anything else.

**SMIv1 vs SMIv2** is the major version split. SMIv1 (RFC 1155) uses ACCESS,
TRAP-TYPE, and status values like `mandatory`. SMIv2 (RFC 2578) uses
MAX-ACCESS, NOTIFICATION-TYPE, status values like `current`, and adds textual
conventions, conformance groups, and MODULE-IDENTITY. Both are still found in
production MIB collections. The language version is inferred conservatively
from explicit base-module identity, imports from base modules, and strong
syntax (`MODULE-IDENTITY` for SMIv2 or `TRAP-TYPE` for SMIv1). Conflicting or
insufficient evidence leaves the language unknown.


## Processing Pipeline

```
Source bytes
  -> Heuristic check (reject non-MIB content)
    -> Lexer (tokenize into SMI token stream)
      -> Parser (build lossless Concrete Syntax Tree)
        -> Lowerer (normalize CST into Module IR)
          -> Validator (structural checks on Module IR)
            -> Resolver (5 phases, build resolved model)
              -> *Mib (queryable, thread-safe output)
```

Each stage produces a distinct representation. Source bytes are released after
parsing, CST after lowering, Module IR after resolution. The final `*Mib` is
the only long-lived object returned to callers.


## Package Layering

Dependencies flow strictly downward:

```
gomib (root)          Public API: Load(), Source, LoadOption
  |
  +-- mib/            Public resolved model + re-exported enums
  |     |
  |     +-- internal/model/     Resolved types (Mib, Node, Object, Type, etc.)
  |
  +-- syntax/          Public CST, tokens, parser, line tables
  |     |
  |     +-- internal/cst/        CST node types
  |     +-- internal/cst/parser/ Recursive descent parser
  |     +-- internal/lexer/      Lexer implementation
  |     +-- internal/types/      Span, LineTable utilities
  |
  +-- internal/
        +-- cst/                CST node types
        |     +-- parser/       Recursive descent parser (tokens -> CST)
        |     +-- lower/        Lowering (CST -> Module IR)
        +-- module/             Module IR types + validation + base modules
        +-- resolver/           Five-phase resolver (Module IR -> *Mib)
        +-- graph/              Topological sort + cycle detection (Tarjan's SCC)
        +-- types/              Shared enums, diagnostics, spans, severity, codes
```

No package imports a sibling or ancestor. `internal/types` is the shared base
layer used by everything else.

The public surface is intentionally small: callers use `gomib.Load()` to get
a `*mib.Mib`, then query it. The `mib` package re-exports all resolved model
types and enums so callers never import `internal/` packages. The `syntax`
package is a secondary entry point for callers who want CST-level, token-level,
or parse-only access without the full resolution pipeline. It re-exports all
CST node types, token types, and provides `Parse()`, `Tokenize()`,
`ReconstructText()`, and `LineTable` for bidirectional offset/position
conversion.


## Stage 1: Source Discovery and Loading

**Files:** `gomib.go`, `load.go`, `source.go`, `searchpath.go`

### Entry Point

```go
mib, err := gomib.Load(ctx,
    gomib.WithSource(gomib.MustDir("/usr/share/snmp/mibs")),
    gomib.WithModules("IF-MIB"),
    gomib.WithSystemPaths(),
)
```

`Load()` accepts functional options (`LoadOption`) and returns a `*mib.Mib`.

### Source Interface

MIB files come from many places - system directories, embedded filesystems,
vendor bundles, user-specified paths. Different tools (net-snmp, libsmi) use
different directory layouts and naming conventions. The `Source` interface
abstracts over all of this:

```go
type Source interface {
    Find(name string) (FindResult, error)
    ListModules() ([]string, error)
}
```

Implementations:
- `Dir(path)` - directory tree, indexed on construction
- `FS(name, fsys)` - any `fs.FS` (including `embed.FS`), lazily indexed
- `File(path)` / `Files(paths...)` - explicit file list
- `Multi(sources...)` - ordered chain, first match wins
- (internal) `embeddedSource` - 7 built-in base modules, always appended last

MIB files have no standardized extension. Different vendors and tools use
different conventions, so the loader recognizes: `""`, `.mib`, `.smi`, `.txt`,
`.my` (configurable via `DefaultExtensions()`).

### System Path Discovery

`WithSystemPaths()` auto-discovers MIB directories from:
- net-snmp: `~/.snmp/mibs`, `/usr/share/snmp/mibs`, config files (`snmp.conf`),
  `MIBDIRS` env var
- libsmi: `/usr/share/mibs/{ietf,iana,irtf,site}`, config files (`smi.conf`),
  `SMIPATH` env var

Config files support replace/prepend/append semantics. Paths are deduplicated
and filtered to existing directories.

### Two Loading Paths

When `WithModules()` is NOT set (load all):
- Enumerate all module names from all sources
- Load in parallel with `runtime.NumCPU()` goroutine semaphore
- File-level caching via `sync.Once` prevents re-parsing multi-module files

When `WithModules()` IS set (load named):
- Sequential recursive loading
- Each module's imports are discovered and loaded depth-first
- Base modules auto-loaded without explicit request

Both paths produce the same `[]*module.Module` list for the resolver.

### Heuristic Check

MIB directories often contain non-MIB files (READMEs, changelogs, license
files, binaries). Before parsing, `looksLikeMIBContent()` rejects files that:
- Contain null bytes (binary files)
- Lack both `DEFINITIONS` and `::=` in the first 128KB


## Stage 2: Lexer

**Files:** `internal/lexer/lexer.go`, `internal/lexer/token.go`,
`internal/lexer/keyword.go`

**Public facade:** `token/token.go` re-exports types and provides
`Tokenize(source) []Token`

### Purpose

Convert raw bytes into a stream of typed tokens. The lexer is a single-pass,
left-to-right scanner with a 4-state machine.

### States

| State | Trigger | Behavior |
|-------|---------|----------|
| `stateNormal` | Default | Scan identifiers, keywords, literals, punctuation |
| `stateInComment` | `--` detected | Consume comment body, emit `TokComment` |
| `stateInMacro` | `MACRO` keyword | Skip body until `END` keyword (boundary-aware) |
| `stateInExports` | `EXPORTS` keyword | Skip body until semicolon |

MACRO and EXPORTS bodies are uninteresting to the parser, so the lexer skips
them entirely for efficiency. MACRO bodies contain ASN.1 grammar notation that
would require a completely different parser; since MIB tools only need the
macro invocations (OBJECT-TYPE, MODULE-IDENTITY, etc.) rather than the macro
definitions, skipping is the right tradeoff.

### Token Categories

~107 token kinds organized into groups:

- **Identifiers:** `TokUppercaseIdent`, `TokLowercaseIdent` (case determines
  which - types are uppercase, values are lowercase per SMI convention)
- **Literals:** `TokNumber`, `TokNegativeNumber`, `TokQuotedString`,
  `TokHexString` (`'FF01'H`), `TokBinString` (`'10110'B`)
- **Operators:** `TokColonColonEqual` (`::=`), `TokDotDot` (`..`)
- **Keywords:** 83 SMI-specific keywords covering structural (`DEFINITIONS`,
  `BEGIN`, `END`), clause (`SYNTAX`, `MAX-ACCESS`, `STATUS`), macro invocation
  (`OBJECT-TYPE`, `MODULE-IDENTITY`), type (`INTEGER`, `Counter32`), and
  status/access values (`current`, `read-only`)
- **Special:** `TokForbiddenKeyword` for ASN.1 reserved words, `TokComment`

### Design Notes

- Tokens store only a `Span` (byte range), not the text itself. Text is
  extracted from the source buffer on demand.
- Keyword lookup is a hash map from string to `TokenKind`.
- Classification helpers (`IsKeyword()`, `IsMacroKeyword()`, `IsTypeKeyword()`,
  etc.) enable grouping without reimplementing keyword tables.
- Error recovery: most lexer errors still produce a token (e.g., an
  unterminated string still yields `TokQuotedString`). Diagnostics are collected
  separately.


## Stage 3: Parser (CST)

**Files:** `internal/cst/parser/parser_*.go`, `internal/cst/nodes_*.go`

### Why a CST?

Most parsers produce an AST (Abstract Syntax Tree) that discards whitespace,
comments, and formatting. gomib's parser instead produces a lossless Concrete
Syntax Tree that preserves every byte of the original source. This matters for
two reasons:

- **Diagnostics need source positions.** When reporting "line 42: missing
  DESCRIPTION clause", the diagnostic system needs to map from tree nodes back
  to exact source locations. A lossless CST makes this straightforward.
- **Normalization/reformatting.** `cst.ReconstructText()` can reproduce the
  original source exactly, which is useful for testing and for future
  tooling (formatting, refactoring).
- **Tooling support.** Language servers, linters, and formatters need
  structural information without running the full resolution pipeline.
  The `syntax` package exposes the CST publicly via `syntax.Parse()`.

The tradeoff is that the CST is bulkier and harder to work with than a
cleaned-up IR, which is why the main `Load()` pipeline lowers it into a
simpler Module IR immediately after parsing (Stage 4). But external tooling
can stop at the CST level when that's all it needs.

### Approach

Recursive descent with manual lookahead. The parser maintains a position in the
token array and uses `peek()`/`peekNth(n)` to look ahead past comments.

Definition dispatch examines the first two meaningful tokens:

```
identifier + OBJECT-TYPE     -> parseObjectType()
identifier + MODULE-IDENTITY -> parseModuleIdentity()
identifier + ::=             -> parseTypeAssignment() or parseValueAssignment()
identifier + TEXTUAL-CONVENTION -> parseTextualConvention()
...
```

### CST Node Types

Each SMI construct has a corresponding node type:

- `ModuleFile` -> `[]ModuleNode` (supports multi-module files)
- `ModuleNode` -> header tokens + `[]DefinitionNode`
- 14 definition node types: `ObjectTypeNode`, `ModuleIdentityNode`,
  `ObjectIdentityNode`, `NotificationTypeNode`, `TrapTypeNode`,
  `TextualConventionNode`, `TypeAssignmentNode`, `ValueAssignmentNode`,
  `ObjectGroupNode`, `NotificationGroupNode`, `ModuleComplianceNode`,
  `AgentCapabilitiesNode`, `MacroDefinitionNode`, `ErrorNode`
- Clause nodes for each clause type (syntax, access, status, description, etc.)
- Type syntax nodes for type expressions and constraints
- `SyntaxToken` wraps each token with leading/trailing trivia

### Error Recovery

Real MIB files frequently contain syntax errors - vendor MIBs especially.
Rather than aborting on the first error, the parser recovers and continues:

When a parse error occurs, `recoverToDefinition()` scans forward for the start
of the next definition (recognized by patterns like `identifier + macro-keyword`
or `identifier + ::=`). Skipped tokens are wrapped in an `ErrorNode`.

If no recovery pattern is found, the parser force-advances by one token to
guarantee progress (prevents infinite loops).

This means a MIB with one broken definition still produces valid parse trees
for all the other definitions, which is important when loading large vendor
MIB collections.

### SMIv1 vs SMIv2

The parser accepts both SMIv1 and SMIv2 syntax without version distinction:
- Both `ACCESS` (SMIv1) and `MAX-ACCESS` (SMIv2) accepted
- Both `TRAP-TYPE` (SMIv1) and `NOTIFICATION-TYPE` (SMIv2) parsed
- Both status value sets accepted (`mandatory`/`optional` and
  `current`/`deprecated`/`obsolete`)

Version detection is deferred to the lowering stage, where explicit base-module
identity, base-module imports, and strong syntax are combined conservatively.


## Stage 4: Lowering (CST -> Module IR)

**Files:** `internal/cst/lower/lower*.go`

### Why Normalize?

The CST faithfully represents what was written in the file, but the same
semantic concept can be expressed in multiple ways in SMI:

- SMIv1's `TRAP-TYPE` and SMIv2's `NOTIFICATION-TYPE` mean the same thing
  (a notification definition) but have completely different syntax.
- `TEXTUAL-CONVENTION` and plain type assignments both define types, but
  with different clause structures.
- DEFVAL values are syntactically ambiguous - the parser sees tokens, but
  whether `{ foo }` is an enum value, a bits value, or an OID component list
  depends on the type (which isn't resolved yet). The lowerer does as much
  classification as possible without type information.
- Imports are grouped by source module in the syntax
  (`symbol1, symbol2 FROM ModuleA`) but the resolver needs them as flat
  per-symbol records.

Lowering collapses these syntactic variations into a uniform Module IR so that
downstream stages don't need to handle multiple representations of the same
concept.

### Key Transformations

- **Language detection:** After definitions are lowered, explicit base-module
  identity, imports from SMIv1/SMIv2 base modules, and strong syntax
  (`MODULE-IDENTITY` or `TRAP-TYPE`) are combined as evidence. A single
  unconflicted version is selected; conflicting or insufficient evidence
  leaves the language unknown.
- **Import flattening:** CST groups imports by module. The lowerer flattens
  these into individual `Import{Module, Symbol, Span}` records.
- **Notification unification:** Both `TrapTypeNode` and `NotificationTypeNode`
  produce the same `Notification` IR type. `Notification.TrapInfo` is non-nil
  only for SMIv1 traps (which derive their OID differently).
- **TypeDef unification:** `TextualConventionNode` and `TypeAssignmentNode`
  both produce `TypeDef`, distinguished by `IsTextualConvention` flag.
- **DEFVAL parsing:** The parser stores DEFVAL content as opaque tokens. The
  lowerer pattern-matches them into typed variants (integer, string, hex, enum,
  bits, OID, etc.).
- **Line table construction:** Built from source bytes to enable span-to-line/col
  conversion for diagnostics. Source bytes can then be released.

### Module IR Structure

```
Module {
    Name, Language, SourcePath, LastUpdated, Span
    Imports []Import
    ObjectTypes, TypeDefs, Notifications, ModuleIdentities,
    ObjectIdentities, ValueAssignments, ObjectGroups,
    NotificationGroups, Compliances, Capabilities
    Diagnostics, LineTable
}
```

All type and OID references remain as unresolved strings/names. Resolution
happens in the next stage.


## Stage 5: Validation

**Files:** `internal/module/validate.go`

### Purpose

Structural checks on the Module IR before resolution. These are checks that
can be performed on a single module in isolation, without cross-module
information. Catching problems here means the resolver doesn't need to
handle them.

### Checks Performed

- SMIv2 modules must have a MODULE-IDENTITY, and it must be the first
  definition (multiple MODULE-IDENTITY definitions produce a warning)
- SMIv2 macro keywords (OBJECT-TYPE, etc.) should be imported
- LAST-UPDATED and REVISION dates validated for format, ranges, and calendar
  correctness
- Revisions must be in reverse chronological order
- LAST-UPDATED must have a matching REVISION entry
- Module naming conventions (SMIv2 modules should end with `-MIB`)

Diagnostics are appended to the module's diagnostic list with line/column
information derived from the line table.


## Stage 6: Resolver

**Files:** `internal/resolver/resolver.go`, `registration.go`, `imports.go`,
`types.go`, `oids.go`, `semantics.go`, `context.go`, `context_lookup.go`,
`well_known.go`, `checks.go`, `checks_style.go`

### Purpose

Transform the collection of unresolved Module IR into a fully resolved `*Mib`
with a connected OID tree, resolved type chains, and classified objects. This is
the most complex stage.

### Why Five Phases?

The resolver can't do everything in one pass because of data dependencies
between different kinds of resolution:

- You can't resolve type references until imports tell you which module defines
  each type name.
- You can't build OIDs until types are resolved (SEQUENCE OF references
  determine table structure, and some OID assignments reference type names).
- You can't classify nodes as scalar/table/row/column until the OID tree
  exists and types are resolved.

Each phase depends on the results of earlier phases, so they run sequentially.

### Phase 1: Registration

**Input:** `[]*module.Module` (from all sources + base modules)
**Output:** Module index, definition name caches, resolved Module shells

- Base modules (SNMPv2-SMI, RFC1155-SMI, etc.) are prepended so primitives
  resolve first
- Creates a resolved `*model.Module` for each input module
- Builds per-module definition name indexes for fast lookup
- Caches references to SNMPv2-SMI, RFC1155-SMI, and SNMPv2-TC for later use

### Phase 2: Imports

**Input:** Per-module import lists
**Output:** `importSources[module][symbol] -> defining module`

Import resolution sounds simple ("look up the symbol in the source module")
but real-world MIBs make it complicated. Modules get renamed over time
(RFC1213-MIB became SNMPv2-MIB), symbols get re-exported through intermediate
modules, and vendor MIBs frequently import from the wrong module name.

The resolver handles this through a multi-level strategy:

1. **Direct lookup:** symbol found in the stated source module
2. **Module aliases** (Normal+): maps old module names to current ones
   (e.g., RFC1213-MIB -> SNMPv2-MIB)
3. **Import forwarding** (Normal+): follows re-export chains through
   intermediate modules
4. **Partial resolution:** resolves what it can, records missing symbols
5. **Transitive collapse:** chains are resolved so `importSources` always
   points to the ultimate definer

When multiple modules define the same symbol, scoring prefers: more matching
symbols in the candidate, newer LAST-UPDATED, then lexicographic module name
for determinism.

### Phase 3: Types

**Input:** All TypeDefs from all modules
**Output:** Resolved `*Type` objects with parent chains and base types

SMI types form inheritance chains. For example, `DisplayString` is defined as
`OCTET STRING (SIZE (0..255))` with a display hint of "255a". When an
OBJECT-TYPE uses DisplayString as its syntax, we need to know its base type
(OctetString), effective size constraint (0..255), and display hint ("255a") -
all inherited from the chain.

The type phase resolves these chains:

1. Seed 4 primitive types: INTEGER, OCTET STRING, OBJECT IDENTIFIER, BITS
2. Create user types from all non-SEQUENCE TypeDefs
3. Build a dependency graph of type references
4. Topological sort (Tarjan's SCC) to resolve parents before children
5. Link parent chains, inherit base types through chain walking (max depth 1000)
6. Map RFC1213 type aliases to SNMPv2-TC equivalents (DisplayString, etc.)
7. Application base types (Counter32, Gauge32, etc.) are preserved through
   inheritance - they don't collapse to INTEGER

Cycles are detected via SCC and reported as unresolved types.

### Phase 4: OIDs

**Input:** OID assignments from all definition types
**Output:** OID trie rooted at `mib.Root()`

MIB files define OIDs symbolically - `{ ifEntry 1 }` means "child arc 1 of
the node named ifEntry." Building the numeric OID tree requires resolving these
symbolic references in dependency order (you have to know where ifEntry is
before you can place ifIndex under it).

1. Collect all OID-bearing definitions (ObjectType, ModuleIdentity,
   ValueAssignment, Notification, etc.)
2. Build dependency graph from OID parent references
3. Topological sort to resolve parents before children
4. For each definition, walk its OID components sequentially:
   - Named: look up in current module, imports, or well-known roots
   - Qualified (Module::Name): look up in specific module
   - Named+Number (e.g., `org(3)`): create named intermediate node
   - Numeric: create or traverse to child arc
5. TRAP-TYPE OIDs derived from enterprise + trapNumber
6. Module preference when multiple modules define the same OID:
   base > SMIv2 > SMIv1, then by LAST-UPDATED, then lexicographic

Well-known roots (`iso`, `ccitt`, `joint-iso-ccitt`) and SMI globals
(`internet`, `mgmt`, `mib-2`, `enterprises`) are available without import
at Normal+ strictness.

### Phase 5: Semantics

**Input:** OID tree, all definition types
**Output:** Objects, Notifications, Groups, Compliances, Capabilities attached
to nodes

With types resolved and the OID tree built, the final phase classifies nodes,
creates resolved entities, and runs conformance checks. This is the most
complex phase.

Key operations:

- **Kind inference:** Classify nodes as Scalar, Table, Row, Column based on
  SEQUENCE OF syntax, INDEX/AUGMENTS clauses, and tree position. This isn't
  declared explicitly in MIB files - it's inferred from patterns (a table
  node has SEQUENCE OF syntax, its single child is the row, the row's children
  are columns).
- **Object creation:** Attach resolved Objects to nodes with type, access,
  units, constraints. Compute effective values by walking type chains. Sizes
  and ranges intersect declared constraints with inherited constraints; an
  empty effective constraint remains distinguished from no constraint.
  Unresolved endpoints are preserved conservatively when an intersection
  cannot be determined. Display hints, enums, and bits use inherited values
  when the object does not declare their replacement.
- **Index resolution:** Link INDEX entries to resolved Objects, classify
  encoding (integer, fixed-string, length-prefixed, implied, ipaddress).
  Index encoding classification matters because SNMP encodes table index
  values into OIDs, and the encoding depends on the index type.
- **AUGMENTS linking:** Bidirectional links between base and augmenting rows.
  AUGMENTS lets one module extend another module's table with additional columns.
- **Entity creation:** Notifications (with resolved object members), Groups,
  Compliances, Capabilities
- **~36 conformance/style checks:** Parent kind validation, integer/enum
  misuse, range overlap, DEFVAL constraint checking, sequence field matching,
  access/status validation, group membership completeness

### Resolver Context

The `resolverContext` is the central working state shared across all phases:

```
resolverContext {
    mib              *model.Mib           // Output container
    modules          []*module.Module     // All input modules
    moduleIndex      map[name][]versions  // Name -> module versions
    importSources    symbolTable          // Module -> Symbol -> DefiningModule
    nodeSymbols      symbolTable          // Module -> Name -> *Node
    typeSymbols      symbolTable          // Module -> Name -> *Type
    strictness       ResolverStrictness   // Strict/Normal/Permissive
    diagConfig       DiagnosticConfig     // Reporting rules
}
```

`symbolTable[T]` is a generic `map[*module.Module]map[string]T` that keeps
lookups scoped to modules while supporting cross-module resolution.


## Resolver Strictness

Real-world MIB collections range from RFC-perfect IETF modules to vendor MIBs
that violate half the spec. Strictness levels let callers choose where they
sit on the correctness-vs-compatibility spectrum:

| Level | Scope | Use Case |
|-------|-------|----------|
| `ResolverStrict` | Import statements only | RFC compliance checking |
| `ResolverNormal` | + well-known modules, module aliases, import forwarding | Default operation |
| `ResolverPermissive` | + global scan across all loaded modules | Best-effort vendor MIB loading |

Strictness is independent of diagnostic configuration. Setting permissive
strictness helps resolution succeed but still emits diagnostics. Users should
configure both for best-effort loading:

```go
gomib.WithResolverStrictness(mib.ResolverPermissive),
gomib.WithDiagnosticConfig(mib.DiagnosticConfig{FailAt: mib.SeverityFatal}),
```


## Diagnostics System

### Why Not Just Return Errors?

A single MIB file might have dozens of issues of varying severity - a missing
description (style), an import from a renamed module (minor), a reference to
an undefined type (error). If `Load()` returned on the first error, callers
would fix one issue at a time and never see the full picture. Instead, gomib
collects diagnostics throughout the entire pipeline and reports them all.

`Load()` returns the `*Mib` even when diagnostics exceed the failure threshold.
The error (`ErrDiagnosticThreshold`) is returned alongside the `*Mib`, letting
callers decide whether partial results are acceptable. This is especially
useful for linting, where you want all diagnostics regardless of severity.

### Flow

Diagnostics are collected at every pipeline stage:

```
Lexer -> SpanDiagnostic (byte offsets)
Parser -> SpanDiagnostic (byte offsets)
  [lowering converts spans to line/col via LineTable]
Validator -> Diagnostic (line/col)
Resolver -> Diagnostic (line/col)
  -> Mib.Diagnostics() (aggregated)
  -> filtered by DiagnosticConfig
  -> checked against FailAt threshold
```

### Severity Levels

7 levels (lower = more severe):

| Level | Name | Meaning |
|-------|------|---------|
| 0 | Fatal | Cannot continue parsing |
| 1 | Severe | Semantics changed to continue |
| 2 | Error | Can continue, should fix |
| 3 | Minor | Minor issue |
| 4 | Style | Style recommendation |
| 5 | Warning | Might be correct |
| 6 | Info | Informational |

### Configuration

```go
DiagnosticConfig {
    Reporting  ReportingLevel       // Silent/Quiet/Default/Verbose
    FailAt     Severity             // Threshold for Load() failure
    Overrides  map[string]Severity  // Per-code severity override
    Ignore     []string             // Glob patterns to suppress
}
```

Presets: `DefaultConfig()` (report Minor+, fail on Severe),
`VerboseConfig()`, `QuietConfig()`, `SilentConfig()`.

### Diagnostic Codes

188 codes organized by pipeline phase:
- Lexer (8): `unexpected-character`, `unterminated-string`, etc.
- Parser (12): `identifier-underscore`, `parse-error`, `bad-identifier-case`, etc.
- Validation (41): `missing-module-identity`, `date-character`, `enum-zero`, etc.
- Resolver (127): `import-not-found`, `type-unknown`, `oid-orphan`,
  `index-unresolved`, `range-overlap`, etc.

Each code has a fixed severity defined at registration time. Using an
unregistered code panics (programmer error, not runtime error).


## Resolved Model (Public API)

The `*mib.Mib` returned by `Load()` is the query interface. All types use
unexported fields with exported accessor methods. Slice accessors return
cloned copies. The Mib is safe for concurrent reads.

### Core Types

- **Mib** - Top-level container. OID tree root, module list, all resolved
  entities, diagnostics. Lookup methods: `Node(name)`, `Object(name)`,
  `Type(name)`, `Module(name)`, `Resolve(query)` (name or OID string).
- **Node** - OID tree node. Arc number, name, kind, parent/children, optional
  entity attachments (Object, Notification, Group, etc.). Lazy-computed OID.
- **Module** - Loaded MIB module. Name, language, source path, MODULE-IDENTITY
  metadata, per-type definition collections, import tracking. Position
  utilities: `LineCol(offset)` converts byte offset to line/col,
  `Offset(line, col)` converts back, `SpanContext(offset)` classifies
  what construct is at a given position (import, OID ref, syntax, definition).
- **Object** - OBJECT-TYPE. Type, access, status, units, description, DEFVAL,
  index entries, AUGMENTS links. Effective values (hint, sizes, ranges, enums,
  bits) computed by walking type chains.
- **Type** - Type definition. Base type, parent chain, status, display hint,
  constraints, enums/bits. `Effective*()` methods walk parent chain.
- **Notification** - NOTIFICATION-TYPE or TRAP-TYPE. Object members, optional
  TrapInfo.
- **Group** - OBJECT-GROUP or NOTIFICATION-GROUP. Member nodes.
- **Compliance** - MODULE-COMPLIANCE. Module/group requirements.
- **Capability** - AGENT-CAPABILITIES. Module/object variations.
- **Symbol** - Union type over all entity types. Used by multi-type lookups.

### Key Enums

- `Kind` (11): Unknown, Internal, Node, Scalar, Table, Row, Column,
  Notification, Group, Compliance, Capability
- `Access` (7): NotAccessible, AccessibleForNotify, ReadOnly, ReadWrite,
  ReadCreate, WriteOnly, NotImplemented
- `Status` (5): Current, Deprecated, Obsolete, Mandatory (SMIv1),
  Optional (SMIv1)
- `BaseType` (13): Integer32, Unsigned32, Counter32, Counter64, Gauge32,
  TimeTicks, IpAddress, OctetString, ObjectIdentifier, Bits, Opaque, Sequence
- `Language` (3): Unknown, SMIv1, SMIv2

### Re-export Chain

```
internal/types/  ->  internal/model/enums.go  ->  mib/enums.go
```

The public `mib` package re-exports all types from `internal/model` via
aliases so external callers never import `internal/` packages.


## Synthetic Base Modules

Every MIB ultimately imports from a small set of base modules that define the
primitive types (INTEGER, Counter32, etc.) and root OID assignments (iso,
internet, enterprises, etc.). Rather than requiring these files to be present
on disk, gomib embeds them so the library works out of the box with no external
dependencies.

Seven base modules are built from embedded source files:

| Module | RFC | Content |
|--------|-----|---------|
| SNMPv2-SMI | 2578 | Base types (Integer32, Counter32, etc.), root OIDs (iso, internet, mgmt, mib-2, enterprises), macros |
| SNMPv2-TC | 2579 | Textual conventions (DisplayString, TruthValue, DateAndTime, etc.) |
| SNMPv2-CONF | 2580 | Conformance macros (OBJECT-GROUP, MODULE-COMPLIANCE, etc.) |
| RFC1155-SMI | 1155 | SMIv1 base types and OIDs |
| RFC1065-SMI | 1065 | Original SMIv1 base (predates RFC 1155) |
| RFC-1212 | 1212 | SMIv1 OBJECT-TYPE macro |
| RFC-1215 | 1215 | SMIv1 TRAP-TYPE macro |

These are always available without explicit source configuration. Each base
module has an explicit SMI version. A regular module's imports from any of
these base modules contribute language evidence alongside strong syntax;
conflicting or insufficient evidence leaves its language unknown.


## Dependency Graph Package

**Files:** `internal/graph/graph.go`

Used by the resolver's type and OID phases for topological ordering. Both
phases need to resolve definitions in dependency order (a type's parent must
be resolved before the type itself, an OID's parent node must exist before
its child can be placed).

- `Graph` maps `Symbol{Module, Name}` to node IDs with edge lists
- `ResolutionOrder()` returns topological order + detected cycles
- Uses Tarjan's SCC algorithm with deterministic iteration (sorted by
  module+name before DFS)
- Cycles (SCCs with >1 node or self-loops) are reported separately so the
  resolver can emit diagnostics for them


## Concurrency Model

- **Loading phase:** Parallel goroutines per module, bounded by
  `runtime.NumCPU()` semaphore. File-level caching prevents duplicate parsing.
- **Resolution phase:** Single-threaded. Cross-module graph traversals don't
  benefit from parallelism and would require complex synchronization.
- **Query phase:** The returned `*Mib` is immutable and safe for concurrent
  reads from multiple goroutines. OID computation uses `sync.Once` for
  thread-safe lazy initialization.


## CLI Tool

**Files:** `cmd/gomib/`

Commands: `load`, `lint`, `get`, `inspect`, `dump`, `normalize`, `trace`,
`paths`, `list`, `find`

Key commands for development:
- `gomib load IF-MIB` - parse/resolve, show counts and diagnostics
- `gomib get IF-MIB::ifIndex` - look up by name or OID
- `gomib lint --level 6 IF-MIB` - show all diagnostics
- `gomib trace IF-MIB ifIndex` - debug resolver decisions
- `gomib paths` - show discovered system MIB paths
