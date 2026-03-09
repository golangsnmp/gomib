package gomib

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

var problemCorpusEnterpriseAllocations = map[string]string{
	"99990":    "PROBLEM-MULTIMOD-BASE-MIB",
	"99997":    "PROBLEM-P3-IMPORTS-MIB",
	"99998.1":  "PROBLEM-SMIv1v2-MIX-MIB",
	"99998.2":  "PROBLEM-IMPORTS-MIB",
	"99998.3":  "PROBLEM-IMPORTS-ALIAS-MIB",
	"99998.4":  "PROBLEM-KEYWORDS-MIB",
	"99998.5":  "PROBLEM-INDEX-MIB",
	"99998.6":  "PROBLEM-DEFVAL-MIB",
	"99998.7":  "PROBLEM-NAMING-MIB",
	"99998.8":  "PROBLEM-NOTIFICATIONS-MIB",
	"99998.9":  "PROBLEM-ACCESS-MIB",
	"99998.10": "PROBLEM-HEXSTRINGS-MIB",
	"99998.11": "PROBLEM-REVISIONS-MIB",
	"99998.20": "PROBLEM-FORWARDING-SOURCE-MIB",
	"99998.21": "PROBLEM-FORWARDING-RELAY-MIB",
	"99998.22": "PROBLEM-FORWARDING-MIB",
	"99998.23": "PROBLEM-TYPECHAINS-MIB",
	"99998.24": "PROBLEM-SEMANTICS-MIB",
	"99998.25": "PROBLEM-DIAGNOSTICS-MIB",
	"99998.26": "PROBLEM-SHADOWING-MIB",
	"99998.30": "PROBLEM-SHARED-OIDS-BASE-MIB",
	"99998.31": "PROBLEM-SHARED-OIDS-MIB",
	"99998.32": "PROBLEM-CASEFOLDING-MIB",
	"99998.33": "PROBLEM-ENUM-SUBTYPE-MIB",
	"99998.34": "PROBLEM-RANGES-MIB",
	"99998.35": "PROBLEM-ENUM-BITS-DUPES-MIB",
	"99998.36": "PROBLEM-SHADOWING-BASE-MIB",
	"99998.37": "PROBLEM-GROUP-MEMBERSHIP-MIB",
	"99998.38": "PROBLEM-PARTIAL-IMPORTS-MIB",
	"99998.39": "PROBLEM-STRICT-METADATA-MIB",
	"99998.40": "PROBLEM-SEMANTIC-GLOBAL-SOURCE-MIB",
	"99998.41": "PROBLEM-SEMANTIC-GLOBAL-MIB",
	"99999":    "PROBLEM-DUPLICATE-IMPORT-MIB",
}

var problemEnterpriseRootPattern = regexp.MustCompile(`::=\s*\{\s*enterprises\s+(\d+)(?:\s+(\d+))?\s*\}`)

func TestProblemCorpusEnterpriseAllocations(t *testing.T) {
	entries, err := os.ReadDir(testutil.ProblemsCorpusDir())
	if err != nil {
		t.Fatalf("ReadDir(problems corpus) failed: %v", err)
	}

	actual := make(map[string]string, len(problemCorpusEnterpriseAllocations))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mib") {
			continue
		}

		path := testutil.ProblemsCorpusDir() + "/" + entry.Name()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) failed: %v", entry.Name(), err)
		}

		matches := problemEnterpriseRootPattern.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			key := m[1]
			if m[2] != "" {
				key += "." + m[2]
			}

			module, ok := problemCorpusEnterpriseAllocations[key]
			if !ok {
				t.Fatalf("%s uses unreserved enterprises root %s", entry.Name(), key)
			}

			if prior, exists := actual[key]; exists {
				t.Fatalf("duplicate enterprises root %s claimed by %s and %s", key, prior, module)
			}
			actual[key] = module
		}
	}

	if len(actual) != len(problemCorpusEnterpriseAllocations) {
		var missing []string
		for key, module := range problemCorpusEnterpriseAllocations {
			if _, ok := actual[key]; !ok {
				missing = append(missing, key+"="+module)
			}
		}
		slices.Sort(missing)
		t.Fatalf("problem corpus allocation table is stale; missing: %v", missing)
	}
}
