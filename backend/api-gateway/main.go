package main

import (
	"encoding/json"
	"log"
	"net/http"

	// Import the gRPC client connection library
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	// Import the generated proto code for user-service
	// This path MUST match the 'go_package' option in your user.proto
	pb "github.com/hoshibmatchi/user-service/proto"
)

// client will hold the persistent gRPC connection
var client pb.UserServiceClient

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
