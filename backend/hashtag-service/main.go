package main

import (
	"context"
	"log"
	"net"
	"strconv"

	// "strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	pb "github.com/hoshibmatchi/hashtag-service/proto"
	postPb "github.com/hoshibmatchi/post-service/proto"
)

// --- GORM Models ---

// Hashtag stores the canonical hashtag name and its post count
type Hashtag struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"type:varchar(100);uniqueIndex;not null"`
	PostCount int64  `gorm:"default:0"`
}

// PostHashtag is the join table between posts and hashtags
type PostHashtag struct {
	PostID    int64 `gorm:"primaryKey"`
	HashtagID uint  `gorm:"primaryKey"`
	CreatedAt time.Time
}

type server struct {
	pb.UnimplementedHashtagServiceServer
	db         *gorm.DB
	postClient postPb.PostServiceClient // Client to get post details
}

func main() {
	// --- Step 1: Connect to hashtag-db ---
	dsn := "host=hashtag-db user=admin password=password dbname=hashtag_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to hashtag-db: %v", err)
	}
	db.AutoMigrate(&Hashtag{}, &PostHashtag{})
	log.Println("Hashtag-service successfully connected to hashtag-db")

	// --- Step 2: Connect to Post Service (gRPC Client) ---
	postConn, err := grpc.Dial("post-service:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to post-service: %v", err)
	}
	defer postConn.Close()
	postClient := postPb.NewPostServiceClient(postConn)
	log.Println("Hashtag-service successfully connected to post-service")

	// --- Step 3: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9007") // Port 9007
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterHashtagServiceServer(s, &server{
		db:         db,
		postClient: postClient,
	})

	log.Println("Hashtag service listening on port 9007...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- gRPC Implementations ---

// AddHashtagsToPost is an INTERNAL RPC called by the worker-service
func (s *server) AddHashtagsToPost(ctx context.Context, req *pb.AddHashtagsToPostRequest) (*pb.AddHashtagsToPostResponse, error) {
	log.Printf("AddHashtagsToPost request for post %d with tags: %v", req.PostId, req.HashtagNames)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, name := range req.HashtagNames {
			// 1. Find or Create the hashtag
			var hashtag Hashtag
			// Use 'clause.OnConflict' to handle the unique constraint gracefully
			// If the name exists, do nothing (it's found). If not, create it.
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Hashtag{Name: name}).Error; err != nil {
				return err
			}
			// Now, find the hashtag (either existing or just created)
			if err := tx.Where("name = ?", name).First(&hashtag).Error; err != nil {
				return err
			}

			// 2. Create the join table entry
			joinEntry := PostHashtag{
				PostID:    req.PostId,
				HashtagID: hashtag.ID,
			}
			// If this entry already exists, do nothing
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&joinEntry).Error; err != nil {
				return err
			}

			// 3. Increment the PostCount on the hashtag
			// We use a SQL expression to avoid race conditions
			if err := tx.Model(&hashtag).Update("post_count", gorm.Expr("post_count + 1")).Error; err != nil {
				return err
			}
		}
		return nil // Commit
	})

	if err != nil {
		log.Printf("Failed to add hashtags to post %d: %v", req.PostId, err)
		return nil, status.Error(codes.Internal, "Failed to process hashtags")
	}

	return &pb.AddHashtagsToPostResponse{Message: "Hashtags added successfully"}, nil
}

// GetTrendingHashtags is a PUBLIC RPC
func (s *server) GetTrendingHashtags(ctx context.Context, req *pb.GetTrendingHashtagsRequest) (*pb.GetTrendingHashtagsResponse, error) {
	log.Printf("GetTrendingHashtags request, limit %d", req.Limit)

	var hashtags []Hashtag
	if err := s.db.Order("post_count DESC").Limit(int(req.Limit)).Find(&hashtags).Error; err != nil {
		log.Printf("Failed to get trending hashtags: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve trending hashtags")
	}

	var grpcHashtags []*pb.Hashtag
	for _, tag := range hashtags {
		grpcHashtags = append(grpcHashtags, &pb.Hashtag{
			Id:        strconv.FormatUint(uint64(tag.ID), 10),
			Name:      tag.Name,
			PostCount: tag.PostCount,
		})
	}

	return &pb.GetTrendingHashtagsResponse{Hashtags: grpcHashtags}, nil
}

// SearchByHashtag is a PUBLIC RPC
func (s *server) SearchByHashtag(ctx context.Context, req *pb.SearchByHashtagRequest) (*pb.SearchByHashtagResponse, error) {
	log.Printf("SearchByHashtag request for tag: %s", req.HashtagName)

	// 1. Find the hashtag
	var hashtag Hashtag
	if err := s.db.Where("name = ?", req.HashtagName).First(&hashtag).Error; err == gorm.ErrRecordNotFound {
		// No posts for this tag
		return &pb.SearchByHashtagResponse{Posts: []*postPb.Post{}, TotalPostCount: 0}, nil
	}

	// 2. Find all PostIDs associated with this hashtag (with pagination)
	var postIDs []int64
	if err := s.db.Model(&PostHashtag{}).
		Where("hashtag_id = ?", hashtag.ID).
		Order("created_at DESC").
		Limit(int(req.PageSize)).
		Offset(int(req.PageOffset)).
		Pluck("post_id", &postIDs).Error; err != nil {
		return nil, status.Error(codes.Internal, "Failed to retrieve post list for hashtag")
	}

	if len(postIDs) == 0 {
		return &pb.SearchByHashtagResponse{Posts: []*postPb.Post{}, TotalPostCount: hashtag.PostCount}, nil
	}

	// 3. Get the actual Post data from post-service using batched call
	postsResp, err := s.postClient.GetPosts(ctx, &postPb.GetPostsRequest{PostIds: postIDs})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get posts from post-service: %v", err)
	}

	return &pb.SearchByHashtagResponse{
		Posts:          postsResp.Posts,
		TotalPostCount: hashtag.PostCount,
	}, nil
}
