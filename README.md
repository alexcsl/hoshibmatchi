# Hoshibmatchi - Instagram Clone

A full-featured Instagram clone built with modern microservices architecture, featuring real-time messaging, stories, posts, video calls, and AI-powered caption generation.

![Architecture](https://img.shields.io/badge/Architecture-Microservices-blue)
![Backend](https://img.shields.io/badge/Backend-Go-00ADD8)
![Frontend](https://img.shields.io/badge/Frontend-Vue.js-4FC08D)
![AI](https://img.shields.io/badge/AI-Python-yellow)

## 🌟 Features

### Core Social Media Features
- **User Management**
  - Registration with email verification (OTP)
  - Login with JWT authentication (access + refresh tokens)
  - Two-Factor Authentication (2FA)
  - Google OAuth integration
  - Password reset functionality
  - User profile with verification badges
  - Profile privacy settings (private/public accounts)

- **Posts**
  - Create posts with images/videos
  - Multiple media uploads per post
  - Video transcoding for optimized playback
  - Like and save posts
  - Comments with nested replies
  - Post summarization using AI
  - Hashtag support with trending topics
  - Explore feed with personalized recommendations

- **Stories**
  - 24-hour ephemeral stories
  - Image and video stories
  - Story filters and stickers
  - Text overlays with customization
  - Close friends feature
  - Story archive
  - View analytics

- **Messaging**
  - Real-time chat via WebSocket
  - Direct messages and group conversations
  - Media sharing (images, videos)
  - Video and audio calls (VideoSDK integration)
  - Message status (sent, delivered, seen)
  - Online status indicators

- **Social Interactions**
  - Follow/unfollow users
  - Block/unblock users
  - Hide stories from specific users
  - Close friends management
  - User search with Jaro-Winkler similarity algorithm
  - Friend recommendations

- **Content Moderation**
  - Post reporting system
  - Admin dashboard for report management
  - User banning capabilities
  - Content visibility controls

## 🏗️ Architecture

### Microservices Design
The application follows a microservices architecture pattern with the following services:

```
┌─────────────────┐
│   Traefik       │  ← Reverse Proxy & Load Balancer
│   (Port 80/443) │
└────────┬────────┘
         │
         ├─────────────┐
         │             │
    ┌────▼────┐   ┌───▼──────┐
    │Frontend │   │   API    │
    │ Vue.js  │   │ Gateway  │
    └─────────┘   └────┬─────┘
                       │
         ┌─────────────┼─────────────┬─────────────┬──────────────┐
         │             │             │             │              │
    ┌────▼────┐   ┌───▼──────┐ ┌───▼──────┐ ┌───▼──────┐  ┌───▼──────┐
    │  User   │   │   Post   │ │  Story   │ │ Message  │  │  Media   │
    │ Service │   │ Service  │ │ Service  │ │ Service  │  │ Service  │
    └────┬────┘   └────┬─────┘ └────┬─────┘ └────┬─────┘  └────┬─────┘
         │             │             │             │              │
    ┌────▼────┐   ┌───▼──────┐ ┌───▼──────┐ ┌───▼──────┐  ┌───▼──────┐
    │User DB  │   │ Post DB  │ │Story DB  │ │Message DB│  │  MinIO   │
    │Postgres │   │ Postgres │ │ Postgres │ │ Postgres │  │(Storage) │
    └─────────┘   └──────────┘ └──────────┘ └──────────┘  └──────────┘
```

Additional services:
- **Hashtag Service**: Manages hashtags and trending topics
- **Report Service**: Handles content reporting and moderation
- **Notification Service**: Manages user notifications
- **Email Service**: Sends transactional emails
- **Worker Service**: Background job processing (video transcoding, story expiration)
- **AI Service**: Caption summarization using T5 transformer model

### Communication Patterns
- **gRPC**: Inter-service communication for high performance
- **REST API**: Client-facing API through API Gateway
- **WebSocket**: Real-time messaging and notifications
- **RabbitMQ**: Asynchronous message queue for background tasks
- **Redis**: Caching and distributed rate limiting

## 🛠️ Tech Stack

### Backend
- **Language**: Go 1.25
- **Framework**: Gin (HTTP), gRPC
- **Database**: PostgreSQL 15 (per-service databases)
- **Cache**: Redis 7
- **Message Queue**: RabbitMQ 3.12
- **Object Storage**: MinIO
- **Authentication**: JWT with bcrypt
- **API Documentation**: Swagger/OpenAPI

### Frontend
- **Framework**: Vue.js 3.5
- **Build Tool**: Vite (Rolldown variant)
- **State Management**: Pinia
- **Routing**: Vue Router
- **Styling**: SCSS with CSS variables for theming
- **HTTP Client**: Axios
- **Testing**: Vitest

### AI/ML
- **Language**: Python 3.11
- **Framework**: FastAPI
- **ML Library**: PyTorch 2.1, Transformers 4.35
- **Model**: T5 (Text-to-Text Transfer Transformer) for caption summarization
- **Model Source**: Hugging Face (`alexcsl10/baption`)

### DevOps
- **Containerization**: Docker & Docker Compose
- **Reverse Proxy**: Traefik 2.10
- **Video Processing**: FFmpeg (for transcoding)
- **Video Calls**: VideoSDK integration

## 📋 Prerequisites

- Docker Desktop (or Docker Engine + Docker Compose)
- Git
- (Optional) Go 1.25+ for local backend development
- (Optional) Node.js 20+ for local frontend development
- (Optional) Python 3.11+ for AI service development

## 🚀 Getting Started

### 1. Clone the Repository
```bash
git clone https://github.com/yourusername/hoshibmatchi.git
cd hoshibmatchi
```

### 2. Configure Environment Variables

**IMPORTANT**: The `.env` file in the repository contains default development values. For production or public deployment, you MUST update these values:

```bash
# Copy and edit the environment file
cp .env .env.production

# Update these critical values:
# - JWT_SECRET (generate a secure random string)
# - Database passwords
# - GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET (obtain from Google Cloud Console)
# - VIDEOSDK_API_KEY and VIDEOSDK_API_SECRET (obtain from VideoSDK)
# - SMTP credentials for email service
# - MinIO credentials
```

**Security Warning**: The current `.env` file contains example credentials and should NOT be used in production!

### 3. Run with Docker Compose

#### Development Mode (with hot reload)
```bash
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

#### Production Mode
```bash
docker-compose up --build -d
```

### 4. Access the Application

- **Frontend**: http://localhost:5173 (dev) or http://localhost:80 (prod)
- **API Gateway**: http://localhost:8000
- **API Documentation**: http://localhost:8000/swagger/index.html
- **Traefik Dashboard**: http://localhost:8080
- **MinIO Console**: http://localhost:9001
- **RabbitMQ Management**: http://localhost:15672 (dev only)

### 5. Initialize the System

On first run, you may need to:
1. Create MinIO buckets (posts, stories, messages, profile-pictures)
2. Run database migrations (automatically handled by services on startup)
3. Register a test account

## 📚 Project Structure

```
hoshibmatchi/
├── backend/
│   ├── api-gateway/          # REST API entry point
│   ├── user-service/         # User management, auth, profiles
│   ├── post-service/         # Post CRUD, likes, comments
│   ├── story-service/        # Story management (24h expiry)
│   ├── message-service/      # Chat and video calls
│   ├── media-service/        # Media upload to MinIO
│   ├── hashtag-service/      # Hashtag tracking
│   ├── report-service/       # Content moderation
│   ├── notification-service/ # Push notifications
│   ├── email-service/        # Email sending
│   ├── worker-service/       # Background jobs
│   └── ai-service/           # AI caption summarization
│
├── frontend/
│   └── hoshi-vue/            # Vue.js SPA
│
├── protos/                   # gRPC protocol definitions
├── traefik/                  # Traefik configuration
├── docker-compose.yml        # Production compose file
├── docker-compose.dev.yml    # Development overrides
└── .env                      # Environment variables
```

## 🔧 Development

### Backend Development

Each microservice is independently buildable:

```bash
cd backend/user-service
go mod download
go run main.go
```

**Generate gRPC code**:
```bash
cd protos
protoc --go_out=. --go-grpc_out=. user.proto
```

### Frontend Development

```bash
cd frontend/hoshi-vue
npm install
npm run dev
```

Build for production:
```bash
npm run build
```

### AI Service Development

```bash
cd backend/ai-service
pip install -r requirements.txt
python main.py
```

## 🧪 Testing

### Backend Tests
```bash
cd backend/user-service
go test ./...
```

### Frontend Tests
```bash
cd frontend/hoshi-vue
npm run test
npm run test:coverage
```

## 📊 Database Schema

Each service has its own PostgreSQL database following the database-per-service pattern:

- **user_service_db**: Users, followers, blocks, close friends
- **post_service_db**: Posts, comments, likes, saves, collections
- **story_service_db**: Stories, story views, story likes
- **message_service_db**: Conversations, messages, message status
- **notification_service_db**: Notifications
- **report_service_db**: Reports
- **hashtag_service_db**: Hashtags, trending data

## 🔐 Security Features

- JWT-based authentication with short-lived access tokens
- Refresh token rotation
- Password hashing with bcrypt
- Rate limiting (Redis-backed distributed limiter)
- CORS configuration
- SQL injection prevention (parameterized queries)
- Input validation and sanitization
- Cloudflare Turnstile CAPTCHA integration
- Content Security Policy headers
- Private account support
- Blocked user filtering

## 🎯 API Endpoints

### Authentication
- `POST /auth/register` - Register new user
- `POST /auth/login` - Login with credentials
- `POST /auth/login/2fa` - Verify 2FA code
- `POST /auth/refresh` - Refresh access token
- `GET /auth/google` - OAuth with Google
- `POST /auth/password-reset` - Request password reset
- `POST /auth/password-reset/verify` - Verify reset token

### Posts
- `GET /feed/home` - Home feed
- `GET /feed/explore` - Explore feed
- `GET /feed/reels` - Reels feed
- `POST /posts` - Create post
- `DELETE /posts/:id` - Delete post
- `POST /posts/:id/like` - Like post
- `POST /posts/:id/save` - Save post
- `POST /posts/:id/summarize` - AI summarize caption

### Stories
- `GET /stories/feed` - Get story feed
- `POST /stories` - Create story
- `POST /stories/:id/view` - Mark story as viewed
- `GET /stories/archive` - Get archived stories

### Messages
- `GET /messages/conversations` - List conversations
- `POST /messages` - Send message
- `GET /messages/token` - Get video call token

See [API Documentation](http://localhost:8000/swagger/index.html) for full details.

## 🚀 Deployment

### Docker Deployment

The project is containerized and can be deployed to any Docker-compatible platform:

- **Docker Swarm**
- **Kubernetes** (K8s manifests needed)
- **AWS ECS**
- **Google Cloud Run**
- **Azure Container Instances**

### Environment Configuration

For production deployment:
1. Update all secrets in `.env`
2. Configure SSL certificates in Traefik
3. Set up proper DNS records
4. Configure backup strategies for PostgreSQL databases
5. Set up monitoring and logging (Prometheus, Grafana)
6. Configure MinIO for production (persistent volumes)

## 🤝 Contributing

This is a personal portfolio project. If you'd like to contribute or provide feedback:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## 📝 License

This project is for educational and portfolio purposes. All rights reserved.

## 🙏 Acknowledgments

- Vue.js and Go communities
- Hugging Face for pre-trained models
- VideoSDK for video calling infrastructure
- MinIO for object storage
- Traefik for reverse proxy

## 📧 Contact

For questions or collaboration opportunities, please reach out via:
- GitHub: [@alexcsl](https://github.com/alexcsl)
- Email: hoshibmatchi@gmail.com

---

**Note**: This is a demonstration project showcasing microservices architecture, full-stack development, and modern DevOps practices. It is not intended for production use without proper security hardening and infrastructure setup. This project is still buggy and is in development. It's a messy project hehe...
