.PHONY: test lint build smoke

# Full suite under the race detector, per PLAT-03 — never split into a
# separate non-race run (see 01-RESEARCH.md Pitfall 4).
test:
	go test ./... -race -count=1

lint:
	golangci-lint run

build:
	go build -o modular .

# Builds the binary and runs `status` against a throwaway MODULAR_HOME so
# a fresh clone can prove the walking skeleton end-to-end with one command.
smoke: build
	MODULAR_HOME=$$(mktemp -d) ./modular status
