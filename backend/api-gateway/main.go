package main

// API Gateway: Main entry point for all client requests

// @title           Hoshibmatchi API
// @version         1.0
// @description     API Gateway for Hoshibmatchi social media platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.hoshibmatchi.com/support
// @contact.email  support@hoshibmatchi.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8000
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	// Import the gRPC client connection library
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// Import the generated proto code for user-service
	// This path MUST match the 'go_package' option in your user.proto
	hashtagPb "github.com/hoshibmatchi/hashtag-service/proto"
	mediaPb "github.com/hoshibmatchi/media-service/proto"
	messagePb "github.com/hoshibmatchi/message-service/proto"
	postPb "github.com/hoshibmatchi/post-service/proto"
	reportPb "github.com/hoshibmatchi/report-service/proto"
	storyPb "github.com/hoshibmatchi/story-service/proto"
	pb "github.com/hoshibmatchi/user-service/proto"

	_ "github.com/hoshibmatchi/api-gateway/docs" // This will be generated
)

// client will hold the persistent gRPC connection
var client pb.UserServiceClient
var postClient postPb.PostServiceClient
var storyClient storyPb.StoryServiceClient
var mediaClient mediaPb.MediaServiceClient
var messageClient messagePb.MessageServiceClient
var reportClient reportPb.ReportServiceClient
var hashtagClient hashtagPb.HashtagServiceClient

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// Redis client for distributed rate limiting
var rdb *redis.Client

// Notification database connection
var notificationDB *gorm.DB

// Rate limiter for in-memory fallback
var globalLimiter *rate.Limiter

type contextKey string

const userIDKey contextKey = "userID"

// Rate limit configuration
type RateLimit struct {
	RequestsPerHour int
	Burst           int
}

var (
	anonymousLimit     = RateLimit{RequestsPerHour: 100, Burst: 20}
	authenticatedLimit = RateLimit{RequestsPerHour: 1000, Burst: 50}
	sensitiveLimit     = RateLimit{RequestsPerHour: 10, Burst: 3} // For login, registration
)

// Notification model (matching notification-service)
type Notification struct {
	gorm.Model
	UserID   int64  `gorm:"index" json:"user_id"`
	ActorID  int64  `json:"actor_id"`
	Type     string `json:"type"`
	EntityID int64  `json:"entity_id"`
	IsRead   bool   `gorm:"default:false" json:"is_read"`
}

