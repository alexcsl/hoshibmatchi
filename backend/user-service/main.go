package main

import (
	"context"
	"log"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"fmt"
	"math/rand"

	"encoding/hex"
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/golang-jwt/jwt/v5"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/go-redis/redis/v8"

	pb "github.com/hoshibmatchi/user-service/proto"
	amqp "github.com/rabbitmq/amqp091-go"
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
	Bio               string    `gorm:"type:varchar(255)"`

	IsActive     bool   `gorm:"default:false"` // For account deactivation
	IsBanned     bool   `gorm:"default:false"` // For admin to ban users
	Is2FAEnabled bool   `gorm:"default:false"` // For 2FA login
	IsSubscribed bool   `gorm:"default:false"` // For newsletters
	IsPrivate    bool   `gorm:"default:false"` // For private accounts
	Role         string `gorm:"type:varchar(10);default:'user'"`
	IsVerified   bool   `gorm:"default:false"` // For verified checkmark
	Provider     string `gorm:"type:varchar(20);default:'local'"`
	ProviderID   string `gorm:"type:varchar(255);index"`
}

// Follow defines the relationship between two users
type Follow struct {
	// Composite primary key (follower_id, following_id)
	FollowerID  int64 `gorm:"primaryKey"` // The user doing the following
	FollowingID int64 `gorm:"primaryKey"` // The user being followed
	CreatedAt   time.Time
}

// server struct holds our database and cache connections
type server struct {
	pb.UnimplementedUserServiceServer
	db     *gorm.DB
	rdb    *redis.Client // Redis client
	amqpCh *amqp.Channel
}

// Block defines a block relationship
type Block struct {
	// Composite primary key (blocker_id, blocked_id)
	BlockerID int64 `gorm:"primaryKey"` // The user doing the blocking
	BlockedID int64 `gorm:"primaryKey"` // The user being blocked
	CreatedAt time.Time
}

// For verification requests
type VerificationRequest struct {
	gorm.Model
	UserID         int64  `gorm:"index;unique"` // A user can only have one open request
	IdCardNumber   string // National ID card number
	FacePictureURL string
	Reason         string
	Status         string `gorm:"type:varchar(10);default:'pending'"` // 'pending', 'approved', 'rejected'
	ResolvedByID   int64  // Admin User ID who resolved it
}

// searchResult is a helper struct for sorting
type searchResult struct {
	user       User
	similarity float64
}

// Not secure
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// emailRegex for validation
var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

var googleOauthConfig *oauth2.Config

// OTP Function
func generateOtp() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000)) // 6-digit code
}

func main() {
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	// Initialize Google OAuth
	googleOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	// --- Step 1: Connect to PostgreSQL ---
	dsn := "host=user-db user=admin password=password dbname=user_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Running database migrations...")
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Follow{})
	db.AutoMigrate(&Block{})
	db.AutoMigrate(&VerificationRequest{})

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

	// --- Step 3: Connect to RabbitMQ (with retries) ---
	var amqpConn *amqp.Connection
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		amqpURI := os.Getenv("RABBITMQ_URI")
		if amqpURI == "" {
			amqpURI = "amqp://guest:guest@rabbitmq:5672/" // Default
		}
		amqpConn, err = amqp.Dial(amqpURI)
		if err == nil {
			log.Println("Successfully connected to RabbitMQ")
			break
		}
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		log.Printf("Retrying in %v... (%d/%d)", retryDelay, i+1, maxRetries)
		time.Sleep(retryDelay)
	}
	if amqpConn == nil {
		log.Fatalf("Could not connect to RabbitMQ after %d retries", maxRetries)
	}
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
	defer amqpCh.Close()

	_, err = amqpCh.QueueDeclare(
		"email_queue", // queue name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare email_queue: %v", err)
	}
	log.Println("RabbitMQ email_queue declared")

	// --- Step 4: Set up and start the gRPC server ---
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on port 9000: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{db: db, rdb: rdb, amqpCh: amqpCh})

	log.Println("User gRPC server is listening on port 9000...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server over port 9000: %v", err)
	}
}

