# Problem Corpus

Synthetic MIBs replicating real-world edge cases found via smilint analysis
of the primary corpus and documented gomib divergences. Each MIB targets a
specific category of problems. References point to real MIBs in the primary
corpus and to gomib source where the handling exists.

Tool behavior key:
- smilint [N] = libsmi flags at severity N (1=error, 2=warning, 3=info)
- net-snmp: silent = net-snmp silently accepts
- net-snmp: error = net-snmp reports an error
- gomib: TESTED = has test coverage, UNTESTED = no test coverage

Test file: resolve_problems_test.go

## MIB Index

- `PROBLEM-SMIv1v2-MIX-MIB.mib` - SMIv1 constructs in SMIv2 context
- `PROBLEM-IMPORTS-MIB.mib` - Import resolution edge cases
- `PROBLEM-IMPORTS-ALIAS-MIB.mib` - Uses aliased module names
- `PROBLEM-KEYWORDS-MIB.mib` - Reserved keywords as identifiers
- `PROBLEM-INDEX-MIB.mib` - Bare types and missing ranges in INDEX
- `PROBLEM-DEFVAL-MIB.mib` - DEFVAL syntax mismatches and edge cases
- `PROBLEM-NAMING-MIB.mib` - Naming convention violations
- `PROBLEM-NOTIFICATIONS-MIB.mib` - Notification varbind issues
- `PROBLEM-ACCESS-MIB.mib` - Access/status value edge cases
- `PROBLEM-HEXSTRINGS-MIB.mib` - Hex/binary string edge cases
- `PROBLEM-REVISIONS-MIB.mib` - Revision ordering violations

---

## PROBLEM-SMIv1v2-MIX-MIB

SMIv1 constructs used inside SMIv2-style modules. Very common in vendor MIBs
that were partially migrated from v1 to v2.

gomib test coverage: PARTIAL. divergences_test.go has status/access
equivalence helpers. No direct tests for TRAP-TYPE parsing, ACCESS keyword
switching, or Counter/Gauge type resolution.

net-snmp: silently accepts all constructs in this MIB.

- [ ] TRAP-TYPE macro in SMIv2 module
  - smilint [2]: "TRAP-TYPE macro is not allowed in SMIv2" (line 72)
  - smilint [2]: "macro TRAP-TYPE has not been imported from module RFC-1215" (line 72)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: misc/RADLAN-MIB.mib (931 occurrences), misc/IFT-SNMP-MIB.mib
- [ ] ACCESS keyword instead of MAX-ACCESS
  - smilint [2]: "ACCESS is SMIv1 style, use MAX-ACCESS in SMIv2 MIBs instead" (lines 32,43,53,93,100)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: misc/RADLAN-MIB.mib (477 occurrences), misc/IFT-SNMP-MIB.mib
- [ ] `mandatory` status in SMIv2 OBJECT-TYPE
  - smilint [2]: "invalid status mandatory in SMIv2 MIB" (lines 33,44,94,101)
  - net-snmp: silent
  - gomib: PARTIAL - divergences_test.go:187-196 has statusEquivalent() but no parse/load test
  - gomib: internal/resolver/types_phase.go:455-469 (status normalization)
  - Real: misc/RADLAN-MIB.mib, misc/IFT-SNMP-MIB.mib
- [ ] `optional` status in SMIv2 OBJECT-TYPE
  - smilint [2]: "invalid status optional in SMIv2 MIB" (line 54)
  - net-snmp: silent
  - gomib: PARTIAL - divergences_test.go has equivalence helper but no parse/load test
- [ ] SMIv1 types (Counter, Gauge) used without proper imports
  - smilint [2]: "type Counter/Gauge does not resolve to a known base type" (lines 91,98)
  - smilint [2]: "unknown type Counter/Gauge" (lines 93,100)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: ietf/RFC1213-MIB.mib (334 occurrences), ietf/RFC1271-MIB.mib

## PROBLEM-IMPORTS-MIB

Import resolution failures and partial imports.

