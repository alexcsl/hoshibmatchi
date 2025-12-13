package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	pb "github.com/hoshibmatchi/post-service/proto"
	userPb "github.com/hoshibmatchi/user-service/proto"

	"github.com/go-redis/redis/v8"
	"github.com/lib/pq" // For string arrays
	amqp "github.com/rabbitmq/amqp091-go"
)

var hashtagRegex = regexp.MustCompile(`#(\w+)`)

// Post defines the GORM model
type Post struct {
	gorm.Model
	AuthorID         int64
	Caption          string
	MediaURLs        pq.StringArray `gorm:"type:text[]"`
	IsReel           bool           `gorm:"default:false"`
	CommentsDisabled bool           `gorm:"default:false"`
	ThumbnailURL     string         `gorm:"type:varchar(255)"`
	LikeCount        int64          `gorm:"default:0"`
	CommentCount     int64          `gorm:"default:0"`
	ShareCount       int64          `gorm:"default:0"`

	Location        string
	CollaboratorIDs pq.Int64Array `gorm:"type:bigint[]"`

	// Denormalized fields from user-service
	AuthorUsername   string
	AuthorProfileURL string
	AuthorIsVerified bool
}

// SharedPost tracks when a user shares a post
type SharedPost struct {
	gorm.Model
	UserID         int64  // Who shared it
	OriginalPostID int64  // The post being shared
	Caption        string // Optional comment when sharing
}

// PostLike defines the GORM model for a like on a post
type PostLike struct {
	// Composite primary key (user_id, post_id)
	UserID    int64 `gorm:"primaryKey"`
	PostID    int64 `gorm:"primaryKey"`
	CreatedAt time.Time
}

// Comment defines the GORM model for a comment
type Comment struct {
	gorm.Model
	UserID  int64
	PostID  int64
	Content string // This can be text or a GIF URL

	// For nested replies
	ParentCommentID uint // GORM's Model.ID is uint

	// Denormalized data from user-service
	AuthorUsername   string
	AuthorProfileURL string
	AuthorIsVerified bool
}

// CommentLike defines the GORM model for a like on a comment
type CommentLike struct {
	// Composite primary key (user_id, comment_id)
	UserID    int64 `gorm:"primaryKey"`
	CommentID int64 `gorm:"primaryKey"`
	CreatedAt time.Time
}

type server struct {
	pb.UnimplementedPostServiceServer
	db          *gorm.DB
	userClient  userPb.UserServiceClient
	amqpCh      *amqp.Channel
	minioClient *minio.Client
	rdb         *redis.Client
}

// Collection defines a user's named collection of posts
type Collection struct {
	gorm.Model
	UserID    int64  `gorm:"index"`
	Name      string `gorm:"type:varchar(100)"`
	IsDefault bool   `gorm:"default:false"`
}

type PostCollaborator struct {
	PostID int64 `gorm:"primaryKey"`
	UserID int64 `gorm:"primaryKey"`
}

// SavedPost is the join table for the many-to-many relationship
// between collections and posts
type SavedPost struct {
	// Composite primary key
	CollectionID uint `gorm:"primaryKey"`
	PostID       uint `gorm:"primaryKey"`
	CreatedAt    time.Time
}

func main() {
	// --- Step 1: Connect to Post DB ---
	dsn := "host=post-db user=admin password=password dbname=post_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to post-db: %v", err)
	}
	db.AutoMigrate(&Post{})
	db.AutoMigrate(&PostLike{})
	db.AutoMigrate(&Comment{})
	db.AutoMigrate(&CommentLike{})
	db.AutoMigrate(&Collection{})
	db.AutoMigrate(&SavedPost{})
	db.AutoMigrate(&PostCollaborator{})
	db.AutoMigrate(&SharedPost{})

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
		amqpURI := os.Getenv("RABBITMQ_URI")
		if amqpURI == "" {
			amqpURI = "amqp://guest:guest@rabbitmq:5672/" // Default
		}
		amqpConn, err = amqp.Dial(amqpURI)
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

	_, err = amqpCh.QueueDeclare(
		"notification_queue", // queue name
		true,                 // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare notification_queue: %v", err)
	}
	log.Println("RabbitMQ notification_queue declared")

	// --- ADDED: Declare video transcoding queue ---
	_, err = amqpCh.QueueDeclare(
		"video_transcoding_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare video_transcoding_queue: %v", err)
	}
	log.Println("RabbitMQ video_transcoding_queue declared")

	// --- ADDED: Declare hashtag processing queue ---
	_, err = amqpCh.QueueDeclare(
		"hashtag_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare hashtag_queue: %v", err)
	}
	log.Println("RabbitMQ hashtag_queue declared")

	// --- ADDED: Connect to MinIO ---
	// Get MinIO credentials from environment
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "minio:9000" // Default
	}
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	if accessKeyID == "" {
		accessKeyID = "minioadmin" // Default
	}
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	if secretAccessKey == "" {
		secretAccessKey = "minioadmin" // Default
	}
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("Failed to connect to minio: %v", err)
	}
	log.Println("Post-service successfully connected to MinIO")

	// --- Step 5: Connect to Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	log.Println("Post-service successfully connected to Redis")

	// --- Step 6: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9001") // Port 9001
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()

	// --- THIS IS THE FIX ---
	// We must pass the amqpCh to the server struct
	pb.RegisterPostServiceServer(s, &server{
		db:          db,
		userClient:  userClient,
		amqpCh:      amqpCh,
		minioClient: minioClient,
		rdb:         rdb,
	})
	// --- END FIX ---

	log.Println("Post service listening on port 9001...")
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

// canViewPost checks if a user can view a post based on privacy settings
func (s *server) canViewPost(ctx context.Context, post *Post, viewerID int64) bool {
	// User can always view their own posts
	if post.AuthorID == viewerID {
		return true
	}

	// Get the author's profile to check privacy settings
	profileResp, err := s.userClient.GetUserProfile(ctx, &userPb.GetUserProfileRequest{
		Username:   post.AuthorUsername,
		SelfUserId: viewerID,
	})
	if err != nil {
		log.Printf("Failed to get author profile: %v", err)
		return false
	}

	// If account is public, anyone can view
	if !profileResp.IsPrivate {
		return true
	}

	// If account is private, check if viewer is following
	followResp, err := s.userClient.IsFollowing(ctx, &userPb.IsFollowingRequest{
		FollowerId:  viewerID,
		FollowingId: post.AuthorID,
	})
	if err != nil {
		log.Printf("Failed to check follow status: %v", err)
		return false
	}

	return followResp.IsFollowing
}