// RegisterUser is the implementation of our gRPC service function
func (s *server) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	log.Println("RegisterUser request received for username:", req.Username)

	// --- Step 1: Validate Business Logic (as per PDF) ---
	if req.Password != req.ConfirmPassword {
		return nil, status.Error(codes.InvalidArgument, "Passwords do not match")
	}
	if len(req.Name) <= 4 {
		return nil, status.Error(codes.InvalidArgument, "Name must be more than 4 characters")
	}
	// Username validation
	if len(req.Username) < 3 || len(req.Username) > 30 {
		return nil, status.Error(codes.InvalidArgument, "Username must be between 3 and 30 characters")
	}
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(req.Username) {
		return nil, status.Error(codes.InvalidArgument, "Username can only contain letters, numbers, and underscores")
	}
	if !emailRegex.MatchString(req.Email) {
		return nil, status.Error(codes.InvalidArgument, "Invalid email format")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
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

	// --- Step 2: Check for unique constraints *before* sending OTP ---
	var existingUser int64
	s.db.Model(&User{}).Where("username = ?", req.Username).Count(&existingUser)
	if existingUser > 0 {
		return nil, status.Error(codes.AlreadyExists, "Username already exists")
	}
	s.db.Model(&User{}).Where("email = ?", req.Email).Count(&existingUser)
	if existingUser > 0 {
		return nil, status.Error(codes.AlreadyExists, "Email already exists")
	}

	// --- Step 3: Hash Password ---
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return nil, status.Error(codes.Internal, "Failed to process password")
	}

	// --- Step 4: Create User in Database (as INACTIVE) ---
	newUser := User{
		Name:        req.Name,
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hashedPassword),
		DateOfBirth: dob,
		Gender:      req.Gender,
		IsActive:    false, // User is inactive until OTP is verified
	}

	result := s.db.Create(&newUser)
	if result.Error != nil {
		log.Printf("Failed to create user in database: %v", result.Error)
		return nil, status.Error(codes.Internal, "Failed to create account")
	}

	// --- Step 5: Send the OTP ---
	// We can just call our other gRPC function internally.
	// It already has the rate-limiting and email-sending logic.
	_, err = s.SendRegistrationOtp(ctx, &pb.SendOtpRequest{Email: req.Email})
	if err != nil {
		// If OTP fails, we don't need to roll back the user creation,
		// they can just request another one.
		log.Printf("Failed to send initial OTP for %s: %v", req.Email, err)
	}

	// --- Step 6: Success ---
	log.Println("Successfully created inactive user with ID:", newUser.ID)
	return &pb.RegisterUserResponse{
		Id:       int64(newUser.ID),
		Username: newUser.Username,
		Email:    newUser.Email,
		Message:  "Registration successful. Please check your email for an OTP code.",
	}, nil
}

