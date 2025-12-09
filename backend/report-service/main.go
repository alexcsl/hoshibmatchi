package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"
	"time"

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
	ReporterID   int64 `gorm:"index"`
	PostID       int64 `gorm:"index"`
	Reason       string
	IsResolved   bool  `gorm:"default:false;index"`
	ResolvedByID int64 // Admin User ID
}

type UserReport struct {
	gorm.Model
	ReporterID     int64 `gorm:"index"`
	ReportedUserID int64 `gorm:"index"`
	Reason         string
	IsResolved     bool  `gorm:"default:false;index"`
	ResolvedByID   int64 // Admin User ID
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
	amqpURI := os.Getenv("RABBITMQ_URI")
	if amqpURI == "" {
		amqpURI = "amqp://guest:guest@rabbitmq:5672/" // Default
	}
	amqpConn, err := amqp.Dial(amqpURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer amqpConn.Close()
	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}
	defer amqpCh.Close()

	_, err = amqpCh.QueueDeclare(
		"admin_notification_queue",
		true, false, false, false, nil,
	)
	if err != nil {
		log.Printf("Failed to declare admin_notification_queue: %v", err)
		// Don't kill the service, just log it
	}

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

	// 1. Validate that the post exists and get post details
	postRes, err := s.postClient.GetPost(ctx, &postPb.GetPostRequest{PostId: req.PostId})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "The post you are trying to report does not exist")
		}
		return nil, status.Error(codes.Internal, "Failed to verify post")
	}

	// 2. Check if the user is trying to report their own post
	if postRes.Post.AuthorId == req.ReporterId {
		return nil, status.Error(codes.InvalidArgument, "You cannot report your own post")
	}

	// 3. Create the report
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

	msgBody, _ := json.Marshal(map[string]string{
		"type":      "new_post_report",
		"report_id": strconv.FormatUint(uint64(newReport.ID), 10),
		"post_id":   strconv.FormatInt(newReport.PostID, 10),
	})
	s.amqpCh.PublishWithContext(ctx, "", "admin_notification_queue", false, false, amqp.Publishing{
		ContentType: "application/json", Body: msgBody,
	})

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

	msgBody, _ := json.Marshal(map[string]string{
		"type":      "new_user_report",
		"report_id": strconv.FormatUint(uint64(newReport.ID), 10),
		"user_id":   strconv.FormatInt(newReport.ReportedUserID, 10),
	})
	s.amqpCh.PublishWithContext(ctx, "", "admin_notification_queue", false, false, amqp.Publishing{
		ContentType: "application/json", Body: msgBody,
	})

	return &pb.ReportResponse{Message: "User reported successfully"}, nil
}

// --- Implement Admin-facing RPCs ---

// gormToGrpcPostReport converts a GORM PostReport to its gRPC representation
func gormToGrpcPostReport(report *PostReport) *pb.PostReport {
	return &pb.PostReport{
		Id:         strconv.FormatUint(uint64(report.ID), 10),
		ReporterId: report.ReporterID,
		PostId:     report.PostID,
		Reason:     report.Reason,
		CreatedAt:  report.CreatedAt.Format(time.RFC3339),
	}
}

// gormToGrpcUserReport converts a GORM UserReport to its gRPC representation
func gormToGrpcUserReport(report *UserReport) *pb.UserReport {
	return &pb.UserReport{
		Id:             strconv.FormatUint(uint64(report.ID), 10),
		ReporterId:     report.ReporterID,
		ReportedUserId: report.ReportedUserID,
		Reason:         report.Reason,
		CreatedAt:      report.CreatedAt.Format(time.RFC3339),
	}
}

func (s *server) GetPostReports(ctx context.Context, req *pb.GetReportsRequest) (*pb.GetPostReportsResponse, error) {
	log.Printf("Admin action: GetPostReports request")

	var reports []PostReport
	query := s.db.Order("created_at DESC").Limit(int(req.PageSize)).Offset(int(req.PageOffset))

	if req.UnresolvedOnly {
		query = query.Where("is_resolved = ?", false)
	}

	if err := query.Find(&reports).Error; err != nil {
		log.Printf("Failed to get post reports from db: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve reports")
	}

	var grpcReports []*pb.PostReport
	for _, report := range reports {
		// Fetch reporter username
		reporterUsername := "unknown"
		userRes, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: report.ReporterID})
		if err == nil {
			reporterUsername = userRes.Username
		}

		grpcReports = append(grpcReports, &pb.PostReport{
			Id:               strconv.FormatUint(uint64(report.ID), 10),
			ReporterId:       report.ReporterID,
			PostId:           report.PostID,
			Reason:           report.Reason,
			CreatedAt:        report.CreatedAt.Format(time.RFC3339),
			IsResolved:       report.IsResolved,
			ReporterUsername: reporterUsername,
			ReportedPostId:   report.PostID,
		})
	}

	return &pb.GetPostReportsResponse{Reports: grpcReports}, nil
}

