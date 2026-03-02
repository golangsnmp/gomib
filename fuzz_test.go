package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// allBaseTypes enumerates all BaseType values for ObjectsByBaseType fuzzing.
var allBaseTypes = []mib.BaseType{
	mib.BaseInteger32, mib.BaseUnsigned32, mib.BaseCounter32,
	mib.BaseCounter64, mib.BaseGauge32, mib.BaseTimeTicks,
	mib.BaseIpAddress, mib.BaseOctetString, mib.BaseObjectIdentifier,
	mib.BaseBits, mib.BaseOpaque, mib.BaseSequence,
}

// pipelineSeeds are shared seed inputs for pipeline fuzz tests.
var pipelineSeeds = [][]byte{
	[]byte("EXAMPLE-MIB DEFINITIONS ::= BEGIN\nEND"),
	[]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			MODULE-IDENTITY, OBJECT-TYPE, Integer32 FROM SNMPv2-SMI
			DisplayString FROM SNMPv2-TC;
		testMIB MODULE-IDENTITY
			LAST-UPDATED "200601010000Z"
			ORGANIZATION "Test"
			CONTACT-INFO "test"
			DESCRIPTION "test"
			::= { enterprises 99999 }
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "test"
			::= { testMIB 1 }
		END`),
	[]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			MODULE-IDENTITY, OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
		testTable OBJECT-TYPE
			SYNTAX SEQUENCE OF TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "table"
			::= { testMIB 1 }
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "row"
			INDEX { testIndex }
			::= { testTable 1 }
		TestEntry ::= SEQUENCE {
			testIndex Integer32
		}
		testIndex OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "index"
			::= { testEntry 1 }
		END`),
	[]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			OBJECT-TYPE, Counter32, enterprises FROM SNMPv2-SMI
			TEXTUAL-CONVENTION, DisplayString FROM SNMPv2-TC;
		MyString ::= TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION ""
			SYNTAX OCTET STRING (SIZE (0..255))
		testObj OBJECT-TYPE
			SYNTAX MyString
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "test"
			DEFVAL { "hello" }
			::= { enterprises 99999 1 }
		END`),
	[]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			TRAP-TYPE FROM RFC-1215
			enterprises FROM RFC1155-SMI;
		testTrap TRAP-TYPE
			ENTERPRISE enterprises
			DESCRIPTION "trap"
			::= 1
		END`),
	[]byte(""),
	[]byte("not a mib at all"),
	[]byte("TEST DEFINITIONS ::= BEGIN"),
}

// FuzzLower fuzzes the parse and lowering stages without resolution.
// This isolates lowering-specific issues and runs faster than FuzzPipeline
// since it skips the resolver.
func FuzzLower(f *testing.F) {
	for _, seed := range pipelineSeeds {
		f.Add(seed)
	}
	testutil.AddProblemCorpusSeeds(f)
	testutil.AddPrimaryCorpusSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip pathologically large inputs to avoid excessive memory use.
		if len(data) > 1<<20 {
			return
		}

		cfg := types.PermissiveConfig()

		p := parser.New(data, nil, cfg)
		astMods := p.ParseModule()
		if len(astMods) == 0 {
			return
		}
		astMod := astMods[0]

		mod := module.Lower(astMod, data, nil, cfg)
		if mod == nil {
			return
		}

		// Module name should match AST.
		if mod.Name == "" && astMod.Name.Name != "" {
			t.Fatal("lowered module lost its name")
		}

		// All diagnostics must have codes.
		for _, d := range mod.Diagnostics {
			if d.Code == "" {
				t.Fatal("lowered module diagnostic has empty Code")
			}
		}

		// All definitions must have valid spans (Start <= End).
		for _, def := range mod.Definitions {
			span := def.DefinitionSpan()
			if span.Start > span.End {
				t.Fatalf("definition %q: span start %d > end %d",
					def.DefinitionName(), span.Start, span.End)
			}
		}

		// Module span should be valid.
		if mod.Span.Start > mod.Span.End {
			t.Fatalf("module span start %d > end %d", mod.Span.Start, mod.Span.End)
		}
	})
}