// validatePassword checks for at least 4 rules
func validatePassword(password string) error {
	var (
		hasMinLen  = len(password) >= 8
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString(password)
		hasNumber  = regexp.MustCompile(`[0-9]`).MatchString(password)
		hasSpecial = regexp.MustCompile(`[\W_]`).MatchString(password)
	)

	rulesMet := 0
	if hasMinLen {
		rulesMet++
	}
	if hasUpper {
		rulesMet++
	}
	if hasLower {
		rulesMet++
	}
	if hasNumber {
		rulesMet++
	}
	if hasSpecial {
		rulesMet++
	}

	// PDF requires "at least 4 (four) different validations"
	if rulesMet >= 4 {
		return nil
	}

	return status.Error(codes.InvalidArgument, "Password must meet at least 4 of the following rules: minimum 8 characters, one uppercase, one lowercase, one number, one special character")
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
		"role":     user.Role,
		"exp":      time.Now().Add(duration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func (s *server) SendRegistrationOtp(ctx context.Context, req *pb.SendOtpRequest) (*pb.SendOtpResponse, error) {
	// --- Step 1: Check if user exists and is *not* active ---
	var user User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err == gorm.ErrRecordNotFound {
		// Don't reveal if user exists
		return nil, status.Error(codes.NotFound, "Cannot send OTP to this email.")
	}
	if user.IsActive {
		return nil, status.Error(codes.AlreadyExists, "This account is already verified.")
	}

	// --- Step 2: Rate Limit Check ---
	rateLimitKey := "rate_limit:otp:" + req.Email
	err := s.rdb.Get(ctx, rateLimitKey).Err()
	if err != redis.Nil {
		// Key exists, user is rate limited
		ttl, _ := s.rdb.TTL(ctx, rateLimitKey).Result()
		return nil, status.Error(codes.ResourceExhausted, fmt.Sprintf("Please wait %d seconds before resending", int(ttl.Seconds())))
	}

	// --- Step 3: Generate, store, and publish OTP ---
	otpKey := "otp:" + req.Email
	otpCode := generateOtp()

	// Code is valid for 5 minutes [cite: 550]
	err = s.rdb.Set(ctx, otpKey, otpCode, 5*time.Minute).Err()
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to store OTP")
	}

	// Set the 60-second rate limit [cite: 552]
	err = s.rdb.Set(ctx, rateLimitKey, "1", 1*time.Minute).Err()
	if err != nil {
		log.Printf("Failed to set rate limit key for %s", req.Email)
	}

	// Publish to RabbitMQ for email-service
	emailBody, _ := json.Marshal(map[string]string{
		"to":      req.Email,
		"type":    "registration_otp",
		"otpCode": otpCode,
	})
	if err := s.publishToQueue(ctx, "email_queue", emailBody); err != nil {
		log.Printf("Failed to publish OTP email to queue: %v", err)
	}

	return &pb.SendOtpResponse{
		Message:          "OTP sent successfully. Please check your email.",
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

func (s *server) publishToQueue(ctx context.Context, queueName string, body []byte) error {
	return s.amqpCh.PublishWithContext(
		ctx,
		"",        // exchange (default)
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // Make message durable
			Body:         body,
		},
	)
}

// --- ADD GPRC FUNCTION 1: SendPasswordReset ---
func (s *server) SendPasswordReset(ctx context.Context, req *pb.SendPasswordResetRequest) (*pb.SendPasswordResetResponse, error) {
	var user User

	// PDF Requirement: "Only registered emails that are not banned can be used"
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

	// --- Step 4: Publish to RabbitMQ for email-service ---
	emailBody, _ := json.Marshal(map[string]string{
		"to":    req.Email,
		"type":  "password_reset",
		"token": token,
	})
	if err := s.publishToQueue(ctx, "email_queue", emailBody); err != nil {
		log.Printf("Failed to publish reset email to queue: %v", err)
		// Don't fail the request, just log it
	}

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
	// PDF Requirement: "Validate the new password can't be the same as the old one"
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

// --- GPRC 3: GetUserData ---
func (s *server) GetUserData(ctx context.Context, req *pb.GetUserDataRequest) (*pb.GetUserDataResponse, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("user:profile:%d", req.UserId)
	cachedData, err := s.rdb.Get(ctx, cacheKey).Result()

	if err == nil {
		// Cache hit - unmarshal and return
		var response pb.GetUserDataResponse
		if json.Unmarshal([]byte(cachedData), &response) == nil {
			log.Printf("Cache hit for user %d", req.UserId)
			return &response, nil
		}
	}

	// Cache miss - query database
	var user User
	if err := s.db.First(&user, req.UserId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "User not found")
	} else if err != nil {
		return nil, status.Error(codes.Internal, "Database error")
	}

	response := &pb.GetUserDataResponse{
		Username:          user.Username,
		ProfilePictureUrl: user.ProfilePictureURL,
		IsVerified:        user.IsVerified,
	}

	// Store in cache with 15 minute TTL
	if responseJSON, err := json.Marshal(response); err == nil {
		s.rdb.Set(ctx, cacheKey, responseJSON, 15*time.Minute)
	}

	return response, nil
}

// --- GPRC: FollowUser ---
func (s *server) FollowUser(ctx context.Context, req *pb.FollowUserRequest) (*pb.FollowUserResponse, error) {
	// 1. Prevent user from following themselves
	if req.FollowerId == req.FollowingId {
		return nil, status.Error(codes.InvalidArgument, "You cannot follow yourself")
	}

	// 2. Check if the user to be followed exists
	var userToFollow User
	if err := s.db.First(&userToFollow, req.FollowingId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "The user you are trying to follow does not exist")
	}

	// 3. Create the follow relationship
	follow := Follow{
		FollowerID:  req.FollowerId,
		FollowingID: req.FollowingId,
	}

	if result := s.db.Create(&follow); result.Error != nil {
		if strings.Contains(result.Error.Error(), "unique constraint") {
			return nil, status.Error(codes.AlreadyExists, "You are already following this user")
		}
		return nil, status.Error(codes.Internal, "Failed to follow user")
	}

	msgBody, _ := json.Marshal(map[string]interface{}{
		"type":      "user.followed",
		"actor_id":  req.FollowerId,
		"user_id":   req.FollowingId, // The user to be notified
		"entity_id": req.FollowerId,  // The entity is the follower
	})
	s.publishToQueue(ctx, "notification_queue", msgBody)

	log.Printf("User %d is now following User %d", req.FollowerId, req.FollowingId)

	return &pb.FollowUserResponse{Message: "Successfully followed user"}, nil
}

// --- GPRC: UnfollowUser ---
func (s *server) UnfollowUser(ctx context.Context, req *pb.UnfollowUserRequest) (*pb.UnfollowUserResponse, error) {
	follow := Follow{
		FollowerID:  req.FollowerId,
		FollowingID: req.FollowingId,
	}

	if result := s.db.Delete(&follow); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to unfollow user")
	} else if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "You are not following this user")
	}

	log.Printf("User %d has unfollowed User %d", req.FollowerId, req.FollowingId)

	return &pb.UnfollowUserResponse{Message: "Successfully unfollowed user"}, nil
}

