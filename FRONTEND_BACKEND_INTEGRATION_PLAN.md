# Frontend-Backend Integration Plan
## HoshiMatchi - Instagram Clone

**Date Created:** November 19, 2025  
**Status:** Draft v1.0  
**Tech Stack:** Vue 3 + Vite + TypeScript + SCSS | Go + gRPC + PostgreSQL + Redis + RabbitMQ

---

## 📋 Table of Contents
1. [Overview](#overview)
2. [Current State Analysis](#current-state-analysis)
3. [Integration Architecture](#integration-architecture)
4. [Phase-by-Phase Implementation](#phase-by-phase-implementation)
5. [API Endpoint Mapping](#api-endpoint-mapping)
6. [State Management Strategy](#state-management-strategy)
7. [Real-time Features](#real-time-features)
8. [Testing Strategy](#testing-strategy)

---

## 🎯 Overview

### Project Goal
Connect the fully functional backend (tested in Postman) with the Vue frontend, implementing all Instagram-like features while maintaining the existing backend structure.

### Key Principles
- ✅ **Backend is working** - minimal changes to backend
- 🎨 **Frontend adapts** - all integration work happens in Vue
- 📦 **Incremental approach** - phase-by-phase implementation
- 🧪 **Test as we go** - validate each phase before moving forward

---

## 📊 Current State Analysis

### Backend Structure
```
API Gateway (Port 8000)
├── Auth Routes (/auth/*)
│   ├── POST /auth/register
│   ├── POST /auth/login
│   ├── POST /auth/send-otp
│   ├── POST /auth/verify-otp
│   ├── POST /auth/login/verify-2fa
│   ├── POST /auth/password-reset/request
│   ├── POST /auth/password-reset/submit
│   └── POST /auth/google/callback
│
├── Protected Routes (JWT Required)
│   ├── Feed & Content
│   │   ├── GET /feed/home
│   │   ├── GET /feed/explore
│   │   └── GET /feed/reels
│   │
│   ├── Posts
│   │   ├── POST /posts
│   │   ├── POST /posts/:id/like
│   │   ├── DELETE /posts/:id/like
│   │   └── POST /posts/:id/summarize (AI)
│   │
│   ├── Stories
│   │   ├── POST /stories
│   │   ├── POST /stories/:id/like
│   │   └── DELETE /stories/:id/like
│   │
│   ├── Comments
│   │   ├── POST /comments
│   │   └── DELETE /comments/:id
│   │
│   ├── Users & Profile
│   │   ├── GET /users/:username
│   │   ├── GET /users/:username/posts
│   │   ├── GET /users/:username/reels
│   │   ├── POST /users/:id/follow
│   │   ├── DELETE /users/:id/follow
│   │   ├── POST /users/:id/block
│   │   ├── DELETE /users/:id/block
│   │   ├── PUT /profile/edit
│   │   ├── PUT /settings/privacy
│   │   └── POST /profile/verify
│   │
│   ├── Collections (Saved Posts)
│   │   ├── POST /collections
│   │   ├── GET /collections
│   │   ├── GET /collections/:id
│   │   ├── POST /collections/:id/posts
│   │   ├── DELETE /collections/:id/posts/:post_id
│   │   ├── DELETE /collections/:id
│   │   └── PUT /collections/:id
│   │
│   ├── Messages
│   │   ├── POST /conversations
│   │   ├── GET /conversations
│   │   ├── GET /conversations/:id/messages
│   │   ├── POST /conversations/:id/messages
│   │   ├── DELETE /messages/:id (unsend)
│   │   ├── DELETE /conversations/:id (soft delete)
│   │   └── GET /conversations/:id/video_token
│   │
│   ├── Search
│   │   ├── GET /search/users?q=
│   │   ├── GET /search/hashtags/:name
│   │   └── GET /trending/hashtags
│   │
│   ├── Reports
│   │   ├── POST /reports/post
│   │   └── POST /reports/user
│   │
│   └── Media Upload
│       └── GET /media/upload-url
│
└── Admin Routes (/admin/*)
    ├── POST /admin/users/:id/ban
    ├── POST /admin/users/:id/unban
    ├── GET /admin/reports/posts
    ├── GET /admin/reports/users
    ├── POST /admin/reports/posts/:id/resolve
    ├── POST /admin/reports/users/:id/resolve
    ├── POST /admin/newsletters
    ├── GET /admin/verifications
    └── POST /admin/verifications/:id/resolve
```

### Backend Services
- **user-service** (Port 9000) - Authentication, profiles, follows, blocks
- **post-service** (Port 9001) - Posts, likes, comments, collections
- **story-service** (Port 9002) - Stories and story interactions
- **message-service** (Port 9003 gRPC, 9004 WebSocket) - Chat, video calls
- **media-service** (Port 9005) - MinIO uploads, presigned URLs
- **report-service** (Port 9006) - User/post reports
- **hashtag-service** (Port 9007) - Hashtag tracking and trending
- **ai-service** (Port 9008) - Caption summarization
- **notification-service** - Push notifications via RabbitMQ
- **email-service** - Emails via RabbitMQ
- **worker-service** - Background jobs (video transcoding, etc.)

### Frontend Structure
```
frontend/hoshi-vue/src/
├── main.ts
├── App.vue
├── router/
│   └── index.ts
├── services/
│   └── api.ts (axios instance, auth helpers)
├── layouts/
│   └── MainLayout.vue
├── pages/
│   ├── Login.vue ✅
│   ├── SignUp.vue ✅
│   ├── LoginOTP.vue ✅
│   ├── ForgotPassword.vue ✅
│   ├── ResetPassword.vue ✅
│   ├── VerifyOTP.vue ✅
│   ├── Feed.vue (placeholder)
│   ├── Explore.vue (placeholder)
│   ├── Reels.vue (placeholder)
│   ├── Messages.vue (placeholder)
│   ├── Profile.vue (placeholder)
│   ├── Settings.vue (placeholder)
│   └── Archive.vue (placeholder)
├── components/
│   ├── Sidebar.vue ✅
│   ├── MiniMessage.vue ✅ (now draggable!)
│   ├── StoryViewer.vue ✅
│   ├── SearchOverlay.vue (placeholder)
│   ├── NotificationOverlay.vue (placeholder)
│   ├── CreatePostOverlay.vue (placeholder)
│   └── PostDetailsOverlay.vue (placeholder)
└── styles/
```

### What's Already Working
✅ Authentication UI (Login, SignUp, OTP)  
✅ Password Reset Flow UI  
✅ Routing & Navigation Guards  
✅ JWT Token Management  
✅ Axios Interceptors  
✅ Basic Layout Structure  
✅ Draggable Mini Message Component

### What Needs Integration
❌ Feed loading (home, explore, reels)  
❌ Post creation & interactions  
❌ Story creation & viewing  
❌ Comments system  
❌ User profiles  
❌ Follow/unfollow functionality  
❌ Search functionality  
❌ Messaging (WebSocket)  
❌ Notifications (WebSocket)  
❌ Collections/Saved posts  
❌ Media uploads  
❌ Real-time updates  

---

## 🏗️ Integration Architecture

### HTTP Communication Flow
```
Vue Component → api.ts → Axios → API Gateway (Port 8000) → gRPC Services
     ↓                                                             ↓
 Display Data ←──────────────── JSON Response ←────────────── Database
```

### WebSocket Communication Flow
```
Vue Component → WebSocket Client → Message Service (Port 9004)
     ↓                                      ↓
 Display Message ←────────── Redis Pub/Sub ←──────── New Messages
```

### Media Upload Flow
```
1. GET /media/upload-url → Presigned URL
2. POST (file) to MinIO directly
3. Save final_media_url in post/story data
```

### State Management Strategy
We'll use **Pinia** (Vue's official state management) for:
- User authentication state
- Current user profile
- Feed cache
- Unread message counts
- Notifications

```typescript
// stores/auth.ts
export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    token: null,
    isAuthenticated: false
  })
})

// stores/feed.ts
export const useFeedStore = defineStore('feed', {
  state: () => ({
    homeFeed: [],
    exploreFeed: [],
    reelsFeed: [],
    page: 1
  })
})

// stores/messages.ts
export const useMessageStore = defineStore('messages', {
  state: () => ({
    conversations: [],
    unreadCount: 0,
    activeConversation: null,
    ws: null
  })
})
```

---

## 🚀 Phase-by-Phase Implementation

### **PHASE 1: Authentication & Profile Foundation** (Week 1)
**Goal:** Complete authentication flow and user profile display

#### 1.1 Install Pinia State Management
```bash
npm install pinia
```

#### 1.2 Create Auth Store
**File:** `src/stores/auth.ts`
```typescript
import { defineStore } from 'pinia'
import { authAPI, userAPI } from '@/services/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: JSON.parse(localStorage.getItem('user') || 'null'),
    token: localStorage.getItem('jwt_token'),
    isAuthenticated: !!localStorage.getItem('jwt_token')
  }),
  
  actions: {
    async login(credentials) {
      const response = await authAPI.login(credentials)
      if (response.is_2fa_required) {
        return { requires2FA: true, email: credentials.email_or_username }
      }
      this.setAuth(response.access_token, response)
    },
    
    setAuth(token, user) {
      this.token = token
      this.user = user
      this.isAuthenticated = true
      localStorage.setItem('jwt_token', token)
      localStorage.setItem('user', JSON.stringify(user))
    },
    
    logout() {
      this.token = null
      this.user = null
      this.isAuthenticated = false
      localStorage.clear()
    }
  }
})
```

#### 1.3 Update API Service
**File:** `src/services/api.ts` - Add all backend endpoints

**Changes:**
- ✅ Add postAPI for posts
- ✅ Add storyAPI for stories
- ✅ Add commentAPI for comments
- ✅ Add feedAPI for feeds
- ✅ Add searchAPI for search
- ✅ Add collectionAPI for saved posts

#### 1.4 Update Login Pages
**Files:** `src/pages/Login.vue`, `src/pages/SignUp.vue`
- Connect to auth store
- Handle loading states
- Display error messages
- Redirect on success

#### 1.5 Create User Profile Page
**File:** `src/pages/Profile.vue`
- Fetch user data via `/users/:username`
- Display posts grid (3 columns)
- Show follower/following counts
- Follow/unfollow button
- Edit profile button (own profile)

**Deliverables:**
- ✅ Working login/logout
- ✅ Token persistence
- ✅ Profile page displays user data
- ✅ Follow/unfollow works

---

### **PHASE 2: Feed & Posts** (Week 2)
**Goal:** Display feeds and implement post interactions

#### 2.1 Create Feed Store
**File:** `src/stores/feed.ts`

#### 2.2 Implement Feed Page
**File:** `src/pages/Feed.vue`
```typescript
// Fetch home feed on mount
onMounted(async () => {
  const response = await feedAPI.getHomeFeed()
  posts.value = response.posts
})

// Infinite scroll
const loadMore = async () => {
  page.value++
  const response = await feedAPI.getHomeFeed(page.value)
  posts.value.push(...response.posts)
}
```

#### 2.3 Create Post Component
**File:** `src/components/PostCard.vue`
- Display image/video
- Like button (optimistic UI)
- Comment count
- Share button
- Caption with hashtags
- Author info
- Timestamp

#### 2.4 Implement Post Creation
**File:** `src/components/CreatePostOverlay.vue`
- Media upload (drag & drop)
- Caption input (with hashtag detection)
- Collaborator selection
- Reel toggle
- Call media upload API
- Create post

#### 2.5 Post Interactions
- Like/unlike (POST/DELETE /posts/:id/like)
- Open comments overlay
- Save to collection

**Deliverables:**
- ✅ Home feed displays posts
- ✅ Can create new posts
- ✅ Like/unlike works
- ✅ Media uploads work
- ✅ Hashtags clickable

---

### **PHASE 3: Comments & Stories** (Week 3)
**Goal:** Interactive comments and story viewing

#### 3.1 Comments System
**File:** `src/components/PostDetailsOverlay.vue`
- Display comments thread
- Reply to comments (nested)
- Like comments
- Delete own comments
- Real-time comment updates

#### 3.2 Story Features
**File:** `src/components/StoryViewer.vue` (already exists!)
- Fetch stories from API
- Auto-advance timer
- Like story
- Progress bar
- Swipe navigation

#### 3.3 Create Story
**File:** `src/components/CreateStoryOverlay.vue`
- Upload photo/video
- Post story
- Preview

**Deliverables:**
- ✅ Comments display and work
- ✅ Stories viewable
- ✅ Can create stories
- ✅ Story interactions work

---

### **PHASE 4: Search & Explore** (Week 4)
**Goal:** Search users, hashtags, and explore feed

#### 4.1 Search Overlay
**File:** `src/components/SearchOverlay.vue`
- Search input with debounce
- User results
- Hashtag results
- Recent searches (localStorage)

#### 4.2 Explore Page
**File:** `src/pages/Explore.vue`
- Grid layout
- Fetch /feed/explore
- Trending hashtags section
- Click to open post details

#### 4.3 Hashtag Page
**File:** `src/pages/Hashtag.vue`
- Display all posts with hashtag
- Hashtag stats
- Related hashtags

**Deliverables:**
- ✅ Search works
- ✅ Explore feed displays
- ✅ Hashtag pages work
- ✅ Trending hashtags

---

### **PHASE 5: Messaging System** (Week 5)
**Goal:** Real-time chat with WebSocket

#### 5.1 WebSocket Service
**File:** `src/services/websocket.ts`
```typescript
class MessageWebSocket {
  ws: WebSocket | null = null
  
  connect(token: string) {
    this.ws = new WebSocket(`ws://localhost:9004/ws?token=${token}`)
    
    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data)
      // Handle incoming messages
      useMessageStore().addMessage(message)
    }
  }
  
  send(conversationId: string, content: string) {
    // Send via HTTP API, WebSocket only receives
  }
}
```

#### 5.2 Message Store
**File:** `src/stores/messages.ts`

#### 5.3 Messages Page
**File:** `src/pages/Messages.vue`
- Conversation list (sidebar)
- Message thread (main area)
- Input box
- Send message
- Real-time updates via WebSocket
- Group chat features
- Video call button

#### 5.4 Mini Message Component (Already Done!)
- Shows recent conversations
- Unread badge
- Draggable position ✅
- Click to open messages page

**Deliverables:**
- ✅ Can send/receive messages
- ✅ WebSocket connected
- ✅ Real-time message updates
- ✅ Group chats work
- ✅ Video call integration

---

### **PHASE 6: Collections & Settings** (Week 6)
**Goal:** Saved posts and user settings

#### 6.1 Collections (Saved Posts)
**File:** `src/pages/Archive.vue`
- Display collections
- Create collection
- Save post to collection
- View posts in collection
- Rename/delete collection

#### 6.2 Settings Page
**File:** `src/pages/Settings.vue`
- Edit profile (name, bio, gender)
- Profile picture upload
- Privacy settings
- 2FA toggle
- Newsletter subscription
- Account verification request
- Logout

**Deliverables:**
- ✅ Collections work
- ✅ Settings functional
- ✅ Profile edits save

---

### **PHASE 7: Notifications & Real-time** (Week 7)
**Goal:** Push notifications system

#### 7.1 Notification WebSocket
**File:** `src/services/notificationWs.ts`
- Connect to notification service
- Receive notifications
- Display toast

#### 7.2 Notification Overlay
**File:** `src/components/NotificationOverlay.vue`
- List notifications
- Mark as read
- Click to navigate

#### 7.3 Real-time Feed Updates
- New posts appear
- Like count updates
- Comment count updates

**Deliverables:**
- ✅ Notifications work
- ✅ Real-time updates
- ✅ Toast notifications

---

### **PHASE 8: Admin Panel** (Week 8)
**Goal:** Admin features (if user is admin)

#### 8.1 Admin Routes
**File:** `src/pages/admin/Dashboard.vue`

#### 8.2 Admin Features
- View reports
- Resolve reports
- Ban/unban users
- Approve verification requests
- Send newsletters

**Deliverables:**
- ✅ Admin panel works
- ✅ Report management
- ✅ User management

---

## 📡 API Endpoint Mapping

### Authentication APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `authAPI.register()` | `/auth/register` | POST | No |
| `authAPI.login()` | `/auth/login` | POST | No |
| `authAPI.requestOTP()` | `/auth/send-otp` | POST | No |
| `authAPI.verifyRegistrationOTP()` | `/auth/verify-otp` | POST | No |
| `authAPI.verify2FA()` | `/auth/login/verify-2fa` | POST | No |
| `authAPI.forgotPassword()` | `/auth/password-reset/request` | POST | No |
| `authAPI.resetPassword()` | `/auth/password-reset/submit` | POST | No |
| `authAPI.googleAuth()` | `/auth/google/callback` | POST | No |

### Feed APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `feedAPI.getHomeFeed()` | `/feed/home?page=1&limit=20` | GET | Yes |
| `feedAPI.getExploreFeed()` | `/feed/explore?page=1&limit=20` | GET | Yes |
| `feedAPI.getReelsFeed()` | `/feed/reels?page=1&limit=20` | GET | Yes |

### Post APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `postAPI.createPost()` | `/posts` | POST | Yes |
| `postAPI.likePost()` | `/posts/:id/like` | POST | Yes |
| `postAPI.unlikePost()` | `/posts/:id/like` | DELETE | Yes |
| `postAPI.summarizeCaption()` | `/posts/:id/summarize` | POST | Yes |

### User APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `userAPI.getProfile()` | `/users/:username` | GET | Yes |
| `userAPI.getUserPosts()` | `/users/:username/posts` | GET | Yes |
| `userAPI.getUserReels()` | `/users/:username/reels` | GET | Yes |
| `userAPI.followUser()` | `/users/:id/follow` | POST | Yes |
| `userAPI.unfollowUser()` | `/users/:id/follow` | DELETE | Yes |
| `userAPI.blockUser()` | `/users/:id/block` | POST | Yes |
| `userAPI.unblockUser()` | `/users/:id/block` | DELETE | Yes |
| `userAPI.updateProfile()` | `/profile/edit` | PUT | Yes |
| `userAPI.setPrivacy()` | `/settings/privacy` | PUT | Yes |

### Comment APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `commentAPI.createComment()` | `/comments` | POST | Yes |
| `commentAPI.deleteComment()` | `/comments/:id` | DELETE | Yes |

### Message APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `messageAPI.createConversation()` | `/conversations` | POST | Yes |
| `messageAPI.getConversations()` | `/conversations` | GET | Yes |
| `messageAPI.getMessages()` | `/conversations/:id/messages` | GET | Yes |
| `messageAPI.sendMessage()` | `/conversations/:id/messages` | POST | Yes |
| `messageAPI.unsendMessage()` | `/messages/:id` | DELETE | Yes |
| `messageAPI.deleteConversation()` | `/conversations/:id` | DELETE | Yes |
| `messageAPI.getVideoToken()` | `/conversations/:id/video_token` | GET | Yes |

### Collection APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `collectionAPI.create()` | `/collections` | POST | Yes |
| `collectionAPI.getAll()` | `/collections` | GET | Yes |
| `collectionAPI.getPosts()` | `/collections/:id` | GET | Yes |
| `collectionAPI.savePost()` | `/collections/:id/posts` | POST | Yes |
| `collectionAPI.unsavePost()` | `/collections/:id/posts/:post_id` | DELETE | Yes |
| `collectionAPI.delete()` | `/collections/:id` | DELETE | Yes |
| `collectionAPI.rename()` | `/collections/:id` | PUT | Yes |

### Search APIs

| Frontend Method | Backend Endpoint | HTTP Method | Auth Required |
|----------------|------------------|-------------|---------------|
| `searchAPI.users()` | `/search/users?q=query` | GET | Yes |
| `searchAPI.hashtag()` | `/search/hashtags/:name` | GET | Yes |
| `searchAPI.trending()` | `/trending/hashtags` | GET | Yes |

---

## 🧪 Testing Strategy

### Phase Validation Checklist

After each phase, verify:

#### Phase 1 Checklist
- [ ] User can register with OTP verification
- [ ] User can login with credentials
- [ ] 2FA works if enabled
- [ ] Password reset flow works
- [ ] Token persists across refreshes
- [ ] Protected routes redirect correctly
- [ ] Profile page displays user info
- [ ] Follow/unfollow updates UI

#### Phase 2 Checklist
- [ ] Home feed loads posts
- [ ] Explore feed loads posts
- [ ] Reels feed loads videos
- [ ] Infinite scroll works
- [ ] Can create posts with media
- [ ] Like/unlike updates count
- [ ] Media uploads successfully
- [ ] Hashtags are clickable

#### Phase 3 Checklist
- [ ] Comments display under posts
- [ ] Can add new comments
- [ ] Can reply to comments
- [ ] Can delete own comments
- [ ] Stories display in viewer
- [ ] Can create stories
- [ ] Story timer works
- [ ] Story likes work

#### Phase 4 Checklist
- [ ] Search returns user results
- [ ] Search returns hashtag results
- [ ] Explore grid displays posts
- [ ] Hashtag page shows posts
- [ ] Trending hashtags display

#### Phase 5 Checklist
- [ ] WebSocket connects
- [ ] Can send messages
- [ ] Receive messages in real-time
- [ ] Conversation list updates
- [ ] Group chats work
- [ ] Mini message is draggable
- [ ] Video call button generates token

#### Phase 6 Checklist
- [ ] Can create collections
- [ ] Can save posts to collections
- [ ] Collections display properly
- [ ] Can rename/delete collections
- [ ] Settings save correctly
- [ ] Profile picture uploads

#### Phase 7 Checklist
- [ ] Notifications appear in real-time
- [ ] Toast notifications work
- [ ] Notification list displays
- [ ] Can mark notifications as read
- [ ] Feed updates in real-time

#### Phase 8 Checklist
- [ ] Admin can view reports
- [ ] Admin can resolve reports
- [ ] Admin can ban users
- [ ] Admin can send newsletters
- [ ] Admin can approve verifications

---

## 🎨 UI/UX Considerations

### Loading States
- Skeleton loaders for feeds
- Spinner for buttons
- Shimmer effect for images

### Error Handling
- Toast notifications for errors
- Inline validation messages
- Retry buttons for failed requests

### Optimistic UI Updates
- Like button instant feedback
- Follow button instant feedback
- Comment appears immediately
- Revert on failure

### Responsive Design
- Mobile-first approach
- Sidebar collapses on mobile
- Touch gestures for stories
- Swipe actions for messages

---

## 🔐 Security Considerations

### Frontend Security
- ✅ JWT tokens in localStorage (consider httpOnly cookies in production)
- ✅ Axios interceptors for token refresh
- ✅ CSRF protection
- ✅ XSS prevention (Vue auto-escapes)
- ✅ Rate limiting handled by backend

### API Security
- ✅ All protected routes require JWT
- ✅ Backend validates all inputs
- ✅ Rate limiting on sensitive endpoints
- ✅ Admin routes require admin role

---

## 📦 Dependencies to Install

### Required NPM Packages
```bash
npm install pinia                    # State management
npm install socket.io-client         # WebSocket (if using Socket.IO)
npm install date-fns                 # Date formatting
npm install @vueuse/core             # Vue composition utilities
npm install vite-plugin-pwa          # PWA support (optional)
```

---

## 🚀 Getting Started

### Step 1: Setup Environment
```bash
cd frontend/hoshi-vue
npm install
npm install pinia
```

### Step 2: Configure API URL
**File:** `.env`
```
VITE_API_URL=http://localhost:8000
VITE_WS_URL=ws://localhost:9004
```

### Step 3: Start Development
```bash
npm run dev
```

### Step 4: Follow Phase 1
Start implementing Phase 1 tasks one by one.

---

## 📝 Notes

### Backend Notes
- API Gateway running on port 8000
- All services are gRPC-based behind the gateway
- WebSocket for messages on port 9004
- MinIO for media storage
- Redis for caching and pub/sub
- RabbitMQ for async jobs

### Frontend Notes
- Vue 3 with Composition API
- TypeScript for type safety
- SCSS for styling
- Axios for HTTP
- Pinia for state
- Vue Router for navigation

### Mini Message Component
- ✅ Now fully draggable across the page
- ✅ Position persists in localStorage
- ✅ Constrained to window bounds
- ✅ Click vs drag detection
- ✅ Appears on all pages except overlays

---

## ✅ Success Criteria

### By End of Integration
- [ ] All 8 phases completed
- [ ] All features functional
- [ ] No console errors
- [ ] Responsive on mobile
- [ ] Real-time features working
- [ ] Media uploads working
- [ ] Authentication flows complete
- [ ] Admin features working (if admin)

---

## 🐛 Known Issues & Solutions

### Issue: CORS Errors
**Solution:** Backend already has CORS enabled in API Gateway

### Issue: WebSocket Connection Fails
**Solution:** Ensure message-service is running on port 9004

### Issue: Media Upload Fails
**Solution:** Check MinIO is running and accessible

### Issue: Token Expires
**Solution:** Implement refresh token logic in axios interceptor

---

## 📞 Support & Resources

### Backend Documentation
- See `BACKEND_FLOW_SUMMARY.md`
- See `TESTING_CHECKLIST.md`
- Postman collections available

### Frontend References
- Vue 3 Docs: https://vuejs.org/
- Pinia Docs: https://pinia.vuejs.org/
- Axios Docs: https://axios-http.com/

---

**End of Integration Plan**

This is a living document. Update as implementation progresses.
