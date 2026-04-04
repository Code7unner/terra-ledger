.PHONY: run build test lint tidy clean

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test -race -count=1 -v ./...

lint:
	golangci-lint run --timeout 5m

tidy:
	go mod tidy

clean:
	rm -rf bin/ tmp/
