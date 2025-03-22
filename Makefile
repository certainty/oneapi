
.PHONY: all build test clean

all: build

build:
	go build -o bin/oneapi cmd/oneapi/main.go

test:
	go test ./...

clean:
	rm -rf bin
