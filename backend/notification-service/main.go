package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Notification defines the GORM model
type Notification struct {
	gorm.Model
	UserID   int64  `gorm:"index"` // The ID of the user who receives the notification
	ActorID  int64  // The ID of the user who performed the action (e.g., liked, followed)
	Type     string // e.g., "post.liked", "user.followed"
	EntityID int64  // The ID of the object (post, user)
	IsRead   bool   `gorm:"default:false"`
}

type server struct {
	db *gorm.DB
}

func main() {
	// --- Step 1: Connect to Notification DB ---
	dsn := "host=notification-db user=admin password=password dbname=notification_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to notification-db: %v", err)
	}
	db.AutoMigrate(&Notification{})

	// Encapsulate DB in a struct
	s := &server{db: db}

	// --- Step 2: Connect to RabbitMQ (with retries) ---
	var amqpConn *amqp.Connection
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		amqpURI := os.Getenv("RABBITMQ_URI")
		if amqpURI == "" {
			amqpURI = "amqp://guest:guest@rabbitmq:5672/" // Default
		}
		amqpConn, err = amqp.Dial(amqpURI)
		if err == nil {
			log.Println("Successfully connected to RabbitMQ")
			break
		}
		log.Printf("Failed to connect to RabbitMQ: %v. Retrying...", err)
		time.Sleep(retryDelay)
	}
	if amqpConn == nil {
		log.Fatalf("Could not connect to RabbitMQ after %d retries", maxRetries)
	}
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
	defer amqpCh.Close()

	q, err := amqpCh.QueueDeclare(
		"notification_queue", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare notification_queue: %v", err)
	}

	// --- Step 3: Start consuming messages ---
	msgs, err := amqpCh.Consume(
		q.Name, "", false, false, false, false, nil, // auto-ack = false
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Println("Notification service is running. Waiting for messages...")

	var forever chan struct{}
	go func() {
		for d := range msgs {
			log.Printf("Received a notification job: %s", d.Body)
			s.processNotification(d.Body)
			d.Ack(false) // Acknowledge the message
		}
	}()
	forever = make(chan struct{})
	<-forever // Block forever
}

// processNotification saves the job to the database
func (s *server) processNotification(body []byte) {
	// Use map[string]interface{} to handle mixed types (string, float64)
	var job map[string]interface{}
	if err := json.Unmarshal(body, &job); err != nil {
		log.Printf("Error decoding notification job: %v", err)
		return
	}

	// Helper to safely convert JSON numbers (float64) to int64
	toInt64 := func(v interface{}) int64 {
		if f, ok := v.(float64); ok {
			return int64(f)
		}
		return 0
	}

	notification := Notification{
		Type:     job["type"].(string),
		UserID:   toInt64(job["user_id"]),
		ActorID:  toInt64(job["actor_id"]),
		EntityID: toInt64(job["entity_id"]),
		IsRead:   false,
	}

	if result := s.db.Create(&notification); result.Error != nil {
		log.Printf("Failed to save notification to db: %v", result.Error)
		return
	}

	log.Printf("Successfully saved notification for user %d", notification.UserID)
}
