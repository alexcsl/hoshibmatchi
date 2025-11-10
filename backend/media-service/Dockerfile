# =================================================================
# Dockerfile for all Go Microservices (user-service, post-service, etc.)
# =================================================================

# Stage 1: The 'builder' stage, where we compile the Go application.
# We use a specific Go version on Alpine Linux for a small base.
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy the go.mod and go.sum files first. This is a Docker layer caching optimization.
# Docker will only re-download dependencies if these files change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code into the container.
COPY . .

# Build the Go application.
# CGO_ENABLED=0 creates a statically linked binary (no external C dependencies).
# GOOS=linux ensures the binary is built for the Linux OS inside our final container.
# -o /app/main specifies the output file name and location.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main .

# Stage 2: The 'final' stage, where we create the minimal production image.
# We start from a fresh, tiny Alpine image.
FROM alpine:latest

WORKDIR /app

# Copy ONLY the compiled binary from the 'builder' stage.
# We don't need the Go toolchain or source code in the final image.
COPY --from=builder /app/main .

# Expose the port the gRPC server will run on.
# We'll use port 9000 as a convention for our internal gRPC services.
EXPOSE 9000

# The command to run when the container starts.
CMD ["/app/main"]