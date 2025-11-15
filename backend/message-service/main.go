package main

import (
	"context"
	"log"
	"net"
	"time"
    "strconv"
    "encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
    "google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"

	// This service's generated proto
	pb "github.com/hoshibmatchi/message-service/proto"
	// User service proto (for gRPC client)
	userPb "github.com/hoshibmatchi/user-service/proto"
)

// --- GORM Models ---

// Conversation represents a single chat, either 1-on-1 or a group.
type Conversation struct {
	gorm.Model
	IsGroup   bool   `gorm:"default:false"`
	GroupName string // e.g., "Study Group"
}

// Participant is the join table between Conversation and User.
type Participant struct {
	// Composite primary key
	ConversationID uint  `gorm:"primaryKey"`
	UserID         int64 `gorm:"primaryKey"`
	JoinedAt       time.Time
}

// Message is a single message within a conversation.
type Message struct {
	gorm.Model
	ConversationID uint  `gorm:"index"` // Foreign key to Conversation
	SenderID       int64 `gorm:"index"` // The UserID of the sender
	Content        string
}

// server struct holds our database, cache, and client connections
type server struct {
	pb.UnimplementedMessageServiceServer
	db         *gorm.DB                 // Postgres connection
	rdb        *redis.Client            // Redis connection
	userClient userPb.UserServiceClient // gRPC client for user-service
}

func main() {
	// --- Step 1: Connect to PostgreSQL (message-db) ---
	dsn := "host=message-db user=admin password=password dbname=message_service_db port=5432 sslmode=disable TimeZone=UTC"
	var db *gorm.DB
	var err error

	// Retry connection
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Println("Successfully connected to message-db")
			break
		}
		log.Printf("Failed to connect to message-db: %v. Retrying...", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to message-db after retries: %v", err)
	}

	// Auto-migrate the schema
	db.AutoMigrate(&Conversation{}, &Participant{}, &Message{})

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

	// --- Step 3: Connect to User Service (gRPC Client) ---
	userConn, err := grpc.Dial("user-service:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer userConn.Close()
	userClient := userPb.NewUserServiceClient(userConn)
	log.Println("Successfully connected to user-service")

	// --- Step 4: Start this gRPC Server (message-service) ---
	lis, err := net.Listen("tcp", ":9003") // Port 9003
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMessageServiceServer(s, &server{
		db:         db,
		rdb:        rdb,
		userClient: userClient,
	})

	// TODO (Solution 4.2): Start Redis Pub/Sub listener goroutine
	// go s.listenForRealtimeMessages()

	log.Println("Message service listening on port 9003...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- Implement gRPC methods ---

func (s *server) CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.Conversation, error) {
	log.Printf("CreateConversation request received from user %d", req.CreatorId)

	// --- Step 1: Validation ---
	if len(req.ParticipantIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "At least one other participant is required to create a conversation")
	}

	// Combine all participant IDs, including the creator
	allParticipantIDs := append(req.ParticipantIds, req.CreatorId)

	// Determine if it's a group
	isGroup := req.GroupName != "" || len(allParticipantIDs) > 2

	// --- Step 2: For 1-on-1 chats, check if a conversation already exists ---
	if !isGroup {
		var existingConversationID uint
		// This query finds a conversation_id that has EXACTLY 2 participants
		// AND where those participants are our two users.
		err := s.db.Table("participants").
			Select("conversation_id").
			Where("user_id IN ?", allParticipantIDs).
			Group("conversation_id").
			Having("COUNT(user_id) = 2").
			Limit(1).
			Pluck("conversation_id", &existingConversationID).Error

		if err == nil && existingConversationID > 0 {
			// A 1-on-1 chat already exists. Find it and return it.
			log.Printf("Found existing 1-on-1 chat (ID: %d) for users %v", existingConversationID, allParticipantIDs)
			var existingConversation Conversation
			if s.db.First(&existingConversation, existingConversationID).Error == nil {
				// We found it, now we must convert it to the gRPC response
				return s.gormToGrpcConversation(ctx, &existingConversation)
			}
		}
	}

	// --- Step 3: Create new conversation in a transaction ---
	newConversation := Conversation{
		IsGroup:   isGroup,
		GroupName: req.GroupName,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the Conversation row
		if err := tx.Create(&newConversation).Error; err != nil {
			return err
		}

		// 2. Create the Participant rows
		participants := []Participant{}
		for _, userID := range allParticipantIDs {
			participants = append(participants, Participant{
				ConversationID: newConversation.ID,
				UserID:         userID,
				JoinedAt:       time.Now(),
			})
		}

		if err := tx.Create(&participants).Error; err != nil {
			return err
		}

		return nil // Commit
	})

	if err != nil {
		log.Printf("Failed to create conversation: %v", err)
		return nil, status.Error(codes.Internal, "Failed to create conversation")
	}

	log.Printf("Created new conversation (ID: %d)", newConversation.ID)

	// --- Step 4: Convert to gRPC response and return ---
	// This helper function will fetch participant user data
	return s.gormToGrpcConversation(ctx, &newConversation)
}

// --- Helper Functions ---

