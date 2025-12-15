package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

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
	AuthorID  int64
	MediaURL  string
	MediaType string // "image" or "video"
	Caption   string
	ExpiresAt time.Time // Crucial for 24h logic

	// Denormalized data
	AuthorUsername   string
	AuthorProfileURL string

	FilterName       string
	StickersJSON     string `gorm:"type:text"`
	CloseFriendsOnly bool
}

type StoryLike struct {
	UserID    int64 `gorm:"primaryKey"`
	StoryID   int64 `gorm:"primaryKey"`
	CreatedAt time.Time
}

type StoryView struct {
	UserID    int64 `gorm:"primaryKey"`
	StoryID   int64 `gorm:"primaryKey"`
	CreatedAt time.Time
}

// server struct holds its DB, user-service client, and RabbitMQ channel
type server struct {
	pb.UnimplementedStoryServiceServer
	db         *gorm.DB
	userClient userPb.UserServiceClient
	amqpCh     *amqp.Channel
}

func main() {
	// --- Step 1: Connect to Story DB ---
	dbHost := os.Getenv("STORY_DB_HOST")
	if dbHost == "" {
		dbHost = "story-db"
	}
	dbUser := os.Getenv("STORY_DB_USER")
	if dbUser == "" {
		dbUser = "admin"
	}
	dbPassword := os.Getenv("STORY_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "password"
	}
	dbName := os.Getenv("STORY_DB_NAME")
	if dbName == "" {
		dbName = "story_service_db"
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=UTC", dbHost, dbUser, dbPassword, dbName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to story-db: %v", err)
	}
	// Update schema with new fields
	db.AutoMigrate(&Story{}, &StoryLike{}, &StoryView{})

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
		if err == nil {
			log.Println("Successfully connected to RabbitMQ")
			break
		}
		log.Printf("Failed to connect to RabbitMQ: %v. Retrying...", err)
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

	// --- Step 3b: Declare RabbitMQ Queues for Delayed Deletion ---

	// 1. The Destination Queue (Worker listens to this)
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

	// 2. The "Waiting Room" Queue (Messages sit here for 24h)
	// We use Dead Letter Exchange (DLX) to route expired messages to the destination
	args := amqp.Table{
		// TTL: 24 hours in milliseconds (24 * 60 * 60 * 1000)
		"x-message-ttl": int32(86400000),
		// When expired, send to default exchange
		"x-dead-letter-exchange": "",
		// With this routing key (the destination queue name)
		"x-dead-letter-routing-key": "story_deletion_queue",
	}

	_, err = amqpCh.QueueDeclare(
		"story_wait_24h",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // <--- Apply the DLX config
	)
	if err != nil {
		log.Fatalf("Failed to declare story_wait_24h queue: %v", err)
	}
	log.Println("RabbitMQ story queues configured")

	// Also declare notification queue for likes
	_, err = amqpCh.QueueDeclare("notification_queue", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare notification_queue: %v", err)
	}

	// --- Step 4: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9002") // Port 9002
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterStoryServiceServer(s, &server{
		db:         db,
		userClient: userClient,
		amqpCh:     amqpCh,
	})

	log.Println("Story service listening on port 9002...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
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
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

// --- 1. Create Story (With Expiry) ---
func (s *server) CreateStory(ctx context.Context, req *pb.CreateStoryRequest) (*pb.CreateStoryResponse, error) {
	log.Println("CreateStory request received")

	// 1. Get User Info for denormalization
	userRes, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: req.AuthorId})
	if err != nil {
		log.Printf("Failed to get user data: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve author details")
	}

	// 2. Calculate Expiry
	expiresAt := time.Now().Add(24 * time.Hour)

	// 3. Create Story Object
	newStory := Story{
		AuthorID:         req.AuthorId,
		MediaURL:         req.MediaUrl,
		MediaType:        req.MediaType,
		Caption:          req.Caption,
		ExpiresAt:        expiresAt,
		AuthorUsername:   userRes.Username,
		AuthorProfileURL: userRes.ProfilePictureUrl,
		FilterName:       req.FilterName,
		StickersJSON:     req.StickersJson,
		CloseFriendsOnly: req.CloseFriendsOnly,
	}

	if err := s.db.Create(&newStory).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to save story to database")
	}

	// 4. Trigger video compression for video stories (Optional for stories since they're temporary)
	// Only compress if it's a video to save bandwidth during 24h lifespan
	if req.MediaType == "video" && isVideoFile(req.MediaUrl) {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		msgBody, _ := json.Marshal(map[string]interface{}{
			"story_id":  newStory.ID,
			"media_url": req.MediaUrl,
			"is_story":  true, // Flag to indicate this is a story, not a post
		})

		err = s.publishToQueue(ctxTimeout, "video_transcoding_queue", msgBody)
		if err != nil {
			log.Printf("Failed to publish video transcoding job for story %d: %v", newStory.ID, err)
			// Don't fail the story creation, just log the error
		} else {
			log.Printf("Published video transcoding job for story %d", newStory.ID)
		}
	}

	// 5. Publish to the "Waiting Room" Queue for 24h deletion
	// The worker will pick this up from 'story_deletion_queue' after 24h
	msgBody, _ := json.Marshal(map[string]interface{}{
		"type":     "story.delete",
		"story_id": newStory.ID,
	})

	err = s.publishToQueue(ctx, "story_wait_24h", msgBody)
	if err != nil {
		log.Printf("Failed to schedule deletion task: %v", err)
		// We don't fail the request, just log warning.
		// In production, you might want a cron backup for cleanup.
	} else {
		log.Printf("Scheduled 24h deletion for story %d", newStory.ID)
	}

	return &pb.CreateStoryResponse{
		Story: &pb.Story{
			Id:               strconv.FormatUint(uint64(newStory.ID), 10),
			MediaUrl:         newStory.MediaURL,
			MediaType:        newStory.MediaType,
			Caption:          newStory.Caption,
			AuthorUsername:   newStory.AuthorUsername,
			AuthorProfileUrl: newStory.AuthorProfileURL,
			CreatedAt:        newStory.CreatedAt.Format(time.RFC3339),
			ExpiresAt:        newStory.ExpiresAt.Format(time.RFC3339),
		},
	}, nil
}