func main() {
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	// Connect to Redis for distributed rate limiting
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed, rate limiting will use in-memory fallback: %v", err)
		globalLimiter = rate.NewLimiter(rate.Limit(authenticatedLimit.RequestsPerHour/3600.0), authenticatedLimit.Burst)
	} else {
		log.Println("Successfully connected to Redis for rate limiting")
	}

	// --- Connect to Notification Database ---
	notificationDSN := "host=notification-db user=admin password=password dbname=notification_service_db port=5432 sslmode=disable"
	var dbErr error
	notificationDB, dbErr = gorm.Open(postgres.Open(notificationDSN), &gorm.Config{})
	if dbErr != nil {
		log.Printf("Warning: Failed to connect to notification database: %v", dbErr)
	} else {
		log.Println("Successfully connected to notification database")
	}

	// --- Connect to all gRPC Services ---
	mustConnect(&client, "user-service:9000")
	mustConnect(&postClient, "post-service:9001")
	mustConnect(&storyClient, "story-service:9002")
	mustConnect(&mediaClient, "media-service:9005")
	mustConnect(&messageClient, "message-service:9003")
	mustConnect(&reportClient, "report-service:9006")
	mustConnect(&hashtagClient, "hashtag-service:9007")

	// --- Set up Gin Router ---
	router := gin.Default()
	router.Use(gin.Logger())   // Add default logger
	router.Use(gin.Recovery()) // Add default panic recovery

	// CORS middleware for frontend connection
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Swagger documentation route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "API Gateway is running")
	})

	router.GET("/media/upload-url", handleGetUploadURL_Gin)
	router.GET("/media/secure-url", handleGetMediaURL_Gin) // New endpoint for pre-signed GET URLs

	// Serve static files from /uploads (media files)
	router.Static("/uploads", "./uploads")

	// Auth routes with sensitive rate limiting (10 req/hour)
	authRoutes := router.Group("/auth")
	authRoutes.Use(SensitiveEndpointLimiter())
	{
		// These handlers don't need params, so gin.WrapF is fine.
		authRoutes.POST("/register", gin.WrapF(handleRegister))
		authRoutes.POST("/send-otp", gin.WrapF(handleSendOtp))
		authRoutes.POST("/verify-otp", gin.WrapF(handleVerifyRegistrationOtp))
		authRoutes.POST("/google/callback", handleGoogleCallback_Gin)
		authRoutes.GET("/check-username/:username", handleCheckUsername_Gin)
		authRoutes.POST("/login", gin.WrapF(handleLogin))
		authRoutes.POST("/login/verify-2fa", gin.WrapF(handleVerify2FA))
		authRoutes.POST("/password-reset/request", gin.WrapF(handleSendPasswordReset))
		authRoutes.POST("/password-reset/submit", gin.WrapF(handleResetPassword))
	}

	// Protected routes (JWT auth required + authenticated rate limit: 1000 req/hour)
	protected := router.Group("/")
	protected.Use(GinAuthMiddleware())
	protected.Use(RateLimitMiddleware(authenticatedLimit))
	{
		// --- THIS IS THE FIX ---
		// We are now calling the Gin-native handlers directly

		// Media (protected)
		protected.POST("/media/generate-thumbnail", handleGenerateThumbnail_Gin) // Generate thumbnail for uploaded video

		// Feeds
		protected.GET("/feed/home", handleGetHomeFeed_Gin)
		protected.GET("/feed/explore", handleGetExploreFeed_Gin)
		protected.GET("/feed/reels", handleGetReelsFeed_Gin)

		// Posts
		protected.POST("/posts", handleCreatePost_Gin)
		protected.POST("/posts/:id/like", handlePostLike_Gin)
		protected.DELETE("/posts/:id/like", handlePostLike_Gin)
		protected.DELETE("/posts/:id", handleDeletePost_Gin)
		protected.POST("/posts/:id/summarize", handleSummarizeCaption_Gin)

		// Stories
		protected.POST("/stories", handleCreateStory_Gin)
		protected.POST("/stories/:id/like", handleStoryLike_Gin)
		protected.DELETE("/stories/:id/like", handleStoryLike_Gin)
		protected.GET("/stories/feed", handleGetStoryFeed_Gin)
		protected.GET("/stories/archive", handleGetUserArchive_Gin)

		// Comments
		protected.POST("/comments", handleCreateComment_Gin)
		protected.GET("/posts/:id/comments", handleGetCommentsByPost_Gin)
		protected.DELETE("/comments/:id", handleDeleteComment_Gin)
		protected.POST("/comments/:id/like", handleLikeComment_Gin)
		protected.DELETE("/comments/:id/like", handleUnlikeComment_Gin)

		// Users
		protected.POST("/users/:id/follow", handleFollowUser_Gin)
		protected.DELETE("/users/:id/follow", handleFollowUser_Gin)
		protected.GET("/users/:id/followers", handleGetFollowersList_Gin)
		protected.GET("/users/:id/following", handleGetFollowingList_Gin)
		protected.GET("/users/top", handleGetTopUsers_Gin)
		protected.GET("/posts/:id/likes", handleGetPostLikers_Gin)

		// Profile
		protected.GET("/users/:id", handleGetUserProfile_Gin)
		protected.GET("/users/:id/posts", handleGetUserPosts_Gin)
		protected.GET("/users/:id/reels", handleGetUserReels_Gin)
		protected.GET("/users/:id/tagged", handleGetUserTaggedPosts_Gin)

		// Edit Profiel
		protected.PUT("/profile/edit", handleUpdateProfile_Gin)
		protected.PUT("/users/complete-profile", handleCompleteProfile_Gin)
		protected.PUT("/settings/privacy", handleSetPrivacy_Gin)

		protected.POST("/users/:id/block", handleBlockUser_Gin)
		protected.DELETE("/users/:id/block", handleBlockUser_Gin)

		protected.POST("/collections", handleCreateCollection_Gin)
		protected.GET("/collections", handleGetUserCollections_Gin)
		protected.GET("/collections/:id", handleGetPostsInCollection_Gin)
		protected.POST("/collections/:id/posts", handleSavePostToCollection_Gin)
		protected.DELETE("/collections/:id/posts/:post_id", handleUnsavePostFromCollection_Gin)
		protected.DELETE("/collections/:id", handleDeleteCollection_Gin)
		protected.PUT("/collections/:id", handleRenameCollection_Gin)

		// Messsage
		protected.POST("/conversations", handleCreateConversation_Gin)
		protected.POST("/conversations/:id/messages", handleSendMessage_Gin)
		protected.POST("/conversations/:id/messages/media", handleSendMessageWithMedia_Gin)

		protected.GET("/conversations", handleGetConversations_Gin)
		protected.GET("/conversations/:id/messages", handleGetMessages_Gin)
		protected.GET("/conversations/:id/messages/search", handleSearchMessages_Gin)

		// Search
		protected.GET("/search/users", handleSearchUsers_Gin)

		// Report Routes
		protected.POST("/reports/post", handleReportPost_Gin)
		protected.POST("/reports/user", handleReportUser_Gin)

		// Verif
		protected.POST("/profile/verify", handleSubmitVerification_Gin)

		// Hashtag
		protected.GET("/search/hashtags/:name", handleSearchHashtag_Gin)
		protected.GET("/trending/hashtags", handleTrendingHashtags_Gin)

		// Video call, delete, unsend
		protected.DELETE("/messages/:id", handleUnsendMessage_Gin)
		protected.DELETE("/conversations/:id", handleDeleteConversation_Gin)
		protected.GET("/conversations/:id/video_token", handleGetVideoToken_Gin)

		// Notifications
		protected.GET("/notifications", handleGetNotifications_Gin)
		protected.PUT("/notifications/:id/read", handleMarkNotificationRead_Gin)
		protected.PUT("/notifications/read-all", handleMarkAllNotificationsRead_Gin)
	}

	admin := router.Group("/admin")
	admin.Use(AdminAuthMiddleware()) // Use our new middleware
	{
		admin.GET("/users", handleGetAllUsers_Gin)
		admin.POST("/users/:id/ban", handleBanUser_Gin)
		admin.POST("/users/:id/unban", handleUnbanUser_Gin)

		admin.GET("/reports/posts", handleGetPostReports_Gin)
		admin.GET("/reports/users", handleGetUserReports_Gin)
		admin.POST("/reports/posts/:id/resolve", handleResolvePostReport_Gin)
		admin.POST("/reports/users/:id/resolve", handleResolveUserReport_Gin)

		// Newsletter & Verification
		admin.POST("/newsletters", handleSendNewsletter_Gin)
		admin.GET("/verifications", handleGetVerifications_Gin)
		admin.POST("/verifications/:id/resolve", handleResolveVerification_Gin)
	}

	log.Println("API Gateway starting on port 8000...")
	if err := router.Run(":8000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// --- Gin-native Auth Middleware ---
func GinAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenString := parts[1]

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID := int64(userIDFloat)

		// We set the userID in the standard http.Request.Context()
		ctx := context.WithValue(c.Request.Context(), userIDKey, userID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// --- NEW: Gin-native Admin Auth Middleware ---
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenString := parts[1]

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// --- THIS IS THE ADMIN CHECK ---
		role, ok := claims["role"].(string)
		if !ok || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: Admin access required"})
			return
		}
		// --- END ADMIN CHECK ---

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID := int64(userIDFloat)

		// Set the userID in the context, just like the normal middleware
		ctx := context.WithValue(c.Request.Context(), userIDKey, userID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// --- gRPC Connection Helper ---
func mustConnect(client interface{}, target string) {
	// Increase max message size to 50MB for video uploads
	conn, err := grpc.Dial(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024), // 50MB
			grpc.MaxCallSendMsgSize(50*1024*1024), // 50MB
		),
	)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", target, err)
	}

	switch c := client.(type) {
	case *pb.UserServiceClient:
		*c = pb.NewUserServiceClient(conn)
	case *postPb.PostServiceClient:
		*c = postPb.NewPostServiceClient(conn)
	case *storyPb.StoryServiceClient:
		*c = storyPb.NewStoryServiceClient(conn)
	case *mediaPb.MediaServiceClient:
		*c = mediaPb.NewMediaServiceClient(conn)
	case *messagePb.MessageServiceClient:
		*c = messagePb.NewMessageServiceClient(conn)
	case *reportPb.ReportServiceClient:
		*c = reportPb.NewReportServiceClient(conn)
	case *hashtagPb.HashtagServiceClient: // <-- ADD THIS
		*c = hashtagPb.NewHashtagServiceClient(conn)
	default:
		log.Fatalf("Unknown client type")
	}
	log.Printf("Successfully connected to %s", target)
}

// RateLimitMiddleware implements distributed rate limiting using Redis
func RateLimitMiddleware(limit RateLimit) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP() // Default to IP

		// If authenticated, use user ID instead
		if userID, exists := c.Get("userID"); exists {
			identifier = fmt.Sprintf("user:%v", userID)
		}

		// Check rate limit
		if !checkRateLimit(c.Request.Context(), identifier, limit) {
			resetTime := time.Now().Add(time.Hour).Unix()
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit.RequestsPerHour))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded. Please try again later.",
				"retry_after": 3600,
			})
			return
		}

		c.Next()
	}
}

// checkRateLimit checks if the request is within rate limits using Redis
func checkRateLimit(ctx context.Context, identifier string, limit RateLimit) bool {
	// If Redis is unavailable, use in-memory rate limiter
	if rdb == nil {
		if globalLimiter == nil {
			return true // No rate limiting if both Redis and limiter failed
		}
		return globalLimiter.Allow()
	}

	key := fmt.Sprintf("rate_limit:%s", identifier)
	now := time.Now()
	windowStart := now.Add(-time.Hour).Unix()

	// Use Redis sorted set for sliding window rate limiting
	pipe := rdb.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Count requests in current window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiration
	pipe.Expire(ctx, key, time.Hour*2)

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Rate limit check failed: %v", err)
		return true // Allow on error
	}

	count := countCmd.Val()
	return count < int64(limit.RequestsPerHour)
}

// SensitiveEndpointLimiter is for login, registration, and other sensitive endpoints
func SensitiveEndpointLimiter() gin.HandlerFunc {
	return RateLimitMiddleware(sensitiveLimit)
}

