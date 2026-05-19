.PHONY: build test run clean

build:
	go build -o bin/server ./cmd/server

test:
	go test -v ./...

run:
	go run ./cmd/server

clean:
	rm -rf bin/