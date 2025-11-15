package main

import (
	"context"
	"log"
	"net"
	"strconv"
	"time"
    "strings"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	amqp "github.com/rabbitmq/amqp091-go"

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

type StoryLike struct {
	UserID int64 `gorm:"primaryKey"`
	StoryID int64 `gorm:"primaryKey"`
	CreatedAt time.Time
}

// server struct holds its DB and the user-service client
type server struct {
	pb.UnimplementedStoryServiceServer
	db         *gorm.DB
	userClient userPb.UserServiceClient
	amqpCh     *amqp.Channel
}

func main() {
	// --- Step 1: Connect to Story DB ---
	dsn := "host=story-db user=admin password=password dbname=story_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to story-db: %v", err)
	}
	db.AutoMigrate(&Story{})
    db.AutoMigrate(&StoryLike{})

	// --- Step 2: Connect to User Service (gRPC Client) ---
	userConn, err := grpc.Dial("user-service:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer userConn.Close()
	userClient := userPb.NewUserServiceClient(userConn)
	log.Println("Successfully connected to user-service")

	// --- Step 3: Connect to RabbitMQ (with retries) ---
	var amqpConn *amqp.Connection
	maxRetries := 10
	retryDelay := 2 * time.Second
	for i := 0; i < maxRetries; i++ {
		amqpConn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil { log.Println("Successfully connected to RabbitMQ"); break }
		log.Printf("Failed to connect to RabbitMQ: %v. Retrying...", err)
		time.Sleep(retryDelay)
	}
	if amqpConn == nil { log.Fatalf("Could not connect to RabbitMQ after %d retries", maxRetries) }
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil { log.Fatalf("Failed to open RabbitMQ channel: %v", err) }
	defer amqpCh.Close()

	// --- ADDED: Declare queues for 24h story deletion ---

	// 1. This is the FINAL queue the worker will listen to.
	_, err = amqpCh.QueueDeclare(
		"story_deletion_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare story_deletion_queue: %v", err)
	}

	// 2. This is the "wait" queue. Messages sit here for 24h.
	_, err = amqpCh.QueueDeclare(
		"story_wait_24h",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			// Set message Time-To-Live to 24 hours (in milliseconds)
			"x-message-ttl": int32(24 * 60 * 60 * 1000),
			// When message expires, send it to the default exchange
			"x-dead-letter-exchange": "",
			// Route it to the "story_deletion_queue"
			"x-dead-letter-routing-key": "story_deletion_queue",
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare story_wait_24h queue: %v", err)
	}
	log.Println("RabbitMQ story deletion queues declared")

	_, err = amqpCh.QueueDeclare(
		"notification_queue", true, false, false, false, nil,
	)
	if err != nil { log.Fatalf("Failed to declare notification_queue: %v", err) }
	log.Println("RabbitMQ notification_queue declared")

	// --- Step 4: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9002") // Correct port 9002
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterStoryServiceServer(s, &server{
		db:         db,
		userClient: userClient,
		amqpCh:     amqpCh, // <-- This was missing
	})

	log.Println("Story service listening on port 9002...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *server) publishToQueue(ctx context.Context, queueName string, body []byte) error {
	return s.amqpCh.PublishWithContext(
		ctx,
		"",          // exchange (default)
		queueName,   // routing key (queue name)
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
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

	// --- ADDED: Publish 24-hour delayed deletion job ---
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgBody, _ := json.Marshal(map[string]uint{
		"story_id": newStory.ID,
	})

	// Publish to the "wait" queue, NOT the final queue
	err = s.amqpCh.PublishWithContext(
		ctxTimeout,
		"",                 // exchange (default)
		"story_wait_24h", // routing key (the "wait" queue)
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // Make message durable
			Body:         msgBody,
		},
	)
	if err != nil {
		// Log the error, but don't fail the user's request
		log.Printf("Failed to publish story deletion job for story %d: %v", newStory.ID, err)
	} else {
		log.Printf("Published 24h deletion job for story %d", newStory.ID)
	}

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

func (s *server) LikeStory(ctx context.Context, req *pb.LikeStoryRequest) (*pb.LikeStoryResponse, error) {
	like := StoryLike{
		UserID:  req.UserId,
		StoryID: req.StoryId,
	}
	if result := s.db.Create(&like); result.Error != nil {
		if strings.Contains(result.Error.Error(), "unique constraint") {
			return nil, status.Error(codes.AlreadyExists, "Story already liked")
		}
		return nil, status.Error(codes.Internal, "Failed to like story")
	}

	var story Story
	s.db.First(&story, req.StoryId)
	if story.AuthorID != req.UserId {
		msgBody, _ := json.Marshal(map[string]interface{}{
			"type":      "story.liked",
			"actor_id":  req.UserId,
			"user_id":   story.AuthorID,
			"entity_id": req.StoryId,
		})
		s.publishToQueue(ctx, "notification_queue", msgBody)
	}

	return &pb.LikeStoryResponse{Message: "Story liked"}, nil
}

// --- Implement UnlikeStory ---
func (s *server) UnlikeStory(ctx context.Context, req *pb.UnlikeStoryRequest) (*pb.UnlikeStoryResponse, error) {
	like := StoryLike{
		UserID:  req.UserId,
		StoryID: req.StoryId,
	}
	if result := s.db.Delete(&like); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to unlike story")
	} else if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "Story was not liked")
	}
	return &pb.UnlikeStoryResponse{Message: "Story unliked"}, nil
}