gomib test coverage: TESTED for SMI base types (Counter64, Gauge32,
Unsigned32, TimeTicks). resolve_problems_test.go:TestProblemImports verifies
permissive fallback resolves missing imports to correct base types (grounded
against net-snmp). SKIP for TC types (DisplayString, TruthValue) where
permissive fallback does not cover SNMPv2-TC textual conventions.
load_test.go:213-309 tests missing `enterprises` import at
strict/normal/permissive levels.

net-snmp: silently accepts all constructs in this MIB (resolves base types
implicitly).

- [x] Missing Counter64 import
  - smilint [2]: "SMIv2 base type Counter64 must be imported from SNMPv2-SMI" (line 41)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemImports
  - Real: adva/F3-SYNC-MIB.mib (138 occurrences)
- [x] Missing Gauge32 import
  - smilint [2]: "SMIv2 base type Gauge32 must be imported from SNMPv2-SMI" (line 49)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemImports
  - Real: huawei/HUAWEI-DISMAN-PING-MIB.mib, misc/ECS4510-MIB.mib (90)
- [x] Missing Unsigned32 import
  - smilint [2]: "SMIv2 base type Unsigned32 must be imported from SNMPv2-SMI" (line 57)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemImports
- [x] Missing TimeTicks import
  - smilint [2]: "SMIv2 base type TimeTicks must be imported from SNMPv2-SMI" (line 65)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemImports
  - Real: adva/CM-SYSTEM-MIB.mib, adva/F3-SYNC-MIB.mib
- [ ] Missing DisplayString import (from SNMPv2-TC)
  - smilint [2]: "unknown type DisplayString" (line 73)
  - smilint [2]: "type [unknown] does not resolve to a known base type" (line 72)
  - net-snmp: silent
  - gomib: SKIP - TC fallback not implemented, resolve_problems_test.go documents divergence
  - Real: misc/RADLAN-MIB.mib (16 occurrences)
- [ ] Missing TruthValue import (from SNMPv2-TC)
  - smilint [2]: "unknown type TruthValue" (line 83)
  - smilint [2]: "type TruthValue does not resolve to a known base type" (line 81)
  - net-snmp: silent
  - gomib: SKIP - TC fallback not implemented, resolve_problems_test.go documents divergence
  - Real: misc/RADLAN-MIB.mib (55 occurrences)
- [ ] Macro names in import list (MODULE-IDENTITY, OBJECT-TYPE, etc.)
  - gomib: UNTESTED
  - gomib: internal/resolver/imports.go:286-295 (isMacroSymbol filter)
- [ ] Import forwarding - symbol re-exported through intermediate module
  - gomib: UNTESTED
  - gomib: internal/resolver/imports.go:133-156 (forwarding)

## PROBLEM-IMPORTS-ALIAS-MIB

Uses renamed/aliased module names that require alias table lookup.

gomib test coverage: NONE.

net-snmp: "Cannot find module (SNMPv2-SMI-v1)" and "Cannot find module
(SNMPv2-TC-v1)" - fails to resolve. Also flags current/MAX-ACCESS as
SMIv1 errors because it treats the module as SMIv1 due to failed imports.

smilint: treats as SMIv1 (since alias resolution fails), then flags
MAX-ACCESS and current as v2-in-v1 issues.

- [ ] SNMPv2-SMI-v1 alias for SNMPv2-SMI
  - smilint [2]: "MAX-ACCESS is SMIv2 style, use ACCESS in SMIv1 MIBs instead" (lines 31,38)
  - smilint [2]: "invalid status current in SMIv1 MIB" (lines 32,39)
  - net-snmp: "Cannot find module (SNMPv2-SMI-v1)" (line 10)
  - gomib: UNTESTED
  - gomib: internal/resolver/imports.go:298-310 (module aliases)
  - Real: misc/RADLAN-MIB.mib
- [ ] SNMPv2-TC-v1 alias for SNMPv2-TC
  - net-snmp: "Cannot find module (SNMPv2-TC-v1)" (line 12)
  - gomib: UNTESTED
  - gomib: internal/resolver/imports.go:298-310

## PROBLEM-KEYWORDS-MIB

Reserved ASN.1 keywords used as identifiers in object definitions or DEFVAL.

