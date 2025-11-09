package main

import (
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"context"

	pb "github.com/hoshibmatchi/user-service/proto" 
	"golang.org/x/crypto/bcrypt"
)

// User defines the data model for a user in our database.
// GORM tags are used to define the table schema.
type User struct {
	gorm.Model                  // Includes ID, CreatedAt, UpdatedAt, DeletedAt
	Name              string    `gorm:"type:varchar(100);not null"`
	Username          string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email             string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Password          string    `gorm:"type:varchar(255);not null"` // Will store the bcrypt hash
	ProfilePictureURL string    `gorm:"type:varchar(255)"`
	DateOfBirth       time.Time `gorm:"not null"`
	Gender            string    `gorm:"type:varchar(10);not null"`
}

// server struct will hold our database connection and implement the gRPC service.
type server struct {
	// UnimplementedUserServiceServer must be embedded to have forward compatible implementations.
	// We will add this line after generating the proto code in the next step.
	pb.UnimplementedUserServiceServer
	db *gorm.DB
}

func main() {
	// Step 1: Connect to the PostgreSQL database using the service name from Docker Compose.
	dsn := "host=user-db user=admin password=password dbname=user_service_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Step 2: Auto-migrate the User struct to create the 'users' table.
	// GORM will automatically create the table if it doesn't exist.
	log.Println("Running database migrations...")
	db.AutoMigrate(&User{})

	// Step 3: Set up and start the gRPC server.
	// It will listen on port 9000 for requests from other services (like the api-gateway).
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on port 9000: %v", err)
	}

	s := grpc.NewServer()

	// We will register our service here in a later step.
	pb.RegisterUserServiceServer(s, &server{db: db})

	log.Println("User gRPC server is listening on port 9000...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server over port 9000: %v", err)
	}
}

// RegisterUser is the implementation of our gRPC service function.
func (s *server) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	log.Println("RegisterUser request received for username:", req.Username)

	// TODO: Add validation here for all fields as per the PDF requirements.
	// (e.g., check name length, email format, password complexity)

	// Hash the user's password for secure storage.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return nil, err // Return an internal server error
	}

	// Parse the date of birth string.
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		log.Printf("Invalid date of birth format: %v", err)
		return nil, err // Return an invalid argument error
	}

	// Create a new User model instance.
	newUser := User{
		Name:        req.Name,
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hashedPassword),
		DateOfBirth: dob,
		Gender:      req.Gender,
	}

	// Save the new user to the database.
	result := s.db.Create(&newUser)
	if result.Error != nil {
		log.Printf("Failed to create user in database: %v", result.Error)
		// TODO: Check for specific errors like 'unique constraint violation'
		// to return a more specific "username/email already exists" error.
		return nil, result.Error
	}

	log.Println("Successfully created user with ID:", newUser.ID)

	// Return a successful response with the new user's details.
	return &pb.RegisterUserResponse{
		Id:       int64(newUser.ID),
		Username: newUser.Username,
		Email:    newUser.Email,
	}, nil
}
