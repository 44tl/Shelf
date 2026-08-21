.PHONY: build test vet install clean

build:
	go build -o bin/ .

test:
	go test ./...

vet:
	go vet ./...

install:
	go install .

clean:
	rm -rf bin/
