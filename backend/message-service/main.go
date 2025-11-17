package main

import (
	"context"
	"log"
	"net"
	"time"
    "strconv"
    "encoding/json"
	"fmt"
	"net/http"
	"sync"
	"gorm.io/gorm/clause"
	"os"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"github.com/golang-jwt/jwt/v5"
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

type HiddenConversation struct {
	UserID         int64 `gorm:"primaryKey"`
	ConversationID uint  `gorm:"primaryKey"`
}

// upgrader specifies the parameters for upgrading an HTTP connection to a WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// This is NOT secure for production, but fine for our local dev
	// It allows connections from any origin (e.g., hoshi.local)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a WebSocket client
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	userID   int64
	convoIDs map[string]bool // Set of conversation IDs this client is listening to
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    sync.Map // Thread-safe map of [int64]*Client (userID -> Client)
	register   chan *Client
	unregister chan *Client
}

// newHub creates a new Hub
func newHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// run starts the hub's event loop
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients.Store(client.userID, client)
			log.Printf("Client registered: %d", client.userID)
		case client := <-h.unregister:
			if _, ok := h.clients.Load(client.userID); ok {
				h.clients.Delete(client.userID)
				close(client.send)
				log.Printf("Client unregistered: %d", client.userID)
			}
		}
	}
}

// server struct holds our database, cache, and client connections
type server struct {
	pb.UnimplementedMessageServiceServer
	db         *gorm.DB                 // Postgres connection
	rdb        *redis.Client            // Redis connection
	userClient userPb.UserServiceClient // gRPC client for user-service
	hub 	  *Hub                     // Hub for managing WebSocket clients
}

func main() {
	// --- Step 1: Connect to PostgreSQL (message-db) ---
	dsn := "host=message-db user=admin password=password dbname=message_service_db port=5432 sslmode=disable TimeZone=UTC"
	var db *gorm.DB
	var err error

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

	db.AutoMigrate(&Conversation{}, &Participant{}, &Message{})
	db.AutoMigrate(&HiddenConversation{})

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

	// --- Step 4: Create Hub and Server Struct ---
	hub := newHub()
	go hub.run() // Start the hub's event loop in a goroutine

	s := &server{
		db:         db,
		rdb:        rdb,
		userClient: userClient,
		hub:        hub,
	}

	// --- Step 5: Start Redis Pub/Sub Listener ---
	go s.listenForRealtimeMessages() // Start in a goroutine

	// --- Step 6: Start gRPC Server (in a goroutine) ---
	lis, err := net.Listen("tcp", ":9003") // Port 9003 for gRPC
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMessageServiceServer(grpcServer, s)

	go func() {
		log.Println("Message service (gRPC) listening on port 9003...")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// --- Step 7: Start WebSocket Server (on main thread) ---
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebSocket(w, r)
	})

	log.Println("Message service (WebSocket) listening on port 9004...")
	if err := http.ListenAndServe(":9004", nil); err != nil {
		log.Fatalf("Failed to serve WebSocket: %v", err)
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
// This involves fetching participant details AND the last message.
func (s *server) gormToGrpcConversation(ctx context.Context, convo *Conversation) (*pb.Conversation, error) {
	// 1. Get all participant IDs for this conversation
	var participantIDs []int64
	if err := s.db.Model(&Participant{}).Where("conversation_id = ?", convo.ID).Pluck("user_id", &participantIDs).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to get participant IDs")
	}

	// 2. Fetch user data for all participants from user-service
	grpcParticipants := []*userPb.GetUserDataResponse{}
	for _, userID := range participantIDs {
		// Create a new context for each gRPC call to avoid cancellation issues
		callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		userData, err := s.userClient.GetUserData(callCtx, &userPb.GetUserDataRequest{UserId: userID})
		if err != nil {
			log.Printf("Failed to get user data for participant %d: %v", userID, err)
			grpcParticipants = append(grpcParticipants, &userPb.GetUserDataResponse{
				Username: "Unknown User",
			})
		} else {
			grpcParticipants = append(grpcParticipants, userData)
		}
		cancel() // Release context
	}

	// --- THIS IS THE FIX ---
	// 3. Get the last message
	var lastMessageGORM Message
	var lastMessage *pb.Message
	err := s.db.Where("conversation_id = ?", convo.ID).Order("created_at DESC").First(&lastMessageGORM).Error

	if err == gorm.ErrRecordNotFound {
		// No messages yet
		lastMessage = &pb.Message{Content: "No messages yet..."}
	} else if err != nil {
		// Database error
		log.Printf("Failed to get last message for convo %d: %v", convo.ID, err)
		lastMessage = &pb.Message{Content: "Error loading message..."}
	} else {
		// Success, convert the last message
		lastMessage, _ = s.gormToGrpcMessage(ctx, &lastMessageGORM)
	}
	// --- END FIX ---

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

func (s *server) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	log.Printf("GetMessages request from user %d for convo %s", req.UserId, req.ConversationId)

	// --- Step 1: Validation (Security Check) ---
	// Check if the user is a participant in this conversation.
	var participantCount int64
	convoID, _ := strconv.ParseUint(req.ConversationId, 10, 64)
	if convoID == 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid conversation ID format")
	}

	s.db.Model(&Participant{}).Where("conversation_id = ? AND user_id = ?", convoID, req.UserId).Count(&participantCount)
	if participantCount == 0 {
		return nil, status.Error(codes.PermissionDenied, "User is not a participant of this conversation")
	}

	// --- Step 2: Fetch Messages (with pagination) ---
	var messages []Message
	if err := s.db.Where("conversation_id = ?", convoID).
		Order("created_at DESC"). // Get newest messages first
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&messages).Error; err != nil {
		log.Printf("Failed to get messages for convo %d: %v", convoID, err)
		return nil, status.Error(codes.Internal, "Failed to retrieve messages")
	}

	// --- Step 3: Convert GORM models to gRPC responses ---
	var grpcMessages []*pb.Message
	for _, msg := range messages {
		// We can re-use the helper we already built
		grpcMsg, err := s.gormToGrpcMessage(ctx, &msg)
		if err != nil {
			// Log, but don't fail the entire request
			log.Printf("Failed to convert message %d: %v", msg.ID, err)
			continue
		}
		grpcMessages = append(grpcMessages, grpcMsg)
	}

	// Note: The frontend will receive these in reverse-chronological order
	// and should display them accordingly (e.g., prepending to a list).

	return &pb.GetMessagesResponse{
		Messages: grpcMessages,
	}, nil
}

