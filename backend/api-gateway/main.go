package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"strconv"
	"context"

	// Import the gRPC client connection library
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
)

// client will hold the persistent gRPC connection
var client pb.UserServiceClient
var postClient postPb.PostServiceClient
var storyClient storyPb.StoryServiceClient

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
	// Connect to the gRPC user-service
	// "user-service:9000" works because Docker Compose provides DNS [cite: 531]
	// We use insecure credentials because it's internal Docker traffic
	conn, err := grpc.Dial("user-service:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer conn.Close()

	// Create a new client stub
	client = pb.NewUserServiceClient(conn)

	// Post
	postConn, err := grpc.Dial("post-service:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to post-service: %v", err)
	}
	defer postConn.Close()
	postClient = postPb.NewPostServiceClient(postConn)
	log.Println("Successfully connected to post-service")

	// Story
	storyConn, err := grpc.Dial("story-service:9002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to story-service: %v", err)
	}
	defer storyConn.Close()
	storyClient = storyPb.NewStoryServiceClient(storyConn)
	log.Println("Successfully connected to story-service")


	// Your existing health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("API Gateway is running"))
	})

	// NEW: Register handler
	http.HandleFunc("/auth/register", handleRegister)
	http.HandleFunc("/auth/send-otp", handleSendOtp)
	// TODO: Add /auth/login, /auth/reset routes as per Phase 1

	// Login
	http.HandleFunc("/auth/login", handleLogin)

	// 2FA
	http.HandleFunc("/auth/login/verify-2fa", handleVerify2FA)

	// Post & PW Reset 
	http.HandleFunc("/auth/password-reset/request", handleSendPasswordReset)
	http.HandleFunc("/auth/password-reset/submit", handleResetPassword)

	http.Handle("/posts", authMiddleware(http.HandlerFunc(handleCreatePost)))
	http.Handle("/stories", authMiddleware(http.HandlerFunc(handleCreateStory)))
	http.Handle("/posts/{id}/like", authMiddleware(http.HandlerFunc(handlePostLike)))
	http.Handle("/comments", authMiddleware(http.HandlerFunc(handleCreateComment)))
	http.Handle("/comments/{id}", authMiddleware(http.HandlerFunc(handleDeleteComment)))
	http.Handle("/stories/{id}/like", authMiddleware(http.HandlerFunc(handleStoryLike)))


	log.Println("API Gateway starting on port 8000...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
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

// --- HANDLER: handlePostLike (Handles POST for Like, DELETE for Unlike) ---
func handlePostLike(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int64)

	postIDStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/posts/"), "/like")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}
	
	if r.Method == http.MethodPost {
		// --- Like Post ---
		req := &postPb.LikePostRequest{UserId: userID, PostId: postID} 
		res, err := postClient.LikePost(r.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)

	} else if r.Method == http.MethodDelete {
		// --- Unlike Post ---
		req := &postPb.LikePostRequest{UserId: userID, PostId: postID}
		res, err := postClient.UnlikePost(r.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
		
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
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

// --- HANDLER: handleDeleteComment ---
func handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	
	userID := r.Context().Value(userIDKey).(int64)

	commentIDStr := strings.TrimPrefix(r.URL.Path, "/comments/")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}
	
	// FIX: Was pb.DeleteCommentRequest, now postPb.DeleteCommentRequest
	grpcReq := &postPb.DeleteCommentRequest{
		UserId:    userID,
		CommentId: commentID,
	}
	
	grpcRes, err := postClient.DeleteComment(r.Context(), grpcReq)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(grpcRes)
}

// --- HANDLER: handleStoryLike (Handles POST for Like, DELETE for Unlike) ---
func handleStoryLike(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int64)

	storyIDStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/stories/"), "/like")
	storyID, err := strconv.ParseInt(storyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}
	
	if r.Method == http.MethodPost {
		// --- Like Story ---
		// This one was already correct, using storyPb
		req := &storyPb.LikeStoryRequest{UserId: userID, StoryId: storyID}
		res, err := storyClient.LikeStory(r.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)

	} else if r.Method == http.MethodDelete {
		// --- Unlike Story ---
		// This one was also correct, using storyPb
		req := &storyPb.UnlikeStoryRequest{UserId: userID, StoryId: storyID}
		res, err := storyClient.UnlikeStory(r.Context(), req)
		if err != nil {
			grpcErr, _ := status.FromError(err)
			http.Error(w, grpcErr.Message(), gRPCToHTTPStatusCode(grpcErr.Code()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
		
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}