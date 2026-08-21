VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test vet install clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/shelf .

test:
	go test ./...

vet:
	go vet ./...

install:
	go install .

clean:
	rm -rf bin/
