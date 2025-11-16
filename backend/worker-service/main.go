package main

// Worker Service: Handles background jobs for story deletion, video transcoding, and hashtag processing

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

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
	MediaURLs pq.StringArray `gorm:"type:text[]"`
}

// server struct holds all our connections
type server struct {
	storyDB       *gorm.DB // Connection to story-db
	postDB        *gorm.DB // Connection to post-db
	amqpCh        *amqp.Channel
	hashtagClient hashtagPb.HashtagServiceClient
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
		amqpConn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
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

	// --- Step 4: Create Server Struct ---
	s := &server{
		storyDB:       storyDB,
		postDB:        postDB,
		amqpCh:        amqpCh,
		hashtagClient: hashtagClient,
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

// processVideoTranscoding simulates transcoding and updates the post-db
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

	log.Printf("--- STARTING VIDEO JOB for Post ID: %d ---", postID)

	// 1. Find the post in the post-db
	var post Post
	if err := s.postDB.First(&post, postID).Error; err != nil {
		log.Printf("Failed to find post %d for transcoding: %v", postID, err)
		return // Can't process if post doesn't exist
	}

	// 2. Simulate a long-running transcode job
	log.Printf("Transcoding video(s): %v", post.MediaURLs)
	time.Sleep(5 * time.Second) // Simulates ffmpeg work

	// 3. Simulate updating the URL to an HLS playlist
	newMediaURLs := []string{}
	for _, url := range post.MediaURLs {
		if strings.HasSuffix(strings.ToLower(url), ".mp4") {
			newURL := strings.Replace(url, ".mp4", ".m3u8", 1)
			newMediaURLs = append(newMediaURLs, newURL)
		} else {
			newMediaURLs = append(newMediaURLs, url) // Keep non-mp4 files as-is
		}
	}

	// 4. Update the post in the post-db
	if err := s.postDB.Model(&post).Update("media_urls", pq.StringArray(newMediaURLs)).Error; err != nil {
		log.Printf("Failed to update post %d with new HLS URLs: %v", postID, err)
		return
	}

	log.Printf("--- FINISHED VIDEO JOB for Post ID: %d. New URLs: %v ---", postID, newMediaURLs)
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
