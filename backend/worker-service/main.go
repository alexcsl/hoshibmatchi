package main

// Worker Service: Handles background jobs for story deletion, video transcoding, and hashtag processing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/lib/pq" // <-- Added for pq.StringArray

	// gRPC Clients
	hashtagPb "github.com/hoshibmatchi/hashtag-service/proto"
)

// --- GORM Models ---

// Story is a minimal struct for story-db deletion
type Story struct {
	gorm.Model
}

// Post is a minimal struct for post-db updates
type Post struct {
	gorm.Model
	MediaURLs    pq.StringArray `gorm:"type:text[]"`
	ThumbnailURL string         `gorm:"type:text"`
}

// server struct holds all our connections
type server struct {
	storyDB       *gorm.DB // Connection to story-db
	postDB        *gorm.DB // Connection to post-db
	amqpCh        *amqp.Channel
	hashtagClient hashtagPb.HashtagServiceClient
	minioClient   *minio.Client
}

func main() {
	// --- Step 1: Connect to Databases ---
	// Connection to story-db (for deleting stories)
	storyDSN := "host=story-db user=admin password=password dbname=story_service_db port=5432 sslmode=disable TimeZone=UTC"
	storyDB, err := gorm.Open(postgres.Open(storyDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to story-db: %v", err)
	}
	storyDB.AutoMigrate(&Story{}) // This migrates the Story struct
	log.Println("Worker successfully connected to story-db")

	// Connection to post-db (for updating transcoded video URLs)
	postDSN := "host=post-db user=admin password=password dbname=post_service_db port=5432 sslmode=disable TimeZone=UTC"
	postDB, err := gorm.Open(postgres.Open(postDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to post-db: %v", err)
	}
	postDB.AutoMigrate(&Post{}) // This migrates the Post struct
	log.Println("Worker successfully connected to post-db")

	// --- Step 2: Connect to RabbitMQ (with retries) ---
	var amqpConn *amqp.Connection
	maxRetries := 30 // Increased retries
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		amqpURI := os.Getenv("RABBITMQ_URI")
		if amqpURI == "" {
			amqpURI = "amqp://guest:guest@rabbitmq:5672/" // Default
		}
		amqpConn, err = amqp.Dial(amqpURI)
		if err == nil {
			log.Println("Worker successfully connected to RabbitMQ")
			break
		}
		log.Printf("Worker failed to connect to RabbitMQ: %v. Retrying...", err)
		time.Sleep(retryDelay)
	}
	if amqpConn == nil {
		log.Fatalf("Worker could not connect to RabbitMQ after %d retries", maxRetries)
	}
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("Worker failed to open RabbitMQ channel: %v", err)
	}
	defer amqpCh.Close()

	// --- Step 3: Connect to Hashtag Service (gRPC Client) ---
	hashtagConn, err := grpc.Dial("hashtag-service:9007", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to hashtag-service: %v", err)
	}
	defer hashtagConn.Close()
	hashtagClient := hashtagPb.NewHashtagServiceClient(hashtagConn)
	log.Println("Worker successfully connected to hashtag-service")

	// --- Step 3.5: Connect to MinIO ---
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

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKeyID, minioSecretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}
	log.Println("Worker successfully connected to MinIO")

	// --- Step 4: Create Server Struct ---
	s := &server{
		storyDB:       storyDB,
		postDB:        postDB,
		amqpCh:        amqpCh,
		hashtagClient: hashtagClient,
		minioClient:   minioClient,
	}

	// --- Step 5: Declare All Queues ---
	// Story deletion queue
	storyQ, err := amqpCh.QueueDeclare("story_deletion_queue", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Worker failed to declare story_deletion_queue: %v", err)
	}
	// Video transcoding queue
	videoQ, err := amqpCh.QueueDeclare("video_transcoding_queue", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Worker failed to declare video_transcoding_queue: %v", err)
	}
	// Hashtag processing queue
	hashtagQ, err := amqpCh.QueueDeclare("hashtag_queue", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Worker failed to declare hashtag_queue: %v", err)
	}

	// --- Step 6: Start Consuming from ALL queues ---
	storyMsgs, err := amqpCh.Consume(storyQ.Name, "story_consumer", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register story consumer: %v", err)
	}

	videoMsgs, err := amqpCh.Consume(videoQ.Name, "video_consumer", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register video consumer: %v", err)
	}

	hashtagMsgs, err := amqpCh.Consume(hashtagQ.Name, "hashtag_consumer", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register hashtag consumer: %v", err)
	}

	var forever chan struct{}

	// Goroutine for story deletion jobs
	go func() {
		for d := range storyMsgs {
			log.Printf("Received a story deletion job: %s", d.Body)
			s.processStoryDeletion(d.Body)
			d.Ack(false) // Acknowledge the message
		}
	}()

	// Goroutine for video transcoding jobs
	go func() {
		for d := range videoMsgs {
			log.Printf("Received a video transcoding job: %s", d.Body)
			s.processVideoTranscoding(d.Body)
			d.Ack(false) // Acknowledge the message
		}
	}()

	// Goroutine for hashtag jobs
	go func() {
		for d := range hashtagMsgs {
			log.Printf("Received a hashtag processing job: %s", d.Body)
			s.processHashtagJob(d.Body)
			d.Ack(false) // Acknowledge the message
		}
	}()

	log.Println("Worker service is running. Waiting for jobs...")
	forever = make(chan struct{})
	<-forever // Block forever
}

