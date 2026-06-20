.PHONY: run lint build

run:
	go run hello.go

lint:
	golangci-lint run

build:
	go build -o bin/hello.exe hello.go