func (s *server) GetConversations(ctx context.Context, req *pb.GetConversationsRequest) (*pb.GetConversationsResponse, error) {
	log.Printf("GetConversations request received for user %d", req.UserId)

	// --- THIS IS THE FIX ---
	// 1. Get list of conversations user has "hidden" (soft-deleted)
	var hiddenConvoIDs []uint
	s.db.Model(&HiddenConversation{}).Where("user_id = ?", req.UserId).Pluck("conversation_id", &hiddenConvoIDs)
	// --- END FIX ---

	// --- Step 2: Find all Conversation IDs the user is a part of ---
	var conversationIDs []uint
	if err := s.db.Model(&Participant{}).
		Where("user_id = ?", req.UserId).
		Pluck("conversation_id", &conversationIDs).Error; err != nil {
		log.Printf("Failed to get conversation IDs for user %d: %v", req.UserId, err)
		return nil, status.Error(codes.Internal, "Failed to get conversation list")
	}

	if len(conversationIDs) == 0 {
		return &pb.GetConversationsResponse{Conversations: []*pb.Conversation{}}, nil
	}

	// --- Step 3: Fetch those conversations, sorted by most recent activity
	// --- AND FILTERING OUT THE HIDDEN ONES ---
	var conversations []Conversation
	query := s.db.Where("id IN ?", conversationIDs).
		Order("updated_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset))

	if len(hiddenConvoIDs) > 0 {
		query = query.Where("id NOT IN ?", hiddenConvoIDs) // <-- ADD THIS
	}

	if err := query.Find(&conversations).Error; err != nil {
		log.Printf("Failed to get conversations for user %d: %v", req.UserId, err)
		return nil, status.Error(codes.Internal, "Failed to retrieve conversations")
	}

	// --- Step 4: Convert GORM models to gRPC responses ---
	var grpcConversations []*pb.Conversation
	for _, convo := range conversations {
		grpcConvo, err := s.gormToGrpcConversation(ctx, &convo)
		if err != nil {
			log.Printf("Failed to convert conversation %d: %v", convo.ID, err)
			continue
		}
		grpcConversations = append(grpcConversations, grpcConvo)
	}

	return &pb.GetConversationsResponse{
		Conversations: grpcConversations,
	}, nil
}

// handleWebSocket upgrades the HTTP request to a WebSocket connection
func (s *server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// TODO: Auth!
	// We MUST get the userID from the auth token.
	// For now, we'll fake it with a query param: /ws?user_id=1
	userIDStr := r.URL.Query().Get("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)
	if userID == 0 {
		http.Error(w, "Unauthorized: Missing user_id query parameter", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}

	// Get all conversation IDs for this user
	convoIDs, err := s.getConversationIDsForUser(userID)
	if err != nil {
		log.Printf("Failed to get convo IDs for user %d: %v", userID, err)
		conn.Close()
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		convoIDs: convoIDs,
	}
	s.hub.register <- client

	// Start goroutines to handle reading and writing for this client
	go client.writePump()
	go client.readPump(s.hub)
}

// getConversationIDsForUser is a helper to find all convos a user is in
func (s *server) getConversationIDsForUser(userID int64) (map[string]bool, error) {
	var conversationIDs []uint
	if err := s.db.Model(&Participant{}).Where("user_id = ?", userID).Pluck("conversation_id", &conversationIDs).Error; err != nil {
		return nil, err
	}

	idMap := make(map[string]bool)
	for _, id := range conversationIDs {
		idMap[strconv.FormatUint(uint64(id), 10)] = true
	}
	return idMap, nil
}

// listenForRealtimeMessages is the Redis subscriber (Solution 4.2)
func (s *server) listenForRealtimeMessages() {
	log.Println("Redis Pub/Sub listener started...")
	// Subscribe to all chat channels
	pubsub := s.rdb.PSubscribe(context.Background(), "chat:*")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		log.Printf("Received message from Redis channel %s", msg.Channel)

		// We don't need to parse msg.Channel, we just need the payload
		// The payload is the JSON of the pb.Message we sent from SendMessage
		var grpcMessage pb.Message
		if err := json.Unmarshal([]byte(msg.Payload), &grpcMessage); err != nil {
			log.Printf("Failed to unmarshal message from redis: %v", err)
			continue
		}

		convoID := grpcMessage.ConversationId

		// Find all clients who are part of this conversation
		s.hub.clients.Range(func(key, value interface{}) bool {
			client, ok := value.(*Client)
			if !ok {
				return true // continue
			}

			// If the client is subscribed to this conversation
			if _, subscribed := client.convoIDs[convoID]; subscribed {
				// Send the message
				select {
				case client.send <- []byte(msg.Payload):
				default:
					// Failed to send, client buffer is full
					log.Printf("Failed to send to client %d, closing", client.userID)
					close(client.send)
					s.hub.clients.Delete(client.userID)
				}
			}
			return true
		})
	}
}

// --- WebSocket Client Helper Methods ---

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	// Set read limits, etc. (omitted for brevity)
	for {
		// Read message from client (e.g., "ping", "user is typing")
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
		// We don't process client messages for now, just keep connection alive
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

// --- ADDED: Phase 6 Completion RPCs ---

func (s *server) UnsendMessage(ctx context.Context, req *pb.UnsendMessageRequest) (*pb.UnsendMessageResponse, error) {
	log.Printf("UnsendMessage request from user %d for msg %s", req.UserId, req.MessageId)

	msgID, _ := strconv.ParseUint(req.MessageId, 10, 64)
	if msgID == 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid message ID format")
	}

	// 1. Find the message
	var msg Message
	if err := s.db.First(&msg, msgID).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Message not found")
	}

	// 2. Security Check: Are you the sender?
	if msg.SenderID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "You are not the sender of this message")
	}

	// 3. Business Logic: Can only unsend within 1 minute
	if time.Since(msg.CreatedAt) > (1 * time.Minute) {
		return nil, status.Error(codes.FailedPrecondition, "Cannot unsend a message after 1 minute")
	}

	// 4. Delete the message
	if err := s.db.Delete(&msg).Error; err != nil {
		log.Printf("Failed to delete message %d: %v", msgID, err)
		return nil, status.Error(codes.Internal, "Failed to delete message")
	}

	// 5. Publish to Redis for real-time update
	// We send a special "delete" type message
	deletePayload := map[string]string{
		"type":       "DELETE",
		"message_id": req.MessageId,
		"convo_id":   strconv.FormatUint(uint64(msg.ConversationID), 10),
	}
	msgBody, _ := json.Marshal(deletePayload)
	channelName := fmt.Sprintf("chat:%d", msg.ConversationID)

	if err := s.rdb.Publish(ctx, channelName, msgBody).Err(); err != nil {
		log.Printf("Failed to publish unsend message to redis channel %s: %v", channelName, err)
	}

	return &pb.UnsendMessageResponse{Message: "Message deleted"}, nil
}