// processStoryDeletion deletes the story from the database
func (s *server) processStoryDeletion(body []byte) {
	var job map[string]uint
	if err := json.Unmarshal(body, &job); err != nil {
		log.Printf("Error decoding story deletion job: %v", err)
		return
	}

	storyID, ok := job["story_id"]
	if !ok {
		log.Printf("Invalid job payload, missing 'story_id'")
		return
	}

	// Delete the story from the story-db
	if result := s.storyDB.Delete(&Story{}, storyID); result.Error != nil {
		log.Printf("Failed to delete story %d: %v", storyID, result.Error)
		return
	}

	log.Printf("Successfully deleted story %d", storyID)
}

// processVideoTranscoding transcodes videos using FFmpeg
func (s *server) processVideoTranscoding(body []byte) {
	var job map[string]interface{}
	if err := json.Unmarshal(body, &job); err != nil {
		log.Printf("Error decoding video transcoding job: %v", err)
		return
	}

	// GORM uses float64 for JSON numbers
	postIDFloat, ok := job["post_id"].(float64)
	if !ok {
		log.Printf("Invalid job payload, missing or invalid 'post_id'")
		return
	}
	postID := uint(postIDFloat)

	log.Printf("--- STARTING VIDEO TRANSCODING for Post ID: %d ---", postID)

	// 1. Find the post in the post-db
	var post Post
	if err := s.postDB.First(&post, postID).Error; err != nil {
		log.Printf("Failed to find post %d for transcoding: %v", postID, err)
		return
	}

	if len(post.MediaURLs) == 0 {
		log.Printf("No media URLs found for Post ID: %d", postID)
		return
	}

	// 2. Create temp directory for processing
	tempDir, err := os.MkdirTemp("", "transcode-*")
	if err != nil {
		log.Printf("Failed to create temp directory: %v", err)
		return
	}
	defer os.RemoveAll(tempDir)

	var transcodedURLs []string

	// 3. Process each media URL
	for idx, mediaURL := range post.MediaURLs {
		if !isVideoFile(mediaURL) {
			// Keep non-video files as-is
			transcodedURLs = append(transcodedURLs, mediaURL)
			continue
		}

		log.Printf("Transcoding video %d/%d: %s", idx+1, len(post.MediaURLs), mediaURL)

		// Extract filename from URL (e.g., "videos/abc123.mp4" -> "abc123")
		parts := strings.Split(mediaURL, "/")
		if len(parts) < 2 {
			log.Printf("Invalid media URL format: %s", mediaURL)
			transcodedURLs = append(transcodedURLs, mediaURL)
			continue
		}
		filename := parts[len(parts)-1]
		filenameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))

		// 4. Download video from MinIO
		inputPath := filepath.Join(tempDir, filename)
		if err := s.downloadFromMinio("hoshibmatchi", mediaURL, inputPath); err != nil {
			log.Printf("Failed to download video from MinIO: %v", err)
			transcodedURLs = append(transcodedURLs, mediaURL)
			continue
		}

		// 5. Transcode to multiple resolutions (720p, 480p, 360p)
		resolutions := []struct {
			name   string
			width  int
			height int
		}{
			{"720p", 1280, 720},
			{"480p", 854, 480},
			{"360p", 640, 360},
		}

		successfulTranscode := false
		for _, res := range resolutions {
			outputFilename := fmt.Sprintf("%s_%s.mp4", filenameNoExt, res.name)
			outputPath := filepath.Join(tempDir, outputFilename)

			// Run FFmpeg transcoding
			cmd := exec.Command("ffmpeg",
				"-i", inputPath,
				"-vf", fmt.Sprintf("scale=%d:%d", res.width, res.height),
				"-c:v", "libx264",
				"-preset", "fast",
				"-crf", "23",
				"-c:a", "aac",
				"-b:a", "128k",
				"-movflags", "+faststart",
				"-y", // Overwrite output file
				outputPath,
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("FFmpeg failed for %s: %v\nOutput: %s", res.name, err, string(output))
				continue
			}

			log.Printf("Successfully transcoded to %s", res.name)

			// 6. Upload transcoded video to MinIO
			minioPath := fmt.Sprintf("videos/transcoded/%s", outputFilename)
			if err := s.uploadToMinio("hoshibmatchi", minioPath, outputPath); err != nil {
				log.Printf("Failed to upload %s to MinIO: %v", res.name, err)
				continue
			}

			log.Printf("Uploaded %s to MinIO: %s", res.name, minioPath)
			transcodedURLs = append(transcodedURLs, minioPath)
			successfulTranscode = true

			// Generate thumbnail from the first resolution (720p)
			if res.name == "720p" && post.ThumbnailURL == "" {
				thumbnailFilename := fmt.Sprintf("%s_thumb.jpg", filenameNoExt)
				thumbnailPath := filepath.Join(tempDir, thumbnailFilename)

				// Extract thumbnail at 1 second mark
				thumbCmd := exec.Command("ffmpeg",
					"-ss", "00:00:01",
					"-i", outputPath,
					"-vframes", "1",
					"-q:v", "2",
					"-y",
					thumbnailPath,
				)

				thumbOutput, thumbErr := thumbCmd.CombinedOutput()
				if thumbErr != nil {
					log.Printf("Failed to generate thumbnail: %v\nOutput: %s", thumbErr, string(thumbOutput))
				} else {
					// Upload thumbnail to MinIO
					thumbnailMinioPath := fmt.Sprintf("thumbnails/%s", thumbnailFilename)
					if err := s.uploadToMinio("media", thumbnailMinioPath, thumbnailPath); err != nil {
						log.Printf("Failed to upload thumbnail to MinIO: %v", err)
					} else {
						thumbnailURL := fmt.Sprintf("http://localhost:9000/media/%s", thumbnailMinioPath)
						log.Printf("Uploaded thumbnail to MinIO: %s", thumbnailURL)

						// Update post with thumbnail URL
						if err := s.postDB.Model(&Post{}).Where("id = ?", postID).Update("thumbnail_url", thumbnailURL).Error; err != nil {
							log.Printf("Failed to update post thumbnail URL: %v", err)
						} else {
							log.Printf("Updated post %d with thumbnail URL", postID)
						}
					}
				}
			}
		}

		// If transcoding failed, keep the original
		if !successfulTranscode {
			log.Printf("All transcoding failed for %s, keeping original", mediaURL)
			transcodedURLs = append(transcodedURLs, mediaURL)
		}
	}

	// 7. Update the post with transcoded URLs
	if err := s.postDB.Model(&post).Update("media_urls", pq.StringArray(transcodedURLs)).Error; err != nil {
		log.Printf("Failed to update post with transcoded URLs: %v", err)
		return
	}

	log.Printf("--- FINISHED VIDEO TRANSCODING for Post ID: %d. New URLs: %v ---", postID, transcodedURLs)
}

