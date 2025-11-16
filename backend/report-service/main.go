package main

import (
	"context"
	"log"
	"net"
	// "time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	amqp "github.com/rabbitmq/amqp091-go"

	pb "github.com/hoshibmatchi/report-service/proto"
	// We need to connect to other services to validate IDs
	postPb "github.com/hoshibmatchi/post-service/proto"
	userPb "github.com/hoshibmatchi/user-service/proto"
)

// --- GORM Models ---

type PostReport struct {
	gorm.Model
	ReporterID  int64  `gorm:"index"`
	PostID      int64  `gorm:"index"`
	Reason      string
	IsResolved  bool   `gorm:"default:false;index"`
	ResolvedByID int64  // Admin User ID
}

type UserReport struct {
	gorm.Model
	ReporterID     int64  `gorm:"index"`
	ReportedUserID int64  `gorm:"index"`
	Reason         string
	IsResolved     bool   `gorm:"default:false;index"`
	ResolvedByID    int64  // Admin User ID
}

type server struct {
	pb.UnimplementedReportServiceServer
	db         *gorm.DB
	amqpCh     *amqp.Channel
	userClient userPb.UserServiceClient
	postClient postPb.PostServiceClient
}

func main() {
	// --- Step 1: Connect to report-db ---
	dsn := "host=report-db user=admin password=password dbname=report_service_db port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to report-db: %v", err)
	}
	db.AutoMigrate(&PostReport{}, &UserReport{})
	log.Println("Report-service successfully connected to report-db")

	// --- Step 2: Connect to RabbitMQ ---
	amqpConn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer amqpConn.Close()
	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
	defer amqpCh.Close()
	// TODO: Declare admin notification queue

	// --- Step 3: Connect to other gRPC Services ---
	userConn, err := grpc.Dial("user-service:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer userConn.Close()
	userClient := userPb.NewUserServiceClient(userConn)
	log.Println("Report-service successfully connected to user-service")

	postConn, err := grpc.Dial("post-service:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to post-service: %v", err)
	}
	defer postConn.Close()
	postClient := postPb.NewPostServiceClient(postConn)
	log.Println("Report-service successfully connected to post-service")

	// --- Step 4: Start this gRPC Server ---
	lis, err := net.Listen("tcp", ":9006") // Port 9006
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterReportServiceServer(s, &server{
		db:         db,
		amqpCh:     amqpCh,
		userClient: userClient,
		postClient: postClient,
	})

	log.Println("Report service listening on port 9006...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// --- Implement User-facing RPCs ---

func (s *server) ReportPost(ctx context.Context, req *pb.ReportPostRequest) (*pb.ReportResponse, error) {
	log.Printf("ReportPost request from user %d for post %d", req.ReporterId, req.PostId)

	// 1. Validate that the post exists
	_, err := s.postClient.GetPost(ctx, &postPb.GetPostRequest{PostId: req.PostId})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "The post you are trying to report does not exist")
		}
		return nil, status.Error(codes.Internal, "Failed to verify post")
	}

	// 2. Create the report
	newReport := PostReport{
		ReporterID: req.ReporterId,
		PostID:     req.PostId,
		Reason:     req.Reason,
		IsResolved: false,
	}
	if err := s.db.Create(&newReport).Error; err != nil {
		log.Printf("Failed to save post report: %v", err)
		return nil, status.Error(codes.Internal, "Failed to submit report")
	}

	// TODO: Publish a "report.created" message to RabbitMQ for admin notifications

	return &pb.ReportResponse{Message: "Post reported successfully"}, nil
}

func (s *server) ReportUser(ctx context.Context, req *pb.ReportUserRequest) (*pb.ReportResponse, error) {
	log.Printf("ReportUser request from user %d for user %d", req.ReporterId, req.ReportedUserId)

	// 1. Validate that the user exists
	_, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: req.ReportedUserId})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "The user you are trying to report does not exist")
		}
		return nil, status.Error(codes.Internal, "Failed to verify user")
	}

	// 2. Create the report
	newReport := UserReport{
		ReporterID:     req.ReporterId,
		ReportedUserID: req.ReportedUserId,
		Reason:         req.Reason,
		IsResolved:     false,
	}
	if err := s.db.Create(&newReport).Error; err != nil {
		log.Printf("Failed to save user report: %v", err)
		return nil, status.Error(codes.Internal, "Failed to submit report")
	}

	// TODO: Publish a "report.created" message to RabbitMQ for admin notifications

	return &pb.ReportResponse{Message: "User reported successfully"}, nil
}