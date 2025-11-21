.PHONY: swagger install-swagger build run

# Install swag CLI tool
install-swagger:
	go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger documentation
swagger:
	swag init -g main.go --output ./docs

# Build the application
build: swagger
	go build -o tmp/main .

# Run the application
run: swagger
	go run main.go

# Clean generated files
clean:
	rm -rf docs/
	rm -rf tmp/