gomib test coverage: TESTED. resolve_problems_test.go:TestProblemKeywordDefvals
verifies all 7 keyword DEFVAL values (mandatory, optional, current, deprecated,
obsolete, true, false) resolve to correct enum labels. Grounded against
net-snmp which preserves keyword DEFVAL values as-is.
parser_test.go:317-330 tests reserved keyword as module name.

net-snmp: silently accepts all keyword-as-DEFVAL constructs.
smilint: also silently accepts (no diagnostics beyond conformance noise).

- [x] DEFVAL with keyword value `mandatory`
  - smilint: silent
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemKeywordDefvals
  - gomib: internal/parser/parser.go parseDefValContent()
  - Real: cisco/CISCO-DOT11-IF-MIB (cd11IfVlanWepEncryptOptions DEFVAL { mandatory })
- [x] DEFVAL with keyword value `optional`
  - smilint: silent
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemKeywordDefvals
- [x] DEFVAL with keyword value `current`, `deprecated`, `obsolete`
  - smilint: silent
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemKeywordDefvals
- [x] DEFVAL with `true` / `false` keywords
  - smilint: silent
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemKeywordDefvals

## PROBLEM-INDEX-MIB

Bare type names and other edge cases in INDEX clauses.

gomib test coverage: NONE. resolve_tables_test.go tests standard indexes
from fixture MIBs only (IF-MIB, SNMPv2-MIB, IP-MIB, ENTITY-MIB, BRIDGE-MIB).

- [ ] Bare INTEGER in INDEX
  - smilint [1]: syntax error (line 41) - rejects entirely
  - net-snmp: silent
  - gomib: UNTESTED
  - gomib: internal/resolver/semantics.go:526-533 (isBareTypeIndex)
- [ ] Index element missing range restriction
  - smilint [2]: "index element must have a range restriction" (lines 165,173)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: adva/CM-FACILITY-MIB.mib (438 occurrences)
- [ ] MacAddress as index type
  - smilint: silent (accepted)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: misc/SIXNET-MIB.mib (8 instances)
- [ ] DisplayString as index
  - smilint: silent (accepted with SIZE)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: misc/RADLAN-MIB.mib

## PROBLEM-DEFVAL-MIB

DEFVAL syntax mismatches, OID references, hex edge cases.

gomib test coverage: MINIMAL. resolve_defval_test.go:10-48 compares fixture
MIBs against net-snmp output. divergences_test.go:204-249 has equivalence
helpers for zeroDotZero and hex zeros. parser_test.go:177-197 tests basic
integer/string DEFVAL parsing. No tests for bad enum labels, binary strings,
empty BITS, or type mismatches.

net-snmp: silently accepts all constructs in this MIB.

- [ ] DEFVAL with OID reference (zeroDotZero)
  - smilint: silent
  - net-snmp: silent
  - gomib: PARTIAL - divergences_test.go:219 has equivalence check, no direct parse test
  - gomib: internal/resolver/semantics.go:377-451
  - gomib normalizes to numeric "0.0"; net-snmp keeps symbolic
- [ ] DEFVAL with raw integer for enum type
  - smilint: silent (integer 5 accepted for ProblemSeverity)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: alcatel/TIMETRA-BGP-MIB.mib (78 occurrences across 10 MIBs)
- [ ] DEFVAL with undefined enum label
  - smilint [2]: "default value syntax does not match object syntax" (line 66)
  - smilint [2]: "default value does not match underlying enumeration type" (line 66)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: alcatel/TIMETRA-FILTER-MIB.mib (29 occurrences)
- [ ] Large hex DEFVAL (16 bytes)
  - smilint: silent
  - net-snmp: silent
  - gomib: PARTIAL - divergences_test.go:215 has hex zero equivalence, no direct test
  - gomib preserves full hex; net-snmp truncates zeros to "0"
- [ ] Binary string DEFVAL (8 bits)
  - smilint: silent
  - net-snmp: silent
  - gomib: UNTESTED
- [ ] Binary string not multiple of 8 bits
  - smilint [2]: "length of binary string 101 is not a multiple of 8" (line 102)
  - net-snmp: silent
  - gomib: UNTESTED - pads to multiple of 8 with leading zeros
- [ ] Empty BITS DEFVAL
  - smilint: silent
  - net-snmp: silent
  - gomib: UNTESTED
