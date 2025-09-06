BINARY := reconciler
MAIN_PKG := ./cmd/reconciler

.PHONY: deps build install run clean

## Install & vendor dependencies
deps:
	go mod tidy && go mod vendor -v

## Build a local binary
build:
	go build -o $(BINARY) $(MAIN_PKG)

## Install to $$GOBIN (so it's on your PATH)
install:
	go install $(MAIN_PKG)

## Run the CLI with example arguments
run: build
	./$(BINARY) \
	  -system="examples/transactions/system_transactions.csv" \
	  -bank="examples/statements/statement_bank_A.csv,examples/statements/statement_bank_B.csv" \
	  -start="2025-09-01" \
	  -end="2025-09-05"

## Clean build artifacts
clean:
	go clean
	rm -f $(BINARY)
