BINARY  := skinner
PKG     := ./cmd/skinner
GOFLAGS ?=

.PHONY: build clean test bench bench-cpu fmt lint vet check install run

build:
	go build $(GOFLAGS) -o $(BINARY) $(PKG)

clean:
	rm -f $(BINARY)

test:
	go test ./...

bench:
	go test -bench=. -benchmem -run=^$$ ./...

bench-cpu:
	go test -bench=. -benchmem -cpuprofile=cpu.prof -run=^$$ ./internal/tui/

fmt:
	gofmt -w .

lint:
	golangci-lint run

vet:
	go vet ./...

check: vet lint test

install:
	go install $(PKG)

run: build
	./$(BINARY)