- [ ] Negative integer DEFVAL
  - smilint: silent
  - net-snmp: silent
  - gomib: UNTESTED

## PROBLEM-NAMING-MIB

Identifier naming convention violations found in vendor MIBs.

gomib test coverage: GOOD for underscores/hyphens/length. NONE for uppercase
identifiers. load_test.go:91-183 tests UNDERSCORE-TEST-MIB,
HYPHEN-END-TEST-MIB, LONG-IDENT-TEST-MIB. parser_test.go:251-315 tests
parser-level diagnostics for the same.

- [ ] Uppercase initial letter on object identifier
  - smilint [2]: "should start with a lower case letter" (lines 30,37,42,47)
  - smilint [1]: syntax error (lines 30,37,42,47) - rejects the declarations
  - net-snmp: silent (accepts uppercase identifiers)
  - gomib: UNTESTED
  - Real: huawei/HUAWEI-MIB.mib (92+ occurrences)
  - gomib: internal/parser/parser.go:460 (lenient in normal mode)
- [ ] Hyphens in SMIv2 object identifier names
  - smilint [5]: "object identifier name should not include hyphens in SMIv2 MIB" (lines 37,42,47)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: huawei/HUAWEI-MIB.mib (S6730-H28Y4C, S5735-L8T4X-A1)

## PROBLEM-NOTIFICATIONS-MIB

Notification varbind and group membership issues.

gomib test coverage: TESTED. resolve_problems_test.go:TestProblemNotifications
verifies normal notification varbinds, not-accessible object inclusion
(matches net-snmp), and undefined varbind exclusion (documents divergence
from net-snmp which preserves undefined names).
resolve_notifications_test.go:12-59 validates varbinds from fixture MIBs.

net-snmp: silently accepts all constructs (including undefined varbind and
not-accessible references).

- [x] not-accessible object referenced in NOTIFICATION-TYPE OBJECTS
  - smilint [3]: "object problemNotifIndex of notification must not be not-accessible" (line 79)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemNotifications/not-accessible_in_OBJECTS
  - Real: adva/CM-PERFORMANCE-MIB.mib, huawei/HUAWEI-ENTITY-EXTENT-MIB.mib (842 occurrences)
- [x] Undefined object referenced in NOTIFICATION-TYPE OBJECTS
  - smilint [1]: "unknown object identifier label problemUndefinedVarbind" (line 92)
  - smilint [3]: "object must be a scalar or column" (line 91)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemNotifications/undefined_varbind
  - gomib: excludes unresolved; net-snmp preserves as strings (documented divergence)
  - Real: misc/ES4552BH2-MIB.mib (trapVarLoginInetAddressTypes - typo)
- [ ] not-accessible index element in OBJECT-GROUP
  - smilint [3]: "node problemNotifIndex is an invalid member of group" (line 108)
  - net-snmp: silent
  - gomib: UNTESTED
  - Real: adva/CM-FACILITY-MIB.mib (ocnStmIndex, stsVcPathIndex)

## PROBLEM-ACCESS-MIB

Access and status value edge cases.

gomib test coverage: TESTED. resolve_problems_test.go:TestProblemAccess verifies
scalar read-create, write-only, and column access values match net-snmp.
resolve_access_test.go:9-77 tests all access and status values against fixtures.
divergences_test.go:171-196 has equivalence helpers.

net-snmp: silently accepts all constructs in this MIB, including scalar
read-create and write-only.

- [x] Scalar object with read-create access (invalid per RFC)
  - smilint [3]: "scalar object must not have a read-create access value" (lines 35,43)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemAccess/scalar_read-create
  - Real: alcatel/TIMETRA-VRTR-MIB.mib, alcatel/TIMETRA-SERV-MIB.mib (35 occurrences)
- [x] write-only access in SMIv2 (deprecated)
  - smilint [2]: "access write-only is no longer allowed in SMIv2" (line 52)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemAccess/write-only
  - Real: misc/SIXNET-MIB.mib
- [x] SMIv1 read-write vs SMIv2 read-create equivalence
  - smilint: silent (no diagnostic for read-write on column)
  - net-snmp: silent
  - gomib: TESTED - divergences_test.go:171-182, resolve_access_test.go:9-40, resolve_problems_test.go:TestProblemAccess/table_column_access_equivalence
  - gomib: internal/resolver/semantics.go:507-523

