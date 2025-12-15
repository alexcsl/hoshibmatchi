package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/hoshibmatchi/media-service/proto"
)

type server struct {
	pb.UnimplementedMediaServiceServer
	internalClient *minio.Client // Connects to minio:9000 (Backend operations)
	signingClient  *minio.Client // Configured for localhost:9000 (Browser signatures)
}

const (
	minioBucketName = "media"
)

func main() {
	// Load environment variables
	minioInternalEndpoint := os.Getenv("MINIO_INTERNAL_ENDPOINT")
	if minioInternalEndpoint == "" {
		minioInternalEndpoint = "minio:9000"
	}
	minioExternalEndpoint := os.Getenv("MINIO_EXTERNAL_ENDPOINT")
	if minioExternalEndpoint == "" {
		minioExternalEndpoint = "localhost:9000"
	}
	minioAccessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	if minioAccessKeyID == "" {
		minioAccessKeyID = "minioadmin"
	}
	minioSecretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	if minioSecretAccessKey == "" {
		minioSecretAccessKey = "minioadmin"
	}
	minioRegion := os.Getenv("MINIO_REGION")
	if minioRegion == "" {
		minioRegion = "us-east-1"
	}
	minioUseSSL := os.Getenv("MINIO_USE_SSL") == "true"

	// 1. Internal Client (Backend Operations)
	var internalClient *minio.Client
	var err error
	for i := 0; i < 10; i++ {
		internalClient, err = minio.New(minioInternalEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(minioAccessKeyID, minioSecretAccessKey, ""),
			Secure: minioUseSSL,
		})
		if err == nil {
			log.Println("Connected to MinIO (Internal)")
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	// 2. Signing Client (Frontend URLs)
	signingClient, err := minio.New(minioExternalEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKeyID, minioSecretAccessKey, ""),
		Secure: minioUseSSL,
		Region: minioRegion,
	})
	if err != nil {
		log.Fatalf("Failed to create signing client: %v", err)
	}

	// 3. Ensure Bucket Exists (Private by default)
	ctx := context.Background()
	exists, err := internalClient.BucketExists(ctx, minioBucketName)
	if err != nil || !exists {
		internalClient.MakeBucket(ctx, minioBucketName, minio.MakeBucketOptions{Region: minioRegion})
		log.Printf("Created bucket '%s'", minioBucketName)
	}

	// --- PRODUCTION BUCKET POLICY: PRIVATE ---
	// Bucket is now PRIVATE by default
	// All media access uses pre-signed URLs for security
	// Remove any existing public policy
	if err := internalClient.SetBucketPolicy(ctx, minioBucketName, ""); err != nil {
		log.Printf("Warning: Failed to clear bucket policy: %v", err)
	} else {
		log.Println("Bucket policy set to PRIVATE (production mode)")
		log.Println("All media access now requires pre-signed URLs")
	}

	// 4. Start gRPC
	lis, err := net.Listen("tcp", ":9005")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Increase max message size to 50MB for video uploads
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024), // 50MB
		grpc.MaxSendMsgSize(50*1024*1024), // 50MB
	)

	pb.RegisterMediaServiceServer(s, &server{
		internalClient: internalClient,
		signingClient:  signingClient,
	})
	log.Println("Media service listening on port 9005...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- RPC: GetUploadURL (For Direct Frontend Uploads) ---
func (s *server) GetUploadURL(ctx context.Context, req *pb.GetUploadURLRequest) (*pb.GetUploadURLResponse, error) {
	objectName := fmt.Sprintf("user-%d/posts/%s", req.UserId, req.Filename)
	expiry := time.Minute * 10

	// Use SIGNING CLIENT to generate the URL the browser needs
	uploadURL, err := s.signingClient.PresignedPutObject(context.Background(), minioBucketName, objectName, expiry)
	if err != nil {
		log.Printf("Failed to generate pre-signed URL: %v", err)
		return nil, status.Error(codes.Internal, "Failed to create upload URL")
	}

	// Store object name (not direct URL) for pre-signed access
	// Frontend will need to call GetMediaURL to get actual viewing URL
	finalURL := objectName // Store the object path, not the full URL

	return &pb.GetUploadURLResponse{
		UploadUrl:     uploadURL.String(),
		FinalMediaUrl: finalURL, // Returns object path like "user-123/posts/abc.jpg"
	}, nil
}

// --- RPC: GetMediaURL (Generate Pre-signed GET URL for viewing media) ---
func (s *server) GetMediaURL(ctx context.Context, req *pb.GetMediaURLRequest) (*pb.GetMediaURLResponse, error) {
	if req.ObjectName == "" {
		return nil, status.Error(codes.InvalidArgument, "object_name is required")
	}

	// Default expiry is 1 hour
	expiry := time.Hour
	if req.ExpirySeconds > 0 {
		expiry = time.Duration(req.ExpirySeconds) * time.Second
	}

	// Generate pre-signed GET URL for viewing/downloading
	presignedURL, err := s.signingClient.PresignedGetObject(
		context.Background(),
		minioBucketName,
		req.ObjectName,
		expiry,
		nil, // No custom request parameters
	)
	if err != nil {
		log.Printf("Failed to generate pre-signed GET URL: %v", err)
		return nil, status.Error(codes.Internal, "Failed to create media URL")
	}

	return &pb.GetMediaURLResponse{
		MediaUrl: presignedURL.String(),
	}, nil
}

// --- RPC: GenerateThumbnail (For videos uploaded via GetUploadURL) ---
func (s *server) GenerateThumbnail(ctx context.Context, req *pb.GenerateThumbnailRequest) (*pb.GenerateThumbnailResponse, error) {
	log.Println("=== GENERATE THUMBNAIL RPC ===")
	log.Printf("Request: object_name=%s, user_id=%d, timestamp=%.2f", req.ObjectName, req.UserId, req.TimestampSeconds)

	if req.ObjectName == "" {
		log.Println("❌ Missing object_name")
		return nil, status.Error(codes.InvalidArgument, "object_name is required")
	}

	// Extract filename from object path
	parts := strings.Split(req.ObjectName, "/")
	if len(parts) == 0 {
		log.Println("❌ Invalid object_name format")
		return nil, status.Error(codes.InvalidArgument, "invalid object_name")
	}
	filename := parts[len(parts)-1]
	log.Printf("Extracted filename: %s", filename)

	// Check if it's a video file
	if !isVideoFile(filename) {
		log.Printf("⚠️ Not a video file, skipping thumbnail generation")
		return &pb.GenerateThumbnailResponse{ThumbnailUrl: ""}, nil
	}

	// Create temp directory
	uniqueID := uuid.New().String()
	tempDir := filepath.Join("/tmp", uniqueID)
	log.Printf("Creating temp directory: %s", tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("❌ Failed to create temp directory: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to create temp directory: %v", err)
	}
	defer func() {
		log.Printf("Cleaning up temp directory: %s", tempDir)
		os.RemoveAll(tempDir)
	}()

	// Download video from MinIO to temp
	videoPath := filepath.Join(tempDir, filename)
	log.Printf("Downloading video from MinIO: %s -> %s", req.ObjectName, videoPath)
	err := s.internalClient.FGetObject(ctx, minioBucketName, req.ObjectName, videoPath, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("❌ Failed to download video: %v", err)
		return nil, status.Error(codes.Internal, "Failed to download video for thumbnail generation")
	}
	log.Printf("✅ Video downloaded successfully")

	// Generate thumbnail
	thumbnailPath := filepath.Join(tempDir, "thumbnail.jpg")
	log.Printf("Generating thumbnail at %.2f seconds -> %s", req.TimestampSeconds, thumbnailPath)
	if err := generateThumbnail(videoPath, thumbnailPath, req.TimestampSeconds); err != nil {
		log.Printf("❌ Failed to generate thumbnail: %v", err)
		return nil, status.Error(codes.Internal, "Failed to generate thumbnail")
	}
	log.Printf("✅ Thumbnail generated successfully")

	// Upload thumbnail to MinIO
	// Extract just the filename without extension for unique ID
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	thumbnailName := fmt.Sprintf("user-%d/thumbnails/%s.jpg", req.UserId, nameWithoutExt)
	log.Printf("Uploading thumbnail to MinIO: %s", thumbnailName)

	if err := s.uploadFileToMinio(ctx, thumbnailPath, thumbnailName, "image/jpeg"); err != nil {
		log.Printf("❌ Failed to upload thumbnail: %v", err)
		return nil, status.Error(codes.Internal, "Failed to upload thumbnail")
	}

	log.Printf("✅ Thumbnail uploaded: %s", thumbnailName)
	log.Println("=== THUMBNAIL GENERATION COMPLETE ===")
	return &pb.GenerateThumbnailResponse{
		ThumbnailUrl: thumbnailName, // Return object path
	}, nil
}

// --- RPC: UploadMedia (For Server-Side Processing) ---
// This restores your original logic for thumbnails/optimization
func (s *server) UploadMedia(ctx context.Context, req *pb.UploadMediaRequest) (*pb.UploadMediaResponse, error) {
	uniqueID := uuid.New().String()
	ext := filepath.Ext(req.Filename)
	objectName := fmt.Sprintf("user-%d/media/%s%s", req.UserId, uniqueID, ext)

	// Create temp directory
	tempDir := filepath.Join("/tmp", uniqueID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save file to temp
	tempFilePath := filepath.Join(tempDir, req.Filename)
	if err := os.WriteFile(tempFilePath, req.FileData, 0644); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to save file: %v", err)
	}

	// Upload original file using INTERNAL client
	if err := s.uploadFileToMinio(ctx, tempFilePath, objectName, req.ContentType); err != nil {
		log.Printf("Failed to upload file to MinIO: %v", err)
		return nil, status.Error(codes.Internal, "Failed to upload file")
	}

	// Store object paths (not full URLs) for presigned URL access
	mediaURL := objectName
	thumbnailURL := ""

	// Video thumbnail generation
	if isVideoFile(req.Filename) {
		thumbnailName := fmt.Sprintf("user-%d/thumbnails/%s.jpg", req.UserId, uniqueID)
		thumbnailPath := filepath.Join(tempDir, "thumbnail.jpg")

		if err := generateThumbnail(tempFilePath, thumbnailPath, 1.0); err != nil {
			log.Printf("Warning: Failed to generate thumbnail: %v", err)
		} else {
			if err := s.uploadFileToMinio(ctx, thumbnailPath, thumbnailName, "image/jpeg"); err != nil {
				log.Printf("Warning: Failed to upload thumbnail: %v", err)
			} else {
				thumbnailURL = thumbnailName // Object path for presigned access
			}
		}
	} else if isImageFile(req.Filename) {
		// Image optimization
		optimizedPaths, err := optimizeImage(tempFilePath, tempDir)
		if err != nil {
			log.Printf("Warning: Failed to optimize image: %v", err)
		} else {
			for sizeName, optimizedPath := range optimizedPaths {
				optimizedObjectName := fmt.Sprintf("user-%d/images/%s/%s.jpg", req.UserId, sizeName, uniqueID)
				if err := s.uploadFileToMinio(ctx, optimizedPath, optimizedObjectName, "image/jpeg"); err != nil {
					log.Printf("Warning: Failed to upload %s version: %v", sizeName, err)
				} else {
					if sizeName == "large" {
						mediaURL = optimizedObjectName // Use optimized version as main
					}
					if sizeName == "thumb" {
						thumbnailURL = optimizedObjectName // Object path for presigned access
					}
				}
			}
		}
	}

	return &pb.UploadMediaResponse{
		MediaUrl:     mediaURL,     // Object path like "user-123/media/abc.mp4"
		ThumbnailUrl: thumbnailURL, // Object path like "user-123/thumbnails/abc.jpg"
	}, nil
}

// --- Restored Helper Functions ---

func (s *server) uploadFileToMinio(ctx context.Context, localPath, objectName, contentType string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	// Use INTERNAL client for backend operations
	_, err = s.internalClient.PutObject(ctx, minioBucketName, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func generateThumbnail(videoPath, outputPath string, timestampSeconds float64) error {
	// Default to 1 second if not specified
	if timestampSeconds <= 0 {
		timestampSeconds = 1.0
	}
	timestamp := fmt.Sprintf("%.2f", timestampSeconds)

	log.Printf("Running ffmpeg: extracting frame at %s seconds", timestamp)
	cmd := exec.Command("ffmpeg", "-ss", timestamp, "-i", videoPath, "-vframes", "1", "-q:v", "2", "-y", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("❌ ffmpeg failed: %v, stderr: %s", err, stderr.String())
		return fmt.Errorf("ffmpeg failed: %v, stderr: %s", err, stderr.String())
	}
	log.Printf("✅ ffmpeg completed successfully")
	return nil
}

func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv"}
	for _, videoExt := range videoExts {
		if ext == videoExt {
			return true
		}
	}
	return false
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

func optimizeImage(inputPath, tempDir string) (map[string]string, error) {
	sizes := map[string]int{"original": 0, "large": 1080, "medium": 640, "thumb": 320}
	optimizedPaths := make(map[string]string)

	for sizeName, width := range sizes {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("%s.jpg", sizeName))
		var cmd *exec.Cmd
		if width == 0 {
			cmd = exec.Command("convert", inputPath, "-strip", "-quality", "85", "-sampling-factor", "4:2:0", "-interlace", "Plane", outputPath)
		} else {
			cmd = exec.Command("convert", inputPath, "-strip", "-resize", fmt.Sprintf("%dx>", width), "-quality", "85", "-sampling-factor", "4:2:0", "-interlace", "Plane", outputPath)
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to optimize: %v, output: %s", err, string(output))
		}
		optimizedPaths[sizeName] = outputPath
	}
	return optimizedPaths, nil
}
