# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app/backend/post-service

# 1. Copy this service's mod files
COPY backend/post-service/go.mod backend/post-service/go.sum ./

# 2. Copy mod files for dependencies (user-service)
COPY backend/user-service/go.mod backend/user-service/go.sum ../user-service/

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

# Correct port for post-service
EXPOSE 9001

CMD ["/app/main"]