// filterPostsByPrivacy filters a slice of posts based on privacy settings
func (s *server) filterPostsByPrivacy(ctx context.Context, posts []Post, viewerID int64) []Post {
	var visiblePosts []Post
	for _, post := range posts {
		if s.canViewPost(ctx, &post, viewerID) {
			visiblePosts = append(visiblePosts, post)
		}
	}
	return visiblePosts
}

// --- Implement CreatePost ---
func (s *server) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	log.Println("CreatePost request received")

	// Input validation
	if len(req.Caption) > 2200 {
		return nil, status.Error(codes.InvalidArgument, "Caption must not exceed 2200 characters")
	}
	if len(req.MediaUrls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "At least one media URL is required")
	}

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
		IsReel:           req.IsReel,
		CommentsDisabled: req.CommentsDisabled,
		ThumbnailURL:     req.ThumbnailUrl,
		// Add denormalized data
		AuthorUsername:   userData.Username,
		AuthorProfileURL: userData.ProfilePictureUrl,
		AuthorIsVerified: userData.IsVerified,

		Location:        req.Location,
		CollaboratorIDs: req.CollaboratorIds,
	}

	// --- Step 3: Create Post and Collaborators in a transaction ---
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the Post
		if result := tx.Create(&newPost); result.Error != nil {
			return result.Error
		}

		// 2. Add author and collaborators to the join table
		collaborators := []PostCollaborator{}
		// Add the author (so their posts appear in their own profile)
		collaborators = append(collaborators, PostCollaborator{PostID: int64(newPost.ID), UserID: req.AuthorId})

		if len(req.CollaboratorIds) > 0 {
			for _, userID := range req.CollaboratorIds {
				if userID != req.AuthorId { // Avoid duplicates
					collaborators = append(collaborators, PostCollaborator{
						PostID: int64(newPost.ID),
						UserID: userID,
					})
				}
			}
		}

		// 3. Save collaborators
		if err := tx.Create(&collaborators).Error; err != nil {
			return err // Rollback
		}

		return nil // Commit
	})
	if err != nil {
		log.Printf("Failed to create post in transaction: %v", err)
		return nil, status.Error(codes.Internal, "Failed to save post to database")
	}

	// We check if it's a Reel OR if any media URLs look like videos.
	isAVideoJob := newPost.IsReel
	if !isAVideoJob {
		for _, url := range newPost.MediaURLs {
			if strings.HasSuffix(url, ".mp4") || strings.HasSuffix(url, ".mov") {
				isAVideoJob = true
				break
			}
		}
	}

	if isAVideoJob {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		msgBody, _ := json.Marshal(map[string]interface{}{
			"post_id":    newPost.ID,
			"media_urls": newPost.MediaURLs,
		})

		err = s.amqpCh.PublishWithContext(
			ctxTimeout,
			"",                        // exchange (default)
			"video_transcoding_queue", // routing key
			false,                     // mandatory
			false,                     // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         msgBody,
			},
		)
		if err != nil {
			log.Printf("Failed to publish video transcoding job for post %d: %v", newPost.ID, err)
		} else {
			log.Printf("Published video transcoding job for post %d", newPost.ID)
		}
	}

	// --- ADDED: Parse caption for hashtags and publish job ---
	matches := hashtagRegex.FindAllStringSubmatch(newPost.Caption, -1)
	if len(matches) > 0 {
		hashtagNames := []string{}
		uniqueTags := make(map[string]bool)
		for _, match := range matches {
			if len(match) > 1 {
				tag := strings.ToLower(match[1]) // Get the tag (group 1) and lowercase it
				if !uniqueTags[tag] {            // Ensure tags are unique per post
					uniqueTags[tag] = true
					hashtagNames = append(hashtagNames, tag)
				}
			}
		}

		if len(hashtagNames) > 0 {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			msgBody, _ := json.Marshal(map[string]interface{}{
				"post_id":       newPost.ID,
				"hashtag_names": hashtagNames,
			})

			err = s.amqpCh.PublishWithContext(
				ctxTimeout,
				"",              // exchange (default)
				"hashtag_queue", // routing key
				false,           // mandatory
				false,           // immediate
				amqp.Publishing{
					ContentType:  "application/json",
					DeliveryMode: amqp.Persistent,
					Body:         msgBody,
				},
			)
			if err != nil {
				log.Printf("Failed to publish hashtag job for post %d: %v", newPost.ID, err)
			} else {
				log.Printf("Published hashtag job for post %d with tags: %v", newPost.ID, hashtagNames)
			}
		}
	}

	// --- Clear feed caches since new post should appear in feeds ---
	feedPattern := "feed:*"
	iter := s.rdb.Scan(ctx, 0, feedPattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Error scanning feed cache keys: %v", err)
	}

	// --- Step 3: Return the created post ---
	return &pb.CreatePostResponse{
		Post: s.enrichPostProto(ctx, &newPost, req.AuthorId), // Requesting user is author
	}, nil
}

// --- Implement LikePost ---
func (s *server) LikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.LikePostResponse, error) {
	like := PostLike{
		UserID: req.UserId,
		PostID: req.PostId,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the like
		if err := tx.Create(&like).Error; err != nil {
			return err
		}
		// 2. Increment the post's like_count
		if err := tx.Model(&Post{}).Where("id = ?", req.PostId).Update("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "Post already liked")
		}
		return nil, status.Error(codes.Internal, "Failed to like post")
	}

	// RabbitMQ Notifications
	// Get Post Author ID
	var post Post
	s.db.First(&post, req.PostId)

	// Don't notify if user likes their own post
	if post.AuthorID != req.UserId {
		msgBody, _ := json.Marshal(map[string]interface{}{
			"type":      "post.liked",
			"actor_id":  req.UserId,
			"user_id":   post.AuthorID, // The user to be notified
			"entity_id": req.PostId,
		})
		s.publishToQueue(ctx, "notification_queue", msgBody)
	}
	// --- END ADD ---

	// --- Clear cache for this post and feed caches ---
	postCacheKey := fmt.Sprintf("post:%d", req.PostId)
	if err := s.rdb.Del(ctx, postCacheKey).Err(); err != nil {
		log.Printf("Failed to delete post cache key %s: %v", postCacheKey, err)
	}

	feedPattern := "feed:*"
	iter := s.rdb.Scan(ctx, 0, feedPattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Error scanning feed cache keys: %v", err)
	}

	return &pb.LikePostResponse{Message: "Post liked"}, nil
}

