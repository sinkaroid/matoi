GOPATH := $(shell go env GOPATH)

.PHONY: dev build prod lint

dev:
	"$(GOPATH)/bin/air"

build:
	go build -o bin/matoi.exe main.go

prod: build
	./bin/matoi.exe

lint:
	"$(GOPATH)/bin/golangci-lint" run
