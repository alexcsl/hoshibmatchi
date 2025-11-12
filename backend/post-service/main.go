package main

import (
	"context"
	"log"
	"net"
	"strconv"
	"time"
	"strings"

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
	IsReel           bool `gorm:"default:false"`
}

// PostLike defines the GORM model for a like on a post
type PostLike struct {
	// Composite primary key (user_id, post_id)
	UserID int64 `gorm:"primaryKey"`
	PostID int64 `gorm:"primaryKey"`
	CreatedAt time.Time
}

// Comment defines the GORM model for a comment
type Comment struct {
	gorm.Model
	UserID   int64
	PostID   int64
	Content  string // This can be text or a GIF URL
	
	// For nested replies
	ParentCommentID uint // GORM's Model.ID is uint

	// Denormalized data from user-service
	AuthorUsername   string
	AuthorProfileURL string
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
	db.AutoMigrate(&PostLike{})
	db.AutoMigrate(&Comment{})

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
		IsReel:           req.IsReel,
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
			IsReel:             newPost.IsReel,
		},
	}, nil
}

// --- Implement LikePost ---
func (s *server) LikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.LikePostResponse, error) {
	like := PostLike{
		UserID: req.UserId,
		PostID: req.PostId,
	}
	
	// GORM's Create will fail if the composite primary key already exists
	if result := s.db.Create(&like); result.Error != nil {
		if strings.Contains(result.Error.Error(), "unique constraint") {
			return nil, status.Error(codes.AlreadyExists, "Post already liked")
		}
		return nil, status.Error(codes.Internal, "Failed to like post")
	}
	
	// TODO: Send a "post.liked" event to RabbitMQ for notifications
	return &pb.LikePostResponse{Message: "Post liked"}, nil
}

// --- Implement UnlikePost ---
func (s *server) UnlikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.UnlikePostResponse, error) {
	like := PostLike{
		UserID: req.UserId,
		PostID: req.PostId,
	}
	
	if result := s.db.Delete(&like); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to unlike post")
	} else if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "Post was not liked")
	}
	
	return &pb.UnlikePostResponse{Message: "Post unliked"}, nil
}

// --- Implement CommentOnPost ---
func (s *server) CommentOnPost(ctx context.Context, req *pb.CommentOnPostRequest) (*pb.CommentResponse, error) {
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
	}

	if result := s.db.Create(&newComment); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to save comment")
	}
	
	// TODO: Send "post.commented" event to RabbitMQ

	// --- Step 3: Return the created comment ---
	return &pb.CommentResponse{
		Id:               strconv.FormatUint(uint64(newComment.ID), 10),
		Content:          newComment.Content,
		AuthorUsername:   newComment.AuthorUsername,
		AuthorProfileUrl: newComment.AuthorProfileURL,
		CreatedAt:        newComment.CreatedAt.Format(time.RFC3339),
		PostId:           newComment.PostID,
		ParentCommentId:  int64(newComment.ParentCommentID),
	}, nil
}

// --- Implement DeleteComment ---
func (s *server) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentResponse, error) {
	var comment Comment
	
	// Find the comment first
	if err := s.db.First(&comment, req.CommentId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Comment not found")
	}
	
	// PDF Requirement: "Users can delete the replies they have posted"
	// This means we must check ownership
	if comment.UserID != req.UserId {
		// TODO: Also allow post author to delete comments
		return nil, status.Error(codes.PermissionDenied, "You do not have permission to delete this comment")
	}

	if result := s.db.Delete(&comment); result.Error != nil {
		return nil, status.Error(codes.Internal, "Failed to delete comment")
	}
	
	return &pb.DeleteCommentResponse{Message: "Comment deleted"}, nil
}

// --- Implement GetHomeFeed ---
func (s *server) GetHomeFeed(ctx context.Context, req *pb.GetHomeFeedRequest) (*pb.GetHomeFeedResponse, error) {
	log.Println("GetHomeFeed request received")

	// --- Step 1: Call User Service to get the list of followed users ---
	followingRes, err := s.userClient.GetFollowingList(ctx, &userPb.GetFollowingListRequest{UserId: req.UserId})
	if err != nil {
		log.Printf("Failed to get following list from user-service: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve user feed")
	}

	followingIDs := followingRes.FollowingUserIds

	// --- Step 2: Query our DB for posts *only* from those users ---
	var posts []Post
	if err := s.db.Where("author_id IN ?", followingIDs).
		Order("created_at DESC"). // PDF: "starting from the most recent posts"
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Find(&posts).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve posts")
	}

	// --- Step 3: Convert GORM models to gRPC responses ---
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, &pb.Post{
			Id:                 strconv.FormatUint(uint64(post.ID), 10),
			Caption:            post.Caption,
			AuthorUsername:     post.AuthorUsername,
			AuthorProfileUrl:   post.AuthorProfileURL,
			AuthorIsVerified:   post.AuthorIsVerified,
			MediaUrls:          post.MediaURLs,
			CreatedAt:          post.CreatedAt.Format(time.RFC3339),
		})
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

	// Convert GORM models to gRPC responses
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, &pb.Post{
			Id:                 strconv.FormatUint(uint64(post.ID), 10),
			Caption:            post.Caption,
			AuthorUsername:     post.AuthorUsername,
			AuthorProfileUrl:   post.AuthorProfileURL,
			AuthorIsVerified:   post.AuthorIsVerified,
			MediaUrls:          post.MediaURLs,
			CreatedAt:          post.CreatedAt.Format(time.RFC3339),
			IsReel:             post.IsReel,
		})
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

	// Convert GORM models to gRPC responses
	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, &pb.Post{
			Id:                 strconv.FormatUint(uint64(post.ID), 10),
			Caption:            post.Caption,
			AuthorUsername:     post.AuthorUsername,
			AuthorProfileUrl:   post.AuthorProfileURL,
			AuthorIsVerified:   post.AuthorIsVerified,
			MediaUrls:          post.MediaURLs,
			CreatedAt:          post.CreatedAt.Format(time.RFC3339),
			IsReel:             post.IsReel,
		})
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

	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, &pb.Post{
			Id:                 strconv.FormatUint(uint64(post.ID), 10),
			Caption:            post.Caption,
			AuthorUsername:     post.AuthorUsername,
			AuthorProfileUrl:   post.AuthorProfileURL,
			AuthorIsVerified:   post.AuthorIsVerified,
			MediaUrls:          post.MediaURLs,
			CreatedAt:          post.CreatedAt.Format(time.RFC3339),
			IsReel:             post.IsReel,
		})
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

	var grpcPosts []*pb.Post
	for _, post := range posts {
		grpcPosts = append(grpcPosts, &pb.Post{
			Id:                 strconv.FormatUint(uint64(post.ID), 10),
			Caption:            post.Caption,
			AuthorUsername:     post.AuthorUsername,
			AuthorProfileUrl:   post.AuthorProfileURL,
			AuthorIsVerified:   post.AuthorIsVerified,
			MediaUrls:          post.MediaURLs,
			CreatedAt:          post.CreatedAt.Format(time.RFC3339),
			IsReel:             post.IsReel,
		})
	}
	return &pb.GetHomeFeedResponse{Posts: grpcPosts}, nil
}