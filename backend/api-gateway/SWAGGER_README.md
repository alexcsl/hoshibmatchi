# API Gateway Swagger Documentation

This API Gateway includes comprehensive Swagger/OpenAPI documentation for all endpoints.

## Quick Start

### 1. Install Swagger Dependencies

#### Windows (PowerShell):
```powershell
.\generate-swagger.ps1
```

#### Linux/Mac:
```bash
# Install swag CLI
go install github.com/swaggo/swag/cmd/swag@latest

# Get dependencies
go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
go mod tidy

# Generate documentation
swag init -g main.go --output ./docs
```

### 2. Access Swagger UI

After starting the API Gateway, visit:
```
http://localhost:8000/swagger/index.html
```

### 3. Using Swagger UI

1. **Authentication**: Click the "Authorize" button at the top
2. Enter your JWT token in the format: `Bearer <your-token>`
3. Try out endpoints directly from the browser

## Swagger Annotations

The API uses Swaggo annotations to generate documentation. Here's the structure:

### General Info (in main.go header):
```go
// @title           Hoshibmatchi API
// @version         1.0
// @description     API Gateway for Hoshibmatchi social media platform
// @host            localhost:8000
// @BasePath        /
```

### Endpoint Annotation Example:
```go
// handleGetHomeFeed_Gin godoc
// @Summary Get home feed
// @Description Get personalized home feed for the authenticated user
// @Tags feed
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {array} object "List of posts"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /feed/home [get]
func handleGetHomeFeed_Gin(c *gin.Context) {
    // handler code
}
```

## Available Endpoints

### Authentication
- `POST /auth/register` - Register new user
- `POST /auth/login` - User login
- `POST /auth/send-otp` - Send OTP for verification
- `POST /auth/verify-otp` - Verify OTP
- `POST /auth/password-reset/request` - Request password reset
- `POST /auth/password-reset/submit` - Submit new password

### Feed
- `GET /feed/home` - Get personalized home feed
- `GET /feed/explore` - Get explore feed
- `GET /feed/reels` - Get reels feed

### Posts
- `POST /posts` - Create new post
- `GET /posts/:id` - Get post by ID
- `POST /posts/:id/like` - Like a post
- `DELETE /posts/:id/like` - Unlike a post
- `DELETE /posts/:id` - Delete a post
- `GET /posts/:id/comments` - Get post comments
- `POST /posts/:id/summarize` - AI summarize caption
- `GET /posts/:id/likes` - Get users who liked the post

### Users
- `GET /users/:id` - Get user profile
- `GET /users/:id/posts` - Get user's posts
- `GET /users/:id/reels` - Get user's reels
- `GET /users/:id/followers` - Get user's followers
- `GET /users/:id/following` - Get users being followed
- `GET /users/top` - Get top users by follower count
- `POST /users/:id/follow` - Follow user
- `DELETE /users/:id/follow` - Unfollow user
- `PUT /profile/edit` - Edit own profile

### Stories
- `POST /stories` - Create new story
- `GET /stories/feed` - Get story feed
- `GET /stories/archive` - Get user's archived stories
- `POST /stories/:id/like` - Like a story
- `DELETE /stories/:id/like` - Unlike a story

### Comments
- `POST /comments` - Create comment
- `DELETE /comments/:id` - Delete comment
- `POST /comments/:id/like` - Like comment
- `DELETE /comments/:id/like` - Unlike comment

### Messages
- `POST /conversations` - Create conversation
- `GET /conversations` - Get user's conversations
- `POST /conversations/:id/messages` - Send message
- `GET /conversations/:id/messages` - Get conversation messages

### Admin
- `GET /admin/users` - Get all users (admin only)
- `POST /admin/users/:id/ban` - Ban user (admin only)
- `DELETE /admin/users/:id/ban` - Unban user (admin only)
- `GET /admin/reports/posts` - Get post reports (admin only)
- `GET /admin/reports/users` - Get user reports (admin only)

## Regenerating Documentation

After adding new endpoints or modifying existing ones:

```bash
# Regenerate docs
swag init -g main.go --output ./docs

# Rebuild and restart the service
docker-compose restart api-gateway
```

## Swagger Annotation Reference

### Common Tags
- `@Summary` - Short description
- `@Description` - Detailed description
- `@Tags` - Group endpoints by category
- `@Accept` - Input format (e.g., json, multipart/form-data)
- `@Produce` - Output format (e.g., json)
- `@Param` - Parameter definition
- `@Success` - Success response
- `@Failure` - Error response
- `@Security` - Security requirement (e.g., BearerAuth)
- `@Router` - Route path and method

### Parameter Types
- `path` - URL path parameter
- `query` - Query string parameter
- `body` - Request body
- `header` - HTTP header
- `formData` - Form data

## Troubleshooting

### Swagger UI not loading
1. Ensure docs folder exists: `ls ./docs`
2. Regenerate documentation: `swag init -g main.go --output ./docs`
3. Check imports in main.go include: `_ "github.com/hoshibmatchi/api-gateway/docs"`

### "docs" package not found
Run: `go mod tidy` then regenerate docs

### Changes not reflected
1. Regenerate docs: `swag init -g main.go --output ./docs`
2. Restart API Gateway: `docker-compose restart api-gateway`

## Resources

- [Swaggo Documentation](https://github.com/swaggo/swag)
- [Swagger/OpenAPI Specification](https://swagger.io/specification/)
- [gin-swagger](https://github.com/swaggo/gin-swagger)
