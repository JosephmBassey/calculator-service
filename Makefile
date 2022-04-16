# SHELL := /bin/bash
export
BINARY=calculator-service
include Makefile.COMMON
include develop.env


# .PHONY: proto
# proto: ## Compile GRPC Protobuf
# 	protoc --go_out=. --go_opt=paths=source_relative \
#     --go-grpc_out=. --go-grpc_opt=paths=source_relative \
#     rpc/proto/*/*.proto

.PHONY: build
build:
	go build -mod=vendor -v -o $(BINARY) ./cmd

.PHONY: run
run: ## Compile and run locally
	go run ./cmd