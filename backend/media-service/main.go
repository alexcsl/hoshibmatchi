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
	// Internal Docker networking (Backend -> MinIO)
	minioInternalEndpoint = "minio:9000"
	// External Browser networking (Frontend -> MinIO)
	minioExternalEndpoint = "localhost:9000"

	minioAccessKeyID     = "minioadmin"
	minioSecretAccessKey = "minioadmin"
	minioUseSSL          = false
	minioBucketName      = "media"
	minioRegion          = "us-east-1" // Critical for avoiding 500 errors
)

func main() {
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

	// 3. Ensure Bucket Exists & Set Public Policy
	ctx := context.Background()
	exists, err := internalClient.BucketExists(ctx, minioBucketName)
	if err != nil || !exists {
		internalClient.MakeBucket(ctx, minioBucketName, minio.MakeBucketOptions{Region: minioRegion})
		log.Printf("Created bucket '%s'", minioBucketName)
	}

	// --- CRITICAL FIX: Make Bucket Publicly Readable ---
	// This allows the browser to load images via <img src="..."> without 403 errors.
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, minioBucketName)

	if err := internalClient.SetBucketPolicy(ctx, minioBucketName, policy); err != nil {
		log.Printf("Warning: Failed to set bucket policy: %v", err)
	} else {
		log.Println("Bucket policy set to public-read")
	}

	// 4. Start gRPC
	lis, err := net.Listen("tcp", ":9005")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
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

	// Construct public URL for database
	finalURL := fmt.Sprintf("http://%s/%s/%s", minioExternalEndpoint, minioBucketName, objectName)

	return &pb.GetUploadURLResponse{
		UploadUrl:     uploadURL.String(),
		FinalMediaUrl: finalURL,
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

	// Construct public URLs
	mediaURL := fmt.Sprintf("http://%s/%s/%s", minioExternalEndpoint, minioBucketName, objectName)
	thumbnailURL := ""

	// Restore Video Logic
	if isVideoFile(req.Filename) {
		thumbnailName := fmt.Sprintf("user-%d/thumbnails/%s.jpg", req.UserId, uniqueID)
		thumbnailPath := filepath.Join(tempDir, "thumbnail.jpg")

		if err := generateThumbnail(tempFilePath, thumbnailPath); err != nil {
			log.Printf("Warning: Failed to generate thumbnail: %v", err)
		} else {
			if err := s.uploadFileToMinio(ctx, thumbnailPath, thumbnailName, "image/jpeg"); err != nil {
				log.Printf("Warning: Failed to upload thumbnail: %v", err)
			} else {
				thumbnailURL = fmt.Sprintf("http://%s/%s/%s", minioExternalEndpoint, minioBucketName, thumbnailName)
			}
		}
	} else if isImageFile(req.Filename) {
		// Restore Image Optimization Logic
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
						mediaURL = fmt.Sprintf("http://%s/%s/%s", minioExternalEndpoint, minioBucketName, optimizedObjectName)
					}
					if sizeName == "thumb" {
						thumbnailURL = fmt.Sprintf("http://%s/%s/%s", minioExternalEndpoint, minioBucketName, optimizedObjectName)
					}
				}
			}
		}
	}

	return &pb.UploadMediaResponse{
		MediaUrl:     mediaURL,
		ThumbnailUrl: thumbnailURL,
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

func generateThumbnail(videoPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-ss", "00:00:01", "-i", videoPath, "-vframes", "1", "-q:v", "2", "-y", outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %v, stderr: %s", err, stderr.String())
	}
	return nil
}

func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv"}
	for _, videoExt := range videoExts {
		if ext == videoExt { return true }
	}
	return false
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
	for _, imgExt := range imageExts {
		if ext == imgExt { return true }
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