package main

import (
	"context"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// Don't forget to run: go get github.com/go-redis/redis/v8
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
	// TODO: Add fields from PDF like 'is_banned', 'is_deactivated'
}

// server struct holds our database and cache connections
type server struct {
	pb.UnimplementedUserServiceServer
	db  *gorm.DB
	rdb *redis.Client // Redis client
}

// emailRegex for validation [cite: 183]
var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

func main() {
	// --- Step 1: Connect to PostgreSQL ---
	dsn := "host=user-db user=admin password=password dbname=user_service_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Running database migrations...")
	db.AutoMigrate(&User{})

	// --- Step 2: Connect to Redis ---
	// "redis:6379" is the service name from docker-compose.yml [cite: 100]
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // default DB
	})

	// Ping Redis to ensure connection is alive
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

	// Register our service with *both* connections
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
	// This assumes another service/endpoint *sends* the OTP and stores it.
	// We are just verifying it here. [cite: 178, 190]
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
	if len(req.Name) <= 4 { // [cite: 179]
		return nil, status.Error(codes.InvalidArgument, "Name must be more than 4 characters")
	}
	// TODO: Add regex for "no symbols or numbers" validation [cite: 179]

	if !emailRegex.MatchString(req.Email) { // [cite: 183]
		return nil, status.Error(codes.InvalidArgument, "Invalid email format")
	}

	// TODO: Add 4+ password validations [cite: 184]
	if len(req.Password) < 8 { // Example validation
		return nil, status.Error(codes.InvalidArgument, "Password must be at least 8 characters")
	}

	if req.Gender != "male" && req.Gender != "female" { // [cite: 186]
		return nil, status.Error(codes.InvalidArgument, "Gender must be male or female")
	}

	// Age validation [cite: 187]
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid date of birth format. Use YYYY-MM-DD")
	}
	if !isAgeValid(dob, 13) {
		return nil, status.Error(codes.InvalidArgument, "You must be at least 13 years old to register")
	}

	// --- Step 3: Hash Password ---
	// Uses bcrypt which includes SALT by default [cite: 128]
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
		ProfilePictureURL: req.ProfilePictureUrl, // Use the new proto field
	}

	result := s.db.Create(&newUser)
	if result.Error != nil {
		log.Printf("Failed to create user in database: %v", result.Error)

		// Check for unique constraint violations [cite: 180, 182]
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
	// The OTP was valid, so we can delete it now
	s.rdb.Del(ctx, otpKey)

	log.Println("Successfully created user with ID:", newUser.ID)
	// TODO: Send successful registration email via RabbitMQ [cite: 195]

	return &pb.RegisterUserResponse{
		Id:       int64(newUser.ID),
		Username: newUser.Username,
		Email:    newUser.Email,
	}, nil
}

// Helper function for age validation [cite: 187]
func isAgeValid(birthDate time.Time, minAge int) bool {
	today := time.Now()
	age := today.Year() - birthDate.Year()
	if today.Month() < birthDate.Month() || (today.Month() == birthDate.Month() && today.Day() < birthDate.Day()) {
		age--
	}
	return age >= minAge
}