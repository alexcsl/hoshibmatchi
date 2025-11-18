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
	minioClient   *minio.Client
	minioEndpoint string
}

const (
	minioUseSSL     = false
	minioBucketName = "media"
)

func main() {
	// Get MinIO credentials from environment
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "minio:9000" // Default
	}
	minioAccessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	if minioAccessKeyID == "" {
		minioAccessKeyID = "minioadmin" // Default
	}
	minioSecretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	if minioSecretAccessKey == "" {
		minioSecretAccessKey = "minioadmin" // Default
	}

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
	pb.RegisterMediaServiceServer(s, &server{
		minioClient:   minioClient,
		minioEndpoint: minioEndpoint,
	})

	log.Println("Media service listening on port 9005...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// generateThumbnail creates a thumbnail from a video at the 1-second mark
func generateThumbnail(videoPath, outputPath string) error {
	// Use FFmpeg to extract a frame at 1 second
	// -ss 00:00:01: seek to 1 second
	// -i: input file
	// -vframes 1: extract only 1 frame
	// -q:v 2: high quality (1-31, lower is better)
	cmd := exec.Command("ffmpeg",
		"-ss", "00:00:01",
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", "2",
		"-y", // Overwrite output file
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

// isVideoFile checks if the filename has a video extension
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

// isImageFile checks if the filename has an image extension
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

// optimizeImage creates optimized versions of an image in multiple sizes
// Returns map[size]filepath for the optimized images
func optimizeImage(inputPath, tempDir string) (map[string]string, error) {
	sizes := map[string]int{
		"original": 0,    // Keep original, just optimize
		"large":    1080, // 1080px width
		"medium":   640,  // 640px width
		"thumb":    320,  // 320px width
	}

	optimizedPaths := make(map[string]string)

	for sizeName, width := range sizes {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("%s.jpg", sizeName))

		var cmd *exec.Cmd
		if width == 0 {
			// Original size, just optimize and convert to JPEG
			cmd = exec.Command("convert",
				inputPath,
				"-strip",         // Remove EXIF metadata
				"-quality", "85", // 85% JPEG quality
				"-sampling-factor", "4:2:0", // Chroma subsampling
				"-interlace", "Plane", // Progressive JPEG
				outputPath,
			)
		} else {
			// Resize and optimize
			cmd = exec.Command("convert",
				inputPath,
				"-strip",                              // Remove EXIF metadata
				"-resize", fmt.Sprintf("%dx>", width), // Resize width, maintain aspect ratio
				"-quality", "85", // 85% JPEG quality
				"-sampling-factor", "4:2:0", // Chroma subsampling
				"-interlace", "Plane", // Progressive JPEG
				outputPath,
			)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to optimize image %s: %v, output: %s", sizeName, err, string(output))
		}

		optimizedPaths[sizeName] = outputPath
		log.Printf("Created %s version: %s", sizeName, outputPath)
	}

	return optimizedPaths, nil
}

// uploadFileToMinio uploads a file from local path to MinIO
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

	_, err = s.minioClient.PutObject(ctx, minioBucketName, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %v", err)
	}

	return nil
}

// --- Implement UploadMedia (New method for actual file upload with thumbnail generation) ---
func (s *server) UploadMedia(ctx context.Context, req *pb.UploadMediaRequest) (*pb.UploadMediaResponse, error) {
	// Generate unique filename
	uniqueID := uuid.New().String()
	ext := filepath.Ext(req.Filename)
	objectName := fmt.Sprintf("user-%d/media/%s%s", req.UserId, uniqueID, ext)

	// Create temp directory for processing
	tempDir := filepath.Join("/tmp", uniqueID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save uploaded file to temp location
	tempFilePath := filepath.Join(tempDir, req.Filename)
	if err := os.WriteFile(tempFilePath, req.FileData, 0644); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to save file: %v", err)
	}

	// Upload original file to MinIO
	if err := s.uploadFileToMinio(ctx, tempFilePath, objectName, req.ContentType); err != nil {
		log.Printf("Failed to upload file to MinIO: %v", err)
		return nil, status.Error(codes.Internal, "Failed to upload file")
	}

	mediaURL := fmt.Sprintf("http://%s/%s/%s", s.minioEndpoint, minioBucketName, objectName)
	thumbnailURL := ""

	// Process based on file type
	if isVideoFile(req.Filename) {
		// Generate thumbnail if it's a video
		thumbnailName := fmt.Sprintf("user-%d/thumbnails/%s.jpg", req.UserId, uniqueID)
		thumbnailPath := filepath.Join(tempDir, "thumbnail.jpg")

		if err := generateThumbnail(tempFilePath, thumbnailPath); err != nil {
			log.Printf("Warning: Failed to generate thumbnail: %v", err)
		} else {
			// Upload thumbnail to MinIO
			if err := s.uploadFileToMinio(ctx, thumbnailPath, thumbnailName, "image/jpeg"); err != nil {
				log.Printf("Warning: Failed to upload thumbnail: %v", err)
			} else {
				thumbnailURL = fmt.Sprintf("http://%s/%s/%s", s.minioEndpoint, minioBucketName, thumbnailName)
				log.Printf("Successfully generated thumbnail for video: %s", thumbnailURL)
			}
		}
	} else if isImageFile(req.Filename) {
		// Optimize images and create multiple sizes
		optimizedPaths, err := optimizeImage(tempFilePath, tempDir)
		if err != nil {
			log.Printf("Warning: Failed to optimize image: %v", err)
		} else {
			// Upload all optimized versions
			for sizeName, optimizedPath := range optimizedPaths {
				optimizedObjectName := fmt.Sprintf("user-%d/images/%s/%s.jpg", req.UserId, sizeName, uniqueID)
				if err := s.uploadFileToMinio(ctx, optimizedPath, optimizedObjectName, "image/jpeg"); err != nil {
					log.Printf("Warning: Failed to upload %s version: %v", sizeName, err)
				} else {
					log.Printf("Uploaded %s version: %s", sizeName, optimizedObjectName)

					// Use the large version as the primary media URL
					if sizeName == "large" {
						mediaURL = fmt.Sprintf("http://%s/%s/%s", s.minioEndpoint, minioBucketName, optimizedObjectName)
					}
					// Use thumb as thumbnail
					if sizeName == "thumb" {
						thumbnailURL = fmt.Sprintf("http://%s/%s/%s", s.minioEndpoint, minioBucketName, optimizedObjectName)
					}
				}
			}
			log.Printf("Successfully optimized and uploaded image in multiple sizes")
		}
	}

	return &pb.UploadMediaResponse{
		MediaUrl:     mediaURL,
		ThumbnailUrl: thumbnailURL,
	}, nil
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
	finalURL := fmt.Sprintf("%s/%s/%s", s.minioEndpoint, minioBucketName, objectName)

	_ = formData // We'd send this to the frontend too

	return &pb.GetUploadURLResponse{
		UploadUrl:     uploadURL.String(),
		FinalMediaUrl: finalURL,
	}, nil
}
