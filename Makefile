.PHONY: build run test clean setup

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=insider-monitor
BINARY_UNIX=$(BINARY_NAME)_unix

# Build directory
BUILD_DIR=bin

all: test build

build: 
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/monitor

clean: 
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf data/*.log

run:
	$(GOCMD) run cmd/monitor/main.go -web

run-api:
	$(GOCMD) run cmd/api/main.go

run-test:
	$(GOCMD) run cmd/monitor/main.go -test

test: 
	$(GOTEST) -v ./...

setup:
	$(GOGET) -v ./...
	cp -n config.example.json config.json || true

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) -v ./cmd/monitor

docker-build:
	docker build -t $(BINARY_NAME) .

# Help target
help:
	@echo "Available targets:"
	@echo "  make build       - Build the binary"
	@echo "  make run        - Run the application"
	@echo "  make run-test   - Run in test mode"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build files"
	@echo "  make setup      - Initial setup"
	@echo "  make build-linux- Build for Linux"
