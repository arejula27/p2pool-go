BINARY := p2pool
BUILD_DIR := build
GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w

.PHONY: all build test clean fmt vet lint run

all: fmt vet test build

build:
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY) ./cmd/p2pool/

test:
	$(GO) test ./... -count=1

test-verbose:
	$(GO) test ./... -count=1 -v

test-race:
	$(GO) test ./... -count=1 -race

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY) $(ARGS)
