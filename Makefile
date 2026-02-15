FIXTURE_MODULES = IF-MIB SNMPv2-MIB IP-MIB ENTITY-MIB BRIDGE-MIB
FIXTURE_DIR = testdata/fixtures/netsnmp
CORPUS_PATH = testdata/corpus/primary

.PHONY: fixtures gomib-netsnmp test lint

gomib-netsnmp:
	CGO_ENABLED=1 go build -tags cgo -o gomib-netsnmp ./cmd/gomib-netsnmp

fixtures: gomib-netsnmp
	@mkdir -p $(FIXTURE_DIR)
	./gomib-netsnmp fixturegen -p $(CORPUS_PATH) -dir $(FIXTURE_DIR) $(FIXTURE_MODULES)

test:
	go test ./...

lint:
	golangci-lint run ./...