// gormToGrpcConversation converts a GORM Conversation model to its gRPC representation
// This involves fetching participant details from the user-service
func (s *server) gormToGrpcConversation(ctx context.Context, convo *Conversation) (*pb.Conversation, error) {
	// 1. Get all participant IDs for this conversation
	var participantIDs []int64
	if err := s.db.Model(&Participant{}).Where("conversation_id = ?", convo.ID).Pluck("user_id", &participantIDs).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to get participant IDs")
	}

	// 2. Fetch user data for all participants from user-service
	// In a real high-performance app, user-service should have a
	// GetUsersData (plural) RPC. We are simulating that by calling in a loop.
	grpcParticipants := []*userPb.GetUserDataResponse{}
	for _, userID := range participantIDs {
		userData, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: userID})
		if err != nil {
			log.Printf("Failed to get user data for participant %d: %v", userID, err)
			// Add a placeholder to not fail the whole request
			grpcParticipants = append(grpcParticipants, &userPb.GetUserDataResponse{
				Username: "Unknown User",
			})
		} else {
			grpcParticipants = append(grpcParticipants, userData)
		}
	}

	// 3. Get the last message (we'll leave this empty for now)
	// TODO: Get the actual last message from the 'messages' table
	lastMessage := &pb.Message{
		Content: "No messages yet...",
	}

	// 4. Assemble and return
	return &pb.Conversation{
		Id:           strconv.FormatUint(uint64(convo.ID), 10),
		Participants: grpcParticipants,
		LastMessage:  lastMessage,
		CreatedAt:    convo.CreatedAt.Format(time.RFC3339),
		IsGroup:      convo.IsGroup,
		GroupName:    convo.GroupName,
	}, nil
}

func (s *server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	log.Printf("SendMessage request from user %d to convo %s", req.SenderId, req.ConversationId)

	// --- Step 1: Validation (Security Check) ---
	// Check if the sender is actually a participant in this conversation.
	var participantCount int64
	// We must convert the conversation ID from string back to uint
	convoID, _ := strconv.ParseUint(req.ConversationId, 10, 64)
	if convoID == 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid conversation ID format")
	}

	s.db.Model(&Participant{}).Where("conversation_id = ? AND user_id = ?", convoID, req.SenderId).Count(&participantCount)
	if participantCount == 0 {
		return nil, status.Error(codes.PermissionDenied, "Sender is not a participant of this conversation")
	}

	// --- Step 2: Create and Save the Message ---
	newMessage := Message{
		ConversationID: uint(convoID),
		SenderID:       req.SenderId,
		Content:        req.Content,
	}

	// We use a transaction to save the message AND update the conversation's timestamp
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Save the new message
		if err := tx.Create(&newMessage).Error; err != nil {
			return err
		}

		// 2. "Touch" the conversation's UpdatedAt timestamp.
		// This is critical for sorting conversations by "most recent".
		if err := tx.Model(&Conversation{}).Where("id = ?", convoID).Update("updated_at", time.Now()).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to save message: %v", err)
		return nil, status.Error(codes.Internal, "Failed to send message")
	}

	// --- Step 3: Publish to Redis Pub/Sub for Real-Time (Solution 4.2) ---
	// Convert to gRPC response first, as this is what we'll send
	grpcMessage, err := s.gormToGrpcMessage(ctx, &newMessage)
	if err != nil {
		// Log the error, but don't fail the send. The message is saved.
		log.Printf("Failed to convert message %d to gRPC: %v", newMessage.ID, err)
	} else {
		// Marshal the gRPC message to JSON
		msgBody, err := json.Marshal(grpcMessage)
		if err != nil {
			log.Printf("Failed to marshal message %d for redis: %v", newMessage.ID, err)
		} else {
			// Publish to a dynamic channel for this specific conversation
			channelName := fmt.Sprintf("chat:%s", req.ConversationId)
			if err := s.rdb.Publish(ctx, channelName, msgBody).Err(); err != nil {
				log.Printf("Failed to publish message to redis channel %s: %v", channelName, err)
			} else {
				log.Printf("Published message to redis channel %s", channelName)
			}
		}
	}

	// --- Step 4: Return the created message ---
	// If we failed to convert/publish, return a manually converted message
	if grpcMessage == nil {
		grpcMessage = &pb.Message{
			Id:             strconv.FormatUint(uint64(newMessage.ID), 10),
			ConversationId: req.ConversationId,
			SenderId:       strconv.FormatInt(newMessage.SenderID, 10),
			Content:        newMessage.Content,
			SentAt:         newMessage.CreatedAt.Format(time.RFC3339),
			SenderUsername: "...", // Denormalization failed
		}
	}

	return &pb.SendMessageResponse{
		Message: grpcMessage,
	}, nil
}

// gormToGrpcMessage converts a GORM Message to its gRPC representation
func (s *server) gormToGrpcMessage(ctx context.Context, msg *Message) (*pb.Message, error) {
	// 1. Get sender's user data
	userData, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: msg.SenderID})
	if err != nil {
		// Don't fail the whole conversion, just log and use a placeholder
		log.Printf("Failed to get user data for sender %d: %v", msg.SenderID, err)
		userData = &userPb.GetUserDataResponse{Username: "Unknown"}
	}

	// 2. Assemble and return
	return &pb.Message{
		Id:             strconv.FormatUint(uint64(msg.ID), 10),
		ConversationId: strconv.FormatUint(uint64(msg.ConversationID), 10),
		SenderId:       strconv.FormatInt(msg.SenderID, 10),
		Content:        msg.Content,
		SentAt:         msg.CreatedAt.Format(time.RFC3339),
		SenderUsername: userData.Username,
	}, nil
}