// --- 2. Get Story Feed (Grouped by User) ---
func (s *server) GetStoryFeed(ctx context.Context, req *pb.GetStoryFeedRequest) (*pb.GetStoryFeedResponse, error) {
	// 1. Get Following List
	followingRes, err := s.userClient.GetFollowingList(ctx, &userPb.GetFollowingListRequest{UserId: req.UserId})
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to get following list")
	}
	// Include Self so user sees their own story
	targetIDs := append(followingRes.FollowingUserIds, req.UserId)

	// Get close friends list
	closeFriendsRes, err := s.userClient.GetCloseFriends(ctx, &userPb.GetCloseFriendsRequest{UserId: req.UserId})
	if err != nil {
		log.Printf("Failed to get close friends: %v", err)
	}
	closeFriendsMap := make(map[int64]bool)
	if closeFriendsRes != nil {
		for _, friend := range closeFriendsRes.Friends {
			closeFriendsMap[friend.UserId] = true
		}
	}

	// Get hidden story users (users who have hidden their stories from viewer)
	hiddenUsersRes, err := s.userClient.GetHiddenStoryUsers(ctx, &userPb.GetHiddenStoryUsersRequest{UserId: req.UserId})
	if err != nil {
		log.Printf("Failed to get hidden story users: %v", err)
	}
	hiddenUsersMap := make(map[int64]bool)
	if hiddenUsersRes != nil {
		for _, user := range hiddenUsersRes.HiddenUsers {
			hiddenUsersMap[user.UserId] = true
		}
	}

	// 2. Fetch Active Stories (ExpiresAt > Now)
	var stories []Story
	if err := s.db.Where("author_id IN ? AND expires_at > ?", targetIDs, time.Now()).
		Order("created_at ASC"). // Oldest first (chronological for stories)
		Find(&stories).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to fetch stories")
	}

	// Filter stories based on close friends, hidden story settings, and blocks
	var filteredStories []Story
	for _, story := range stories {
		// Skip if author has hidden stories from this user
		if hiddenUsersMap[story.AuthorID] {
			continue
		}

		// Check if there's a block relationship (either direction)
		if story.AuthorID != req.UserId {
			blockCheckAuthorToViewer, err := s.userClient.IsBlocked(ctx, &userPb.IsBlockedRequest{
				BlockerId: story.AuthorID,
				BlockedId: req.UserId,
			})
			if err == nil && blockCheckAuthorToViewer.IsBlocked {
				continue // Author blocked the viewer
			}

			blockCheckViewerToAuthor, err := s.userClient.IsBlocked(ctx, &userPb.IsBlockedRequest{
				BlockerId: req.UserId,
				BlockedId: story.AuthorID,
			})
			if err == nil && blockCheckViewerToAuthor.IsBlocked {
				continue // Viewer blocked the author
			}
		}

		// Skip close friends only stories if viewer is not a close friend (unless it's their own story)
		if story.CloseFriendsOnly && story.AuthorID != req.UserId && !closeFriendsMap[story.AuthorID] {
			continue
		}

		filteredStories = append(filteredStories, story)
	}
	stories = filteredStories

	// 3. Get user's likes and views for these stories
	storyIDs := make([]uint, len(stories))
	for i, story := range stories {
		storyIDs[i] = story.ID
	}

	// Query likes
	var userLikes []StoryLike
	s.db.Where("user_id = ? AND story_id IN ?", req.UserId, storyIDs).Find(&userLikes)
	likedMap := make(map[uint]bool)
	for _, like := range userLikes {
		likedMap[uint(like.StoryID)] = true
	}

	// Query views
	var userViews []StoryView
	s.db.Where("user_id = ? AND story_id IN ?", req.UserId, storyIDs).Find(&userViews)
	viewedMap := make(map[uint]bool)
	for _, view := range userViews {
		viewedMap[uint(view.StoryID)] = true
	}

	// 4. Group Stories by AuthorID
	groupedMap := make(map[int64]*pb.UserStoryGroup)

	for _, story := range stories {
		// Initialize group if not exists
		if _, exists := groupedMap[story.AuthorID]; !exists {
			groupedMap[story.AuthorID] = &pb.UserStoryGroup{
				UserId:         story.AuthorID,
				Username:       story.AuthorUsername,
				UserProfileUrl: story.AuthorProfileURL,
				Stories:        []*pb.Story{},
				AllSeen:        false,
			}
		}

		// Add story to group with is_liked status
		groupedMap[story.AuthorID].Stories = append(groupedMap[story.AuthorID].Stories, &pb.Story{
			Id:               strconv.FormatUint(uint64(story.ID), 10),
			MediaUrl:         story.MediaURL,
			MediaType:        story.MediaType,
			Caption:          story.Caption,
			CreatedAt:        story.CreatedAt.Format(time.RFC3339),
			ExpiresAt:        story.ExpiresAt.Format(time.RFC3339),
			IsLiked:          likedMap[story.ID],
			FilterName:       story.FilterName,
			StickersJson:     story.StickersJSON,
			CloseFriendsOnly: story.CloseFriendsOnly,
		})
	}

	// 5. Calculate all_seen for each group
	for _, group := range groupedMap {
		allSeen := true
		for _, story := range group.Stories {
			storyID, _ := strconv.ParseUint(story.Id, 10, 32)
			if !viewedMap[uint(storyID)] {
				allSeen = false
				break
			}
		}
		group.AllSeen = allSeen
	}

	// 6. Convert Map to Slice
	var responseGroups []*pb.UserStoryGroup
	for _, group := range groupedMap {
		// Optional: Put "Self" (current user) at the very front
		if group.UserId == req.UserId {
			responseGroups = append([]*pb.UserStoryGroup{group}, responseGroups...)
		} else {
			responseGroups = append(responseGroups, group)
		}
	}

	return &pb.GetStoryFeedResponse{StoryGroups: responseGroups}, nil
}

