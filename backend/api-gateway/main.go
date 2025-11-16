package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	// Import the gRPC client connection library
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v5"

	// Import the generated proto code for user-service
	// This path MUST match the 'go_package' option in your user.proto
	mediaPb "github.com/hoshibmatchi/media-service/proto"
	messagePb "github.com/hoshibmatchi/message-service/proto"
	postPb "github.com/hoshibmatchi/post-service/proto"
	reportPb "github.com/hoshibmatchi/report-service/proto"
	storyPb "github.com/hoshibmatchi/story-service/proto"
	pb "github.com/hoshibmatchi/user-service/proto"
)

// client will hold the persistent gRPC connection
var client pb.UserServiceClient
var postClient postPb.PostServiceClient
var storyClient storyPb.StoryServiceClient
var mediaClient mediaPb.MediaServiceClient
var messageClient messagePb.MessageServiceClient
var reportClient reportPb.ReportServiceClient


// ADD THIS (must match user-service)
// TODO: Load this from an environment variable, not hardcoded
var jwtSecret = []byte("my-super-secret-key-that-is-not-secure")

type contextKey string
const userIDKey contextKey = "userID"

func main() {
	// --- Connect to all gRPC Services ---
	mustConnect(&client, "user-service:9000")
	mustConnect(&postClient, "post-service:9001")
	mustConnect(&storyClient, "story-service:9002")
	mustConnect(&mediaClient, "media-service:9005")
	mustConnect(&messageClient, "message-service:9003")
	mustConnect(&reportClient, "report-service:9006")
	
	// --- Set up Gin Router ---
	router := gin.Default()
	router.Use(gin.Logger())   // Add default logger
	router.Use(gin.Recovery()) // Add default panic recovery

	// Public routes (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "API Gateway is running")
	})
	
	authRoutes := router.Group("/auth")
	{
		// These handlers don't need params, so gin.WrapF is fine.
		authRoutes.POST("/register", gin.WrapF(handleRegister))
		authRoutes.POST("/send-otp", gin.WrapF(handleSendOtp))
		authRoutes.POST("/verify-otp", gin.WrapF(handleVerifyRegistrationOtp))
		authRoutes.POST("/login", gin.WrapF(handleLogin))
		authRoutes.POST("/login/verify-2fa", gin.WrapF(handleVerify2FA))
		authRoutes.POST("/password-reset/request", gin.WrapF(handleSendPasswordReset))
		authRoutes.POST("/password-reset/submit", gin.WrapF(handleResetPassword))
	}

	// Protected routes (JWT auth required)
	protected := router.Group("/")
	protected.Use(GinAuthMiddleware())
	{
		// --- THIS IS THE FIX ---
		// We are now calling the Gin-native handlers directly

		// Feeds
		protected.GET("/feed/home", handleGetHomeFeed_Gin)
		protected.GET("/feed/explore", handleGetExploreFeed_Gin)
		protected.GET("/feed/reels", handleGetReelsFeed_Gin)

		// Posts
		protected.POST("/posts", handleCreatePost_Gin)
		protected.POST("/posts/:id/like", handlePostLike_Gin)
		protected.DELETE("/posts/:id/like", handlePostLike_Gin)
		
		// Stories
		protected.POST("/stories", handleCreateStory_Gin)
		protected.POST("/stories/:id/like", handleStoryLike_Gin)
		protected.DELETE("/stories/:id/like", handleStoryLike_Gin)

		// Comments
		protected.POST("/comments", handleCreateComment_Gin)
		protected.DELETE("/comments/:id", handleDeleteComment_Gin)

		// Users
		protected.POST("/users/:id/follow", handleFollowUser_Gin)
		protected.DELETE("/users/:id/follow", handleFollowUser_Gin)
		
		// Media
		protected.GET("/media/upload-url", handleGetUploadURL_Gin)

		// Profile
		protected.GET("/users/:username", handleGetUserProfile_Gin)
		protected.GET("/users/:username/posts", handleGetUserPosts_Gin)
		protected.GET("/users/:username/reels", handleGetUserReels_Gin)

		// Edit Profiel
		protected.PUT("/profile/edit", handleUpdateProfile_Gin)
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

		protected.GET("/conversations", handleGetConversations_Gin)
		protected.GET("/conversations/:id/messages", handleGetMessages_Gin)

		// Search
		protected.GET("/search/users", handleSearchUsers_Gin)

		// AI
		protected.POST("/posts/:id/summarize", handleSummarizeCaption_Gin)

		// Report Routes
		protected.POST("/reports/post", handleReportPost_Gin)
		protected.POST("/reports/user", handleReportUser_Gin)

		// Verif
		protected.POST("/profile/verify", handleSubmitVerification_Gin)

	}

	admin := router.Group("/admin")
	admin.Use(AdminAuthMiddleware()) // Use our new middleware
	{
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
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	default:
		log.Fatalf("Unknown client type")
	}
	log.Printf("Successfully connected to %s", target)
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
        Name            string `json:"name"`
        Username        string `json:"username"`
        Email           string `json:"email"`
        Password        string `json:"password"`
        ConfirmPassword string `json:"confirm_password"` // ADDED
        DateOfBirth     string `json:"date_of_birth"`
        Gender          string `json:"gender"`
        // ProfilePictureURL string `json:"profile_picture_url"` // REMOVED
        // OtpCode           string `json:"otp_code"` // REMOVED
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 3. Call the gRPC service
    grpcReq := &pb.RegisterUserRequest{
        Name:            req.Name,
        Username:        req.Username,
        Email:           req.Email,
        Password:        req.Password,
        ConfirmPassword: req.ConfirmPassword, // ADDED
        DateOfBirth:     req.DateOfBirth,
        Gender:          req.Gender,
        // ProfilePictureUrl: req.ProfilePictureURL, // REMOVED
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
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	var req struct { Email string `json:"email"` }
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
func handleCreatePost_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

	var req struct {
		Caption          string   `json:"caption"`
		MediaURLs        []string `json:"media_urls"`
		CommentsDisabled bool     `json:"comments_disabled"`
		IsReel           bool     `json:"is_reel"`
	}
	// Use ShouldBindJSON for Gin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &postPb.CreatePostRequest{
		AuthorId:         userID,
		Caption:          req.Caption,
		MediaUrls:        req.MediaURLs,
		CommentsDisabled: req.CommentsDisabled,
		IsReel:           req.IsReel,
	}

	grpcRes, err := postClient.CreatePost(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to CreatePost failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusCreated, grpcRes.Post)
}