// FuzzPipeline fuzzes the full processing pipeline: parse -> lower -> resolve.
// This catches issues that only manifest when phases interact.
func FuzzPipeline(f *testing.F) {
	for _, seed := range pipelineSeeds {
		f.Add(seed)
	}
	testutil.AddProblemCorpusSeeds(f)
	testutil.AddPrimaryCorpusSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip pathologically large inputs to avoid excessive memory use.
		if len(data) > 1<<20 {
			return
		}

		cfg := types.PermissiveConfig()

		p := parser.New(data, nil, cfg)
		astMods := p.ParseModule()
		if len(astMods) == 0 {
			return
		}

		mod := module.Lower(astMods[0], data, nil, cfg)
		if mod == nil {
			return
		}

		resolverCfg := mib.PermissiveConfig()
		m := mib.Resolve([]*module.Module{mod}, nil, &resolverCfg)
		if m == nil {
			t.Fatal("Resolve returned nil")
		}

		// Exercise all collection accessors.
		_ = m.Modules()
		_ = m.Objects()
		_ = m.Types()
		_ = m.Notifications()
		_ = m.Groups()
		_ = m.Compliances()
		_ = m.Capabilities()
		_ = m.Diagnostics()
		_ = m.Unresolved()
		_ = m.Tables()
		_ = m.Scalars()
		_ = m.Columns()
		_ = m.Rows()

		// All diagnostics must have codes.
		for _, d := range m.Diagnostics() {
			if d.Code == "" {
				t.Fatal("resolved diagnostic has empty Code")
			}
		}

		// Validate node tree invariants.
		root := m.Root()
		if root == nil {
			t.Fatal("Root() returned nil")
		}
		if !root.IsRoot() {
			t.Fatal("Root() is not IsRoot()")
		}
		if root.Parent() != nil {
			t.Fatal("Root() has non-nil Parent")
		}
		if root.OID() != nil {
			t.Fatal("Root() has non-nil OID")
		}

		// Walk all nodes and check parent-child consistency.
		for nd := range m.Nodes() {
			oid := nd.OID()
			parent := nd.Parent()

			// Every non-root node must have a parent.
			if parent == nil {
				t.Fatalf("non-root node %q has nil parent", nd.Name())
			}

			// Parent's child at our arc must be us.
			if parent.Child(nd.Arc()) != nd {
				t.Fatalf("parent-child inconsistency: parent.Child(%d) != node %q",
					nd.Arc(), nd.Name())
			}

			// OID should have at least one arc.
			if len(oid) == 0 {
				t.Fatalf("non-root node %q has empty OID", nd.Name())
			}

			// OID's last arc should match node's arc.
			if oid.LastArc() != nd.Arc() {
				t.Fatalf("node %q: OID last arc %d != node arc %d",
					nd.Name(), oid.LastArc(), nd.Arc())
			}

			// If node has an object, check Object<->Node binding.
			if obj := nd.Object(); obj != nil {
				if obj.Node() != nd {
					t.Fatalf("object %q: Node() does not point back to its node", obj.Name())
				}
				objOID := obj.OID()
				if objOID != nil && !objOID.Equal(oid) {
					t.Fatalf("object %q: OID %v != node OID %v", obj.Name(), objOID, oid)
				}
			}

			// Exercise Children() to catch panics.
			_ = nd.Children()
		}

		// Validate type chain acyclicity.
		for _, typ := range m.Types() {
			visited := make(map[*mib.Type]bool)
			for cur := typ; cur != nil; cur = cur.Parent() {
				if visited[cur] {
					t.Fatalf("type chain cycle detected at %q", cur.Name())
				}
				visited[cur] = true
				// Bound the walk to catch unreasonably long chains.
				if len(visited) > 100 {
					t.Fatalf("type chain for %q exceeds 100 links", typ.Name())
				}
			}

			// EffectiveBase must match walking the chain manually.
			eb := typ.EffectiveBase()
			if typ.Base() != 0 && eb != typ.Base() {
				t.Fatalf("type %q: EffectiveBase %v != Base %v", typ.Name(), eb, typ.Base())
			}
		}

		// Validate OID lookup consistency.
		for nd := range m.Nodes() {
			oid := nd.OID()
			if oid == nil {
				continue
			}
			found := m.NodeByOID(oid)
			if found != nd {
				t.Fatalf("NodeByOID(%v) returned different node for %q", oid, nd.Name())
			}
			prefix := m.LongestPrefixByOID(oid)
			if prefix == nil {
				t.Fatalf("LongestPrefixByOID(%v) returned nil for existing node %q", oid, nd.Name())
			}
		}

		// Validate kind-specific object accessors.
		for _, obj := range m.Tables() {
			if obj.Kind() != mib.KindTable {
				t.Fatalf("Tables() returned object %q with kind %v", obj.Name(), obj.Kind())
			}
		}
		for _, obj := range m.Scalars() {
			if obj.Kind() != mib.KindScalar {
				t.Fatalf("Scalars() returned object %q with kind %v", obj.Name(), obj.Kind())
			}
		}
		for _, obj := range m.Columns() {
			if obj.Kind() != mib.KindColumn {
				t.Fatalf("Columns() returned object %q with kind %v", obj.Name(), obj.Kind())
			}
		}
		for _, obj := range m.Rows() {
			if obj.Kind() != mib.KindRow {
				t.Fatalf("Rows() returned object %q with kind %v", obj.Name(), obj.Kind())
			}
		}

		// Exercise deep object accessors to catch nil-pointer panics.
		for _, obj := range m.Objects() {
			_ = obj.String()
			_ = obj.Access()
			_ = obj.Status()
			_ = obj.Description()
			_ = obj.Reference()
			_ = obj.Units()
			_ = obj.IsTable()
			_ = obj.IsRow()
			_ = obj.IsColumn()
			_ = obj.IsScalar()
			_ = obj.Table()
			_ = obj.Row()
			_ = obj.Entry()
			_ = obj.Columns()
			_ = obj.EffectiveIndexes()
			_ = obj.EffectiveDisplayHint()
			_ = obj.EffectiveSizes()
			_ = obj.EffectiveRanges()
			_ = obj.EffectiveEnums()
			_ = obj.EffectiveBits()
			_ = obj.Index()
			_ = obj.Augments()

			dv := obj.DefaultValue()
			_ = dv.Kind()
			_ = dv.Kind().String()
			_ = dv.Value()
			_ = dv.Raw()
			_ = dv.String()
			_ = dv.IsZero()

			if typ := obj.Type(); typ != nil {
				_ = typ.String()
				_ = typ.IsCounter()
				_ = typ.IsGauge()
				_ = typ.IsString()
				_ = typ.IsEnumeration()
				_ = typ.IsBits()
				_ = typ.IsTextualConvention()
			}

			// Exercise named lookups using the object's name.
			_ = m.Object(obj.Name())
			_ = m.Node(obj.Name())
		}

		// Exercise Type chain methods.
		for _, typ := range m.Types() {
			_ = typ.String()
			_ = typ.EffectiveDisplayHint()
			_ = typ.EffectiveSizes()
			_ = typ.EffectiveRanges()
			_ = typ.EffectiveEnums()
			_ = typ.EffectiveBits()
			_ = typ.IsCounter()
			_ = typ.IsGauge()
			_ = typ.IsString()
			_ = typ.IsEnumeration()
			_ = typ.IsBits()
			_ = typ.IsTextualConvention()
			_ = typ.DisplayHint()
			_ = typ.Description()
			_ = typ.Reference()
			_ = typ.Sizes()
			_ = typ.Ranges()
			_ = typ.Enums()
			_ = typ.Bits()
			_ = typ.Status()

			// Exercise named lookups.
			_ = m.Type(typ.Name())
		}

		// Exercise Notification accessors.
		for _, notif := range m.Notifications() {
			_ = notif.String()
			_ = notif.Name()
			_ = notif.OID()
			_ = notif.Status()
			_ = notif.Description()
			_ = notif.Reference()
			_ = notif.Objects()
			_ = notif.TrapInfo()
			_ = m.Notification(notif.Name())
		}

		// Exercise Group accessors.
		for _, grp := range m.Groups() {
			_ = grp.String()
			_ = grp.Name()
			_ = grp.OID()
			_ = grp.Status()
			_ = grp.Description()
			_ = grp.Reference()
			_ = grp.Members()
			_ = grp.IsNotificationGroup()
			_ = m.Group(grp.Name())
		}

		// Exercise Compliance accessors.
		for _, comp := range m.Compliances() {
			_ = comp.String()
			_ = comp.Name()
			_ = comp.OID()
			_ = comp.Status()
			_ = comp.Description()
			_ = comp.Reference()
			_ = comp.Modules()
			_ = m.Compliance(comp.Name())
		}

		// Exercise Capability accessors.
		for _, cap := range m.Capabilities() {
			_ = cap.String()
			_ = cap.Name()
			_ = cap.OID()
			_ = cap.Status()
			_ = cap.Description()
			_ = cap.Reference()
			_ = cap.ProductRelease()
			_ = cap.Supports()
			_ = m.Capability(cap.Name())
		}

		// Exercise Module-level accessors.
		for _, mod := range m.Modules() {
			_ = mod.Language()
			_ = mod.SourcePath()
			_ = mod.OID()
			_ = mod.Organization()
			_ = mod.ContactInfo()
			_ = mod.Description()
			_ = mod.LastUpdated()
			_ = mod.Revisions()
			_ = mod.Imports()
			_ = mod.Tables()
			_ = mod.Scalars()
			_ = mod.Columns()
			_ = mod.Rows()
			_ = mod.Nodes()
			_ = mod.Objects()
			_ = mod.Types()
			_ = mod.Notifications()
			_ = mod.Groups()
			_ = mod.Compliances()
			_ = mod.Capabilities()
		}

		// Exercise FormatOID on every node.
		for nd := range m.Nodes() {
			oid := nd.OID()
			if oid != nil {
				_ = m.FormatOID(oid)
			}
			// Exercise Subtree iterator.
			for sub := range nd.Subtree() {
				_ = sub.Name()
			}
		}

		// Exercise ObjectsByType and ObjectsByBaseType.
		for _, typ := range m.Types() {
			_ = m.ObjectsByType(typ.Name())
		}
		for _, bt := range allBaseTypes {
			_ = m.ObjectsByBaseType(bt)
		}

		// Exercise HasErrors and NodeCount.
		_ = m.HasErrors()
		_ = m.NodeCount()
	})
}

