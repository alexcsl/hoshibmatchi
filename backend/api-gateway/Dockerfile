# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# CGO_ENABLED=0 is important for a static binary
# -o /app/main creates the output file in the specified path
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main .

# Stage 2: Create the final, minimal image
FROM alpine:latest

WORKDIR /app

# Copy only the built binary from the builder stage
COPY --from=builder /app/main .

# Expose the port the app runs on (e.g., 8000)
EXPOSE 8000

# Command to run the executable
CMD ["/app/main"]