// --- GPRC: IsFollowing ---
func (s *server) IsFollowing(ctx context.Context, req *pb.IsFollowingRequest) (*pb.IsFollowingResponse, error) {
	var count int64
	err := s.db.Model(&Follow{}).
		Where("follower_id = ? AND following_id = ?", req.FollowerId, req.FollowingId).
		Count(&count).Error

	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to check follow status")
	}

	return &pb.IsFollowingResponse{IsFollowing: count > 0}, nil
}

// --- GPRC: GetFollowingList ---
func (s *server) GetFollowingList(ctx context.Context, req *pb.GetFollowingListRequest) (*pb.GetFollowingListResponse, error) {
	var followingIDs []int64

	// Find all 'Follow' records where the follower_id is our user
	// Then, select only the 'following_id' column
	err := s.db.Model(&Follow{}).
		Where("follower_id = ?", req.UserId).
		Pluck("following_id", &followingIDs).Error

	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve following list")
	}

	// Also add the user's own ID to the list
	// This ensures a user always sees their *own* posts in their feed
	followingIDs = append(followingIDs, req.UserId)

	return &pb.GetFollowingListResponse{
		FollowingUserIds: followingIDs,
	}, nil
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

// --- GPRC: GetUserProfile ---
func (s *server) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest) (*pb.GetUserProfileResponse, error) {
	var user User

	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "User not found")
	} else if err != nil {
		return nil, status.Error(codes.Internal, "Database error")
	}

	var followerCount int64
	var followingCount int64
	var mutualFollowerCount int64
	var isFollowedBySelf bool

	// 2. Get follower count
	s.db.Model(&Follow{}).Where("following_id = ?", user.ID).Count(&followerCount)

	// 3. Get following count
	s.db.Model(&Follow{}).Where("follower_id = ?", user.ID).Count(&followingCount)

	// 4. Check if the requestor is following this user
	// --- THIS IS THE FIX ---
	if req.SelfUserId != int64(user.ID) { // Cast user.ID to int64
		var followCheck int64
		// And cast here as well
		s.db.Model(&Follow{}).Where("follower_id = ? AND following_id = ?", req.SelfUserId, int64(user.ID)).Count(&followCheck)
		isFollowedBySelf = (followCheck > 0)
	}
	// --- END FIX ---

	mutualFollowerCount = 0

	return &pb.GetUserProfileResponse{
		UserId:              int64(user.ID), // Also cast here
		Name:                user.Name,
		Username:            user.Username,
		Bio:                 user.Bio,
		ProfilePictureUrl:   user.ProfilePictureURL,
		IsVerified:          false,
		FollowerCount:       followerCount,
		FollowingCount:      followingCount,
		IsFollowedBySelf:    isFollowedBySelf,
		MutualFollowerCount: mutualFollowerCount,
		Gender:              user.Gender,
		IsPrivate:           user.IsPrivate,
	}, nil
}

// --- GPRC: UpdateUserProfile ---
func (s *server) UpdateUserProfile(ctx context.Context, req *pb.UpdateUserProfileRequest) (*pb.GetUserProfileResponse, error) {
	// 1. Find the user
	var user User
	if err := s.db.First(&user, req.UserId).Error; err != nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	// 2. Validate new data (as per PDF)
	if len(req.Name) <= 4 {
		return nil, status.Error(codes.InvalidArgument, "Name must be more than 4 characters")
	}
	if len(req.Bio) > 150 {
		return nil, status.Error(codes.InvalidArgument, "Bio must not exceed 150 characters")
	}
	if req.Gender != "male" && req.Gender != "female" {
		return nil, status.Error(codes.InvalidArgument, "Gender must be male or female")
	}

	// 3. Update the fields
	user.Name = req.Name
	user.Bio = req.Bio
	user.Gender = req.Gender

	if err := s.db.Save(&user).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to update profile")
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:profile:%d", req.UserId)
	s.rdb.Del(ctx, cacheKey)

	log.Printf("User profile updated for user_id: %d", req.UserId)

	// 4. Return the new, updated profile data (by calling our other function)
	// This is good practice to avoid duplicating logic
	return s.GetUserProfile(ctx, &pb.GetUserProfileRequest{
		Username:   user.Username,
		SelfUserId: req.UserId,
	})
}