// handleRegister translates the HTTP request to a gRPC call
func handleRegister(w http.ResponseWriter, r *http.Request) {
	// 1. We only accept POST methods
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode the JSON body from the client
	var req struct {
		Name                  string `json:"name"`
		Username              string `json:"username"`
		Email                 string `json:"email"`
		Password              string `json:"password"`
		ConfirmPassword       string `json:"confirm_password"` // ADDED
		DateOfBirth           string `json:"date_of_birth"`
		Gender                string `json:"gender"`
		Enable2FA             bool   `json:"enable_2fa"` // ADDED for 2FA
		ProfilePictureURL     string `json:"profile_picture_url"`
		SubscribeToNewsletter bool   `json:"subscribe_to_newsletter"` // Newsletter subscription
		TurnstileToken        string `json:"turnstile_token"`         // Cloudflare Turnstile
		// OtpCode           string `json:"otp_code"` // REMOVED
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2.5. Verify Turnstile token
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	isValid, err := VerifyTurnstileToken(req.TurnstileToken, clientIP)
	if err != nil || !isValid {
		log.Printf("Turnstile verification failed for %s: %v", clientIP, err)
		http.Error(w, "Verification challenge failed. Please try again.", http.StatusBadRequest)
		return
	}

	log.Printf("Register request - username: %s, subscribe_to_newsletter: %v", req.Username, req.SubscribeToNewsletter)

	// 3. Call the gRPC service
	grpcReq := &pb.RegisterUserRequest{
		Name:              req.Name,
		Username:          req.Username,
		Email:             req.Email,
		Password:          req.Password,
		ConfirmPassword:   req.ConfirmPassword, // ADDED
		DateOfBirth:       req.DateOfBirth,
		Gender:            req.Gender,
		Enable_2Fa:        req.Enable2FA, // ADDED for 2FA
		ProfilePictureUrl: req.ProfilePictureURL,
		IsSubscribed:      req.SubscribeToNewsletter, // Newsletter subscription
		// OtpCode:           req.OtpCode, // REMOVED
	}

	res, err := client.RegisterUser(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}

	// 4. Send the successful JSON response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func handleSendOtp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.SendOtpRequest{Email: req.Email}
	res, err := client.SendRegistrationOtp(r.Context(), grpcReq)
	if err != nil {
		// This will correctly pass gRPC errors (like 429 ResourceExhausted) to the client
		grpcErr, _ := status.FromError(err)
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// --- ADD THIS HELPER FUNCTION TO main.go ---
// (We'll use this to translate gRPC errors to HTTP)
func gRPCToHTTPStatusCode(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// handleLogin translates the HTTP request to a gRPC call
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EmailOrUsername string `json:"email_or_username"`
		Password        string `json:"password"`
		TurnstileToken  string `json:"turnstile_token"` // Cloudflare Turnstile
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify Turnstile token
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	isValid, err := VerifyTurnstileToken(req.TurnstileToken, clientIP)
	if err != nil || !isValid {
		log.Printf("Turnstile verification failed for %s: %v", clientIP, err)
		http.Error(w, "Verification challenge failed. Please try again.", http.StatusBadRequest)
		return
	}

	grpcReq := &pb.LoginRequest{
		EmailOrUsername: req.EmailOrUsername,
		Password:        req.Password,
	}

	// 'grpcRes' is the response from the gRPC service
	grpcRes, err := client.LoginUser(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}

	// --- THIS IS THE FIX ---
	// We create our own JSON response struct
	// This gives us full control over the JSON output
	type jsonResponse struct {
		Message       string `json:"message"`
		AccessToken   string `json:"access_token,omitempty"` // omitempty is fine here
		RefreshToken  string `json:"refresh_token,omitempty"`
		Is2FARequired bool   `json:"is_2fa_required"` // No omitempty!
	}

	res := jsonResponse{
		Message:       grpcRes.Message,
		AccessToken:   grpcRes.AccessToken,
		RefreshToken:  grpcRes.RefreshToken,
		Is2FARequired: grpcRes.Is_2FaRequired,
	}
	// --- END FIX ---

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res) // Encode our custom struct
}

func handleVerify2FA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email   string `json:"email"`
		OtpCode string `json:"otp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.Verify2FARequest{
		Email:   req.Email,
		OtpCode: req.OtpCode,
	}

	// 'grpcRes' is the response from the gRPC service
	grpcRes, err := client.Verify2FA(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}

	// We can reuse the same JSON response struct from handleLogin
	// to return the tokens in the same format.
	type jsonResponse struct {
		Message       string `json:"message"`
		AccessToken   string `json:"access_token,omitempty"`
		RefreshToken  string `json:"refresh_token,omitempty"`
		Is2FARequired bool   `json:"is_2fa_required"`
	}

	res := jsonResponse{
		Message:       "2FA verification successful. Logged in.",
		AccessToken:   grpcRes.AccessToken,
		RefreshToken:  grpcRes.RefreshToken,
		Is2FARequired: false, // We've just completed it
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// --- HANDLER 1: handleSendPasswordReset ---
func handleSendPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	grpcReq := &pb.SendPasswordResetRequest{Email: req.Email}
	grpcRes, err := client.SendPasswordReset(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(grpcRes)
}

// --- HANDLER 2: handleResetPassword ---
func handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email       string `json:"email"`
		OtpCode     string `json:"otp_code"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	grpcReq := &pb.ResetPasswordRequest{
		Email:       req.Email,
		OtpCode:     req.OtpCode,
		NewPassword: req.NewPassword,
	}
	grpcRes, err := client.ResetPassword(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(grpcRes)
}

// --- GIN-NATIVE HANDLER: handleCreatePost ---
// handleCreatePost_Gin godoc
// @Summary Create a new post
// @Description Create a new post with caption and media
// @Tags posts
// @Accept json
// @Produce json
// @Param request body object true "Post data"
// @Success 201 {object} object "Created post"
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /posts [post]
func handleCreatePost_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		Caption          string   `json:"caption"`
		MediaURLs        []string `json:"media_urls"`
		CommentsDisabled bool     `json:"comments_disabled"`
		IsReel           bool     `json:"is_reel"`
		CollaboratorIDs  []int64  `json:"collaborator_ids"` // Added
		ThumbnailURL     string   `json:"thumbnail_url"`    // Added
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.MediaURLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one media_url is required"})
		return
	}

	grpcReq := &postPb.CreatePostRequest{
		AuthorId:         userID,
		Caption:          req.Caption,
		MediaUrls:        req.MediaURLs,
		CommentsDisabled: req.CommentsDisabled,
		IsReel:           req.IsReel,
		CollaboratorIds:  req.CollaboratorIDs, // Added
		ThumbnailUrl:     req.ThumbnailURL,    // Added
	}

	grpcRes, err := postClient.CreatePost(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusCreated, grpcRes.Post) // Return the Post object inside the response
}

// --- GIN-NATIVE HANDLER: handleCreateStory ---
func handleCreateStory_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		MediaURL  string `json:"media_url"`
		MediaType string `json:"media_type"` // Ensure frontend sends this ("image" or "video")
		Caption   string `json:"caption"`
		Filter    string `json:"filter_name"`
		Stickers  string `json:"stickers_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &storyPb.CreateStoryRequest{
		AuthorId:     userID,
		MediaUrl:     req.MediaURL,
		MediaType:    req.MediaType,
		Caption:      req.Caption,
		FilterName:   req.Filter,
		StickersJson: req.Stickers,
	}

	grpcRes, err := storyClient.CreateStory(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to story-service failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusCreated, grpcRes.Story)
}

// --- GIN-NATIVE HANDLER: handleCreateComment ---
func handleCreateComment_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		PostID          int64  `json:"post_id"`
		Content         string `json:"content"`
		ParentCommentID int64  `json:"parent_comment_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &postPb.CommentOnPostRequest{
		UserId:          userID,
		PostId:          req.PostID,
		Content:         req.Content,
		ParentCommentId: req.ParentCommentID,
	}

	grpcRes, err := postClient.CommentOnPost(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusCreated, grpcRes)
}

