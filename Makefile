# Variables
BINARY_NAME := discache
BINARY_DIR := bin
BINARY_PATH := $(BINARY_DIR)/$(BINARY_NAME)
GO_FILES := $(shell find . -type f -name '*.go')
LISTEN_ADDR := :4000
LEADER_ADDR := :3000

# Targets
.PHONY: all build clean run runfollower test lint fmt

all: build

build: $(GO_FILES)
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_PATH)

run: build
	$(BINARY_PATH)

leader: build
	$(BINARY_PATH) start node $(LISTEN_ADDR) leader $(LEADER_ADDR)

test:
	@go test -v ./...

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...

clean:
	@rm -rf $(BINARY_DIR)
