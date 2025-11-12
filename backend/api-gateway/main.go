package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"strconv"
	"context"

	// Import the gRPC client connection library
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v5"

	// Import the generated proto code for user-service
	// This path MUST match the 'go_package' option in your user.proto
	pb "github.com/hoshibmatchi/user-service/proto"
	postPb "github.com/hoshibmatchi/post-service/proto"
	storyPb "github.com/hoshibmatchi/story-service/proto"
	mediaPb "github.com/hoshibmatchi/media-service/proto"
)

// client will hold the persistent gRPC connection
var client pb.UserServiceClient
var postClient postPb.PostServiceClient
var storyClient storyPb.StoryServiceClient
var mediaClient mediaPb.MediaServiceClient

// ADD THIS (must match user-service)
// TODO: Load this from an environment variable, not hardcoded
var jwtSecret = []byte("my-super-secret-key-that-is-not-secure")

type contextKey string
const userIDKey contextKey = "userID"

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// 2. The header should be in the format "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		// 3. Parse and validate the token
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil // Use our shared secret
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// 4. Extract the user_id from the token's "claims"
		// We must convert it from float64 (default for JSON numbers) to int64
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}
		userID := int64(userIDFloat)

		// 5. Add the userID to the request's context
		ctx := context.WithValue(r.Context(), userIDKey, userID)

		// 6. Call the *next* handler (e.g., handleCreatePost) with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	// --- Connect to all gRPC Services ---
	mustConnect(&client, "user-service:9000")
	mustConnect(&postClient, "post-service:9001")
	mustConnect(&storyClient, "story-service:9002")
	mustConnect(&mediaClient, "media-service:9005")
	
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
		authRoutes.POST("/login", gin.WrapF(handleLogin))
		authRoutes.POST("/login/verify-2fa", gin.WrapF(handleVerify2FA))
		authRoutes.POST("/password-reset/request", gin.WrapF(handleSendPasswordReset))
		authRoutes.POST("/password-reset/submit", gin.WrapF(handleResetPassword))
	}

	// Protected routes (JWT auth required)
	protected := router.Group("/")
	protected.Use(GinAuthMiddleware())
	{
		// These handlers ALSO don't need URL params, gin.WrapF is fine.
		protected.POST("/posts", gin.WrapF(handleCreatePost))
		protected.POST("/stories", gin.WrapF(handleCreateStory))
		protected.POST("/comments", gin.WrapF(handleCreateComment))
		protected.GET("/media/upload-url", gin.WrapF(handleGetUploadURL))

		// --- THIS IS THE FIX ---
		// These routes NEED URL params, so they get native Gin handlers.
		protected.POST("/users/:id/follow", handleFollowUser_Gin)
		protected.DELETE("/users/:id/follow", handleFollowUser_Gin)

		protected.POST("/posts/:id/like", handlePostLike_Gin)
		protected.DELETE("/posts/:id/like", handlePostLike_Gin)

		protected.POST("/stories/:id/like", handleStoryLike_Gin)
		protected.DELETE("/stories/:id/like", handleStoryLike_Gin)

		protected.DELETE("/comments/:id", handleDeleteComment_Gin)
	}

	log.Println("API Gateway starting on port 8000...")
	if err := router.Run(":8000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

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
		
		// Add the userID to the request context
		ctx := context.WithValue(c.Request.Context(), userIDKey, int64(userIDFloat))
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
		Name              string `json:"name"`
		Username          string `json:"username"`
		Email             string `json:"email"`
		Password          string `json:"password"`
		DateOfBirth       string `json:"date_of_birth"`
		Gender            string `json:"gender"`
		ProfilePictureURL string `json:"profile_picture_url"`
		OtpCode           string `json:"otp_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 3. Call the gRPC service
	grpcReq := &pb.RegisterUserRequest{
		Name:              req.Name,
		Username:          req.Username,
		Email:             req.Email,
		Password:          req.Password,
		DateOfBirth:       req.DateOfBirth,
		Gender:            req.Gender,
		ProfilePictureUrl: req.ProfilePictureURL,
		OtpCode:           req.OtpCode,
	}

	res, err := client.RegisterUser(r.Context(), grpcReq)
	if err != nil {
		// --- THIS IS THE FIX ---
		// We now translate the gRPC error into a proper HTTP status
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		// --- END FIX ---
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

// --- HANDLER 3: handleCreatePost ---
func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(userIDKey).(int64)

	var req struct {
		Caption          string   `json:"caption"`
		MediaURLs        []string `json:"media_urls"`
		CommentsDisabled bool     `json:"comments_disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &postPb.CreatePostRequest{
		AuthorId:         userID,
		Caption:          req.Caption,
		MediaUrls:        req.MediaURLs,
		CommentsDisabled: req.CommentsDisabled,
	}

	grpcRes, err := postClient.CreatePost(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to post-service failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(grpcRes.Post)
}

func handleCreateStory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(userIDKey).(int64)

	var req struct {
		MediaURL string `json:"media_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &storyPb.CreateStoryRequest{
		AuthorId: userID,
		MediaUrl: req.MediaURL,
	}

	grpcRes, err := storyClient.CreateStory(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		log.Printf("gRPC call to story-service failed (%s): %v", grpcErr.Code(), grpcErr.Message())
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(grpcRes.Story)
}

// --- HANDLER: handleCreateComment ---
func handleCreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	
	userID := r.Context().Value(userIDKey).(int64)

	var req struct {
		PostID          int64  `json:"post_id"`
		Content         string `json:"content"`
		ParentCommentID int64  `json:"parent_comment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// FIX: Was pb.CommentOnPostRequest, now postPb.CommentOnPostRequest
	grpcReq := &postPb.CommentOnPostRequest{
		UserId:          userID,
		PostId:          req.PostID,
		Content:         req.Content,
		ParentCommentId: req.ParentCommentID,
	}
	
	grpcRes, err := postClient.CommentOnPost(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(grpcRes)
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

func handleGetUploadURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(userIDKey).(int64) // Get from JWT

	// Get query params, e.g., /media/upload-url?filename=foo.jpg&type=image/jpeg
	filename := r.URL.Query().Get("filename")
	contentType := r.URL.Query().Get("type")

	if filename == "" || contentType == "" {
		http.Error(w, "Missing 'filename' or 'type' query parameters", http.StatusBadRequest)
		return
	}

	grpcReq := &mediaPb.GetUploadURLRequest{
		Filename:    filename,
		ContentType: contentType,
		UserId:      userID,
	}

	grpcRes, err := mediaClient.GetUploadURL(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(grpcRes)
}