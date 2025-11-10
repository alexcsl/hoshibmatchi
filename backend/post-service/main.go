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

	pb "github.com/hoshibmatchi/post-service/proto"
	userPb "github.com/hoshibmatchi/user-service/proto"
	
	"github.com/lib/pq" // For string arrays
)

// Post defines the GORM model
type Post struct {
	gorm.Model
	AuthorID         int64
	Caption          string
	MediaURLs        pq.StringArray `gorm:"type:text[]"`
	CommentsDisabled bool
	AuthorUsername   string
	AuthorProfileURL string
	AuthorIsVerified bool
}

type server struct {
	pb.UnimplementedPostServiceServer
	db         *gorm.DB
	userClient userPb.UserServiceClient
}

func main() {
	// --- Step 1: Connect to Post DB ---
	dsn := "host=post-db user=admin password=password dbname=post_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to post-db: %v", err)
	}
	db.AutoMigrate(&Post{})

	// --- Step 2: Connect to User Service (gRPC Client) ---
	userConn, err := grpc.Dial("user-service:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer userConn.Close()
	userClient := userPb.NewUserServiceClient(userConn)
	log.Println("Successfully connected to user-service")

	// --- Step 3: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9001") // Port 9001
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterPostServiceServer(s, &server{db: db, userClient: userClient})

	log.Println("Post service listening on port 9001...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- Implement CreatePost ---
func (s *server) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	log.Println("CreatePost request received")

	// --- Step 1: Call User Service for Denormalization ---
	userData, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: req.AuthorId})
	if err != nil {
		log.Printf("Failed to get user data from user-service: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve author details")
	}

	// --- Step 2: Create the Post in our DB ---
	newPost := Post{
		AuthorID:         req.AuthorId,
		Caption:          req.Caption,
		MediaURLs:        req.MediaUrls,
		CommentsDisabled: req.CommentsDisabled,
		AuthorUsername:   userData.Username,
		AuthorProfileURL: userData.ProfilePictureUrl,
		AuthorIsVerified: userData.IsVerified,
	}

	if result := s.db.Create(&newPost); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to save post to database")
	}

	// --- Step 3: Return the created post ---
	return &pb.CreatePostResponse{
		Post: &pb.Post{
			Id:                 strconv.FormatUint(uint64(newPost.ID), 10),
			Caption:            newPost.Caption,
			AuthorUsername:     newPost.AuthorUsername,
			AuthorProfileUrl:   newPost.AuthorProfileURL,
			AuthorIsVerified:   newPost.AuthorIsVerified,
			MediaUrls:          newPost.MediaURLs,
			CreatedAt:          newPost.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}