## PROBLEM-HEXSTRINGS-MIB

Hex and binary string parsing edge cases.

gomib test coverage: TESTED. resolve_problems_test.go:TestProblemHexStrings
verifies DEFVAL String() output matches net-snmp ground truth (decimal format
for <=8 bytes). TestProblemHexStringBytes verifies raw []byte values from
hexToBytes() and binaryToBytes() conversion.

net-snmp: silently accepts all constructs in this MIB (odd hex, empty hex,
binary strings).

- [x] Odd-length hex string (7 chars)
  - smilint [2]: "length of hexadecimal string ABCDEF0 is not a multiple of 2" (line 31)
  - net-snmp: DEFVAL { 180150000 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
  - Real: misc/ECS4510-MIB.mib, misc/ES4552BH2-MIB.mib
- [x] Odd-length hex string (3 chars: FFF)
  - smilint [2]: "length of hexadecimal string FFF is not a multiple of 2" (line 41)
  - net-snmp: DEFVAL { 4095 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
  - Real: misc/RAPID-CITY.mib
- [x] Odd-length hex string (1 char: 0)
  - smilint [2]: "length of hexadecimal string 0 is not a multiple of 2" (line 51)
  - net-snmp: DEFVAL { 0 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
  - Real: misc/ECS4510-MIB.mib
- [x] Long hex string (128 chars)
  - smilint: silent
  - net-snmp: DEFVAL { 0 } (truncates leading zeros)
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings (known divergence: gomib preserves full hex)
  - Real: alcatel/ALCATEL-IND1-TIMETRA-PORT-MIB.mib (600 chars)
- [x] Empty hex string
  - smilint: silent
  - net-snmp: DEFVAL { 0 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
- [x] Binary string (8 bits, exact)
  - smilint: silent
  - net-snmp: DEFVAL { 240 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
- [x] Binary string not multiple of 8 (5 bits)
  - smilint [2]: "length of binary string 10101 is not a multiple of 8" (line 92)
  - net-snmp: DEFVAL { 21 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
- [x] Binary string not multiple of 8 (12 bits)
  - smilint [2]: "length of binary string 101010101010 is not a multiple of 8" (line 102)
  - net-snmp: DEFVAL { 2730 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
- [x] Lowercase hex characters
  - smilint: silent
  - net-snmp: DEFVAL { 3735928559 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes
- [x] All-zeros hex
  - smilint: silent
  - net-snmp: DEFVAL { 0 }
  - gomib: TESTED - resolve_problems_test.go:TestProblemHexStrings, TestProblemHexStringBytes

## PROBLEM-REVISIONS-MIB

MODULE-IDENTITY revision ordering and placement issues.

gomib test coverage: TESTED. resolve_problems_test.go:TestProblemRevisions
verifies pre-identity object OID resolution, post-identity object OID
resolution, MODULE-IDENTITY OID, and revision count. Grounded against
net-snmp OID output. load_test.go:185-209 tests missing MODULE-IDENTITY.

net-snmp: silently accepts all constructs (out-of-order revisions,
pre-identity declarations, missing revision for LAST-UPDATED).

- [x] Revisions not in reverse chronological order
  - smilint [3]: "revision not in reverse chronological order" (line 31)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemRevisions/revisions_are_parsed
  - Real: 15 MIBs (99 occurrences) across multiple vendors
- [ ] Missing revision for LAST-UPDATED date
  - smilint [3]: "revision for last update is missing" (line 41)
  - net-snmp: silent
  - gomib: UNTESTED (no diagnostic for this)
  - Real: 24 MIBs
- [x] MODULE-IDENTITY not first declaration
  - smilint [3]: "MODULE-IDENTITY clause must be the first declaration in a module" (line 19)
  - net-snmp: silent
  - gomib: TESTED - resolve_problems_test.go:TestProblemRevisions/pre-identity_object_resolves
  - Real: cisco/CISCOSB-MIB.mib, ieee/IEEE802dot11-MIB.mib (7 occurrences)
