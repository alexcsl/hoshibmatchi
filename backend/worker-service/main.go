package main

import (
	// "context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Define a minimal Story struct for deletion
type Story struct {
	gorm.Model
}

type server struct {
	db     *gorm.DB
	amqpCh *amqp.Channel
}

func main() {
	// --- Step 1: Connect to Story DB ---
	// The worker needs to talk to the story-db to delete stories
	dsn := "host=story-db user=admin password=password dbname=story_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to story-db: %v", err)
	}
	db.AutoMigrate(&Story{})
	log.Println("Worker successfully connected to story-db")

	// --- Step 2: Connect to RabbitMQ (with retries) ---
	var amqpConn *amqp.Connection
	maxRetries := 10
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

	s := &server{db: db, amqpCh: amqpCh}

	// --- Step 3: Declare the FINAL queue we will consume from ---
	q, err := amqpCh.QueueDeclare(
		"story_deletion_queue", // This is the final "work" queue
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		log.Fatalf("Worker failed to declare story_deletion_queue: %v", err)
	}

	// --- ADDED: Declare video transcoding queue ---
	videoQ, err := amqpCh.QueueDeclare(
		"video_transcoding_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Worker failed to declare video_transcoding_queue: %v", err)
	}

	// --- REVISED: Consume from BOTH queues ---
	storyMsgs, err := amqpCh.Consume(
		q.Name,           // story_deletion_queue
		"story_consumer", // consumer tag
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		log.Fatalf("Failed to register story consumer: %v", err)
	}

	videoMsgs, err := amqpCh.Consume(
		videoQ.Name,      // video_transcoding_queue
		"video_consumer", // consumer tag
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		log.Fatalf("Failed to register video consumer: %v", err)
	}

	var forever chan struct{}

	// Consumer for story deletion jobs
	go func() {
		for d := range storyMsgs {
			log.Printf("Received a story deletion job: %s", d.Body)
			s.processStoryDeletion(d.Body)
			d.Ack(false) // Acknowledge the message
		}
	}()

	// Consumer for video transcoding jobs
	go func() {
		for d := range videoMsgs {
			log.Printf("Received a video transcoding job: %s", d.Body)
			s.processVideoTranscoding(d.Body)
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
	if result := s.db.Delete(&Story{}, storyID); result.Error != nil {
		log.Printf("Failed to delete story %d: %v", storyID, result.Error)
		return
	}

	log.Printf("Successfully deleted story %d", storyID)
}

func (s *server) processVideoTranscoding(body []byte) {
	var job map[string]interface{}
	if err := json.Unmarshal(body, &job); err != nil {
		log.Printf("Error decoding video transcoding job: %v", err)
		return
	}

	postID, ok := job["post_id"]
	if !ok {
		log.Printf("Invalid job payload, missing 'post_id'")
		return
	}

	log.Printf("--- STARTING VIDEO JOB for Post ID: %v ---", postID)

	// Simulate a long-running transcode job
	time.Sleep(5 * time.Second)

	// TODO:
	// 1. Download video from MinIO (using media_urls)
	// 2. Run 'ffmpeg' command to convert to HLS (.m3u8)
	// 3. Upload new HLS files back to MinIO
	// 4. Connect to post-db and update the Post row's media_urls
	//    to point to the new .m3u8 file.

	log.Printf("--- FINISHED VIDEO JOB for Post ID: %v ---", postID)
}
