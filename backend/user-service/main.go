package main

import (
	"context"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"fmt"
	"math/rand"

	"encoding/hex" // <-- To make the token a string

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v5"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/go-redis/redis/v8"

	pb "github.com/hoshibmatchi/user-service/proto"
	"golang.org/x/crypto/bcrypt"
)

// User defines the data model as per GORM tags
type User struct {
	gorm.Model
	Name              string    `gorm:"type:varchar(100);not null"`
	Username          string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email             string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Password          string    `gorm:"type:varchar(255);not null"`
	ProfilePictureURL string    `gorm:"type:varchar(255)"`
	DateOfBirth       time.Time `gorm:"not null"`
	Gender            string    `gorm:"type:varchar(10);not null"`

	IsActive     bool `gorm:"default:true"`  // For account deactivation
	IsBanned     bool `gorm:"default:false"` // For admin to ban users
	Is2FAEnabled bool `gorm:"default:false"` // For 2FA login
	IsSubscribed bool `gorm:"default:false"` // For newsletters
}

// server struct holds our database and cache connections
type server struct {
	pb.UnimplementedUserServiceServer
	db  *gorm.DB
	rdb *redis.Client // Redis client
}

// Not secure
var jwtSecret = []byte("my-super-secret-key-that-is-not-secure")

// emailRegex for validation
var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

// OTP Function
func generateOtp() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000)) // 6-digit code
}

func main() {
	// --- Step 1: Connect to PostgreSQL ---
	//
	// THIS IS THE FIXED LINE (Line 51):
	// It now correctly declares the 'dsn' variable and uses 'UTC'.
	//
	dsn := "host=user-db user=admin password=password dbname=user_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Running database migrations...")
	db.AutoMigrate(&User{})

	// --- Step 2: Connect to Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // default DB
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	log.Println("Successfully connected to Redis.")

	// --- Step 3: Set up and start the gRPC server ---
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on port 9000: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{db: db, rdb: rdb})

	log.Println("User gRPC server is listening on port 9000...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server over port 9000: %v", err)
	}
}

// RegisterUser is the implementation of our gRPC service function
func (s *server) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	log.Println("RegisterUser request received for username:", req.Username)

	// --- Step 1: Validate OTP ---
	otpKey := "otp:" + req.Email
	storedOtp, err := s.rdb.Get(ctx, otpKey).Result()
	if err == redis.Nil {
		log.Printf("OTP not found or expired for: %s", req.Email)
		return nil, status.Error(codes.InvalidArgument, "OTP expired or not requested")
	} else if err != nil {
		log.Printf("Redis error checking OTP: %v", err)
		return nil, status.Error(codes.Internal, "Failed to verify OTP")
	}

	if storedOtp != req.OtpCode {
		log.Printf("Invalid OTP for: %s. Expected %s, got %s", req.Email, storedOtp, req.OtpCode)
		return nil, status.Error(codes.InvalidArgument, "Invalid OTP code")
	}

	// --- Step 2: Validate Business Logic (as per PDF) ---
	if len(req.Name) <= 4 {
		return nil, status.Error(codes.InvalidArgument, "Name must be more than 4 characters")
	}
	if !emailRegex.MatchString(req.Email) {
		return nil, status.Error(codes.InvalidArgument, "Invalid email format")
	}
	if len(req.Password) < 8 { // Example validation
		return nil, status.Error(codes.InvalidArgument, "Password must be at least 8 characters")
	}
	if req.Gender != "male" && req.Gender != "female" {
		return nil, status.Error(codes.InvalidArgument, "Gender must be male or female")
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid date of birth format. Use YYYY-MM-DD")
	}
	if !isAgeValid(dob, 13) {
		return nil, status.Error(codes.InvalidArgument, "You must be at least 13 years old to register")
	}

	// --- Step 3: Hash Password ---
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return nil, status.Error(codes.Internal, "Failed to process password")
	}

	// --- Step 4: Create User in Database ---
	newUser := User{
		Name:              req.Name,
		Username:          req.Username,
		Email:             req.Email,
		Password:          string(hashedPassword),
		DateOfBirth:       dob,
		Gender:            req.Gender,
		ProfilePictureURL: req.ProfilePictureUrl,
	}

	result := s.db.Create(&newUser)
	if result.Error != nil {
		log.Printf("Failed to create user in database: %v", result.Error)
		if strings.Contains(result.Error.Error(), "unique constraint") {
			if strings.Contains(result.Error.Error(), "username") {
				return nil, status.Error(codes.AlreadyExists, "Username already exists")
			}
			if strings.Contains(result.Error.Error(), "email") {
				return nil, status.Error(codes.AlreadyExists, "Email already exists")
			}
		}
		return nil, status.Error(codes.Internal, "Failed to create account")
	}

	// --- Step 5: Success ---
	s.rdb.Del(ctx, otpKey)
	log.Println("Successfully created user with ID:", newUser.ID)
	return &pb.RegisterUserResponse{
		Id:       int64(newUser.ID),
		Username: newUser.Username,
		Email:    newUser.Email,
	}, nil
}

