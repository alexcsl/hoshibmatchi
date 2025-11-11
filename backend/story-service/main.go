package main

import (
	"context"
	"log"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// --- This service's generated proto ---
	pb "github.com/hoshibmatchi/story-service/proto"
	// --- Proto from the user-service (which we will call) ---
	userPb "github.com/hoshibmatchi/user-service/proto"
)

// Story defines the GORM model
type Story struct {
	gorm.Model
	AuthorID         int64
	MediaURL         string
	
	// Denormalized data
	AuthorUsername   string
	AuthorProfileURL string
}

// server struct holds its DB and the user-service client
type server struct {
	pb.UnimplementedStoryServiceServer
	db         *gorm.DB
	userClient userPb.UserServiceClient
}

func main() {
	// --- Step 1: Connect to Story DB ---
	dsn := "host=story-db user=admin password=password dbname=story_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to story-db: %v", err)
	}
	db.AutoMigrate(&Story{})

	// --- Step 2: Connect to User Service (gRPC Client) ---
	userConn, err := grpc.Dial("user-service:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer userConn.Close()
	userClient := userPb.NewUserServiceClient(userConn)
	log.Println("Successfully connected to user-service")

	// --- Step 3: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9002") // Correct port 9002
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterStoryServiceServer(s, &server{db: db, userClient: userClient})

	log.Println("Story service listening on port 9002...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- Implement CreateStory ---
func (s *server) CreateStory(ctx context.Context, req *pb.CreateStoryRequest) (*pb.CreateStoryResponse, error) {
	log.Println("CreateStory request received")

	// --- Step 1: Call User Service for Denormalization ---
	userData, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: req.AuthorId})
	if err != nil {
		log.Printf("Failed to get user data from user-service: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve author details")
	}

	// --- Step 2: Create the Story in our DB ---
	newStory := Story{
		AuthorID:         req.AuthorId,
		MediaURL:         req.MediaUrl,
		AuthorUsername:   userData.Username,
		AuthorProfileURL: userData.ProfilePictureUrl,
	}

	if result := s.db.Create(&newStory); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to save story to database")
	}

	// TODO: As per blueprint,
	// publish a "story.created" message to RabbitMQ with a 24-hour delay
	// for asynchronous deletion. We will add this in the RabbitMQ phase.

	// --- Step 3: Return the created story ---
	return &pb.CreateStoryResponse{
		Story: &pb.Story{
			Id:               strconv.FormatUint(uint64(newStory.ID), 10),
			MediaUrl:         newStory.MediaURL,
			AuthorUsername:   newStory.AuthorUsername,
			AuthorProfileUrl: newStory.AuthorProfileURL,
			CreatedAt:        newStory.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}