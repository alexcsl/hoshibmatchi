package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"

	pb "github.com/hoshibmatchi/media-service/proto"
)

type server struct {
	pb.UnimplementedMediaServiceServer
	minioClient *minio.Client
}

const (
	minioEndpoint        = "minio:9000" // From docker-compose
	minioAccessKeyID     = "minioadmin" // From docker-compose
	minioSecretAccessKey = "minioadmin" // From docker-compose
	minioUseSSL          = false
	minioBucketName      = "media" // We'll have to create this
)

func main() {
	// --- Step 1: Connect to MinIO (with retries) ---
	var minioClient *minio.Client
	var err error

	for i := 0; i < 10; i++ {
		minioClient, err = minio.New(minioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(minioAccessKeyID, minioSecretAccessKey, ""),
			Secure: minioUseSSL,
		})
		if err == nil {
			log.Println("Successfully connected to MinIO")
			break
		}
		log.Printf("Failed to connect to MinIO: %v. Retrying...", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to MinIO after retries: %v", err)
	}

	// --- Step 2: Ensure our bucket exists ---
	err = minioClient.MakeBucket(context.Background(), minioBucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), minioBucketName)
		if errBucketExists == nil && exists {
			log.Printf("Bucket '%s' already exists.", minioBucketName)
		} else {
			log.Fatalf("Failed to create or find bucket: %v", err)
		}
	} else {
		log.Printf("Successfully created bucket '%s'", minioBucketName)
	}

	// --- Step 3: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9005") // Port 9005
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMediaServiceServer(s, &server{minioClient: minioClient})

	log.Println("Media service listening on port 9005...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- Implement GetUploadURL ---
func (s *server) GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error) {
	// Create a unique object name, e.g., "user-1/posts/my-photo.jpg"
	objectName := fmt.Sprintf("user-%d/posts/%s", req.UserId, req.Filename)
	
	// Set upload policy
	policy := minio.NewPostPolicy()
	policy.SetBucket(minioBucketName)
	policy.SetKey(objectName)
	policy.SetExpires(time.Now().Add(10 * time.Minute)) // URL is valid for 10 minutes
	policy.SetContentType(req.ContentType)

	// Get the pre-signed URL
	// We use PresignedPostPolicy to allow browser-based uploads
	uploadURL, formData, err := s.minioClient.PresignedPostPolicy(context.Background(), policy)
	if err != nil {
		log.Printf("Failed to generate pre-signed URL: %v", err)
		return nil, status.Error(codes.Internal, "Failed to create upload URL")
	}

	// The `upload_url` is what the frontend will POST to.
	// But the `final_media_url` is the simple S3-compatible URL
	// we want to save in our Postgres database.
	finalURL := fmt.Sprintf("%s/%s/%s", minioEndpoint, minioBucketName, objectName)

	_ = formData // We'd send this to the frontend too

	return &pb.GetUploadURLResponse{
		UploadUrl:     uploadURL.String(),
		FinalMediaUrl: finalURL,
	}, nil
}