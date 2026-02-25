FIXTURE_MODULES = IF-MIB SNMPv2-MIB IP-MIB ENTITY-MIB BRIDGE-MIB
FIXTURE_DIR = testdata/fixtures/netsnmp
CORPUS_PATH = testdata/corpus/primary

.PHONY: fixtures gomib-netsnmp test lint fuzz

gomib-netsnmp:
	CGO_ENABLED=1 go build -tags cgo -o gomib-netsnmp ./cmd/gomib-netsnmp

fixtures: gomib-netsnmp
	@mkdir -p $(FIXTURE_DIR)
	./gomib-netsnmp fixturegen -p $(CORPUS_PATH) -dir $(FIXTURE_DIR) $(FIXTURE_MODULES)

test:
	go test ./...

lint:
	golangci-lint run ./...

fuzz:
	@echo "Running all fuzz targets for $(or $(FUZZ_TIME),30s) each..."
	go test -fuzz=FuzzTokenize -fuzztime=$(or $(FUZZ_TIME),30s) ./internal/lexer/
	go test -fuzz=FuzzParseModule -fuzztime=$(or $(FUZZ_TIME),30s) ./internal/parser/
	go test -fuzz=FuzzConstraintParsing -fuzztime=$(or $(FUZZ_TIME),30s) ./internal/parser/
	go test -fuzz=FuzzResolutionOrder -fuzztime=$(or $(FUZZ_TIME),30s) ./internal/graph/
	go test -fuzz=FuzzParseOID -fuzztime=$(or $(FUZZ_TIME),30s) ./mib/
	go test -fuzz=FuzzFormatOID -fuzztime=$(or $(FUZZ_TIME),30s) ./mib/
	go test -fuzz=FuzzLower -fuzztime=$(or $(FUZZ_TIME),30s) .
	go test -fuzz=FuzzPipeline -fuzztime=$(or $(FUZZ_TIME),30s) .
	go test -fuzz=FuzzMultiModule -fuzztime=$(or $(FUZZ_TIME),30s) .