// --- Implement UnlikePost ---
func (s *server) UnlikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.UnlikePostResponse, error) {
	like := PostLike{
		UserID: req.UserId,
		PostID: req.PostId,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Delete the like
		result := tx.Delete(&like)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return status.Error(codes.NotFound, "Post was not liked")
		}
		// 2. Decrement the post's like_count
		if err := tx.Model(&Post{}).Where("id = ?", req.PostId).Update("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// --- Clear cache for this post and feed caches ---
	postCacheKey := fmt.Sprintf("post:%d", req.PostId)
	if err := s.rdb.Del(ctx, postCacheKey).Err(); err != nil {
		log.Printf("Failed to delete post cache key %s: %v", postCacheKey, err)
	}

	feedPattern := "feed:*"
	iter := s.rdb.Scan(ctx, 0, feedPattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Error scanning feed cache keys: %v", err)
	}

	return &pb.UnlikePostResponse{Message: "Post unliked"}, nil
}

// --- Implement CommentOnPost ---
func (s *server) CommentOnPost(ctx context.Context, req *pb.CommentOnPostRequest) (*pb.CommentResponse, error) {
	// Input validation
	if len(strings.TrimSpace(req.Content)) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Comment content cannot be empty")
	}
	if len(req.Content) > 500 {
		return nil, status.Error(codes.InvalidArgument, "Comment must not exceed 500 characters")
	}

	// --- Step 1: Call User Service for Denormalization (like in CreatePost) ---
	userData, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: req.UserId})
	if err != nil {
		log.Printf("Failed to get user data from user-service: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve author details")
	}

	// --- Step 2: Create the Comment in our DB ---
	newComment := Comment{
		UserID:           req.UserId,
		PostID:           req.PostId,
		Content:          req.Content,
		ParentCommentID:  uint(req.ParentCommentId), // 0 is fine
		AuthorUsername:   userData.Username,
		AuthorProfileURL: userData.ProfilePictureUrl,
		AuthorIsVerified: userData.IsVerified,
	}
	// Note: Do NOT set ID manually - let GORM auto-generate it

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the comment
		if err := tx.Create(&newComment).Error; err != nil {
			return err
		}
		// 2. Increment the post's comment_count
		if err := tx.Model(&Post{}).Where("id = ?", req.PostId).Update("comment_count", gorm.Expr("comment_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to create comment: %v", err)
		return nil, status.Error(codes.Internal, "Failed to save comment")
	}

	// Notification for comments
	// Get Post Author ID
	var post Post
	s.db.First(&post, req.PostId)

	// Don't notify if user comments on their own post
	if post.AuthorID != req.UserId {
		msgBody, _ := json.Marshal(map[string]interface{}{
			"type":      "post.commented",
			"actor_id":  req.UserId,
			"user_id":   post.AuthorID,
			"entity_id": req.PostId,
		})
		s.publishToQueue(ctx, "notification_queue", msgBody)
	}

	// --- Clear cache for this post and feed caches ---
	postCacheKey := fmt.Sprintf("post:%d", req.PostId)
	if err := s.rdb.Del(ctx, postCacheKey).Err(); err != nil {
		log.Printf("Failed to delete post cache key %s: %v", postCacheKey, err)
	}

	feedPattern := "feed:*"
	iter := s.rdb.Scan(ctx, 0, feedPattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Error scanning feed cache keys: %v", err)
	}

	// --- Step 3: Return the created comment ---
	return &pb.CommentResponse{
		Id:               strconv.FormatUint(uint64(newComment.ID), 10),
		Content:          newComment.Content,
		AuthorUsername:   newComment.AuthorUsername,
		AuthorProfileUrl: newComment.AuthorProfileURL,
		CreatedAt:        newComment.CreatedAt.Format(time.RFC3339),
		PostId:           newComment.PostID,
		ParentCommentId:  int64(newComment.ParentCommentID),
		UserId:           newComment.UserID,
		LikeCount:        0,
		IsLiked:          false,
		ReplyCount:       0,
		AuthorIsVerified: newComment.AuthorIsVerified,
	}, nil
}

// --- Implement DeleteComment ---
func (s *server) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentResponse, error) {
	var comment Comment

	// Find the comment first
	if err := s.db.First(&comment, req.CommentId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Comment not found")
	}

	// Check if user is comment owner OR post author
	var post Post
	if err := s.db.First(&post, comment.PostID).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve post")
	}

	isCommentOwner := comment.UserID == req.UserId
	isPostAuthor := post.AuthorID == req.UserId

	if !isCommentOwner && !isPostAuthor {
		return nil, status.Error(codes.PermissionDenied, "You do not have permission to delete this comment")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Delete the comment
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}
		// 2. Decrement the post's comment_count
		if err := tx.Model(&Post{}).Where("id = ?", comment.PostID).Update("comment_count", gorm.Expr("comment_count - 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to delete comment")
	}

	return &pb.DeleteCommentResponse{Message: "Comment deleted"}, nil
}