// --- GPRC: SetAccountPrivacy ---
func (s *server) SetAccountPrivacy(ctx context.Context, req *pb.SetAccountPrivacyRequest) (*pb.SetAccountPrivacyResponse, error) {
	// We can use a simple 'Update' for this one field
	if err := s.db.Model(&User{}).Where("id = ?", req.UserId).Update("is_private", req.IsPrivate).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to update account privacy")
	}

	log.Printf("Account privacy set to %t for user_id: %d", req.IsPrivate, req.UserId)

	return &pb.SetAccountPrivacyResponse{Message: "Account privacy updated successfully"}, nil
}

// --- GPRC: BlockUser ---
func (s *server) BlockUser(ctx context.Context, req *pb.BlockUserRequest) (*pb.BlockUserResponse, error) {
	if req.BlockerId == req.BlockedId {
		return nil, status.Error(codes.InvalidArgument, "You cannot block yourself")
	}

	// Check if the user to be blocked exists
	var userToBlock User
	if err := s.db.First(&userToBlock, req.BlockedId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "The user you are trying to block does not exist")
	}

	// Create the block relationship
	block := Block{
		BlockerID: req.BlockerId,
		BlockedID: req.BlockedId,
	}

	// Use a database transaction to ensure all or nothing
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the block
		if err := tx.Create(&block).Error; err != nil {
			if strings.Contains(err.Error(), "unique constraint") {
				return status.Error(codes.AlreadyExists, "You are already blocking this user")
			}
			return status.Error(codes.Internal, "Failed to block user")
		}

		// 2. The blocker unfollows the blocked user
		tx.Delete(&Follow{}, "follower_id = ? AND following_id = ?", req.BlockerId, req.BlockedId)

		// 3. The blocked user unfollows the blocker
		tx.Delete(&Follow{}, "follower_id = ? AND following_id = ?", req.BlockedId, req.BlockerId)

		return nil // Commit
	})

	if err != nil {
		// Transaction failed
		return nil, err
	}

	log.Printf("User %d is now blocking User %d", req.BlockerId, req.BlockedId)

	return &pb.BlockUserResponse{Message: "Successfully blocked user"}, nil
}

// --- GPRC: UnblockUser ---
func (s *server) UnblockUser(ctx context.Context, req *pb.UnblockUserRequest) (*pb.UnblockUserResponse, error) {
	block := Block{
		BlockerID: req.BlockerId,
		BlockedID: req.BlockedId,
	}

	if result := s.db.Delete(&block); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to unblock user")
	} else if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "You are not blocking this user")
	}

	log.Printf("User %d has unblocked User %d", req.BlockerId, req.BlockedId)

	return &pb.UnblockUserResponse{Message: "Successfully unblocked user"}, nil
}