// --- GIN-NATIVE HANDLERS (FOR URL PARAMS) ---

func handleFollowUser_Gin(c *gin.Context) {
	// --- THIS IS THE FIX ---
	// Read from the request's context, not Gin's context
	followerID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	// --- END FIX ---

	followingIDStr := c.Param("id")
	followingID, err := strconv.ParseInt(followingIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if c.Request.Method == http.MethodPost {
		// ... (rest of the function is the same)
		grpcReq := &pb.FollowUserRequest{FollowerId: followerID, FollowingId: followingID}
		grpcRes, err := client.FollowUser(c.Request.Context(), grpcReq)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, grpcRes)

	} else if c.Request.Method == http.MethodDelete {
		// ... (rest of the function is the same)
		grpcReq := &pb.UnfollowUserRequest{FollowerId: followerID, FollowingId: followingID}
		grpcRes, err := client.UnfollowUser(c.Request.Context(), grpcReq)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, grpcRes)
	}
}

// handlePostLike_Gin godoc
// @Summary Like or unlike a post
// @Description Like a post (POST) or unlike a post (DELETE)
// @Tags posts
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} object "Success message"
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /posts/{id}/like [post]
func handlePostLike_Gin(c *gin.Context) {
	// --- THIS IS THE FIX ---
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	// --- END FIX ---

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	if c.Request.Method == http.MethodPost {
		req := &postPb.LikePostRequest{UserId: userID, PostId: postID}
		res, err := postClient.LikePost(c.Request.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, res)

	} else if c.Request.Method == http.MethodDelete {
		req := &postPb.LikePostRequest{UserId: userID, PostId: postID}
		res, err := postClient.UnlikePost(c.Request.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func handleStoryLike_Gin(c *gin.Context) {
	// --- THIS IS THE FIX ---
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	// --- END FIX ---

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseInt(storyIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid story ID"})
		return
	}

	if c.Request.Method == http.MethodPost {
		req := &storyPb.LikeStoryRequest{UserId: userID, StoryId: storyID}
		res, err := storyClient.LikeStory(c.Request.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, res)

	} else if c.Request.Method == http.MethodDelete {
		req := &storyPb.UnlikeStoryRequest{UserId: userID, StoryId: storyID}
		res, err := storyClient.UnlikeStory(c.Request.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func handleDeleteComment_Gin(c *gin.Context) {
	// --- THIS IS THE FIX ---
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	// --- END FIX ---

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	grpcReq := &postPb.DeleteCommentRequest{UserId: userID, CommentId: commentID}
	grpcRes, err := postClient.DeleteComment(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleLikeComment ---
func handleLikeComment_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	grpcReq := &postPb.LikeCommentRequest{UserId: userID, CommentId: commentID}
	grpcRes, err := postClient.LikeComment(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleUnlikeComment ---
func handleUnlikeComment_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	grpcReq := &postPb.LikeCommentRequest{UserId: userID, CommentId: commentID}
	grpcRes, err := postClient.UnlikeComment(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleDeletePost ---
func handleDeletePost_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	grpcReq := &postPb.DeletePostRequest{PostId: postID, AdminUserId: userID}
	grpcRes, err := postClient.DeletePost(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetCommentsByPost ---
func handleGetCommentsByPost_Gin(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	// Get pagination params
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	offset := (page - 1) * limit

	grpcReq := &postPb.GetCommentsByPostRequest{
		PostId:     postID,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	}

	grpcRes, err := postClient.GetCommentsByPost(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Comments)
}

// --- GIN-NATIVE HANDLER: handleGetUploadURL ---
func handleGetUploadURL_Gin(c *gin.Context) {
	var userID int64 = 0
	if val, ok := c.Request.Context().Value(userIDKey).(int64); ok {
		userID = val
	}

	// Get query params, e.g., /media/upload-url?filename=foo.jpg&type=image/jpeg
	filename := c.Query("filename")
	contentType := c.Query("type")

	if filename == "" || contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'filename' or 'type' query parameters"})
		return
	}

	grpcReq := &mediaPb.GetUploadURLRequest{
		Filename:    filename,
		ContentType: contentType,
		UserId:      userID,
	}

	grpcRes, err := mediaClient.GetUploadURL(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetMediaURL ---
func handleGetMediaURL_Gin(c *gin.Context) {
	// Get query param, e.g., /media/secure-url?object_name=user-123/posts/abc.jpg
	objectName := c.Query("object_name")

	if objectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'object_name' query parameter"})
		return
	}

	// Optional expiry parameter (default handled by media-service)
	expirySeconds, _ := strconv.Atoi(c.DefaultQuery("expiry_seconds", "3600"))

	grpcReq := &mediaPb.GetMediaURLRequest{
		ObjectName:    objectName,
		ExpirySeconds: int32(expirySeconds),
	}

	grpcRes, err := mediaClient.GetMediaURL(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGenerateThumbnail ---
func handleGenerateThumbnail_Gin(c *gin.Context) {
	log.Println("=== THUMBNAIL GENERATION REQUEST ===")

	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		log.Println("❌ Failed to get user ID from token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	log.Printf("✅ User ID: %d", userID)

	var req struct {
		ObjectName       string  `json:"object_name"`
		TimestampSeconds float64 `json:"timestamp_seconds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ObjectName == "" {
		log.Println("❌ Missing object_name field")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'object_name' field"})
		return
	}

	log.Printf("Request data: object_name=%s, timestamp=%.2f", req.ObjectName, req.TimestampSeconds)

	grpcReq := &mediaPb.GenerateThumbnailRequest{
		ObjectName:       req.ObjectName,
		UserId:           userID,
		TimestampSeconds: req.TimestampSeconds,
	}

	log.Println("Calling media-service GenerateThumbnail RPC...")
	grpcRes, err := mediaClient.GenerateThumbnail(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("❌ gRPC error: %v", grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	log.Printf("✅ Thumbnail generated: %s", grpcRes.ThumbnailUrl)
	log.Println("=== THUMBNAIL GENERATION COMPLETE ===")
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetHomeFeed ---
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
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// Get pagination query params, e.g., /feed/home?page=1&limit=20
	// We'll default to page 1, limit 20
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	grpcReq := &postPb.GetHomeFeedRequest{
		UserId:     userID,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	}

	grpcRes, err := postClient.GetHomeFeed(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetHomeFeed failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": grpcRes.Posts})
}

// --- GIN-NATIVE HANDLER: handleGetExploreFeed ---
func handleGetExploreFeed_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	grpcReq := &postPb.GetHomeFeedRequest{
		UserId:     userID,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	}

	grpcRes, err := postClient.GetExploreFeed(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetExploreFeed failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"posts": grpcRes.Posts})
}

// --- GIN-NATIVE HANDLER: handleGetReelsFeed ---
func handleGetReelsFeed_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	grpcReq := &postPb.GetHomeFeedRequest{
		UserId:     userID,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	}

	grpcRes, err := postClient.GetReelsFeed(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetReelsFeed failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"posts": grpcRes.Posts})
}

// --- GIN-NATIVE HANDLER: handleGetUserProfile ---
// This is a complex aggregator handler
// handleGetUserProfile_Gin godoc
// @Summary Get user profile
// @Description Get user profile by user ID or username
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID or username"
// @Success 200 {object} object "User profile"
// @Failure 404 {object} object "User not found"
// @Security BearerAuth
// @Router /users/{id} [get]
func handleGetUserProfile_Gin(c *gin.Context) {
	selfUserID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	usernameToFind := c.Param("id") // Get username from URL

	// --- 1. Get Profile Data from User-Service ---
	userReq := &pb.GetUserProfileRequest{
		Username:   usernameToFind,
		SelfUserId: selfUserID,
	}
	userRes, err := client.GetUserProfile(c.Request.Context(), userReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	// --- 2. Get Post/Reel Counts from Post-Service (THIS IS THE FIX) ---
	countReq := &postPb.GetUserContentCountRequest{UserId: userRes.UserId}
	countRes, err := postClient.GetUserContentCount(c.Request.Context(), countReq)
	if err != nil {
		// Don't fail the whole request if this fails, just log it
		log.Printf("Failed to get post counts: %v", err)
	}

	// --- 3. Aggregate the response ---
	type ProfileResponse struct {
		User      *pb.GetUserProfileResponse `json:"user"`
		PostCount int64                      `json:"post_count"`
		ReelCount int64                      `json:"reel_count"`
	}

	var postCount int64 = 0
	var reelCount int64 = 0
	if countRes != nil {
		postCount = countRes.PostCount
		reelCount = countRes.ReelCount
	}

	res := ProfileResponse{
		User:      userRes,
		PostCount: postCount,
		ReelCount: reelCount,
	}

	c.JSON(http.StatusOK, res)
}

// --- GIN-NATIVE HANDLER: handleGetUserPosts ---
func handleGetUserPosts_Gin(c *gin.Context) {
	usernameToFind := c.Param("id")

	userRes, err := client.GetUserProfile(c.Request.Context(), &pb.GetUserProfileRequest{Username: usernameToFind})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
	offset := (page - 1) * limit

	// --- THIS IS THE FIX ---
	grpcReq := &postPb.GetUserContentRequest{ // Was pb.
		UserId:     userRes.UserId,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	}
	// --- END FIX ---

	grpcRes, err := postClient.GetUserPosts(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes.Posts)
}

// --- GIN-NATIVE HANDLER: handleGetUserReels ---
func handleGetUserReels_Gin(c *gin.Context) {
	usernameToFind := c.Param("id")

	userRes, err := client.GetUserProfile(c.Request.Context(), &pb.GetUserProfileRequest{Username: usernameToFind})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
	offset := (page - 1) * limit

	// --- THIS IS THE FIX ---
	grpcReq := &postPb.GetUserContentRequest{ // Was pb.
		UserId:     userRes.UserId,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	}
	// --- END FIX ---

	grpcRes, err := postClient.GetUserReels(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes.Posts)
}

// --- GIN-NATIVE HANDLER: handleCompleteProfile ---
func handleCompleteProfile_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		Username    string `json:"username"`
		DateOfBirth string `json:"date_of_birth"`
		Gender      string `json:"gender"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &pb.CompleteProfileRequest{
		UserId:      userID,
		Username:    req.Username,
		DateOfBirth: req.DateOfBirth,
		Gender:      req.Gender,
	}

	grpcRes, err := client.CompleteProfile(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": grpcRes.Message})
}

// --- GIN-NATIVE HANDLER: handleUpdateProfile ---
func handleUpdateProfile_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		Name              string `json:"name"`
		Bio               string `json:"bio"`
		Gender            string `json:"gender"`
		ProfilePictureURL string `json:"profile_picture_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &pb.UpdateUserProfileRequest{
		UserId:            userID,
		Name:              req.Name,
		Bio:               req.Bio,
		Gender:            req.Gender,
		ProfilePictureUrl: req.ProfilePictureURL,
	}

	grpcRes, err := client.UpdateUserProfile(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes) // Return the full updated profile
}

// --- GIN-NATIVE HANDLER: handleSetPrivacy ---
func handleSetPrivacy_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		IsPrivate bool `json:"is_private"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &pb.SetAccountPrivacyRequest{
		UserId:    userID,
		IsPrivate: req.IsPrivate,
	}

	grpcRes, err := client.SetAccountPrivacy(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleBlockUser (Handles POST for Block, DELETE for Unblock) ---
func handleBlockUser_Gin(c *gin.Context) {
	// 1. Get the current user's ID from the JWT
	blockerID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// 2. Get the target user's ID from the URL
	blockedIDStr := c.Param("id")
	blockedID, err := strconv.ParseInt(blockedIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if c.Request.Method == http.MethodPost {
		// --- Block User ---
		grpcReq := &pb.BlockUserRequest{
			BlockerId: blockerID,
			BlockedId: blockedID,
		}
		grpcRes, err := client.BlockUser(c.Request.Context(), grpcReq)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, grpcRes)

	} else if c.Request.Method == http.MethodDelete {
		// --- Unblock User ---
		grpcReq := &pb.UnblockUserRequest{
			BlockerId: blockerID,
			BlockedId: blockedID,
		}
		grpcRes, err := client.UnblockUser(c.Request.Context(), grpcReq)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
			return
		}
		c.JSON(http.StatusOK, grpcRes)

	} else {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Invalid request method"})
	}
}

// --- GIN-NATIVE HANDLER: handleCreateCollection ---
func handleCreateCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &postPb.CreateCollectionRequest{UserId: userID, Name: req.Name}
	grpcRes, err := postClient.CreateCollection(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusCreated, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetUserCollections ---
func handleGetUserCollections_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	grpcReq := &postPb.GetUserCollectionsRequest{UserId: userID}
	grpcRes, err := postClient.GetUserCollections(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes.Collections)
}

// --- GIN-NATIVE HANDLER: handleGetPostsInCollection ---
func handleGetPostsInCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
	offset := (page - 1) * limit

	grpcReq := &postPb.GetPostsInCollectionRequest{
		UserId:       userID,
		CollectionId: collectionID,
		PageSize:     int32(limit),
		PageOffset:   int32(offset),
	}
	grpcRes, err := postClient.GetPostsInCollection(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes.Posts)
}

// --- GIN-NATIVE HANDLER: handleSavePostToCollection ---
func handleSavePostToCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	var req struct {
		PostID int64 `json:"post_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'post_id'"})
		return
	}

	grpcReq := &postPb.SavePostToCollectionRequest{
		UserId:       userID,
		CollectionId: collectionID,
		PostId:       req.PostID,
	}
	grpcRes, err := postClient.SavePostToCollection(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleUnsavePostFromCollection ---
func handleUnsavePostFromCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}
	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	grpcReq := &postPb.UnsavePostFromCollectionRequest{
		UserId:       userID,
		CollectionId: collectionID,
		PostId:       postID,
	}
	grpcRes, err := postClient.UnsavePostFromCollection(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleDeleteCollection ---
func handleDeleteCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	grpcReq := &postPb.DeleteCollectionRequest{UserId: userID, CollectionId: collectionID}
	grpcRes, err := postClient.DeleteCollection(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleRenameCollection ---
func handleRenameCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	var req struct {
		NewName string `json:"new_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'new_name'"})
		return
	}

	grpcReq := &postPb.RenameCollectionRequest{
		UserId:       userID,
		CollectionId: collectionID,
		NewName:      req.NewName,
	}
	grpcRes, err := postClient.RenameCollection(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes)
}

func handleVerifyRegistrationOtp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email   string `json:"email"`
		OtpCode string `json:"otp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.VerifyRegistrationOtpRequest{
		Email:   req.Email,
		OtpCode: req.OtpCode,
	}

	grpcRes, err := client.VerifyRegistrationOtp(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}

	// Return the tokens
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(grpcRes)
}

// --- GIN-NATIVE HANDLER: handleCreateConversation ---
func handleCreateConversation_Gin(c *gin.Context) {
	creatorID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		ParticipantIDs []int64 `json:"participant_ids"`
		GroupName      string  `json:"group_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.ParticipantIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "participant_ids must not be empty"})
		return
	}

	grpcReq := &messagePb.CreateConversationRequest{
		CreatorId:      creatorID,
		ParticipantIds: req.ParticipantIDs,
		GroupName:      req.GroupName,
	}

	grpcRes, err := messageClient.CreateConversation(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to CreateConversation failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusCreated, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleSendMessage ---
func handleSendMessage_Gin(c *gin.Context) {
	senderID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// Get conversation ID from URL param
	convoID := c.Param("id")

	var req struct {
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'content' in request body"})
		return
	}

	grpcReq := &messagePb.SendMessageRequest{
		SenderId:       senderID,
		ConversationId: convoID,
		Content:        req.Content,
	}

	grpcRes, err := messageClient.SendMessage(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to SendMessage failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusCreated, grpcRes.Message)
}

// --- GIN-NATIVE HANDLER: handleSendMessageWithMedia ---
func handleSendMessageWithMedia_Gin(c *gin.Context) {
	senderID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// Get conversation ID from URL param
	convoID := c.Param("id")

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	// Get optional text content
	content := c.PostForm("content")

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'file' in request"})
		return
	}
	defer file.Close()

	// Determine media type based on content type
	contentType := header.Header.Get("Content-Type")
	var mediaType string
	if strings.HasPrefix(contentType, "image/gif") {
		mediaType = "gif"
	} else if strings.HasPrefix(contentType, "image/") {
		mediaType = "image"
	} else if strings.HasPrefix(contentType, "video/") {
		mediaType = "video"
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported media type. Only images, GIFs, and videos are allowed"})
		return
	}

	// Read file content
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Upload file directly via media service gRPC
	uploadReq := &mediaPb.UploadMediaRequest{
		Filename:    header.Filename,
		ContentType: contentType,
		UserId:      senderID,
		FileData:    fileBytes,
	}

	uploadRes, err := mediaClient.UploadMedia(c.Request.Context(), uploadReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("Failed to upload media: %v", err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": "Failed to upload media"})
		return
	}

	// Send message with media URL
	grpcReq := &messagePb.SendMessageRequest{
		SenderId:       senderID,
		ConversationId: convoID,
		Content:        content,
		MediaUrl:       uploadRes.MediaUrl,
		MediaType:      mediaType,
	}

	grpcRes, err := messageClient.SendMessage(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to SendMessage failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusCreated, grpcRes.Message)
}

// --- GIN-NATIVE HANDLER: handleGetConversations ---
func handleGetConversations_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// Get pagination params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	grpcReq := &messagePb.GetConversationsRequest{
		UserId:     userID,
		PageSize:   int32(limit),
		PageOffset: int32(offset),
	}

	grpcRes, err := messageClient.GetConversations(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetConversations failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Conversations)
}

// --- GIN-NATIVE HANDLER: handleGetMessages ---
func handleGetMessages_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// Get conversation ID from URL param
	convoID := c.Param("id")

	// Get pagination params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50")) // Get more messages by default
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit

	grpcReq := &messagePb.GetMessagesRequest{
		UserId:         userID,
		ConversationId: convoID,
		PageSize:       int32(limit),
		PageOffset:     int32(offset),
	}

	grpcRes, err := messageClient.GetMessages(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetMessages failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Messages)
}

func handleSearchMessages_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// Get conversation ID from URL param
	convoID := c.Param("id")

	// Get search query from query param
	searchQuery := c.Query("q")
	if searchQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query 'q' is required"})
		return
	}

	grpcReq := &messagePb.SearchMessagesRequest{
		UserId:         userID,
		ConversationId: convoID,
		Query:          searchQuery,
	}

	grpcRes, err := messageClient.SearchMessages(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to SearchMessages failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Messages)
}

// --- GIN-NATIVE HANDLER: handleSearchUsers ---
func handleSearchUsers_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	// Get search query from URL param ?q=
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing search query 'q'"})
		return
	}

	grpcReq := &pb.SearchUsersRequest{
		Query:      query,
		SelfUserId: userID,
	}

	grpcRes, err := client.SearchUsers(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to SearchUsers failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Users)
}

// --- GIN-NATIVE HANDLER: handleSummarizeCaption (BapTion) ---
func handleSummarizeCaption_Gin(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	// --- Step 1: Get Post Caption from post-service ---
	postReq := &postPb.GetPostRequest{PostId: postID}
	postRes, err := postClient.GetPost(c.Request.Context(), postReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetPost failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	if postRes.Caption == "" {
		c.JSON(http.StatusOK, gin.H{"summary": ""}) // No caption to summarize
		return
	}

	// --- Step 2: Call ai-service (HTTP) ---
	aiServiceURL := "http://ai-service:9008/summarize"
	requestBody, _ := json.Marshal(map[string]string{
		"caption": postRes.Caption,
	})

	resp, err := http.Post(aiServiceURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Failed to call ai-service: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service is unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("ai-service returned non-200 status: %s", string(bodyBytes))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI service failed to process request"})
		return
	}

	// --- Step 3: Decode and return ai-service's response ---
	var aiResponse struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aiResponse); err != nil {
		log.Printf("Failed to decode ai-service response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	c.JSON(http.StatusOK, aiResponse)
}

// --- GIN-NATIVE HANDLER: handleReportPost ---
func handleReportPost_Gin(c *gin.Context) {
	reporterID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		PostID int64  `json:"post_id"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.PostID == 0 || req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'post_id' or 'reason'"})
		return
	}

	grpcReq := &reportPb.ReportPostRequest{
		ReporterId: reporterID,
		PostId:     req.PostID,
		Reason:     req.Reason,
	}

	grpcRes, err := reportClient.ReportPost(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to ReportPost failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusCreated, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleReportUser ---
func handleReportUser_Gin(c *gin.Context) {
	reporterID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		ReportedUserID int64  `json:"reported_user_id"`
		Reason         string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.ReportedUserID == 0 || req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'reported_user_id' or 'reason'"})
		return
	}

	grpcReq := &reportPb.ReportUserRequest{
		ReporterId:     reporterID,
		ReportedUserId: req.ReportedUserID,
		Reason:         req.Reason,
	}

	grpcRes, err := reportClient.ReportUser(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to ReportUser failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusCreated, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetAllUsers (Admin) ---
func handleGetAllUsers_Gin(c *gin.Context) {
	adminID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get admin ID from token"})
		return
	}

	// Use search with empty/wildcard query to get all users
	grpcReq := &pb.SearchUsersRequest{
		Query:      " ", // Space as query to match all
		SelfUserId: adminID,
	}

	grpcRes, err := client.SearchUsers(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to SearchUsers (GetAllUsers) failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Users)
}

// --- GIN-NATIVE HANDLER: handleBanUser ---
func handleBanUser_Gin(c *gin.Context) {
	adminID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get admin ID from token"})
		return
	}

	userToBanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	grpcReq := &pb.BanUserRequest{
		AdminUserId: adminID,
		UserToBanId: userToBanID,
	}

	grpcRes, err := client.BanUser(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to BanUser failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleUnbanUser ---
func handleUnbanUser_Gin(c *gin.Context) {
	adminID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get admin ID from token"})
		return
	}

	userToUnbanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	grpcReq := &pb.UnbanUserRequest{
		AdminUserId:   adminID,
		UserToUnbanId: userToUnbanID,
	}

	grpcRes, err := client.UnbanUser(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to UnbanUser failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetPostReports ---
func handleGetPostReports_Gin(c *gin.Context) {
	// Pagination and filtering
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	unresolvedOnly, _ := strconv.ParseBool(c.DefaultQuery("unresolved_only", "true"))

	grpcReq := &reportPb.GetReportsRequest{
		PageSize:       int32(limit),
		PageOffset:     int32((page - 1) * limit),
		UnresolvedOnly: unresolvedOnly,
	}

	grpcRes, err := reportClient.GetPostReports(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Reports)
}

// --- GIN-NATIVE HANDLER: handleGetUserReports ---
func handleGetUserReports_Gin(c *gin.Context) {
	// Pagination and filtering
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	unresolvedOnly, _ := strconv.ParseBool(c.DefaultQuery("unresolved_only", "true"))

	grpcReq := &reportPb.GetReportsRequest{
		PageSize:       int32(limit),
		PageOffset:     int32((page - 1) * limit),
		UnresolvedOnly: unresolvedOnly,
	}

	grpcRes, err := reportClient.GetUserReports(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Reports)
}

// --- GIN-NATIVE HANDLER: handleResolvePostReport ---
func handleResolvePostReport_Gin(c *gin.Context) {
	adminID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get admin ID from token"})
		return
	}

	reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	var req struct {
		Action string `json:"action"` // "ACCEPT" or "REJECT"
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Action != "ACCEPT" && req.Action != "REJECT") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'action', must be 'ACCEPT' or 'REJECT'"})
		return
	}

	grpcReq := &reportPb.ResolveReportRequest{
		AdminUserId: adminID,
		ReportId:    reportID,
		Action:      req.Action,
	}

	grpcRes, err := reportClient.ResolvePostReport(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleResolveUserReport ---
func handleResolveUserReport_Gin(c *gin.Context) {
	adminID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get admin ID from token"})
		return
	}

	reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	var req struct {
		Action string `json:"action"` // "ACCEPT" or "REJECT"
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Action != "ACCEPT" && req.Action != "REJECT") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'action', must be 'ACCEPT' or 'REJECT'"})
		return
	}

	grpcReq := &reportPb.ResolveReportRequest{
		AdminUserId: adminID,
		ReportId:    reportID,
		Action:      req.Action,
	}

	grpcRes, err := reportClient.ResolveUserReport(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleSubmitVerification (User-facing) ---
func handleSubmitVerification_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	var req struct {
		IdCardNumber   string `json:"id_card_number"`
		FacePictureURL string `json:"face_picture_url"`
		Reason         string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.IdCardNumber == "" || req.FacePictureURL == "" || req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'id_card_number', 'face_picture_url', or 'reason'"})
		return
	}

	grpcReq := &pb.SubmitVerificationRequestRequest{
		UserId:         userID,
		IdCardNumber:   req.IdCardNumber,
		FacePictureUrl: req.FacePictureURL,
		Reason:         req.Reason,
	}

	grpcRes, err := client.SubmitVerificationRequest(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusCreated, grpcRes.Request)
}

// --- GIN-NATIVE HANDLER: handleSendNewsletter (Admin) ---
func handleSendNewsletter_Gin(c *gin.Context) {
	adminID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get admin ID from token"})
		return
	}

	var req struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Subject == "" || req.Body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'subject' or 'body'"})
		return
	}

	grpcReq := &pb.SendNewsletterRequest{
		AdminUserId: adminID,
		Subject:     req.Subject,
		Body:        req.Body,
	}

	grpcRes, err := client.SendNewsletter(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetVerifications (Admin) ---
func handleGetVerifications_Gin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	// --- THIS IS THE FIX ---
	reqStatus := c.DefaultQuery("status", "pending") // Renamed 'status' to 'reqStatus'
	// --- END FIX ---

	grpcReq := &pb.GetVerificationRequestsRequest{
		PageSize:   int32(limit),
		PageOffset: int32((page - 1) * limit),
		Status:     reqStatus, // Use the new variable
	}

	grpcRes, err := client.GetVerificationRequests(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err) // This line will now work
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Requests)
}

// --- GIN-NATIVE HANDLER: handleResolveVerification (Admin) ---
func handleResolveVerification_Gin(c *gin.Context) {
	adminID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get admin ID from token"})
		return
	}

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var req struct {
		Action string `json:"action"` // "APPROVE" or "REJECT"
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Action != "APPROVE" && req.Action != "REJECT") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'action', must be 'APPROVE' or 'REJECT'"})
		return
	}

	grpcReq := &pb.ResolveVerificationRequestRequest{
		AdminUserId: adminID,
		RequestId:   requestID,
		Action:      req.Action,
	}

	grpcRes, err := client.ResolveVerificationRequest(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleSearchHashtag ---
func handleSearchHashtag_Gin(c *gin.Context) {
	hashtagName := c.Param("name")
	if hashtagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hashtag name is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	grpcReq := &hashtagPb.SearchByHashtagRequest{
		HashtagName: strings.ToLower(hashtagName),
		PageSize:    int32(limit),
		PageOffset:  int32((page - 1) * limit),
	}

	grpcRes, err := hashtagClient.SearchByHashtag(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes) // Returns { "posts": [...], "total_post_count": X }
}

// --- GIN-NATIVE HANDLER: handleTrendingHashtags ---
func handleTrendingHashtags_Gin(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	grpcReq := &hashtagPb.GetTrendingHashtagsRequest{
		Limit: int32(limit),
	}

	grpcRes, err := hashtagClient.GetTrendingHashtags(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Hashtags)
}

// --- GIN-NATIVE HANDLER: handleUnsendMessage ---
func handleUnsendMessage_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	msgID := c.Param("id")

	grpcReq := &messagePb.UnsendMessageRequest{
		UserId:    userID,
		MessageId: msgID,
	}

	grpcRes, err := messageClient.UnsendMessage(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleDeleteConversation ---
func handleDeleteConversation_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	convoID := c.Param("id")

	grpcReq := &messagePb.DeleteConversationRequest{
		UserId:         userID,
		ConversationId: convoID,
	}

	grpcRes, err := messageClient.DeleteConversation(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetVideoToken ---
func handleGetVideoToken_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	convoID := c.Param("id")

	grpcReq := &messagePb.GetVideoCallTokenRequest{
		UserId:         userID,
		ConversationId: convoID,
	}

	grpcRes, err := messageClient.GetVideoCallToken(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGoogleCallback ---
func handleGoogleCallback_Gin(c *gin.Context) {
	var req struct {
		AuthCode string `json:"auth_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AuthCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'auth_code'"})
		return
	}

	grpcReq := &pb.HandleGoogleAuthRequest{AuthCode: req.AuthCode}
	grpcRes, err := client.HandleGoogleAuth(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	// Return the same as login: tokens
	c.JSON(http.StatusOK, gin.H{
		"message":       grpcRes.Message,
		"access_token":  grpcRes.AccessToken,
		"refresh_token": grpcRes.RefreshToken,
	})
}

// --- GIN-NATIVE HANDLER: handleCheckUsername ---
func handleCheckUsername_Gin(c *gin.Context) {
	username := c.Param("username")

	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	// Try to get user by username
	_, err := client.GetUserProfile(c.Request.Context(), &pb.GetUserProfileRequest{
		Username: username,
	})

	if err != nil {
		// User not found, username is available
		grpcErr, _ := status.FromError(err)
		if grpcErr.Code() == codes.NotFound {
			c.JSON(http.StatusOK, gin.H{"exists": false, "available": true})
			return
		}
		// Other error
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	// User found, username is taken
	c.JSON(http.StatusOK, gin.H{"exists": true, "available": false})
}

// --- HANDLER: GetUserTaggedPosts ---
func handleGetUserTaggedPosts_Gin(c *gin.Context) {
	requesterID, _ := c.Request.Context().Value(userIDKey).(int64)
	username := c.Param("id")

	// 1. Resolve username to ID (reuse existing logic or call GetUserProfile)
	userRes, err := client.GetUserProfile(c.Request.Context(), &pb.GetUserProfileRequest{Username: username})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 2. Call Post Service
	grpcReq := &postPb.GetUserContentRequest{
		UserId:      userRes.UserId,
		RequesterId: requesterID,
		PageSize:    20,
		PageOffset:  0,
	}
	res, err := postClient.GetUserTaggedPosts(c.Request.Context(), grpcReq)
	if err != nil {
		// handle error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tagged posts"})
		return
	}
	c.JSON(http.StatusOK, res.Posts)
}

// --- HANDLER: GetFollowersList ---
func handleGetFollowersList_Gin(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	res, err := client.GetFollowersList(c.Request.Context(), &pb.GetFollowersListRequest{UserId: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get followers"})
		return
	}

	// Hydrate user IDs with full user data
	users := make([]map[string]interface{}, 0)
	for _, followerID := range res.FollowerUserIds {
		userRes, err := client.GetUserData(c.Request.Context(), &pb.GetUserDataRequest{UserId: followerID})
		if err != nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"user_id":             userRes.Id,
			"username":            userRes.Username,
			"profile_picture_url": userRes.ProfilePictureUrl,
			"is_verified":         userRes.IsVerified,
		})
	}
	c.JSON(http.StatusOK, users)
}

// --- HANDLER: GetFollowingList ---
func handleGetFollowingList_Gin(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	res, err := client.GetFollowingList(c.Request.Context(), &pb.GetFollowingListRequest{UserId: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get following"})
		return
	}

	// Hydrate user IDs with full user data
	users := make([]map[string]interface{}, 0)
	for _, followingID := range res.FollowingUserIds {
		userRes, err := client.GetUserData(c.Request.Context(), &pb.GetUserDataRequest{UserId: followingID})
		if err != nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"user_id":             userRes.Id,
			"username":            userRes.Username,
			"profile_picture_url": userRes.ProfilePictureUrl,
			"is_verified":         userRes.IsVerified,
		})
	}
	c.JSON(http.StatusOK, users)
}

// --- HANDLER: GetPostLikers ---
func handleGetPostLikers_Gin(c *gin.Context) {
	postIDStr := c.Param("id")

	// For now, return empty array as this requires querying post_likes table
	// which needs to be added to post-service proto
	// TODO: Add GetPostLikers RPC to post-service
	_ = postIDStr
	c.JSON(http.StatusOK, []map[string]interface{}{})
}

// --- HANDLER: GetTopUsers ---
func handleGetTopUsers_Gin(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50 // Cap at 50
	}

	userID, _ := c.Request.Context().Value(userIDKey).(int64)

	// Use SearchUsers with empty query to get all users
	res, err := client.SearchUsers(c.Request.Context(), &pb.SearchUsersRequest{
		Query:      "",
		SelfUserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Convert to slice for sorting
	type userWithFollowers struct {
		UserID            int64  `json:"user_id"`
		Username          string `json:"username"`
		ProfilePictureURL string `json:"profile_picture_url"`
		IsVerified        bool   `json:"is_verified"`
		FollowerCount     int64  `json:"follower_count"`
	}

	users := make([]userWithFollowers, 0)
	for _, user := range res.Users {
		users = append(users, userWithFollowers{
			UserID:            user.UserId,
			Username:          user.Username,
			ProfilePictureURL: user.ProfilePictureUrl,
			IsVerified:        user.IsVerified,
			FollowerCount:     user.FollowerCount,
		})
	}

	// Sort by follower count descending
	sort.Slice(users, func(i, j int) bool {
		return users[i].FollowerCount > users[j].FollowerCount
	})

	// Take top N
	if len(users) > limit {
		users = users[:limit]
	}

	c.JSON(http.StatusOK, users)
}

// --- HANDLER: GetStoryFeed ---
func handleGetStoryFeed_Gin(c *gin.Context) {
	userID, _ := c.Request.Context().Value(userIDKey).(int64)

	res, err := storyClient.GetStoryFeed(c.Request.Context(), &storyPb.GetStoryFeedRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stories"})
		return
	}

	c.JSON(http.StatusOK, res.StoryGroups)
}

// --- HANDLER: GetUserArchive ---
func handleGetUserArchive_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"})
		return
	}

	res, err := storyClient.GetUserArchive(c.Request.Context(), &storyPb.GetUserArchiveRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch archive"})
		return
	}

	c.JSON(http.StatusOK, res.Stories)
}

// --- NOTIFICATION HANDLERS ---

// NotificationResponse includes actor details
type NotificationResponse struct {
	ID                     uint   `json:"id"`
	UserID                 int64  `json:"user_id"`
	ActorID                int64  `json:"actor_id"`
	ActorUsername          string `json:"actor_username"`
	ActorProfilePictureURL string `json:"actor_profile_picture_url"`
	ActorIsVerified        bool   `json:"actor_is_verified"`
	Type                   string `json:"type"`
	EntityID               int64  `json:"entity_id"`
	IsRead                 bool   `json:"is_read"`
	CreatedAt              string `json:"created_at"`
}

// handleGetNotifications returns notifications for the current user
func handleGetNotifications_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if notificationDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service unavailable"})
		return
	}

	// Get limit from query params (default 50)
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Query notifications
	var notifications []Notification
	result := notificationDB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications)

	if result.Error != nil {
		log.Printf("Failed to query notifications: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	// Enrich notifications with actor data
	enrichedNotifications := make([]NotificationResponse, 0, len(notifications))
	for _, notif := range notifications {
		// Get actor user data
		actorRes, err := client.GetUserData(c.Request.Context(), &pb.GetUserDataRequest{
			UserId: notif.ActorID,
		})

		actorUsername := "User"
		actorProfileURL := ""
		actorIsVerified := false
		if err == nil {
			actorUsername = actorRes.Username
			actorProfileURL = actorRes.ProfilePictureUrl
			actorIsVerified = actorRes.IsVerified
		}

		enrichedNotifications = append(enrichedNotifications, NotificationResponse{
			ID:                     notif.Model.ID,
			UserID:                 notif.UserID,
			ActorID:                notif.ActorID,
			ActorUsername:          actorUsername,
			ActorProfilePictureURL: actorProfileURL,
			ActorIsVerified:        actorIsVerified,
			Type:                   notif.Type,
			EntityID:               notif.EntityID,
			IsRead:                 notif.IsRead,
			CreatedAt:              notif.Model.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": enrichedNotifications,
		"unread_count":  countUnread(notifications),
	})
}

// handleMarkNotificationRead marks a single notification as read
func handleMarkNotificationRead_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if notificationDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service unavailable"})
		return
	}

	notifID := c.Param("id")

	// Update notification - ensure it belongs to the user
	result := notificationDB.Model(&Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Update("is_read", true)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// handleMarkAllNotificationsRead marks all notifications as read for the current user
func handleMarkAllNotificationsRead_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if notificationDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notification service unavailable"})
		return
	}

	// Update all unread notifications for this user
	result := notificationDB.Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All notifications marked as read",
		"count":   result.RowsAffected,
	})
}

// Helper function to count unread notifications
func countUnread(notifications []Notification) int {
	count := 0
	for _, n := range notifications {
		if !n.IsRead {
			count++
		}
	}
	return count
}
