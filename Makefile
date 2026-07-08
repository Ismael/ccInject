BIN := bin/ccinject

.PHONY: build test clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/ccinject

test:
	go vet ./...
	go test ./...

clean:
	rm -rf bin