// --- ADD NEW GRPC FUNCTION: VerifyRegistrationOtp ---
func (s *server) VerifyRegistrationOtp(ctx context.Context, req *pb.VerifyRegistrationOtpRequest) (*pb.VerifyRegistrationOtpResponse, error) {
	log.Printf("VerifyRegistrationOtp request received for: %s", req.Email)

	// --- Step 1: Validate OTP ---
	otpKey := "otp:" + req.Email
	storedOtp, err := s.rdb.Get(ctx, otpKey).Result()
	if err == redis.Nil {
		log.Printf("OTP not found or expired for: %s", req.Email)
		return nil, status.Error(codes.InvalidArgument, "Invalid or expired OTP code")
	} else if err != nil {
		log.Printf("Redis error checking OTP: %v", err)
		return nil, status.Error(codes.Internal, "Failed to verify OTP")
	}

	if storedOtp != req.OtpCode {
		log.Printf("Invalid OTP for: %s. Expected %s, got %s", req.Email, storedOtp, req.OtpCode)
		return nil, status.Error(codes.InvalidArgument, "Invalid or expired OTP code")
	}

	// --- Step 2: Code is valid, get user and activate them ---
	var user User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve user data")
	}

	if user.IsActive {
		return nil, status.Error(codes.AlreadyExists, "This account is already verified.")
	}

	// Activate the user
	if err := s.db.Model(&user).Update("is_active", true).Error; err != nil {
		log.Printf("Failed to activate user %s: %v", req.Email, err)
		return nil, status.Error(codes.Internal, "Failed to activate account")
	}

	// Code is correct, delete it from Redis so it can't be reused
	s.rdb.Del(ctx, otpKey)

	// --- Step 3: Create tokens (as per blueprint) ---
	accessToken, err := createToken(user, 1*time.Hour) // 1 hour expiry
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to create access token")
	}

	refreshToken, err := createToken(user, 7*24*time.Hour) // 7 day expiry
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to create refresh token")
	}

	log.Printf("OTP verification successful for: %s", req.Email)

	return &pb.VerifyRegistrationOtpResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *server) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	log.Printf("SearchUsers request received for query: '%s'", req.Query)

	if req.Query == "" {
		return &pb.SearchUsersResponse{Users: []*pb.GetUserProfileResponse{}}, nil
	}

	// --- Step 1: Broad database query ---
	// We find all users who are NOT the user searching
	// AND whose username starts with the query.
	var users []User
	query := req.Query + "%"
	if err := s.db.Where("username LIKE ? AND id != ?", query, req.SelfUserId).
		Limit(50). // Limit the initial pull from DB for performance
		Find(&users).Error; err != nil {
		log.Printf("Failed to search users in DB: %v", err)
		return nil, status.Error(codes.Internal, "Failed to perform search")
	}

	// --- Step 2: Calculate Jaro-Winkler distance (as required by PDF) ---
	jw := metrics.NewJaroWinkler()
	jw.CaseSensitive = false // Make search case-insensitive

	var results []searchResult
	for _, user := range users {
		similarity := strutil.Similarity(req.Query, user.Username, jw)
		results = append(results, searchResult{
			user:       user,
			similarity: similarity,
		})
	}

	// --- Step 3: Sort by similarity (highest first) ---
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// --- Step 4: Get top 5 and convert to gRPC response ---
	// PDF requires "5 recommended user profiles"
	var grpcUsers []*pb.GetUserProfileResponse
	limit := 5
	if len(results) < 5 {
		limit = len(results)
	}

	for _, res := range results[:limit] {
		// We can re-use our existing helper function!
		grpcUser, err := s.GetUserProfile(ctx, &pb.GetUserProfileRequest{
			Username:   res.user.Username,
			SelfUserId: req.SelfUserId, // Pass this along
		})
		if err != nil {
			log.Printf("Failed to convert user %s for search: %v", res.user.Username, err)
			continue
		}
		grpcUsers = append(grpcUsers, grpcUser)
	}

	return &pb.SearchUsersResponse{
		Users: grpcUsers,
	}, nil
}

// --- ADD NEW ADMIN GRPC FUNCTIONS ---

func (s *server) BanUser(ctx context.Context, req *pb.BanUserRequest) (*pb.BanUserResponse, error) {
	log.Printf("Admin action: BanUser request from admin %d for user %d", req.AdminUserId, req.UserToBanId)

	// Find the user to ban
	var userToBan User
	if err := s.db.First(&userToBan, req.UserToBanId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "User to ban not found")
	}

	// Ban them
	if err := s.db.Model(&userToBan).Update("is_banned", true).Error; err != nil {
		log.Printf("Failed to ban user %d: %v", req.UserToBanId, err)
		return nil, status.Error(codes.Internal, "Failed to update user status")
	}

	return &pb.BanUserResponse{Message: "User banned successfully"}, nil
}

func (s *server) UnbanUser(ctx context.Context, req *pb.UnbanUserRequest) (*pb.UnbanUserResponse, error) {
	log.Printf("Admin action: UnbanUser request from admin %d for user %d", req.AdminUserId, req.UserToUnbanId)

	// Find the user to unban
	var userToUnban User
	if err := s.db.First(&userToUnban, req.UserToUnbanId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "User to unban not found")
	}

	// Unban them
	if err := s.db.Model(&userToUnban).Update("is_banned", false).Error; err != nil {
		log.Printf("Failed to unban user %d: %v", req.UserToUnbanId, err)
		return nil, status.Error(codes.Internal, "Failed to update user status")
	}

	return &pb.UnbanUserResponse{Message: "User unbanned successfully"}, nil
}

