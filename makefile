.PHONY: build run clean test deps

build:
	@echo "Building aServ..."
	@go build -o bin/aServ ./cmd/app

run:
	@echo "Starting aServ..."
	@go run ./cmd/app/main.go

clean:
	@echo "Cleaning..."
	@rm -rf bin/

test:
	@echo "Running tests..."
	@go test ./...

deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go get gopkg.in/yaml.v2
	@go get github.com/google/uuid
	@go get github.com/gorilla/mux

help:
	@echo "Available commands:"
	@echo "  build  - Build the application"
	@echo "  run    - Run the application"
	@echo "  clean  - Clean build artifacts"
	@echo "  test   - Run tests"
	@echo "  deps   - Install dependencies"
