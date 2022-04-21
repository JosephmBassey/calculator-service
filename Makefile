# SHELL := /bin/bash
export
BINARY=server
include develop.env


.PHONY: proto
proto: ## Compile proto
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    rpc/proto/*/*.proto

.PHONY: test
test:
	go test -v ./services/calculatorservice/calculator_test.go

.PHONY: build
build:
	go build -mod=vendor -v -o $(BINARY) ./cmd

.PHONY: run
run: ## Compile and run locally
	go run ./cmd