// FuzzMultiModule fuzzes the resolver with two interacting modules.
// This catches cross-module issues (import resolution, OID references,
// type resolution across module boundaries) that FuzzPipeline misses
// since it only tests single-module resolution.
func FuzzMultiModule(f *testing.F) {
	// Module B defines base objects, module A imports from B.
	f.Add(
		[]byte(`A-MIB DEFINITIONS ::= BEGIN
			IMPORTS testObj FROM B-MIB;
			END`),
		[]byte(`B-MIB DEFINITIONS ::= BEGIN
			IMPORTS OBJECT-TYPE, Integer32, enterprises FROM SNMPv2-SMI;
			testObj OBJECT-TYPE
				SYNTAX Integer32
				MAX-ACCESS read-only
				STATUS current
				DESCRIPTION ""
				::= { enterprises 1 1 }
			END`),
	)
	// Module A imports a textual convention from B.
	f.Add(
		[]byte(`A-MIB DEFINITIONS ::= BEGIN
			IMPORTS OBJECT-TYPE, Integer32, enterprises FROM SNMPv2-SMI
				MyString FROM B-MIB;
			aObj OBJECT-TYPE
				SYNTAX MyString
				MAX-ACCESS read-only
				STATUS current
				DESCRIPTION ""
				::= { enterprises 2 1 }
			END`),
		[]byte(`B-MIB DEFINITIONS ::= BEGIN
			IMPORTS TEXTUAL-CONVENTION FROM SNMPv2-TC;
			MyString ::= TEXTUAL-CONVENTION
				STATUS current
				DESCRIPTION ""
				SYNTAX OCTET STRING (SIZE (0..255))
			END`),
	)
	// Both modules defining OIDs in the same subtree.
	f.Add(
		[]byte(`A-MIB DEFINITIONS ::= BEGIN
			IMPORTS MODULE-IDENTITY, enterprises FROM SNMPv2-SMI;
			aMIB MODULE-IDENTITY
				LAST-UPDATED "200601010000Z"
				ORGANIZATION ""
				CONTACT-INFO ""
				DESCRIPTION ""
				::= { enterprises 1 }
			END`),
		[]byte(`B-MIB DEFINITIONS ::= BEGIN
			IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises FROM SNMPv2-SMI;
			bMIB MODULE-IDENTITY
				LAST-UPDATED "200601010000Z"
				ORGANIZATION ""
				CONTACT-INFO ""
				DESCRIPTION ""
				::= { enterprises 2 }
			bObj OBJECT-TYPE
				SYNTAX Integer32
				MAX-ACCESS read-only
				STATUS current
				DESCRIPTION ""
				::= { bMIB 1 }
			END`),
	)
	// Empty modules.
	f.Add(
		[]byte("A-MIB DEFINITIONS ::= BEGIN\nEND"),
		[]byte("B-MIB DEFINITIONS ::= BEGIN\nEND"),
	)
	// One garbage, one valid.
	f.Add(
		[]byte("not a module"),
		[]byte("B-MIB DEFINITIONS ::= BEGIN\nEND"),
	)

	f.Fuzz(func(t *testing.T, dataA, dataB []byte) {
		if len(dataA) > 1<<20 || len(dataB) > 1<<20 {
			return
		}

		cfg := types.PermissiveConfig()

		var mods []*module.Module
		for _, data := range [][]byte{dataA, dataB} {
			p := parser.New(data, nil, cfg)
			astMods := p.ParseModule()
			if len(astMods) == 0 {
				continue
			}
			mod := module.Lower(astMods[0], data, nil, cfg)
			if mod != nil {
				mods = append(mods, mod)
			}
		}
		if len(mods) == 0 {
			return
		}

		resolverCfg := mib.PermissiveConfig()
		m := mib.Resolve(mods, nil, &resolverCfg)
		if m == nil {
			t.Fatal("Resolve returned nil for multi-module input")
		}

		// All diagnostics must have codes.
		for _, d := range m.Diagnostics() {
			if d.Code == "" {
				t.Fatal("multi-module diagnostic has empty Code")
			}
		}

		// Root invariants.
		root := m.Root()
		if root == nil {
			t.Fatal("Root() returned nil")
		}
		if !root.IsRoot() {
			t.Fatal("Root() is not IsRoot()")
		}

		// Walk nodes - check parent-child consistency.
		for nd := range m.Nodes() {
			parent := nd.Parent()
			if parent == nil {
				t.Fatalf("non-root node %q has nil parent", nd.Name())
			}
			if parent.Child(nd.Arc()) != nd {
				t.Fatalf("parent-child inconsistency at node %q", nd.Name())
			}
			oid := nd.OID()
			if len(oid) == 0 {
				t.Fatalf("non-root node %q has empty OID", nd.Name())
			}

			// Validate OID lookups.
			if found := m.NodeByOID(oid); found != nd {
				t.Fatalf("NodeByOID mismatch for %q", nd.Name())
			}

			// Exercise FormatOID.
			_ = m.FormatOID(oid)

			// Object-node binding.
			if obj := nd.Object(); obj != nil {
				if obj.Node() != nd {
					t.Fatalf("object %q: Node() does not point back", obj.Name())
				}
				_ = obj.String()
				_ = obj.Table()
				_ = obj.Row()
				_ = obj.Columns()
				_ = obj.EffectiveIndexes()
			}
		}

		// Validate cross-module type chain acyclicity.
		for _, typ := range m.Types() {
			visited := make(map[*mib.Type]bool)
			for cur := typ; cur != nil; cur = cur.Parent() {
				if visited[cur] {
					t.Fatalf("type chain cycle at %q", cur.Name())
				}
				visited[cur] = true
				if len(visited) > 100 {
					t.Fatalf("type chain for %q exceeds 100 links", typ.Name())
				}
			}
		}
	})
}
