
.PHONY: all build test clean

all: build

build: tidy
	go build -o bin/oneapi cmd/oneapi/main.go

test:
	go test ./...

clean:
	rm -rf bin

tidy:
	go mod tidy

docker-build:
	docker build -f docker/Dockerfile . 