// --- GRPC: LikeComment ---
func (s *server) LikeComment(ctx context.Context, req *pb.LikeCommentRequest) (*pb.LikeCommentResponse, error) {
	// Check if comment exists
	var comment Comment
	if err := s.db.First(&comment, req.CommentId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Comment not found")
	}

	// Check if already liked
	var existingLike CommentLike
	err := s.db.Where("user_id = ? AND comment_id = ?", req.UserId, req.CommentId).First(&existingLike).Error
	if err == nil {
		// Already liked
		return &pb.LikeCommentResponse{Message: "Comment already liked"}, nil
	}

	// Create new like
	like := CommentLike{
		UserID:    req.UserId,
		CommentID: req.CommentId,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(&like).Error; err != nil {
		log.Printf("Failed to like comment: %v", err)
		return nil, status.Error(codes.Internal, "Failed to like comment")
	}

	return &pb.LikeCommentResponse{Message: "Comment liked"}, nil
}

// --- GRPC: UnlikeComment ---
func (s *server) UnlikeComment(ctx context.Context, req *pb.LikeCommentRequest) (*pb.UnlikeCommentResponse, error) {
	// Delete the like
	result := s.db.Where("user_id = ? AND comment_id = ?", req.UserId, req.CommentId).Delete(&CommentLike{})

	if result.Error != nil {
		log.Printf("Failed to unlike comment: %v", result.Error)
		return nil, status.Error(codes.Internal, "Failed to unlike comment")
	}

	if result.RowsAffected == 0 {
		return &pb.UnlikeCommentResponse{Message: "Comment was not liked"}, nil
	}

	return &pb.UnlikeCommentResponse{Message: "Comment unliked"}, nil
}

// --- GRPC: GetCommentsByPost ---
func (s *server) GetCommentsByPost(ctx context.Context, req *pb.GetCommentsByPostRequest) (*pb.GetCommentsByPostResponse, error) {
	// Get requesting user ID from context (if available)
	var requestingUserID int64
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userIDs := md.Get("user_id"); len(userIDs) > 0 {
			requestingUserID, _ = strconv.ParseInt(userIDs[0], 10, 64)
		}
	}

	var comments []Comment

	// Fetch ALL comments for the post (both top-level and replies)
	err := s.db.Where("post_id = ?", req.PostId).
		Order("created_at ASC").
		Find(&comments).Error

	if err != nil {
		log.Printf("Failed to fetch comments: %v", err)
		return nil, status.Error(codes.Internal, "Failed to fetch comments")
	}

	// Convert to proto response with additional fields
	var commentResponses []*pb.CommentResponse
	for _, comment := range comments {
		// Count likes for this comment
		var likeCount int64
		s.db.Model(&CommentLike{}).Where("comment_id = ?", comment.ID).Count(&likeCount)

		// Check if requesting user liked this comment
		var isLiked bool
		if requestingUserID > 0 {
			var like CommentLike
			err := s.db.Where("user_id = ? AND comment_id = ?", requestingUserID, comment.ID).First(&like).Error
			isLiked = err == nil
		}

		// Count replies for this comment
		var replyCount int64
		s.db.Model(&Comment{}).Where("parent_comment_id = ?", comment.ID).Count(&replyCount)

		commentResponses = append(commentResponses, &pb.CommentResponse{
			Id:               strconv.FormatUint(uint64(comment.ID), 10),
			Content:          comment.Content,
			AuthorUsername:   comment.AuthorUsername,
			AuthorProfileUrl: comment.AuthorProfileURL,
			CreatedAt:        comment.CreatedAt.Format(time.RFC3339),
			PostId:           comment.PostID,
			ParentCommentId:  int64(comment.ParentCommentID),
			UserId:           comment.UserID,
			LikeCount:        likeCount,
			IsLiked:          isLiked,
			ReplyCount:       replyCount,
			AuthorIsVerified: comment.AuthorIsVerified,
		})
	}

	return &pb.GetCommentsByPostResponse{
		Comments: commentResponses,
	}, nil
}

// --- GPRC: GetHomeFeed ---
func (s *server) GetHomeFeed(ctx context.Context, req *pb.GetHomeFeedRequest) (*pb.GetHomeFeedResponse, error) {
	log.Printf("GetHomeFeed request received for user %d", req.UserId)

	// // Try cache first
	cacheKey := strconv.FormatInt(req.UserId, 10) + ":" + strconv.FormatInt(int64(req.PageSize), 10) + ":" + strconv.FormatInt(int64(req.PageOffset), 10)
	cacheKey = "feed:home:" + cacheKey
	cachedData, err := s.rdb.Get(ctx, cacheKey).Result()

	if err == nil {
		// Cache hit
		var response pb.GetHomeFeedResponse
		if json.Unmarshal([]byte(cachedData), &response) == nil {
			log.Printf("Cache hit for home feed user %d", req.UserId)
			return &response, nil
		}
	}

	// --- Step 1: Get user's following list ---
	followingRes, err := s.userClient.GetFollowingList(ctx, &userPb.GetFollowingListRequest{UserId: req.UserId})
	if err != nil {
		log.Printf("Failed to get following list from user-service: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve user feed")
	}
	followingIDs := followingRes.FollowingUserIds

	// --- Step 2: Get posts where user is a collaborator ---
	var collaboratorPostIDs []int64
	// We *don't* want to re-show the user's *own* posts, so we filter them out
	s.db.Model(&PostCollaborator{}).
		Where("user_id = ? AND user_id NOT IN (SELECT author_id FROM posts WHERE id = post_collaborators.post_id)", req.UserId).
		Pluck("post_id", &collaboratorPostIDs)

	// --- Step 3: Query our DB for posts ---
	var posts []Post
	query := s.db.Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset))

	// Build the complex WHERE clause:
	// (author_id IN [followingIDs]) OR (id IN [collaboratorPostIDs])
	if len(followingIDs) > 0 && len(collaboratorPostIDs) > 0 {
		query = query.Where("author_id IN ? OR id IN ?", followingIDs, collaboratorPostIDs)
	} else if len(followingIDs) > 0 {
		query = query.Where("author_id IN ?", followingIDs)
	} else if len(collaboratorPostIDs) > 0 {
		query = query.Where("id IN ?", collaboratorPostIDs)
	} else {
		// Not following anyone and not a collaborator on any posts
		return &pb.GetHomeFeedResponse{Posts: []*pb.Post{}}, nil
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve posts")
	}

	// --- Step 4: Filter posts by privacy settings ---
	posts = s.filterPostsByPrivacy(ctx, posts, req.UserId)

	// --- Step 5: Convert GORM models to gRPC responses ---
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, s.enrichPostProto(ctx, &post, req.UserId))
	}

	response := &pb.GetHomeFeedResponse{Posts: grpcPosts}

	// Cache with 5 minute TTL
	if responseJSON, err := json.Marshal(response); err == nil {
		s.rdb.Set(ctx, cacheKey, responseJSON, 2*time.Minute)
	}

	return response, nil
}

// --- NEW: GetUserTaggedPosts ---
func (s *server) GetUserTaggedPosts(ctx context.Context, req *pb.GetUserContentRequest) (*pb.GetHomeFeedResponse, error) {
	var postIDs []int64

	// Find all posts where this user is a collaborator
	if err := s.db.Model(&PostCollaborator{}).
		Where("user_id = ?", req.UserId).
		Pluck("post_id", &postIDs).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to fetch tagged posts")
	}

	if len(postIDs) == 0 {
		return &pb.GetHomeFeedResponse{Posts: []*pb.Post{}}, nil
	}

	var posts []Post
	if err := s.db.Where("? = ANY(collaborator_ids) AND author_id != ?", req.UserId, req.UserId).
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to fetch tagged posts")
	}

	// Filter privacy & Enrich
	posts = s.filterPostsByPrivacy(ctx, posts, req.RequesterId)
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, s.enrichPostProto(ctx, &post, req.RequesterId))
	}

	return &pb.GetHomeFeedResponse{Posts: grpcPosts}, nil
}

