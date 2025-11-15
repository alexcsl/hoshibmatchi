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

	// --- Step 4: Start consuming messages ---
	msgs, err := amqpCh.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack (set to false)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Worker failed to register consumer: %v", err)
	}

	log.Println("Worker service is running. Waiting for story deletion jobs...")

	var forever chan struct{}
	go func() {
		for d := range msgs {
			log.Printf("Received a story deletion job: %s", d.Body)
			s.processStoryDeletion(d.Body)
			d.Ack(false) // Acknowledge the message *after* processing
		}
	}()
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