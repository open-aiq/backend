.PHONY: help dev build run swagger clean

## help: Show available commands
help:
	@echo "Available commands:"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

## dev: Live reload with swagger regeneration
dev:
	air

## swagger: Generate swagger docs
swagger:
	swag init -g cmd/server/main.go -o docs

## build: Build the binary (runs swagger first)
build: swagger
	go build -o bin/server ./cmd/server/

## run: Build and run the server
run: build
	./bin/server

## clean: Remove build artifacts
clean:
	rm -rf bin/ tmp/