func (s *server) LoginUser(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	var user User

	// Find user by email OR username
	err := s.db.Where("email = ? OR username = ?", req.EmailOrUsername, req.EmailOrUsername).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Invalid credentials")
	} else if err != nil {
		return nil, status.Error(codes.Internal, "Database error")
	}

	// PDF Requirement: "Only activated accounts that are not banned and not deactivated"
	if user.IsBanned {
		return nil, status.Error(codes.PermissionDenied, "This account is banned")
	}
	if !user.IsActive {
		return nil, status.Error(codes.PermissionDenied, "This account is deactivated")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		// Password doesn't match
		return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
	}

	// --- Password is correct, proceed ---

	// PDF Requirement: "If the user's account has 2FA enabled, send a verification code" [cite: 214]
	if user.Is2FAEnabled {
		// Send a 2FA code
		otpKey := "2fa:" + user.Email
		otpCode := generateOtp() // Use the same 6-digit helper
		err = s.rdb.Set(ctx, otpKey, otpCode, 5*time.Minute).Err()
		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to send 2FA code")
		}

		// --- SIMULATE SENDING 2FA EMAIL ---
		log.Printf("***********************************")
		log.Printf("2FA Code for %s: %s", user.Email, otpCode)
		log.Printf("***********************************")

		return &pb.LoginResponse{
			Message:        "Login successful. Please enter your 2FA code.",
			Is_2FaRequired: true,
		}, nil
	}

	// --- User is logged in (No 2FA) ---
	// PDF Requirement: "Implement access tokens and refresh tokens"

	// Create Access Token (short-lived)
	accessToken, err := createToken(user, 1*time.Hour) // 1 hour expiry
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to create access token")
	}

	// Create Refresh Token (long-lived)
	refreshToken, err := createToken(user, 7*24*time.Hour) // 7 day expiry
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to create refresh token")
	}

	return &pb.LoginResponse{
		Message:        "Login successful",
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		Is_2FaRequired: false,
	}, nil
}

// --- ADD THIS TOKEN HELPER FUNCTION ---
func createToken(user User, duration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(duration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func (s *server) SendRegistrationOtp(ctx context.Context, req *pb.SendOtpRequest) (*pb.SendOtpResponse, error) {
	// PDF Requirement: "Rate limit OTP resend to 1 request every 60 seconds per email" [cite: 183]
	rateLimitKey := "rate_limit:otp:" + req.Email
	err := s.rdb.Get(ctx, rateLimitKey).Err()
	if err != redis.Nil {
		// Key exists, user is rate limited
		ttl, _ := s.rdb.TTL(ctx, rateLimitKey).Result()
		return nil, status.Error(codes.ResourceExhausted, fmt.Sprintf("Please wait %d seconds before resending", int(ttl.Seconds())))
	}

	// TODO: Validate email format [cite: 174]
	// TODO: Check if email is already in use [cite: 173]

	// Generate and store OTP
	otpKey := "otp:" + req.Email
	otpCode := generateOtp()

	// PDF Requirement: "The code is valid for 5 minutes"
	err = s.rdb.Set(ctx, otpKey, otpCode, 5*time.Minute).Err()
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to store OTP")
	}

	// Set the 60-second rate limit [cite: 183]
	err = s.rdb.Set(ctx, rateLimitKey, "1", 1*time.Minute).Err()
	if err != nil {
		// Not a fatal error, just log it
		log.Printf("Failed to set rate limit key for %s", req.Email)
	}

	// --- SIMULATE SENDING EMAIL ---
	// Later, this will be a RabbitMQ message
	log.Printf("***********************************")
	log.Printf("OTP for %s: %s", req.Email, otpCode)
	log.Printf("***********************************")

	return &pb.SendOtpResponse{
		Message:          "OTP sent successfully. Please check your email (and the console).",
		RateLimitSeconds: 60,
	}, nil
}

func (s *server) Verify2FA(ctx context.Context, req *pb.Verify2FARequest) (*pb.Verify2FAResponse, error) {
	log.Printf("Verify2FA request received for: %s", req.Email)

	// --- Step 1: Validate 2FA OTP ---
	otpKey := "2fa:" + req.Email
	storedOtp, err := s.rdb.Get(ctx, otpKey).Result()
	if err == redis.Nil {
		log.Printf("2FA code not found or expired for: %s", req.Email)
		return nil, status.Error(codes.Unauthenticated, "Invalid or expired 2FA code")
	} else if err != nil {
		log.Printf("Redis error checking 2FA OTP: %v", err)
		return nil, status.Error(codes.Internal, "Failed to verify 2FA code")
	}

	if storedOtp != req.OtpCode {
		log.Printf("Invalid 2FA code for: %s. Expected %s, got %s", req.Email, storedOtp, req.OtpCode)
		return nil, status.Error(codes.Unauthenticated, "Invalid or expired 2FA code")
	}

	// --- Step 2: Code is valid, get user and generate tokens ---
	var user User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// This should be rare, but good to check
		log.Printf("Failed to find user %s after 2FA success", req.Email)
		return nil, status.Error(codes.Internal, "Failed to retrieve user data")
	}

	// Code is correct, delete it from Redis so it can't be reused
	s.rdb.Del(ctx, otpKey)

	// --- Step 3: Create tokens (using the helper we already have) ---
	accessToken, err := createToken(user, 1*time.Hour) // 1 hour expiry
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to create access token")
	}

	refreshToken, err := createToken(user, 7*24*time.Hour) // 7 day expiry
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to create refresh token")
	}

	log.Printf("2FA verification successful for: %s", req.Email)

	return &pb.Verify2FAResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// --- ADD GPRC FUNCTION 1: SendPasswordReset ---
