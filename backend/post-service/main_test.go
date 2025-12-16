package main

import (
	"testing"

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
	db.AutoMigrate(&Post{})
	db.AutoMigrate(&PostLike{})
	db.AutoMigrate(&Comment{})
	db.AutoMigrate(&CommentLike{})
	db.AutoMigrate(&Collection{})
	db.AutoMigrate(&SavedPost{})
	db.AutoMigrate(&PostCollaborator{})
	db.AutoMigrate(&SharedPost{})

	return db, nil
}

func TestPostCreation(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	post := Post{
		AuthorID:         1,
		Caption:          "Test post caption",
		IsReel:           false,
		CommentsDisabled: false,
		AuthorUsername:   "testuser",
		AuthorProfileURL: "https://example.com/profile.jpg",
		AuthorIsVerified: true,
		LikeCount:        0,
		CommentCount:     0,
	}

	result := db.Create(&post)
	if result.Error != nil {
		t.Fatalf("Failed to create post: %v", result.Error)
	}

	// Assertions
	if post.ID == 0 {
		t.Error("Expected post ID to be set")
	}

	if post.Caption != "Test post caption" {
		t.Errorf("Expected caption 'Test post caption', got '%s'", post.Caption)
	}

	if post.AuthorID != 1 {
		t.Errorf("Expected author ID 1, got %d", post.AuthorID)
	}
}

func TestPostLike(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create a post
	post := Post{
		AuthorID:       1,
		Caption:        "Test post",
		AuthorUsername: "testuser",
	}
	db.Create(&post)

	// Create a like
	like := PostLike{
		UserID: 2,
		PostID: int64(post.ID),
	}

	result := db.Create(&like)
	if result.Error != nil {
		t.Fatalf("Failed to create like: %v", result.Error)
	}

	// Verify like exists
	var foundLike PostLike
	db.Where("user_id = ? AND post_id = ?", 2, post.ID).First(&foundLike)

	if foundLike.UserID != 2 {
		t.Errorf("Expected user_id 2, got %d", foundLike.UserID)
	}

	if foundLike.PostID != int64(post.ID) {
		t.Errorf("Expected post_id %d, got %d", post.ID, foundLike.PostID)
	}
}

func TestCommentCreation(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create a post
	post := Post{
		AuthorID:       1,
		Caption:        "Test post",
		AuthorUsername: "testuser",
	}
	db.Create(&post)

	// Create a comment
	comment := Comment{
		UserID:           2,
		PostID:           int64(post.ID),
		Content:          "Great post!",
		AuthorUsername:   "commenter",
		AuthorProfileURL: "https://example.com/commenter.jpg",
		AuthorIsVerified: false,
	}

	result := db.Create(&comment)
	if result.Error != nil {
		t.Fatalf("Failed to create comment: %v", result.Error)
	}

	// Assertions
	if comment.ID == 0 {
		t.Error("Expected comment ID to be set")
	}

	if comment.Content != "Great post!" {
		t.Errorf("Expected content 'Great post!', got '%s'", comment.Content)
	}

	if comment.PostID != int64(post.ID) {
		t.Errorf("Expected post_id %d, got %d", post.ID, comment.PostID)
	}
}

func TestCollectionCreation(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	collection := Collection{
		UserID:    1,
		Name:      "My Favorites",
		IsDefault: false,
	}

	result := db.Create(&collection)
	if result.Error != nil {
		t.Fatalf("Failed to create collection: %v", result.Error)
	}

	// Assertions
	if collection.ID == 0 {
		t.Error("Expected collection ID to be set")
	}

	if collection.Name != "My Favorites" {
		t.Errorf("Expected name 'My Favorites', got '%s'", collection.Name)
	}

	if collection.UserID != 1 {
		t.Errorf("Expected user_id 1, got %d", collection.UserID)
	}
}

func TestSavedPost(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create a collection
	collection := Collection{
		UserID: 1,
		Name:   "Saved Posts",
	}
	db.Create(&collection)

	// Create a post
	post := Post{
		AuthorID:       1,
		Caption:        "Test post",
		AuthorUsername: "testuser",
	}
	db.Create(&post)

	// Save post to collection
	savedPost := SavedPost{
		CollectionID: collection.ID,
		PostID:       post.ID,
	}

	result := db.Create(&savedPost)
	if result.Error != nil {
		t.Fatalf("Failed to save post: %v", result.Error)
	}

	// Verify saved post
	var found SavedPost
	db.Where("collection_id = ? AND post_id = ?", collection.ID, post.ID).First(&found)

	if found.CollectionID != collection.ID {
		t.Errorf("Expected collection_id %d, got %d", collection.ID, found.CollectionID)
	}

	if found.PostID != post.ID {
		t.Errorf("Expected post_id %d, got %d", post.ID, found.PostID)
	}
}

func TestHashtagRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello #world", []string{"#world"}},
		{"#first #second #third", []string{"#first", "#second", "#third"}},
		{"No hashtags here", []string{}},
		{"#123numbers", []string{"#123numbers"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := hashtagRegex.FindAllString(tt.input, -1)

			if len(matches) != len(tt.expected) {
				t.Errorf("Expected %d matches, got %d", len(tt.expected), len(matches))
				return
			}

			for i, match := range matches {
				if match != tt.expected[i] {
					t.Errorf("Expected match '%s', got '%s'", tt.expected[i], match)
				}
			}
		})
	}
}

func TestSharedPost(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create original post
	post := Post{
		AuthorID:       1,
		Caption:        "Original post",
		AuthorUsername: "author",
	}
	db.Create(&post)

	// Share the post
	sharedPost := SharedPost{
		UserID:         2,
		OriginalPostID: int64(post.ID),
		Caption:        "Check this out!",
	}

	result := db.Create(&sharedPost)
	if result.Error != nil {
		t.Fatalf("Failed to create shared post: %v", result.Error)
	}

	// Assertions
	if sharedPost.ID == 0 {
		t.Error("Expected shared post ID to be set")
	}

	if sharedPost.OriginalPostID != int64(post.ID) {
		t.Errorf("Expected original_post_id %d, got %d", post.ID, sharedPost.OriginalPostID)
	}

	if sharedPost.Caption != "Check this out!" {
		t.Errorf("Expected caption 'Check this out!', got '%s'", sharedPost.Caption)
	}
}

func TestCommentLike(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// Create a post
	post := Post{
		AuthorID:       1,
		Caption:        "Test post",
		AuthorUsername: "testuser",
	}
	db.Create(&post)

	// Create a comment
	comment := Comment{
		UserID:         1,
		PostID:         int64(post.ID),
		Content:        "Test comment",
		AuthorUsername: "testuser",
	}
	db.Create(&comment)

	// Like the comment
	commentLike := CommentLike{
		UserID:    2,
		CommentID: int64(comment.ID),
	}

	result := db.Create(&commentLike)
	if result.Error != nil {
		t.Fatalf("Failed to create comment like: %v", result.Error)
	}

	// Verify like
	var found CommentLike
	db.Where("user_id = ? AND comment_id = ?", 2, comment.ID).First(&found)

	if found.UserID != 2 {
		t.Errorf("Expected user_id 2, got %d", found.UserID)
	}

	if found.CommentID != int64(comment.ID) {
		t.Errorf("Expected comment_id %d, got %d", comment.ID, found.CommentID)
	}
}
