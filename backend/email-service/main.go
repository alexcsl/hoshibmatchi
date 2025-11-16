package main

import (
	"encoding/json"
	"log"
	"time" // <-- ADD THIS IMPORT

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// --- Step 1: Connect to RabbitMQ (with retries) ---
	var amqpConn *amqp.Connection
	var err error
	
	maxRetries := 30
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		amqpConn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil {
			// Success!
			log.Println("Successfully connected to RabbitMQ")
			break
		}

		log.Printf("Failed to connect to RabbitMQ: %v", err)
		log.Printf("Retrying in %v... (%d/%d)", retryDelay, i+1, maxRetries)
		time.Sleep(retryDelay)
	}

	if amqpConn == nil {
		log.Fatalf("Could not connect to RabbitMQ after %d retries", maxRetries)
	}
	defer amqpConn.Close()

	// --- Step 2: Open Channel ---
	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
	defer amqpCh.Close()

	// --- Step 3: Ensure the queue exists ---
	q, err := amqpCh.QueueDeclare(
		"email_queue", // queue name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare email_queue: %v", err)
	}

	// --- Step 4: Start consuming messages ---
	msgs, err := amqpCh.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Println("Email service is running. Waiting for messages...")

	// --- Step 5: Run forever ---
	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf("Received an email job: %s", d.Body)
			processEmail(d.Body)
		}
	}()

	<-forever // Block forever
}

// processEmail function (remains the same)
func processEmail(body []byte) {
	var job map[string]string
	if err := json.Unmarshal(body, &job); err != nil {
		log.Printf("Error decoding email job: %v", err)
		return
	}

	log.Println("=====================================")
	log.Printf("📫 SIMULATING SENDING EMAIL 📫")
	log.Printf("TO: %s", job["to"])
	
	switch job["type"] {
	case "registration_otp":
		log.Printf("SUBJECT: Your hoshiBmaTchi Verification Code")
		log.Printf("BODY: Your OTP code is: %s", job["otpCode"])
	case "password_reset":
		log.Printf("SUBJECT: Your hoshiBmaTchi Password Reset")
		log.Printf("BODY: Your reset token is: %s", job["token"])
	case "newsletter":
		log.Printf("SUBJECT: [Newsletter] %s", job["subject"])
		log.Printf("BODY: %s", job["body"])
	case "verification_accepted":
		log.Printf("SUBJECT: Your hoshiBmaTchi Verification is Approved!")
		log.Printf("BODY: Hello %s, congratulations! Your account is now verified.", job["username"])
	case "verification_rejected":
		log.Printf("SUBJECT: hoshiBmaTchi Verification Status")
		log.Printf("BODY: Hello %s, after a review, we're unable to verify your account at this time.", job["username"])
	default:
		log.Printf("Unknown email type: %s", job["type"])
	}
	log.Println("=====================================")
}