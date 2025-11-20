# Phase 3: User Profile, Search & Direct Messages

## Overview
Implement user profiles, search functionality, and real-time messaging system.

## Components to Build

### 1. User Profile Page (`Profile.vue`)
**Features:**
- Profile header with avatar, bio, stats (posts/followers/following)
- Edit profile button (own profile only)
- Post grid display (3 columns)
- Tabs: Posts, Saved, Tagged
- Follow/Unfollow button (other profiles)
- Settings dropdown

**Backend:**
- `GET /users/:username` - Get user profile
- `GET /users/:id/posts` - Get user's posts
- `POST /users/:id/follow` - Follow user
- `DELETE /users/:id/follow` - Unfollow user
- `GET /users/:id/followers` - Get followers list
- `GET /users/:id/following` - Get following list

### 2. Profile Edit Modal
**Features:**
- Change profile picture
- Edit name, username, bio, website
- Change email, phone
- Privacy settings
- Account management (change password, 2FA)

**Backend:**
- `PUT /users/profile` - Update profile
- `PUT /users/profile/picture` - Update profile picture

### 3. Search Functionality (`SearchOverlay.vue` enhancement)
**Features:**
- Real-time search as you type
- Search users, hashtags, locations
- Recent searches (stored locally)
- Clear search history
- Navigate to profiles/hashtags

**Backend:**
- `GET /search?q=query&type=users` - Search users
- `GET /search?q=query&type=hashtags` - Search hashtags
- `GET /search?q=query&type=all` - Search all

### 4. Direct Messages (`Messages.vue`)
**Features:**
- Conversation list (left sidebar)
- Active conversation (right panel)
- Send text messages
- Send images via upload
- Real-time message updates (WebSocket/polling)
- Read receipts
- Typing indicators
- Message reactions
- Delete messages

**Backend (message-service already exists):**
- `GET /messages/conversations` - Get user's conversations
- `GET /messages/:conversationId` - Get messages in conversation
- `POST /messages` - Send new message
- `PUT /messages/:id/read` - Mark message as read
- `DELETE /messages/:id` - Delete message
- WebSocket: `/ws/messages` - Real-time updates

### 5. Story Functionality
**Features:**
- Story ring on profile picture
- Story viewer (fullscreen swipeable)
- Create story (photo/video with text overlay)
- View who seen your story
- 24h auto-deletion

**Backend (story-service):**
- `POST /stories` - Create story
- `GET /stories/feed` - Get stories from following
- `GET /stories/:id/viewers` - Get story viewers
- `POST /stories/:id/view` - Mark story as viewed

## Implementation Order

### Week 1: Profile & Search
1. **Day 1-2**: User Profile Page
   - Profile header component
   - Post grid display
   - Follow/unfollow functionality
   - Backend: Profile endpoints in user-service

2. **Day 3-4**: Profile Editing
   - Edit profile modal
   - Profile picture upload
   - Settings page skeleton
   - Backend: Update profile endpoints

3. **Day 5**: Search Enhancement
   - Search API integration
   - Search results display
   - Recent searches feature

### Week 2: Direct Messages
4. **Day 6-7**: Message UI
   - Conversation list component
   - Message thread component
   - Message composer

5. **Day 8-9**: Message Functionality
   - Send/receive messages
   - Image upload in messages
   - Backend: Message CRUD operations

6. **Day 10**: Real-time Updates
   - Implement WebSocket/SSE for messages
   - Typing indicators
   - Read receipts

### Week 3: Stories (Optional Enhancement)
7. **Day 11-12**: Story Creation
   - Story camera/upload
   - Text overlays
   - Backend: Story storage

8. **Day 13-14**: Story Viewing
   - Story viewer component
   - Story ring indicators
   - View tracking

## API Endpoints Summary

### User Service (user-service/main.go)
```go
GET    /users/:username        // Get profile by username
GET    /users/:id/posts        // Get user posts
GET    /users/:id/followers    // Get followers
GET    /users/:id/following    // Get following
POST   /users/:id/follow       // Follow user
DELETE /users/:id/follow       // Unfollow user
PUT    /users/profile          // Update profile
PUT    /users/profile/picture  // Update avatar
```

### Search (api-gateway)
```go
GET /search?q=query&type=users     // Search users
GET /search?q=query&type=hashtags  // Search hashtags
```

### Message Service (message-service/main.go)
```go
GET    /messages/conversations      // List conversations
GET    /messages/:conversationId    // Get messages
POST   /messages                    // Send message
PUT    /messages/:id/read           // Mark as read
DELETE /messages/:id                // Delete message
WS     /ws/messages                 // WebSocket connection
```

### Story Service (story-service/main.go)
```go
GET    /stories/feed        // Get stories from following
POST   /stories             // Create story
GET    /stories/:id         // Get story
POST   /stories/:id/view    // Mark as viewed
GET    /stories/:id/viewers // Get viewers
```

## Database Schema Updates

### users table (already exists)
```sql
-- Add fields if missing:
ALTER TABLE users ADD COLUMN IF NOT EXISTS website VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS followers_count INT DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS following_count INT DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS posts_count INT DEFAULT 0;
```

### messages table (message-service)
```sql
CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL,
    sender_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    media_url VARCHAR(500),
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS conversations (
    id SERIAL PRIMARY KEY,
    participant_1_id BIGINT NOT NULL,
    participant_2_id BIGINT NOT NULL,
    last_message_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(participant_1_id, participant_2_id)
);
```

### stories table (story-service)
```sql
CREATE TABLE IF NOT EXISTS stories (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    media_url VARCHAR(500) NOT NULL,
    media_type VARCHAR(20),
    caption TEXT,
    views_count INT DEFAULT 0,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS story_views (
    id SERIAL PRIMARY KEY,
    story_id BIGINT NOT NULL,
    viewer_id BIGINT NOT NULL,
    viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(story_id, viewer_id)
);
```

## Testing Checklist
- [ ] View own profile
- [ ] View other user's profile
- [ ] Follow/unfollow users
- [ ] Edit profile details
- [ ] Upload profile picture
- [ ] Search for users
- [ ] Search for hashtags
- [ ] Send direct message
- [ ] Receive direct message
- [ ] Real-time message updates
- [ ] Create story
- [ ] View stories
- [ ] Story expiration

## Next Steps
1. Start with Profile page implementation
2. Test follow/unfollow flow
3. Implement search
4. Build messaging system
5. Add stories (optional)
