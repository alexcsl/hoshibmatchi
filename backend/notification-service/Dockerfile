# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app/backend/notification-service

# 1. Copy this service's mod files
COPY backend/notification-service/go.mod backend/notification-service/go.sum ./

# 2. (No local dependencies)

# 3. Download dependencies
RUN go mod download

# 4. Copy all backend source
COPY backend/ /app/backend/

# 5. Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main .

# Stage 2: Final
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .

# This is a worker, it does not need to EXPOSE a port
CMD ["/app/main"]