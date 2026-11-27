
# Variables
BINARY_NAME := discache
BINARY_DIR := bin
BINARY_PATH := $(BINARY_DIR)/$(BINARY_NAME)
GO_FILES := $(shell find . -type f -name '*.go')

.PHONY: all build clean run follower test lint fmt

all: build

CONFIG ?= config.json

build: $(GO_FILES)
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_PATH)

run: build
	$(BINARY_PATH) start node $(or $(NAME),:node1) config $(CONFIG)

follower: build
	$(BINARY_PATH) start node $(or $(NAME),:node2) config $(CONFIG) leader $(or $(Leader_NAME),:node1)

test:
	@go test -v ./...

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...

clean:
	@rm -rf $(BINARY_DIR)
