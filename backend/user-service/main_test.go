package main

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Run migrations
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Follow{})
	db.AutoMigrate(&Block{})
	db.AutoMigrate(&VerificationRequest{})
	db.AutoMigrate(&CloseFriend{})
	db.AutoMigrate(&HiddenStoryUser{})
	db.AutoMigrate(&NotificationSetting{})

	return db, nil
}

func TestUserCreation(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	user := User{
		Name:        "Test User",
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "hashedpassword",
		DateOfBirth: time.Now().AddDate(-20, 0, 0),
		Gender:      "male",
		IsActive:    true,
		Role:        "user",
	}

	result := db.Create(&user)
	if result.Error != nil {
		t.Fatalf("Failed to create user: %v", result.Error)
	}

	// Assertions
	if user.ID == 0 {
		t.Error("Expected user ID to be set")
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	// Verify user exists in database
	var foundUser User
	db.First(&foundUser, user.ID)

	if foundUser.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", foundUser.Email)
	}
}

func TestFollowRelationship(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create two users
	user1 := User{
		Name:        "User One",
		Username:    "user1",
		Email:       "user1@example.com",
		Password:    "hashedpassword",
		DateOfBirth: time.Now().AddDate(-20, 0, 0),
		Gender:      "male",
		IsActive:    true,
	}
	user2 := User{
		Name:        "User Two",
		Username:    "user2",
		Email:       "user2@example.com",
		Password:    "hashedpassword",
		DateOfBirth: time.Now().AddDate(-20, 0, 0),
		Gender:      "female",
		IsActive:    true,
	}

	db.Create(&user1)
	db.Create(&user2)

	// Create follow relationship
	follow := Follow{
		FollowerID:  int64(user1.ID),
		FollowingID: int64(user2.ID),
		Status:      "approved",
	}

	result := db.Create(&follow)
	if result.Error != nil {
		t.Fatalf("Failed to create follow relationship: %v", result.Error)
	}

	// Verify follow relationship
	var foundFollow Follow
	db.Where("follower_id = ? AND following_id = ?", user1.ID, user2.ID).First(&foundFollow)

	if foundFollow.FollowerID != int64(user1.ID) {
		t.Errorf("Expected follower_id %d, got %d", user1.ID, foundFollow.FollowerID)
	}

	if foundFollow.FollowingID != int64(user2.ID) {
		t.Errorf("Expected following_id %d, got %d", user2.ID, foundFollow.FollowingID)
	}

	if foundFollow.Status != "approved" {
		t.Errorf("Expected status 'approved', got '%s'", foundFollow.Status)
	}
}

func TestBlockRelationship(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create two users
	user1 := User{
		Name:        "User One",
		Username:    "blocker",
		Email:       "blocker@example.com",
		Password:    "hashedpassword",
		DateOfBirth: time.Now().AddDate(-20, 0, 0),
		Gender:      "male",
		IsActive:    true,
	}
	user2 := User{
		Name:        "User Two",
		Username:    "blocked",
		Email:       "blocked@example.com",
		Password:    "hashedpassword",
		DateOfBirth: time.Now().AddDate(-20, 0, 0),
		Gender:      "female",
		IsActive:    true,
	}

	db.Create(&user1)
	db.Create(&user2)

	// Create block relationship
	block := Block{
		BlockerID: int64(user1.ID),
		BlockedID: int64(user2.ID),
	}

	result := db.Create(&block)
	if result.Error != nil {
		t.Fatalf("Failed to create block relationship: %v", result.Error)
	}

	// Verify block relationship
	var foundBlock Block
	db.Where("blocker_id = ? AND blocked_id = ?", user1.ID, user2.ID).First(&foundBlock)

	if foundBlock.BlockerID != int64(user1.ID) {
		t.Errorf("Expected blocker_id %d, got %d", user1.ID, foundBlock.BlockerID)
	}

	if foundBlock.BlockedID != int64(user2.ID) {
		t.Errorf("Expected blocked_id %d, got %d", user2.ID, foundBlock.BlockedID)
	}
}

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"valid@example.com", true},
		{"another.valid@test.co.uk", true},
		{"invalid.email", false},
		{"@example.com", false},
		{"invalid@", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			isValid := emailRegex.MatchString(tt.email)
			if isValid != tt.expected {
				t.Errorf("For email '%s', expected %v, got %v", tt.email, tt.expected, isValid)
			}
		})
	}
}

func TestGenerateOTP(t *testing.T) {
	otp := generateOtp()

	// Check length
	if len(otp) != 6 {
		t.Errorf("Expected OTP length 6, got %d", len(otp))
	}

	// Check all characters are digits
	for _, char := range otp {
		if char < '0' || char > '9' {
			t.Errorf("Expected all digits in OTP, got '%s'", otp)
			break
		}
	}
}

func TestJaroWinklerDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		minScore float64
	}{
		{"john", "john", 1.0},
		{"john", "johnny", 0.8},
		{"martha", "marhta", 0.9},
		{"completely", "different", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_vs_"+tt.s2, func(t *testing.T) {
			score := jaroWinklerDistance(tt.s1, tt.s2)
			if tt.s1 == tt.s2 && score != tt.minScore {
				t.Errorf("Expected exact match score %.2f, got %.2f", tt.minScore, score)
			} else if tt.s1 != tt.s2 && score < tt.minScore {
				t.Errorf("Expected minimum score %.2f, got %.2f", tt.minScore, score)
			}
		})
	}
}

func TestCloseFriendRelationship(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create two users
	user1 := User{
		Name:        "User One",
		Username:    "user1",
		Email:       "user1@example.com",
		Password:    "hashedpassword",
		DateOfBirth: time.Now().AddDate(-20, 0, 0),
		Gender:      "male",
		IsActive:    true,
	}
	user2 := User{
		Name:        "User Two",
		Username:    "user2",
		Email:       "user2@example.com",
		Password:    "hashedpassword",
		DateOfBirth: time.Now().AddDate(-20, 0, 0),
		Gender:      "female",
		IsActive:    true,
	}

	db.Create(&user1)
	db.Create(&user2)

	// Create close friend relationship
	closeFriend := CloseFriend{
		UserID:   int64(user1.ID),
		FriendID: int64(user2.ID),
	}

	result := db.Create(&closeFriend)
	if result.Error != nil {
		t.Fatalf("Failed to create close friend relationship: %v", result.Error)
	}

	// Verify relationship
	var found CloseFriend
	db.Where("user_id = ? AND friend_id = ?", user1.ID, user2.ID).First(&found)

	if found.UserID != int64(user1.ID) {
		t.Errorf("Expected user_id %d, got %d", user1.ID, found.UserID)
	}

	if found.FriendID != int64(user2.ID) {
		t.Errorf("Expected friend_id %d, got %d", user2.ID, found.FriendID)
	}
}
