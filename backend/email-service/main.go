package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gopkg.in/gomail.v2"
)

// SMTP configuration from environment variables
var (
	smtpHost     = os.Getenv("SMTP_HOST")
	smtpPort     = getEnvInt("SMTP_PORT", 587)
	smtpUser     = os.Getenv("SMTP_USER")
	smtpPassword = os.Getenv("SMTP_PASSWORD")
	smtpFrom     = os.Getenv("SMTP_FROM")
)

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func main() {
	// Check SMTP configuration
	if smtpHost == "" || smtpUser == "" || smtpPassword == "" {
		log.Println("WARNING: SMTP not configured. Set SMTP_HOST, SMTP_USER, SMTP_PASSWORD")
		log.Println("Email service will simulate sending emails (development mode)")
	} else {
		if smtpFrom == "" {
			smtpFrom = smtpUser
		}
		log.Printf("SMTP configured: %s:%d (from: %s)", smtpHost, smtpPort, smtpFrom)
	}

	// --- Step 1: Connect to RabbitMQ (with retries) ---
	var amqpConn *amqp.Connection
	var err error

	maxRetries := 30
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
		false,  // auto-ack (changed to false for retry logic)
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
			log.Printf("Received email job: %s", d.Body)
			if processEmail(d.Body) {
				d.Ack(false) // Acknowledge on success
			} else {
				d.Nack(false, true) // Requeue on failure
			}
		}
	}()

	<-forever // Block forever
}

// processEmail processes and sends an email with retry logic
func processEmail(body []byte) bool {
	var job map[string]string
	if err := json.Unmarshal(body, &job); err != nil {
		log.Printf("Error decoding email job: %v", err)
		return false // Don't retry malformed jobs
	}

	to := job["to"]
	emailType := job["type"]

	// Generate email content based on type
	subject, htmlBody := generateEmailContent(emailType, job)

	// If SMTP is not configured, simulate sending
	if smtpHost == "" || smtpUser == "" || smtpPassword == "" {
		simulateEmail(to, subject, htmlBody)
		return true
	}

	// Send email with retry logic (exponential backoff)
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := sendEmail(to, subject, htmlBody); err != nil {
			log.Printf("Attempt %d/%d failed to send email to %s: %v", attempt, maxRetries, to, err)
			if attempt < maxRetries {
				backoff := time.Duration(attempt*attempt) * time.Second
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
			} else {
				log.Printf("Failed to send email to %s after %d attempts", to, maxRetries)
				return false // Requeue for later retry
			}
		} else {
			log.Printf("✅ Successfully sent %s email to %s", emailType, to)
			return true
		}
	}

	return false
}

// sendEmail sends an email using SMTP
func sendEmail(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", smtpFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)

	return d.DialAndSend(m)
}

// simulateEmail logs email content (development mode)
func simulateEmail(to, subject, body string) {
	log.Println("=====================================")
	log.Printf("📫 SIMULATING SENDING EMAIL 📫")
	log.Printf("TO: %s", to)
	log.Printf("SUBJECT: %s", subject)
	log.Printf("BODY (truncated): %s...", truncate(body, 100))
	log.Println("=====================================")
}

// generateEmailContent creates subject and HTML body for each email type
func generateEmailContent(emailType string, data map[string]string) (string, string) {
	switch emailType {
	case "registration_otp":
		return "Your hoshiBmaTchi Verification Code",
			fmt.Sprintf(templateRegistrationOTP, data["otpCode"])

	case "password_reset":
		return "Reset Your hoshiBmaTchi Password",
			fmt.Sprintf(templatePasswordReset, data["otpCode"])

	case "newsletter":
		return data["subject"],
			fmt.Sprintf(templateNewsletter, data["subject"], data["body"])

	case "verification_accepted":
		return "🎉 Your hoshiBmaTchi Account is Verified!",
			fmt.Sprintf(templateVerificationAccepted, data["username"])

	case "verification_rejected":
		return "hoshiBmaTchi Verification Status Update",
			fmt.Sprintf(templateVerificationRejected, data["username"])

	default:
		return "hoshiBmaTchi Notification",
			fmt.Sprintf(templateGeneric, "You have a new notification", "Please check your account for more details.")
	}
}

// truncate string for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// HTML Email Templates
const templateRegistrationOTP = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .otp-code { font-size: 32px; font-weight: bold; color: #667eea; text-align: center; padding: 20px; background: white; border-radius: 8px; margin: 20px 0; letter-spacing: 5px; }
        .footer { text-align: center; margin-top: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌟 Welcome to hoshiBmaTchi!</h1>
        </div>
        <div class="content">
            <p>Hello!</p>
            <p>Thank you for signing up. Please use the following verification code to complete your registration:</p>
            <div class="otp-code">%s</div>
            <p>This code will expire in 10 minutes.</p>
            <p>If you didn't request this code, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>© 2025 hoshiBmaTchi. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const templatePasswordReset = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .otp-code { font-size: 32px; font-weight: bold; color: #667eea; text-align: center; padding: 20px; background: white; border-radius: 8px; margin: 20px 0; letter-spacing: 5px; }
        .warning { background: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔒 Password Reset Request</h1>
        </div>
        <div class="content">
            <p>Hello!</p>
            <p>We received a request to reset your hoshiBmaTchi password. Use the following verification code:</p>
            <div class="otp-code">%s</div>
            <p>This code will expire in 15 minutes.</p>
            <div class="warning">
                <strong>⚠️ Security Notice:</strong> If you didn't request a password reset, please ignore this email and ensure your account is secure.
            </div>
        </div>
        <div class="footer">
            <p>© 2025 hoshiBmaTchi. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const templateNewsletter = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .footer { text-align: center; margin-top: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📰 %s</h1>
        </div>
        <div class="content">
            %s
        </div>
        <div class="footer">
            <p>© 2025 hoshiBmaTchi. All rights reserved.</p>
            <p><a href="#">Unsubscribe</a> from newsletters</p>
        </div>
    </div>
</body>
</html>
`

const templateVerificationAccepted = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .badge { font-size: 64px; text-align: center; margin: 20px 0; }
        .success { background: #d4edda; border-left: 4px solid #28a745; padding: 15px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Congratulations!</h1>
        </div>
        <div class="content">
            <div class="badge">✓</div>
            <p>Hello %s!</p>
            <div class="success">
                <strong>Your account is now verified!</strong>
            </div>
            <p>Your hoshiBmaTchi account has been successfully verified. You now have access to all verified user features.</p>
            <p>Thank you for being part of our community!</p>
        </div>
        <div class="footer">
            <p>© 2025 hoshiBmaTchi. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const templateVerificationRejected = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .info { background: #d1ecf1; border-left: 4px solid #0c5460; padding: 15px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Verification Status Update</h1>
        </div>
        <div class="content">
            <p>Hello %s,</p>
            <div class="info">
                After reviewing your verification request, we're unable to verify your account at this time.
            </div>
            <p>This may be due to incomplete documentation or information that doesn't meet our verification criteria.</p>
            <p>You can submit a new verification request with updated information anytime.</p>
        </div>
        <div class="footer">
            <p>© 2025 hoshiBmaTchi. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const templateGeneric = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .footer { text-align: center; margin-top: 20px; color: #777; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>hoshiBmaTchi</h1>
        </div>
        <div class="content">
            <h2>%s</h2>
            <p>%s</p>
        </div>
        <div class="footer">
            <p>© 2025 hoshiBmaTchi. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`
