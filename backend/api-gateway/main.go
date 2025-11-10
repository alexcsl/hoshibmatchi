package main

import (
	"encoding/json"
	"log"
	"net/http"

	// Import the gRPC client connection library
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
    // TODO: Add /auth/login, /auth/reset routes as per Phase 1 

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
	// This struct is a *temporary* DTO (Data Transfer Object)
	// It's good practice to have this separate from the gRPC proto struct
	var req struct {
		Name        string `json:"name"`
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		DateOfBirth string `json:"date_of_birth"` // "YYYY-MM-DD"
		Gender      string `json:"gender"`
		// The PDF also requires Profile Picture and OTP 
		// We'll fix that in the proto in the next step.
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 3. Call the gRPC service
	// The gRPC client's RegisterUser function handles the network call
	grpcReq := &pb.RegisterUserRequest{
		Name:        req.Name,
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DateOfBirth: req.DateOfBirth,
		Gender:      req.Gender,
	}

	res, err := client.RegisterUser(r.Context(), grpcReq)
	if err != nil {
		// TODO: Translate gRPC errors (e.g., codes.AlreadyExists)
		// into proper HTTP status codes (e.g., http.StatusConflict)
		log.Printf("gRPC call failed: %v", err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	// 4. Send the successful JSON response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}