func (s *server) GetUserReports(ctx context.Context, req *pb.GetReportsRequest) (*pb.GetUserReportsResponse, error) {
	log.Printf("Admin action: GetUserReports request")

	var reports []UserReport
	query := s.db.Order("created_at DESC").Limit(int(req.PageSize)).Offset(int(req.PageOffset))

	if req.UnresolvedOnly {
		query = query.Where("is_resolved = ?", false)
	}

	if err := query.Find(&reports).Error; err != nil {
		log.Printf("Failed to get user reports from db: %v", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve reports")
	}

	var grpcReports []*pb.UserReport
	for _, report := range reports {
		// Fetch reporter and reported usernames
		reporterUsername := "unknown"
		reportedUsername := "unknown"

		if userRes, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: report.ReporterID}); err == nil {
			reporterUsername = userRes.Username
		}

		if userRes, err := s.userClient.GetUserData(ctx, &userPb.GetUserDataRequest{UserId: report.ReportedUserID}); err == nil {
			reportedUsername = userRes.Username
		}

		grpcReports = append(grpcReports, &pb.UserReport{
			Id:               strconv.FormatUint(uint64(report.ID), 10),
			ReporterId:       report.ReporterID,
			ReportedUserId:   report.ReportedUserID,
			Reason:           report.Reason,
			CreatedAt:        report.CreatedAt.Format(time.RFC3339),
			IsResolved:       report.IsResolved,
			ReporterUsername: reporterUsername,
			ReportedUsername: reportedUsername,
		})
	}

	return &pb.GetUserReportsResponse{Reports: grpcReports}, nil
}

func (s *server) ResolvePostReport(ctx context.Context, req *pb.ResolveReportRequest) (*pb.ReportResponse, error) {
	log.Printf("Admin action: ResolvePostReport request for report %d with action '%s'", req.ReportId, req.Action)

	// 1. Find the report
	var report PostReport
	if err := s.db.First(&report, req.ReportId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Report not found")
	}

	if report.IsResolved {
		return nil, status.Error(codes.AlreadyExists, "This report has already been resolved")
	}

	// 2. Perform the action
	if req.Action == "ACCEPT" {
		// Call post-service to delete the post
		_, err := s.postClient.DeletePost(ctx, &postPb.DeletePostRequest{
			PostId:      report.PostID,
			AdminUserId: req.AdminUserId,
		})
		if err != nil {
			// Don't fail the report resolution, just log it.
			log.Printf("Failed to delete post %d as part of report %d: %v", report.PostID, req.ReportId, err)
		} else {
			log.Printf("Successfully deleted post %d as part of report %d", report.PostID, req.ReportId)
		}
	}

	// 3. Mark the report as resolved
	if err := s.db.Model(&report).Updates(PostReport{IsResolved: true, ResolvedByID: req.AdminUserId}).Error; err != nil {
		log.Printf("Failed to mark post report %d as resolved: %v", req.ReportId, err)
		return nil, status.Error(codes.Internal, "Failed to resolve report")
	}

	return &pb.ReportResponse{Message: "Post report resolved successfully"}, nil
}

func (s *server) ResolveUserReport(ctx context.Context, req *pb.ResolveReportRequest) (*pb.ReportResponse, error) {
	log.Printf("Admin action: ResolveUserReport request for report %d with action '%s'", req.ReportId, req.Action)

	// 1. Find the report
	var report UserReport
	if err := s.db.First(&report, req.ReportId).Error; err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "Report not found")
	}

	if report.IsResolved {
		return nil, status.Error(codes.AlreadyExists, "This report has already been resolved")
	}

	// 2. Perform the action
	if req.Action == "ACCEPT" {
		// Call user-service to ban the user
		_, err := s.userClient.BanUser(ctx, &userPb.BanUserRequest{
			AdminUserId: req.AdminUserId,
			UserToBanId: report.ReportedUserID,
		})
		if err != nil {
			log.Printf("Failed to ban user %d as part of report %d: %v", report.ReportedUserID, req.ReportId, err)
		} else {
			log.Printf("Successfully banned user %d as part of report %d", report.ReportedUserID, req.ReportId)
		}
	}

	// 3. Mark the report as resolved
	if err := s.db.Model(&report).Updates(UserReport{IsResolved: true, ResolvedByID: req.AdminUserId}).Error; err != nil {
		log.Printf("Failed to mark user report %d as resolved: %v", req.ReportId, err)
		return nil, status.Error(codes.Internal, "Failed to resolve report")
	}

	return &pb.ReportResponse{Message: "User report resolved successfully"}, nil
}