// --- Implement GetExploreFeed ---
func (s *server) GetExploreFeed(ctx context.Context, req *pb.GetHomeFeedRequest) (*pb.GetHomeFeedResponse, error) {
	log.Println("GetExploreFeed request received")

	var posts []Post
	// This feed gets ALL posts (not just from followed users)
	// and filters out Reels
	if err := s.db.Where("is_reel = ?", false).
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve posts")
	}

	// Filter by privacy settings
	posts = s.filterPostsByPrivacy(ctx, posts, req.UserId)

	// Convert GORM models to gRPC responses
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, s.enrichPostProto(ctx, &post, req.UserId))
	}

	return &pb.GetHomeFeedResponse{Posts: grpcPosts}, nil
}

// --- Implement GetReelsFeed ---
func (s *server) GetReelsFeed(ctx context.Context, req *pb.GetHomeFeedRequest) (*pb.GetHomeFeedResponse, error) {
	log.Println("GetReelsFeed request received")

	var posts []Post
	// This feed gets ONLY posts that are Reels
	if err := s.db.Where("is_reel = ?", true).
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve posts")
	}

	// Filter by privacy settings
	posts = s.filterPostsByPrivacy(ctx, posts, req.UserId)

	// Convert GORM models to gRPC responses
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, s.enrichPostProto(ctx, &post, req.UserId))
	}

	return &pb.GetHomeFeedResponse{Posts: grpcPosts}, nil
}

// --- Implement GetUserPosts ---
func (s *server) GetUserPosts(ctx context.Context, req *pb.GetUserContentRequest) (*pb.GetHomeFeedResponse, error) {
	var posts []Post

	// Query for posts by author_id, filtering OUT reels
	if err := s.db.Where("author_id = ? AND is_reel = ?", req.UserId, false).
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve posts")
	}

	// Filter by privacy settings
	posts = s.filterPostsByPrivacy(ctx, posts, req.RequesterId)

	var grpcPosts []*pb.Post
	for _, post := range posts {
		// We pass req.RequesterId so we know if the viewer liked/saved them
		grpcPosts = append(grpcPosts, s.enrichPostProto(ctx, &post, req.RequesterId))
	}
	return &pb.GetHomeFeedResponse{Posts: grpcPosts}, nil
}

// --- Implement GetUserReels ---
func (s *server) GetUserReels(ctx context.Context, req *pb.GetUserContentRequest) (*pb.GetHomeFeedResponse, error) {
	var posts []Post

	// Query for posts by author_id, filtering FOR reels
	if err := s.db.Where("author_id = ? AND is_reel = ?", req.UserId, true).
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve reels")
	}

	// Filter by privacy settings
	posts = s.filterPostsByPrivacy(ctx, posts, req.RequesterId)

	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, &pb.Post{
			Id:               strconv.FormatUint(uint64(post.ID), 10),
			AuthorId:         post.AuthorID, // FIXED: Include author_id
			Caption:          post.Caption,
			AuthorUsername:   post.AuthorUsername,
			AuthorProfileUrl: post.AuthorProfileURL,
			AuthorIsVerified: post.AuthorIsVerified,
			MediaUrls:        post.MediaURLs,
			CreatedAt:        post.CreatedAt.Format(time.RFC3339),
			IsReel:           post.IsReel,
		})
	}
	return &pb.GetHomeFeedResponse{Posts: grpcPosts}, nil
}

// --- Implement GetUserContentCount ---
func (s *server) GetUserContentCount(ctx context.Context, req *pb.GetUserContentCountRequest) (*pb.GetUserContentCountResponse, error) {
	var postCount int64
	var reelCount int64

	// 1. Get count of regular posts
	s.db.Model(&Post{}).
		Where("author_id = ? AND is_reel = ?", req.UserId, false).
		Count(&postCount)

	// 2. Get count of reels
	s.db.Model(&Post{}).
		Where("author_id = ? AND is_reel = ?", req.UserId, true).
		Count(&reelCount)

	return &pb.GetUserContentCountResponse{
		PostCount: postCount,
		ReelCount: reelCount,
	}, nil
}

// --- Helper function to convert GORM Collection to gRPC Collection ---
func (s *server) gormToGrpcCollection(collection *Collection) *pb.Collection {
	// TODO: Get 4 cover image URLs
	return &pb.Collection{
		Id:        strconv.FormatUint(uint64(collection.ID), 10),
		UserId:    strconv.FormatInt(collection.UserID, 10),
		Name:      collection.Name,
		IsDefault: collection.IsDefault,
	}
}

// --- GPRC: CreateCollection ---
func (s *server) CreateCollection(ctx context.Context, req *pb.CreateCollectionRequest) (*pb.Collection, error) {
	newCollection := Collection{
		UserID: req.UserId,
		Name:   req.Name,
	}
	if result := s.db.Create(&newCollection); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to create collection")
	}
	return s.gormToGrpcCollection(&newCollection), nil
}

// --- GPRC: GetUserCollections ---
func (s *server) GetUserCollections(ctx context.Context, req *pb.GetUserCollectionsRequest) (*pb.GetUserCollectionsResponse, error) {
	var collections []Collection
	if err := s.db.Where("user_id = ?", req.UserId).Order("created_at DESC").Find(&collections).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve collections")
	}

	var grpcCollections []*pb.Collection
	for _, c := range collections {
		grpcCollections = append(grpcCollections, s.gormToGrpcCollection(&c))
	}

	return &pb.GetUserCollectionsResponse{Collections: grpcCollections}, nil
}

// --- GPRC: GetPostsInCollection ---
func (s *server) GetPostsInCollection(ctx context.Context, req *pb.GetPostsInCollectionRequest) (*pb.GetHomeFeedResponse, error) {
	// 1. Verify this user owns this collection
	var collection Collection
	if err := s.db.First(&collection, req.CollectionId).Error; err != nil {
		return nil, status.Error(codes.NotFound, "Collection not found")
	}
	if collection.UserID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "You do not own this collection")
	}

	// 2. Get all Post IDs from the join table
	var postIDs []uint
	result := s.db.Model(&SavedPost{}).Where("collection_id = ?", req.CollectionId).Order("created_at DESC").Pluck("post_id", &postIDs)
	if result.Error != nil {
		log.Printf("Error querying SavedPost: %v", result.Error)
	}
	log.Printf("GetPostsInCollection: collection_id=%d, found %d post IDs: %v (rows affected: %d)", req.CollectionId, len(postIDs), postIDs, result.RowsAffected)

	// Debug: Let's also count total records in SavedPost
	var totalCount int64
	s.db.Model(&SavedPost{}).Where("collection_id = ?", req.CollectionId).Count(&totalCount)
	log.Printf("Total SavedPost records for collection %d: %d", req.CollectionId, totalCount)

	if len(postIDs) == 0 {
		return &pb.GetHomeFeedResponse{Posts: []*pb.Post{}}, nil
	}

	// 3. Get all posts matching those IDs
	var posts []Post
	if err := s.db.Where("id IN ?", postIDs).
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve posts")
	}

	// 4. Convert and return
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, s.gormToGrpcPost(&post)) // Assumes you have a gormToGrpcPost helper
	}
	return &pb.GetHomeFeedResponse{Posts: grpcPosts}, nil
}

