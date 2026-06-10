.PHONY: dev build run swagger clean

# Live reload with swagger regeneration
dev:
	air

# Generate swagger docs
swagger:
	swag init -g cmd/server/main.go -o docs

# Build the binary
build: swagger
	go build -o bin/server ./cmd/server/

# Run the binary
run: build
	./bin/server

# Remove build artifacts
clean:
	rm -rf bin/ tmp/