// --- GPRC: SendNewsletter ---
func (s *server) SendNewsletter(ctx context.Context, req *pb.SendNewsletterRequest) (*pb.SendNewsletterResponse, error) {
	log.Printf("Admin action: SendNewsletter request from admin %d", req.AdminUserId)

	// 1. Find all subscribed users
	var subscribedUsers []User
	if err := s.db.Where("is_subscribed = ?", true).Find(&subscribedUsers).Error; err != nil {
		log.Printf("Failed to get subscribed users: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve user list")
	}

	if len(subscribedUsers) == 0 {
		return &pb.SendNewsletterResponse{Message: "No subscribed users found", RecipientsCount: 0}, nil
	}

	// 2. Publish one job *per user* to the email_queue
	// This is more resilient than sending one giant job.
	for _, user := range subscribedUsers {
		emailBody, _ := json.Marshal(map[string]string{
			"to":      user.Email,
			"type":    "newsletter",
			"subject": req.Subject,
			"body":    req.Body,
		})
		if err := s.publishToQueue(ctx, "email_queue", emailBody); err != nil {
			log.Printf("Failed to publish newsletter job for user %s: %v", user.Email, err)
			// Don't stop, try to send to other users
		}
	}

	log.Printf("Published newsletter jobs for %d users", len(subscribedUsers))
	return &pb.SendNewsletterResponse{
		Message:         "Newsletter batch queued successfully",
		RecipientsCount: int32(len(subscribedUsers)),
	}, nil
}