// --- GPRC: GetCollectionsForPost ---
func (s *server) GetCollectionsForPost(ctx context.Context, req *pb.GetCollectionsForPostRequest) (*pb.GetCollectionsForPostResponse, error) {
	// Get all collection IDs that contain this post for this user
	var savedPosts []SavedPost
	if err := s.db.
		Joins("JOIN collections ON saved_posts.collection_id = collections.id").
		Where("saved_posts.post_id = ? AND collections.user_id = ?", req.PostId, req.UserId).
		Find(&savedPosts).Error; err != nil {
		log.Printf("Error getting collections for post %d: %v", req.PostId, err)
		return nil, status.Error(codes.Internal, "Failed to get collections for post")
	}

	// Convert collection IDs to strings
	collectionIds := make([]string, len(savedPosts))
	for i, sp := range savedPosts {
		collectionIds[i] = strconv.FormatUint(uint64(sp.CollectionID), 10)
	}

	log.Printf("GetCollectionsForPost: post_id=%d, user_id=%d, found %d collections: %v", req.PostId, req.UserId, len(collectionIds), collectionIds)
	return &pb.GetCollectionsForPostResponse{CollectionIds: collectionIds}, nil
}

// --- Helper: Get or create default collection for user ---
func (s *server) getOrCreateDefaultCollection(userID int64) (*Collection, error) {
	// Try to find existing default collection for this user
	var collection Collection
	err := s.db.Where("user_id = ? AND is_default = ?", userID, true).First(&collection).Error

	if err == gorm.ErrRecordNotFound {
		// No default collection exists, create "All Posts" collection
		collection = Collection{
			UserID:    userID,
			Name:      "All Posts",
			IsDefault: true,
		}
		if createErr := s.db.Create(&collection).Error; createErr != nil {
			log.Printf("Failed to create default collection for user %d: %v", userID, createErr)
			return nil, createErr
		}
		log.Printf("Created default collection (ID: %d) for user %d", collection.ID, userID)
		return &collection, nil
	} else if err != nil {
		log.Printf("Database error when fetching collection: %v", err)
		return nil, err
	}

	// Found existing collection
	return &collection, nil
}

// --- GPRC: SavePostToCollection ---
func (s *server) SavePostToCollection(ctx context.Context, req *pb.SavePostToCollectionRequest) (*pb.SavePostToCollectionResponse, error) {
	// Special handling: If collection ID is 1 (common default), get or create user's first collection
	var collection *Collection
	var err error

	if req.CollectionId == 1 {
		// Use the user's first/default collection
		collection, err = s.getOrCreateDefaultCollection(req.UserId)
		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to get or create default collection")
		}
		log.Printf("Using default collection ID %d for user %d", collection.ID, req.UserId)
	} else {
		// Use the specified collection
		var col Collection
		if err := s.db.First(&col, req.CollectionId).Error; err != nil {
			log.Printf("Collection %d not found: %v", req.CollectionId, err)
			return nil, status.Error(codes.NotFound, "Collection not found")
		}
		collection = &col

		log.Printf("SavePostToCollection: collection.UserID=%d, req.UserId=%d", collection.UserID, req.UserId)
		if collection.UserID != req.UserId {
			return nil, status.Error(codes.PermissionDenied, "You do not own this collection")
		}
	}

	// 2. Save the post to the actual collection (use collection.ID, not req.CollectionId)
	savedPost := SavedPost{
		CollectionID: uint(collection.ID),
		PostID:       uint(req.PostId),
	}
	log.Printf("Attempting to save post %d to collection %d", req.PostId, collection.ID)
	result := s.db.Create(&savedPost)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "unique constraint") {
			log.Printf("Post %d already exists in collection %d", req.PostId, collection.ID)
			return nil, status.Error(codes.AlreadyExists, "Post already saved to this collection")
		}
		log.Printf("Failed to save post %d to collection %d: %v", req.PostId, collection.ID, result.Error)
		return nil, status.Error(codes.Internal, "Failed to save post")
	}
	log.Printf("Successfully saved post %d to collection %d (Rows affected: %d)", req.PostId, collection.ID, result.RowsAffected)

	// Verify the save by immediately querying
	var count int64
	s.db.Model(&SavedPost{}).Where("collection_id = ? AND post_id = ?", collection.ID, req.PostId).Count(&count)
	log.Printf("Verification: Found %d records for collection %d, post %d", count, collection.ID, req.PostId)

	return &pb.SavePostToCollectionResponse{Message: "Post saved successfully"}, nil
}

// --- GPRC: UnsavePostFromCollection ---
func (s *server) UnsavePostFromCollection(ctx context.Context, req *pb.UnsavePostFromCollectionRequest) (*pb.UnsavePostFromCollectionResponse, error) {
	log.Printf("UnsavePostFromCollection: collection_id=%d, post_id=%d, user_id=%d", req.CollectionId, req.PostId, req.UserId)

	// 1. Verify this user owns this collection
	var collection Collection
	if err := s.db.First(&collection, req.CollectionId).Error; err != nil {
		return nil, status.Error(codes.NotFound, "Collection not found")
	}
	if collection.UserID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "You do not own this collection")
	}

	// 2. Unsave the post - use explicit WHERE to ensure we only delete from the specified collection
	result := s.db.Where("collection_id = ? AND post_id = ?", req.CollectionId, req.PostId).Delete(&SavedPost{})
	log.Printf("UnsavePostFromCollection: Deleted %d rows from collection %d, post %d", result.RowsAffected, req.CollectionId, req.PostId)

	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "Post was not saved in this collection")
	}

	return &pb.UnsavePostFromCollectionResponse{Message: "Post unsaved successfully"}, nil
}

