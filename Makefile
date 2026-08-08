# MockServer Makefile — Go cross-platform, works on Linux, macOS, and Windows (Git Bash/MSYS).

BINARY := mockserver
ifeq ($(OS),Windows_NT)
	BINARY := $(BINARY).exe
endif

.PHONY: all build test test-race vet fmt run run-tls clean

all: fmt vet test build

build:
	go build -o $(BINARY) ./cmd/mockserver/

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

run: build
	./$(BINARY) --config testdata/example.json --port 8080

run-tls: build
	./$(BINARY) --tls-self-signed --config testdata/example.json --port 8443

clean:
	rm -f $(BINARY)