// --- GIN-NATIVE HANDLER: handleCreateStory ---
func handleCreateStory_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

	var req struct {
		MediaURL string `json:"media_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &storyPb.CreateStoryRequest{
		AuthorId: userID,
		MediaUrl: req.MediaURL,
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
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

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
		if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
		c.JSON(http.StatusOK, res)

	} else if c.Request.Method == http.MethodDelete {
		req := &postPb.LikePostRequest{UserId: userID, PostId: postID}
		res, err := postClient.UnlikePost(c.Request.Context(), req)
		if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
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
		if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
		c.JSON(http.StatusOK, res)

	} else if c.Request.Method == http.MethodDelete {
		req := &storyPb.UnlikeStoryRequest{UserId: userID, StoryId: storyID}
		res, err := storyClient.UnlikeStory(c.Request.Context(), req)
		if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
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
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetUploadURL ---
func handleGetUploadURL_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	
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

// --- GIN-NATIVE HANDLER: handleGetHomeFeed ---
func handleGetHomeFeed_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

	// Get pagination query params, e.g., /feed/home?page=1&limit=20
	// We'll default to page 1, limit 20
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }

	offset := (page - 1) * limit

	grpcReq := &postPb.GetHomeFeedRequest{
		UserId:      userID,
		PageSize:    int32(limit),
		PageOffset:  int32(offset),
	}

	grpcRes, err := postClient.GetHomeFeed(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetHomeFeed failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}

	c.JSON(http.StatusOK, grpcRes.Posts)
}

// --- GIN-NATIVE HANDLER: handleGetExploreFeed ---
func handleGetExploreFeed_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	grpcReq := &postPb.GetHomeFeedRequest{
		UserId:      userID,
		PageSize:    int32(limit),
		PageOffset:  int32(offset),
	}

	grpcRes, err := postClient.GetExploreFeed(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetExploreFeed failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes.Posts)
}

// --- GIN-NATIVE HANDLER: handleGetReelsFeed ---
func handleGetReelsFeed_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	grpcReq := &postPb.GetHomeFeedRequest{
		UserId:      userID,
		PageSize:    int32(limit),
		PageOffset:  int32(offset),
	}

	grpcRes, err := postClient.GetReelsFeed(c.Request.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to GetReelsFeed failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()})
		return
	}
	c.JSON(http.StatusOK, grpcRes.Posts)
}

// --- GIN-NATIVE HANDLER: handleGetUserProfile ---
// This is a complex aggregator handler
func handleGetUserProfile_Gin(c *gin.Context) {
	selfUserID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	usernameToFind := c.Param("username")    // Get username from URL

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
		User          *pb.GetUserProfileResponse `json:"user"`
		PostCount     int64                      `json:"post_count"`
		ReelCount     int64                      `json:"reel_count"`
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
	usernameToFind := c.Param("username")
	
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
		UserId:      userRes.UserId,
		PageSize:    int32(limit),
		PageOffset:  int32(offset),
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
	usernameToFind := c.Param("username")
	
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
		UserId:      userRes.UserId,
		PageSize:    int32(limit),
		PageOffset:  int32(offset),
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

// --- GIN-NATIVE HANDLER: handleUpdateProfile ---
func handleUpdateProfile_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

	var req struct {
		Name   string `json:"name"`
		Bio    string `json:"bio"`
		Gender string `json:"gender"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	grpcReq := &pb.UpdateUserProfileRequest{
		UserId: userID,
		Name:   req.Name,
		Bio:    req.Bio,
		Gender: req.Gender,
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
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

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
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }

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
			BlockerId:  blockerID,
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
			BlockerId:  blockerID,
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
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	var req struct { Name string `json:"name"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	grpcReq := &postPb.CreateCollectionRequest{UserId: userID, Name: req.Name}
	grpcRes, err := postClient.CreateCollection(c.Request.Context(), grpcReq)
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
	c.JSON(http.StatusCreated, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleGetUserCollections ---
func handleGetUserCollections_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	grpcReq := &postPb.GetUserCollectionsRequest{UserId: userID}
	grpcRes, err := postClient.GetUserCollections(c.Request.Context(), grpcReq)
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
	c.JSON(http.StatusOK, grpcRes.Collections)
}

// --- GIN-NATIVE HANDLER: handleGetPostsInCollection ---
func handleGetPostsInCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"}); return }

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
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
	c.JSON(http.StatusOK, grpcRes.Posts)
}

// --- GIN-NATIVE HANDLER: handleSavePostToCollection ---
func handleSavePostToCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"}); return }

	var req struct { PostID int64 `json:"post_id"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'post_id'"}); return
	}

	grpcReq := &postPb.SavePostToCollectionRequest{
		UserId:       userID,
		CollectionId: collectionID,
		PostId:       req.PostID,
	}
	grpcRes, err := postClient.SavePostToCollection(c.Request.Context(), grpcReq)
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleUnsavePostFromCollection ---
func handleUnsavePostFromCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"}); return }
	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"}); return }

	grpcReq := &postPb.UnsavePostFromCollectionRequest{
		UserId:       userID,
		CollectionId: collectionID,
		PostId:       postID,
	}
	grpcRes, err := postClient.UnsavePostFromCollection(c.Request.Context(), grpcReq)
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleDeleteCollection ---
func handleDeleteCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"}); return }

	grpcReq := &postPb.DeleteCollectionRequest{UserId: userID, CollectionId: collectionID}
	grpcRes, err := postClient.DeleteCollection(c.Request.Context(), grpcReq)
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
	c.JSON(http.StatusOK, grpcRes)
}

// --- GIN-NATIVE HANDLER: handleRenameCollection ---
func handleRenameCollection_Gin(c *gin.Context) {
	userID, ok := c.Request.Context().Value(userIDKey).(int64)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get user ID from token"}); return }
	collectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"}); return }

	var req struct { NewName string `json:"new_name"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'new_name'"}); return
	}

	grpcReq := &postPb.RenameCollectionRequest{
		UserId:       userID,
		CollectionId: collectionID,
		NewName:      req.NewName,
	}
	grpcRes, err := postClient.RenameCollection(c.Request.Context(), grpcReq)
	if err != nil { grpcErr, _ := status.FromError(err); c.JSON(gRPCToHTTPStatusCode(grpcErr.Code()), gin.H{"error": grpcErr.Message()}); return }
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
		CreatorId:       creatorID,
		ParticipantIds:  req.ParticipantIDs,
		GroupName:       req.GroupName,
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
	if page < 1 { page = 1 }
	if limit < 1 { limit = 20 }
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
	if page < 1 { page = 1 }
	if limit < 1 { limit = 50 }
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
		AdminUserId:  adminID,
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
		AdminUserId:    adminID,
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