// --- GPRC: DeleteCollection ---
func (s *server) DeleteCollection(ctx context.Context, req *pb.DeleteCollectionRequest) (*pb.DeleteCollectionResponse, error) {
	var collection Collection
	if err := s.db.First(&collection, req.CollectionId).Error; err != nil {
		return nil, status.Error(codes.NotFound, "Collection not found")
	}
	if collection.UserID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "You do not own this collection")
	}
	if collection.IsDefault {
		return nil, status.Error(codes.PermissionDenied, "Cannot delete default collection")
	}

	// Delete from collections table (GORM will handle cascade deletes if set up)
	// For simplicity, we'll manually delete from the join table first
	if err := s.db.Where("collection_id = ?", req.CollectionId).Delete(&SavedPost{}).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to clear collection items")
	}
	if err := s.db.Delete(&collection).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to delete collection")
	}

	return &pb.DeleteCollectionResponse{Message: "Collection deleted successfully"}, nil
}

// --- GPRC: RenameCollection ---
func (s *server) RenameCollection(ctx context.Context, req *pb.RenameCollectionRequest) (*pb.Collection, error) {
	var collection Collection
	if err := s.db.First(&collection, req.CollectionId).Error; err != nil {
		return nil, status.Error(codes.NotFound, "Collection not found")
	}
	if collection.UserID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "You do not own this collection")
	}
	if collection.IsDefault {
		return nil, status.Error(codes.PermissionDenied, "Cannot rename default collection")
	}

	collection.Name = req.NewName
	if err := s.db.Save(&collection).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to rename collection")
	}

	return s.gormToGrpcCollection(&collection), nil
}

// gormToGrpcPost converts our GORM Post model to the gRPC Post message
func (s *server) gormToGrpcPost(post *Post) *pb.Post {
	// Get real LikeCount and CommentCount
	var likeCount int64
	s.db.Model(&PostLike{}).Where("post_id = ?", post.ID).Count(&likeCount)
	var commentCount int64
	s.db.Model(&Comment{}).Where("post_id = ?", post.ID).Count(&commentCount)

	return &pb.Post{
		Id:               strconv.FormatUint(uint64(post.ID), 10),
		AuthorId:         post.AuthorID,
		Caption:          post.Caption,
		MediaUrls:        post.MediaURLs,
		CreatedAt:        post.CreatedAt.Format(time.RFC3339),
		IsReel:           post.IsReel,
		CommentsDisabled: post.CommentsDisabled,
		ThumbnailUrl:     post.ThumbnailURL,

		// Use the saved denormalized data
		AuthorUsername:   post.AuthorUsername,
		AuthorProfileUrl: post.AuthorProfileURL,
		AuthorIsVerified: post.AuthorIsVerified,

		// Real-time counts from database
		LikeCount:    likeCount,
		CommentCount: commentCount,
	}
}

// --- GPRC: GetPost ---
func (s *server) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.Post, error) {
	// Note: We can't use cache here because is_liked and is_saved are viewer-specific

	// Query database
	var post Post
	if err := s.db.First(&post, req.PostId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Post not found")
	} else if err != nil {
		return nil, status.Error(codes.Internal, "Database error")
	}

	// Use enrichPostProto to get viewer-specific data (is_liked, is_saved)
	grpcPost := s.enrichPostProto(ctx, &post, req.ViewerId)

	return grpcPost, nil
}

// --- GPRC: GetPosts (Batched) ---
func (s *server) GetPosts(ctx context.Context, req *pb.GetPostsRequest) (*pb.GetPostsResponse, error) {
	if len(req.PostIds) == 0 {
		return &pb.GetPostsResponse{Posts: []*pb.Post{}}, nil
	}

	var posts []Post
	if err := s.db.Where("id IN ?", req.PostIds).Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve posts")
	}

	// Convert to gRPC posts
	grpcPosts := make([]*pb.Post, 0, len(posts))
	for i := range posts {
		grpcPosts = append(grpcPosts, s.gormToGrpcPost(&posts[i]))
	}

	return &pb.GetPostsResponse{Posts: grpcPosts}, nil
}

// --- GPRC: DeletePost (Admin) ---
func (s *server) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostResponse, error) {
	log.Printf("DeletePost request from user %d for post %d", req.AdminUserId, req.PostId)

	var post Post
	if err := s.db.First(&post, req.PostId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Post not found")
	}

	// Only allow deletion if user is the post owner
	// (Admin check can be added later if needed via user-service gRPC call)
	if post.AuthorID != req.AdminUserId {
		return nil, status.Error(codes.PermissionDenied, "You can only delete your own posts")
	}

	// We can't delete a composite primary key (like PostLike) with just GORM,
	// so we'll use a transaction and delete related data manually.

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Delete associated comments
		if err := tx.Where("post_id = ?", req.PostId).Delete(&Comment{}).Error; err != nil {
			return err
		}

		// 2. Delete associated likes
		if err := tx.Where("post_id = ?", req.PostId).Delete(&PostLike{}).Error; err != nil {
			return err
		}

		// 3. Delete from "Saved" collections (join table)
		if err := tx.Where("post_id = ?", req.PostId).Delete(&SavedPost{}).Error; err != nil {
			return err
		}

		// 4. Finally, delete the post itself
		if result := tx.Delete(&Post{}, req.PostId); result.Error != nil {
			return result.Error
		} else if result.RowsAffected == 0 {
			return status.Error(codes.NotFound, "Post not found")
		}

		return nil // Commit
	})

	if err != nil {
		log.Printf("Failed to delete post %d: %v", req.PostId, err)
		// Check if it's the "not found" error from our transaction
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, st.Err()
		}
		return nil, status.Error(codes.Internal, "Failed to delete post and associated data")
	}

	// --- ADDED: Delete media from MinIO ---
	if len(post.MediaURLs) > 0 {
		bucketName := "media" // Standard bucket name used across all services
		for _, url := range post.MediaURLs {
			// MediaURLs now store object paths directly (e.g., "user-123/posts/video.mp4")
			// not full URLs, so we can use them directly as object names
			objectName := url
			// Handle legacy URLs if they still exist (contains http://)
			if strings.Contains(url, "http://") || strings.Contains(url, "https://") {
				objectName = url[strings.LastIndex(url, "/")+1:]
			}
			if objectName != "" {
				err := s.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
				if err != nil {
					log.Printf("Failed to delete object %s from MinIO: %v", objectName, err)
					// Do not fail the whole request, just log it
				} else {
					log.Printf("Successfully deleted object %s from MinIO bucket '%s'", objectName, bucketName)
				}
			}
		}
	}

	// --- Clear Redis cache for this post ---
	cacheKey := fmt.Sprintf("post:%d", req.PostId)
	if err := s.rdb.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to delete cache key %s: %v", cacheKey, err)
	}

	// Also clear feed caches that might contain this post
	feedPattern := "feed:*"
	iter := s.rdb.Scan(ctx, 0, feedPattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Error scanning feed cache keys: %v", err)
	}

	log.Printf("Successfully deleted post %d and its associations", req.PostId)

	return &pb.DeletePostResponse{Message: "Post deleted successfully"}, nil
}