// --- 3. Delete Story (Manual or Worker Triggered) ---
func (s *server) DeleteStory(ctx context.Context, req *pb.DeleteStoryRequest) (*pb.DeleteStoryResponse, error) {
	// In a real app, you'd check if req.UserId owns the story or is admin/worker
	if err := s.db.Delete(&Story{}, req.StoryId).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to delete story")
	}
	log.Printf("Story %d deleted successfully", req.StoryId)
	return &pb.DeleteStoryResponse{Message: "Story deleted"}, nil
}

// --- 4. Like Story ---
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

	// Send notification (unless liking own story)
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

// --- 5. Unlike Story ---
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

// --- 6. View Story ---
func (s *server) ViewStory(ctx context.Context, req *pb.ViewStoryRequest) (*pb.ViewStoryResponse, error) {
	view := StoryView{
		UserID:  req.UserId,
		StoryID: req.StoryId,
	}
	// Use FirstOrCreate to avoid duplicate errors
	if result := s.db.FirstOrCreate(&view, view); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to mark story as viewed")
	}
	return &pb.ViewStoryResponse{Message: "Story viewed"}, nil
}

// --- 6. Get User Archive (All Stories for a Specific User) ---
func (s *server) GetUserArchive(ctx context.Context, req *pb.GetUserArchiveRequest) (*pb.GetUserArchiveResponse, error) {
	// Fetch all stories by this user (including expired ones for archive)
	var stories []Story
	if err := s.db.Where("author_id = ?", req.UserId).
		Order("created_at DESC"). // Newest first
		Find(&stories).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to fetch archive stories")
	}

	// Convert to protobuf
	var pbStories []*pb.Story
	for _, story := range stories {
		pbStories = append(pbStories, &pb.Story{
			Id:               strconv.FormatUint(uint64(story.ID), 10),
			MediaUrl:         story.MediaURL,
			MediaType:        story.MediaType,
			Caption:          story.Caption,
			AuthorUsername:   story.AuthorUsername,
			AuthorProfileUrl: story.AuthorProfileURL,
			CreatedAt:        story.CreatedAt.Format(time.RFC3339),
			ExpiresAt:        story.ExpiresAt.Format(time.RFC3339),
			FilterName:       story.FilterName,
			StickersJson:     story.StickersJSON,
		})
	}

	return &pb.GetUserArchiveResponse{Stories: pbStories}, nil
}

// --- Helper Function: Check if URL is a video file ---
func isVideoFile(url string) bool {
	if url == "" {
		return false
	}
	videoExtensions := []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv"}
	urlLower := strings.ToLower(url)
	for _, ext := range videoExtensions {
		if strings.HasSuffix(urlLower, ext) || strings.Contains(urlLower, ext) {
			return true
		}
	}
	return false
}