// downloadFromMinio downloads a file from MinIO to local filesystem
func (s *server) downloadFromMinio(bucketName, objectName, filePath string) error {
	ctx := context.Background()

	// Get object from MinIO
	object, err := s.minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Create local file
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Copy data
	if _, err := io.Copy(outFile, object); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

// uploadToMinio uploads a local file to MinIO
func (s *server) uploadToMinio(bucketName, objectName, filePath string) error {
	ctx := context.Background()

	// Open local file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Upload to MinIO
	_, err = s.minioClient.PutObject(ctx, bucketName, objectName, file, stat.Size(), minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

// isVideoFile checks if a URL is a video based on extension
func isVideoFile(url string) bool {
	lowerURL := strings.ToLower(url)
	return strings.HasSuffix(lowerURL, ".mp4") ||
		strings.HasSuffix(lowerURL, ".mov") ||
		strings.HasSuffix(lowerURL, ".avi") ||
		strings.HasSuffix(lowerURL, ".webm") ||
		strings.HasSuffix(lowerURL, ".mkv")
}

// processHashtagJob calls the hashtag-service
func (s *server) processHashtagJob(body []byte) {
	var job struct {
		PostID       int64    `json:"post_id"`
		HashtagNames []string `json:"hashtag_names"`
	}
	if err := json.Unmarshal(body, &job); err != nil {
		log.Printf("Error decoding hashtag job: %v", err)
		return
	}

	log.Printf("Processing hashtag job for Post ID: %d", job.PostID)

	// Call the hashtag-service gRPC method
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.hashtagClient.AddHashtagsToPost(ctx, &hashtagPb.AddHashtagsToPostRequest{
		PostId:       job.PostID,
		HashtagNames: job.HashtagNames,
	})

	if err != nil {
		log.Printf("Failed to call hashtag-service for post %d: %v", job.PostID, err)
	} else {
		log.Printf("Successfully processed hashtag job for post %d", job.PostID)
	}
}