func (s *server) DeleteConversation(ctx context.Context, req *pb.DeleteConversationRequest) (*pb.DeleteConversationResponse, error) {
	log.Printf("DeleteConversation (soft) request from user %d for convo %s", req.UserId, req.ConversationId)

	convoID, _ := strconv.ParseUint(req.ConversationId, 10, 64)
	if convoID == 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid conversation ID format")
	}

	// This is a soft delete. We just add an entry to the HiddenConversation table.
	// Our GetConversations RPC will now filter this out.
	hiddenEntry := HiddenConversation{
		UserID:         req.UserId,
		ConversationID: uint(convoID),
	}

	// Use 'clause.OnConflict{DoNothing: true}' in case they try to delete it twice
	if err := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&hiddenEntry).Error; err != nil {
		log.Printf("Failed to soft-delete conversation %d for user %d: %v", convoID, req.UserId, err)
		return nil, status.Error(codes.Internal, "Failed to hide conversation")
	}

	return &pb.DeleteConversationResponse{Message: "Conversation hidden"}, nil
}

func (s *server) GetVideoCallToken(ctx context.Context, req *pb.GetVideoCallTokenRequest) (*pb.GetVideoCallTokenResponse, error) {
	log.Printf("GetVideoCallToken request from user %d for convo %s", req.UserId, req.ConversationId)

	// --- Step 1: Validation (Security Check) ---
	var participantCount int64
	convoID, _ := strconv.ParseUint(req.ConversationId, 10, 64)
	if convoID == 0 {
		return nil, status.Error(codes.InvalidArgument, "Invalid conversation ID format")
	}

	s.db.Model(&Participant{}).Where("conversation_id = ? AND user_id = ?", convoID, req.UserId).Count(&participantCount)
	if participantCount == 0 {
		return nil, status.Error(codes.PermissionDenied, "User is not a participant of this conversation")
	}

	// --- Step 2: Get API Keys from Environment ---
	apiKey := os.Getenv("VIDEOSDK_API_KEY")
	apiSecret := os.Getenv("VIDEOSDK_API_SECRET")

	if apiKey == "" || apiSecret == "" || strings.Contains(apiKey, "YOUR_") {
		log.Println("VIDEOSDK_API_KEY or VIDEOSDK_API_SECRET is not set in environment")
		return nil, status.Error(codes.Internal, "Video service is not configured on the server")
	}

	// --- Step 3: Create VideoSDK JWT Token ---
	// This token is valid for 10 minutes
	expirationTime := time.Now().Add(10 * time.Minute).Unix()

	claims := jwt.MapClaims{
		"apikey":        apiKey,
		"permissions":   []string{"allow_join"}, // User can join a room
		"version":       2,                      // Use v2 of VideoSDK tokens
		"roomId":        req.ConversationId,     // Use our convo ID as the room ID
		"participantId": strconv.FormatInt(req.UserId, 10), // User's ID as a string
		"iat":           time.Now().Unix(),
		"exp":           expirationTime,
	}

	// Create a new token object, specifying signing method and claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	signedToken, err := token.SignedString([]byte(apiSecret))
	if err != nil {
		log.Printf("Failed to sign VideoSDK token: %v", err)
		return nil, status.Error(codes.Internal, "Failed to generate video token")
	}

	return &pb.GetVideoCallTokenResponse{
		Token:  signedToken,
		RoomId: req.ConversationId,
	}, nil
}