func (s *server) SendPasswordReset(ctx context.Context, req *pb.SendPasswordResetRequest) (*pb.SendPasswordResetResponse, error) {
	var user User
	
	// PDF Requirement: "Only registered emails that are not banned can be used" [cite: 324]
	err := s.db.Where("email = ?", req.Email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		// Don't tell the user if the email exists or not.
		return &pb.SendPasswordResetResponse{Message: "If an account with that email exists, a reset link has been sent."}, nil
	}
	if user.IsBanned {
		return &pb.SendPasswordResetResponse{Message: "If an account with that email exists, a reset link has been sent."}, nil
	}

	// Generate a secure token
	token, err := generateSecureToken(32) // 32 bytes = 64-char string
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to generate reset token")
	}

	// Store token in Redis with a 15-minute expiry
	tokenKey := "reset_token:" + req.Email
	if err := s.rdb.Set(ctx, tokenKey, token, 15*time.Minute).Err(); err != nil {
		return nil, status.Error(codes.Internal, "Failed to store reset token")
	}
	
	// --- SIMULATE SENDING RESET EMAIL ---
	log.Printf("***********************************")
	log.Printf("Password Reset Token for %s: %s", req.Email, token)
	log.Printf("***********************************")
	
	return &pb.SendPasswordResetResponse{Message: "If an account with that email exists, a reset link has been sent."}, nil
}

// --- ADD GPRC FUNCTION 2: ResetPassword ---
func (s *server) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	// --- Step 1: Verify the token ---
	tokenKey := "reset_token:" + req.Email
	storedToken, err := s.rdb.Get(ctx, tokenKey).Result()
	if err == redis.Nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid or expired reset token")
	} else if err != nil {
		return nil, status.Error(codes.Internal, "Failed to verify token")
	}

	if storedToken != req.OtpCode {
		return nil, status.Error(codes.InvalidArgument, "Invalid or expired reset token")
	}

	// --- Step 2: Token is good, find user ---
	var user User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to find user")
	}

	// --- Step 3: Validate new password ---
	// PDF Requirement: "Validate the new password can't be the same as the old one" [cite: 326]
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword))
	if err == nil {
		// 'err == nil' means the passwords *match*, which is an error
		return nil, status.Error(codes.InvalidArgument, "New password cannot be the same as the old one")
	}

	// --- Step 4: Hash and save new password ---
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to hash new password")
	}

	if err := s.db.Model(&user).Update("password", string(newHashedPassword)).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to update password")
	}

	// --- Step 5: Success. Delete the token. ---
	s.rdb.Del(ctx, tokenKey)
	log.Printf("Password successfully reset for %s", req.Email)

	return &pb.ResetPasswordResponse{Message: "Password has been reset successfully. You can now log in."}, nil
}

// Helper function for age validation
func isAgeValid(birthDate time.Time, minAge int) bool {
	today := time.Now()
	age := today.Year() - birthDate.Year()
	if today.Month() < birthDate.Month() || (today.Month() == birthDate.Month() && today.Day() < birthDate.Day()) {
		age--
	}
	return age >= minAge
}