// --- Implement SharePost ---
func (s *server) SharePost(ctx context.Context, req *pb.SharePostRequest) (*pb.SharePostResponse, error) {
	// Verify post exists
	var post Post
	if err := s.db.First(&post, req.PostId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "Post not found")
		}
		return nil, status.Error(codes.Internal, "Failed to fetch post")
	}

	// Create SharedPost record
	sharedPost := SharedPost{
		UserID:         req.UserId,
		OriginalPostID: req.PostId,
		Caption:        req.Caption,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the share
		if err := tx.Create(&sharedPost).Error; err != nil {
			return err
		}
		// 2. Increment the post's share_count
		if err := tx.Model(&Post{}).Where("id = ?", req.PostId).Update("share_count", gorm.Expr("share_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to share post: %v", err)
		return nil, status.Error(codes.Internal, "Failed to share post")
	}

	// Send notification to post author (if not sharing own post)
	if post.AuthorID != req.UserId {
		msgBody, _ := json.Marshal(map[string]interface{}{
			"type":      "post.shared",
			"actor_id":  req.UserId,
			"user_id":   post.AuthorID,
			"entity_id": req.PostId,
		})
		s.publishToQueue(ctx, "notification_queue", msgBody)
	}

	return &pb.SharePostResponse{
		Message:      "Post shared successfully",
		SharedPostId: int64(sharedPost.ID),
	}, nil
}

// --- Implement UnsharePost ---
func (s *server) UnsharePost(ctx context.Context, req *pb.UnsharePostRequest) (*pb.UnsharePostResponse, error) {
	var sharedPost SharedPost

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Find and delete the SharedPost
		result := tx.Where("user_id = ? AND original_post_id = ?", req.UserId, req.PostId).Delete(&sharedPost)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return status.Error(codes.NotFound, "Shared post not found")
		}

		// 2. Decrement the post's share_count
		if err := tx.Model(&Post{}).Where("id = ?", req.PostId).Update("share_count", gorm.Expr("share_count - 1")).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, st.Err()
		}
		log.Printf("Failed to unshare post: %v", err)
		return nil, status.Error(codes.Internal, "Failed to unshare post")
	}

	return &pb.UnsharePostResponse{Message: "Post unshared successfully"}, nil
}

// --- Implement GetSharedPosts ---
func (s *server) GetSharedPosts(ctx context.Context, req *pb.GetSharedPostsRequest) (*pb.GetSharedPostsResponse, error) {
	var sharedPosts []SharedPost

	// Fetch shared posts with pagination
	offset := int(req.PageOffset * req.PageSize)
	if err := s.db.Where("user_id = ?", req.UserId).
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(offset).
		Find(&sharedPosts).Error; err != nil {
		log.Printf("Failed to fetch shared posts: %v", err)
		return nil, status.Error(codes.Internal, "Failed to fetch shared posts")
	}

	// Get user info for the sharer
	userResp, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: req.UserId})
	if err != nil {
		log.Printf("Failed to fetch user info: %v", err)
		return nil, status.Error(codes.Internal, "Failed to fetch user info")
	}

	// Build response
	var items []*pb.SharedPostItem
	for _, sp := range sharedPosts {
		// Fetch the original post
		var post Post
		if err := s.db.First(&post, sp.OriginalPostID).Error; err != nil {
			log.Printf("Warning: Shared post %d references non-existent post %d", sp.ID, sp.OriginalPostID)
			continue
		}

		// Convert to proto Post
		protoPost := &pb.Post{
			Id:               strconv.FormatUint(uint64(post.ID), 10),
			AuthorId:         post.AuthorID,
			Caption:          post.Caption,
			AuthorUsername:   post.AuthorUsername,
			AuthorProfileUrl: post.AuthorProfileURL,
			AuthorIsVerified: post.AuthorIsVerified,
			MediaUrls:        post.MediaURLs,
			CreatedAt:        post.CreatedAt.Format(time.RFC3339),
			IsReel:           post.IsReel,
			CommentsDisabled: post.CommentsDisabled,
			ThumbnailUrl:     post.ThumbnailURL,
			LikeCount:        post.LikeCount,
			CommentCount:     post.CommentCount,
			ShareCount:       post.ShareCount,
		}

		items = append(items, &pb.SharedPostItem{
			Id:               strconv.FormatUint(uint64(sp.ID), 10),
			OriginalPost:     protoPost,
			SharedByUsername: userResp.Username,
			SharedCaption:    sp.Caption,
			SharedAt:         sp.CreatedAt.Format(time.RFC3339),
		})
	}

	return &pb.GetSharedPostsResponse{SharedPosts: items}, nil
}

func (s *server) enrichPostProto(ctx context.Context, post *Post, viewerID int64) *pb.Post {
	var likeCount int64
	var commentCount int64
	var isLiked bool
	var isSaved bool

	// 1. Count Likes
	s.db.Model(&PostLike{}).Where("post_id = ?", post.ID).Count(&likeCount)

	// 2. Count Comments
	s.db.Model(&Comment{}).Where("post_id = ?", post.ID).Count(&commentCount)

	// 3. Check if Viewer Liked
	if viewerID != 0 {
		var count int64
		s.db.Model(&PostLike{}).Where("post_id = ? AND user_id = ?", post.ID, viewerID).Count(&count)
		isLiked = count > 0
	}

	// 4. Check if Viewer Saved
	if viewerID != 0 {
		// We need to check if the post ID exists in ANY of the user's collections
		// Join SavedPost with Collection
		var count int64
		s.db.Table("saved_posts").
			Joins("JOIN collections ON saved_posts.collection_id = collections.id").
			Where("saved_posts.post_id = ? AND collections.user_id = ?", post.ID, viewerID).
			Count(&count)
		isSaved = count > 0
	}

	return &pb.Post{
		Id:               strconv.FormatUint(uint64(post.ID), 10),
		AuthorId:         post.AuthorID, // FIXED: Include author_id
		Caption:          post.Caption,
		AuthorUsername:   post.AuthorUsername,
		AuthorProfileUrl: post.AuthorProfileURL,
		AuthorIsVerified: post.AuthorIsVerified,
		MediaUrls:        post.MediaURLs,
		CreatedAt:        post.CreatedAt.Format(time.RFC3339),
		IsReel:           post.IsReel,
		Location:         post.Location,
		LikeCount:        likeCount,
		CommentCount:     commentCount,
		IsLiked:          isLiked,
		IsSaved:          isSaved,
	}
}
