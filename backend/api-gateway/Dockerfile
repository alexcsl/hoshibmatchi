# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app/backend/api-gateway

# 1. Copy this service's mod files
COPY backend/api-gateway/go.mod backend/api-gateway/go.sum ./

# 2. Copy mod files for ALL local dependencies
COPY backend/user-service/go.mod backend/user-service/go.sum ../user-service/
COPY backend/post-service/go.mod backend/post-service/go.sum ../post-service/
COPY backend/story-service/go.mod backend/story-service/go.sum ../story-service/
COPY backend/media-service/go.mod backend/media-service/go.sum ../media-service/
COPY backend/message-service/go.mod backend/message-service/go.sum ../message-service/
COPY backend/report-service/go.mod backend/report-service/go.sum ../report-service/
COPY backend/hashtag-service/go.mod backend/hashtag-service/go.sum ../hashtag-service/

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
EXPOSE 8000
CMD ["/app/main"]