// --- GPRC: SubmitVerificationRequest ---
func (s *server) SubmitVerificationRequest(ctx context.Context, req *pb.SubmitVerificationRequestRequest) (*pb.SubmitVerificationRequestResponse, error) {
	log.Printf("SubmitVerificationRequest received from user %d", req.UserId)

	newRequest := VerificationRequest{
		UserID:         req.UserId,
		IdCardNumber:   req.IdCardNumber,
		FacePictureURL: req.FacePictureUrl,
		Reason:         req.Reason,
		Status:         "pending",
	}

	// GORM's Create will fail if the unique constraint on UserID is violated
	if err := s.db.Create(&newRequest).Error; err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return nil, status.Error(codes.AlreadyExists, "A verification request for this user already exists")
		}
		log.Printf("Failed to create verification request: %v", err)
		return nil, status.Error(codes.Internal, "Failed to submit request")
	}

	msgBody, _ := json.Marshal(map[string]string{
		"type":       "new_verification_request",
		"user_id":    strconv.FormatInt(newRequest.UserID, 10),
		"request_id": strconv.FormatUint(uint64(newRequest.ID), 10),
	})
	s.publishToQueue(ctx, "admin_notification_queue", msgBody)

	// We can't use a helper here as we need to return the request model
	return &pb.SubmitVerificationRequestResponse{
		Request: &pb.VerificationRequest{
			Id:             strconv.FormatUint(uint64(newRequest.ID), 10),
			UserId:         newRequest.UserID,
			IdCardNumber:   "REDACTED",
			FacePictureUrl: newRequest.FacePictureURL,
			Reason:         newRequest.Reason,
			Status:         newRequest.Status,
			CreatedAt:      newRequest.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// --- GPRC: GetVerificationRequests ---
func (s *server) GetVerificationRequests(ctx context.Context, req *pb.GetVerificationRequestsRequest) (*pb.GetVerificationRequestsResponse, error) {
	log.Printf("Admin action: GetVerificationRequests request")

	var requests []VerificationRequest
	query := s.db.Order("created_at DESC").Limit(int(req.PageSize)).Offset(int(req.PageOffset))

	if req.Status == "pending" || req.Status == "approved" || req.Status == "rejected" {
		query = query.Where("status = ?", req.Status)
	}

	if err := query.Find(&requests).Error; err != nil {
		log.Printf("Failed to get verification requests from db: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve requests")
	}

	// We need user data (usernames) for each request
	var grpcRequests []*pb.VerificationRequest
	for _, req := range requests {

		// --- THIS IS THE FIX ---
		// Get username for this request directly from our own DB
		var user User
		username := "Unknown"
		// We only select the username for efficiency
		if err := s.db.Model(&User{}).Select("username").First(&user, req.UserID).Error; err == nil {
			username = user.Username
		} else {
			log.Printf("Could not find user %d for verification request %d", req.UserID, req.ID)
		}
		// --- END FIX ---

		grpcRequests = append(grpcRequests, &pb.VerificationRequest{
			Id:             strconv.FormatUint(uint64(req.ID), 10),
			UserId:         req.UserID,
			IdCardNumber:   "REDACTED", // Don't send sensitive info
			FacePictureUrl: req.FacePictureURL,
			Reason:         req.Reason,
			Status:         req.Status,
			CreatedAt:      req.CreatedAt.Format(time.RFC3339),
			Username:       username,
		})
	}

	return &pb.GetVerificationRequestsResponse{Requests: grpcRequests}, nil
}

// --- GPRC: ResolveVerificationRequest ---
func (s *server) ResolveVerificationRequest(ctx context.Context, req *pb.ResolveVerificationRequestRequest) (*pb.ResolveVerificationRequestResponse, error) {
	log.Printf("Admin action: ResolveVerificationRequest for request %d with action '%s'", req.RequestId, req.Action)

	// 1. Find the request
	var request VerificationRequest
	if err := s.db.First(&request, req.RequestId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Request not found")
	}

	if request.Status != "pending" {
		return nil, status.Error(codes.AlreadyExists, "This request has already been resolved")
	}

	// 2. Find the user
	var user User
	if err := s.db.First(&user, request.UserID).Error; err != nil {
		return nil, status.Error(codes.NotFound, "The user for this request no longer exists")
	}

	// 3. Perform the action
	var newStatus string
	var emailType string
	if req.Action == "APPROVE" {
		newStatus = "approved"
		emailType = "verification_accepted"
		// --- UPDATE THE USER TABLE ---
		if err := s.db.Model(&user).Update("is_verified", true).Error; err != nil {
			log.Printf("Failed to set user %d as verified: %v", user.ID, err)
			return nil, status.Error(codes.Internal, "Failed to update user status")
		}
	} else if req.Action == "REJECT" {
		newStatus = "rejected"
		emailType = "verification_rejected"
	} else {
		return nil, status.Error(codes.InvalidArgument, "Action must be 'APPROVE' or 'REJECT'")
	}

	// 4. Mark the request as resolved
	if err := s.db.Model(&request).Updates(VerificationRequest{Status: newStatus, ResolvedByID: req.AdminUserId}).Error; err != nil {
		log.Printf("Failed to mark request %d as resolved: %v", req.RequestId, err)
		return nil, status.Error(codes.Internal, "Failed to resolve request")
	}

	// 5. Send email to user
	emailBody, _ := json.Marshal(map[string]string{
		"to":       user.Email,
		"type":     emailType,
		"username": user.Username,
	})
	if err := s.publishToQueue(ctx, "email_queue", emailBody); err != nil {
		log.Printf("Failed to publish verification email for user %s: %v", user.Email, err)
	}

	return &pb.ResolveVerificationRequestResponse{Message: "Request resolved successfully"}, nil
}

// --- GPRC: HandleGoogleAuth ---
func (s *server) HandleGoogleAuth(ctx context.Context, req *pb.HandleGoogleAuthRequest) (*pb.LoginResponse, error) {
	// 1. Exchange auth code for Google token
	token, err := googleOauthConfig.Exchange(ctx, req.AuthCode)
	if err != nil {
		log.Printf("Failed to exchange Google auth code: %v", err)
		return nil, status.Error(codes.InvalidArgument, "Invalid Google authorization code")
	}

	// 2. Get user info from Google
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to get user info from Google")
	}
	defer response.Body.Close()

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(response.Body).Decode(&googleUser); err != nil {
		return nil, status.Error(codes.Internal, "Failed to parse Google user info")
	}

	// 3. Find or Create user in our DB
	var user User
	err = s.db.Where("provider = 'google' AND provider_id = ?", googleUser.ID).First(&user).Error

	if err == gorm.ErrRecordNotFound {
		// User not found, create them
		log.Printf("Creating new user for Google login: %s", googleUser.Email)
		newUser := User{
			Name:              googleUser.Name,
			Username:          googleUser.Email, // Can be changed later
			Email:             googleUser.Email,
			Password:          "", // No password for OAuth users
			Provider:          "google",
			ProviderID:        googleUser.ID,
			ProfilePictureURL: googleUser.Picture,
			IsActive:          true, // Google emails are pre-verified
			IsVerified:        false,
			Role:              "user",
		}

		if err := s.db.Create(&newUser).Error; err != nil {
			// Handle username/email conflict
			if strings.Contains(err.Error(), "unique constraint") {
				return nil, status.Error(codes.AlreadyExists, "A user with this email or username already exists. Please log in normally.")
			}
			return nil, status.Error(codes.Internal, "Failed to create user")
		}
		user = newUser // Use the new user

	} else if err != nil {
		return nil, status.Error(codes.Internal, "Database error")
	}

	// 4. User exists, check if banned
	if user.IsBanned {
		return nil, status.Error(codes.PermissionDenied, "This account is banned")
	}

	// 5. Create tokens and return login response
	accessToken, _ := createToken(user, 1*time.Hour)
	refreshToken, _ := createToken(user, 7*24*time.Hour)

	return &pb.LoginResponse{
		Message:      "Google login successful",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
