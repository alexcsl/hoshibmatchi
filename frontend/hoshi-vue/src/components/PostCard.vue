<template>
  <div class="post" @click="handlePostClick">
    <div class="post-header">
      <div class="user-info">
        <img 
          :src="post.author_profile_url || '/placeholder.svg?height=32&width=32'" 
          :alt="post.author_username" 
          class="avatar" 
        />
        <div>
          <div class="username">
            {{ post.author_username }}
            <span v-if="post.author_is_verified" class="verified">✓</span>
          </div>
          <div class="timestamp">{{ formatTimestamp(post.created_at) }}</div>
        </div>
      </div>
      <button class="options-btn" @click.stop="handleOptions">⋯</button>
    </div>

    <div class="post-media">
      <img 
        v-if="post.media_urls?.length > 0" 
        :src="getMediaUrl(post.media_urls[0])" 
        :alt="post.caption" 
        class="post-image"
        @error="handleImageError"
      />
      <div v-else class="no-media-placeholder">
        <span>📷</span>
        <p>No media</p>
      </div>
      <div v-if="post.media_urls?.length > 1" class="media-indicator">
        📷 {{ currentMediaIndex + 1 }}/{{ post.media_urls.length }}
      </div>
    </div>

    <div class="post-actions">
      <div class="action-buttons">
        <button 
          class="icon-btn" 
          :class="{ liked: post.is_liked }" 
          @click.stop="handleLike"
          :aria-label="post.is_liked ? 'Unlike' : 'Like'"
        >
          {{ post.is_liked ? '❤️' : '🤍' }}
        </button>
        <button class="icon-btn" @click.stop="handleComment" aria-label="Comment">💬</button>
        <button class="icon-btn" @click.stop="handleShare" aria-label="Share">📤</button>
      </div>
      <button 
        class="icon-btn" 
        :class="{ saved: post.is_saved }" 
        @click.stop="handleSave"
        :aria-label="post.is_saved ? 'Unsave' : 'Save'"
      >
        {{ post.is_saved ? '🔖' : '🏷️' }}
      </button>
    </div>

    <div class="post-content">
      <div class="likes">
        <strong>{{ formatLikes(post.like_count) }}</strong>
      </div>
      <div v-if="post.caption" class="caption">
        <strong>{{ post.author_username }}</strong>
        {{ post.caption }}
      </div>
      <div v-if="post.comment_count > 0" class="comments-link" @click.stop="handleViewComments">
        View all {{ post.comment_count }} comments
      </div>
    </div>

    <div class="comment-input">
      <input 
        type="text" 
        v-model="commentText"
        placeholder="Add a comment..." 
        @keyup.enter="handleAddComment"
        @click.stop
      />
      <button 
        v-if="commentText.trim()" 
        @click.stop="handleAddComment"
        :disabled="isSubmitting"
      >
        Post
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface Post {
  id: string
  author_id: number
  caption: string
  author_username: string
  author_profile_url: string
  author_is_verified: boolean
  media_urls: string[]
  created_at: string
  is_reel: boolean
  like_count: number
  comment_count: number
  share_count?: number
  is_liked?: boolean
  is_saved?: boolean
}

const props = defineProps<{
  post: Post
}>()

const emit = defineEmits<{
  like: [postId: string]
  save: [postId: string]
  comment: [postId: string, content: string]
  share: [postId: string]
  openDetails: [postId: string]
  openOptions: [postId: string]
}>()

const commentText = ref('')
const isSubmitting = ref(false)
const currentMediaIndex = ref(0)

const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diffInMs = now.getTime() - date.getTime()
  const diffInSecs = Math.floor(diffInMs / 1000)
  const diffInMins = Math.floor(diffInSecs / 60)
  const diffInHours = Math.floor(diffInMins / 60)
  const diffInDays = Math.floor(diffInHours / 24)

  if (diffInDays > 7) {
    return date.toLocaleDateString()
  } else if (diffInDays > 0) {
    return `${diffInDays} day${diffInDays > 1 ? 's' : ''} ago`
  } else if (diffInHours > 0) {
    return `${diffInHours} hour${diffInHours > 1 ? 's' : ''} ago`
  } else if (diffInMins > 0) {
    return `${diffInMins} minute${diffInMins > 1 ? 's' : ''} ago`
  } else {
    return 'Just now'
  }
}

const formatLikes = (count: number) => {
  const likeCount = count || 0
  if (likeCount >= 1000000) {
    return `${(likeCount / 1000000).toFixed(1)}M likes`
  } else if (likeCount >= 1000) {
    return `${(likeCount / 1000).toFixed(1)}K likes`
  } else if (likeCount === 0) {
    return '0 likes'
  } else {
    return `${likeCount} like${likeCount !== 1 ? 's' : ''}`
  }
}

const getMediaUrl = (url: string) => {
  // Check if it's a valid URL
  if (!url || url.trim() === '') {
    return '/placeholder.svg?height=600&width=600&text=No+Image'
  }
  
  // If it's a relative path, prepend the API base URL
  if (url.startsWith('/uploads/') || url.startsWith('uploads/')) {
    return `http://localhost:8000${url.startsWith('/') ? url : '/' + url}`
  }
  
  return url
}

const handleImageError = (event: Event) => {
  const img = event.target as HTMLImageElement
  img.src = '/placeholder.svg?height=600&width=600&text=Image+Not+Found'
}

const handleLike = () => {
  emit('like', props.post.id)
}

const handleSave = () => {
  emit('save', props.post.id)
}

const handleComment = () => {
  emit('openDetails', props.post.id)
}

const handleShare = () => {
  emit('share', props.post.id)
}

const handlePostClick = () => {
  emit('openDetails', props.post.id)
}

const handleViewComments = () => {
  emit('openDetails', props.post.id)
}

const handleOptions = () => {
  emit('openOptions', props.post.id)
}

const handleAddComment = async () => {
  if (!commentText.value.trim() || isSubmitting.value) return
  
  isSubmitting.value = true
  try {
    emit('comment', props.post.id, commentText.value.trim())
    commentText.value = ''
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped lang="scss">
.post {
  border: 1px solid #262626;
  border-radius: 1px;
  cursor: pointer;
  background-color: #000;
}

.post-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;

  .user-info {
    display: flex;
    align-items: center;
    gap: 12px;

    .avatar {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      object-fit: cover;
    }

    .username {
      font-weight: 600;
      font-size: 14px;
      display: flex;
      align-items: center;
      gap: 4px;

      .verified {
        color: #0a66c2;
        font-size: 12px;
      }
    }

    .timestamp {
      font-size: 12px;
      color: #a8a8a8;
    }
  }

  .options-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
    padding: 0;

    &:hover {
      opacity: 0.7;
    }
  }
}

.post-media {
  position: relative;
  width: 100%;
  
  .post-image {
    width: 100%;
    display: block;
    object-fit: cover;
    max-height: 600px;
  }

  .no-media-placeholder {
    width: 100%;
    height: 400px;
    background-color: #1a1a1a;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #a8a8a8;

    span {
      font-size: 48px;
      margin-bottom: 12px;
    }

    p {
      font-size: 14px;
    }
  }

  .media-indicator {
    position: absolute;
    top: 12px;
    right: 12px;
    background: rgba(0, 0, 0, 0.7);
    color: #fff;
    padding: 4px 12px;
    border-radius: 16px;
    font-size: 12px;
    font-weight: 600;
  }
}

.post-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;

  .action-buttons {
    display: flex;
    gap: 12px;
  }

  .icon-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
    padding: 8px;
    transition: opacity 0.2s, transform 0.1s;

    &:hover {
      opacity: 0.7;
    }

    &:active {
      transform: scale(0.9);
    }

    &.liked, &.saved {
      animation: pop 0.3s ease;
    }
  }
}

@keyframes pop {
  0% { transform: scale(1); }
  50% { transform: scale(1.2); }
  100% { transform: scale(1); }
}

.post-content {
  padding: 0 16px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;

  .likes {
    font-size: 14px;
  }

  .caption {
    font-size: 14px;
    line-height: 1.5;
    
    strong {
      margin-right: 4px;
    }
  }

  .comments-link {
    font-size: 14px;
    color: #a8a8a8;
    cursor: pointer;

    &:hover {
      color: #fff;
    }
  }
}

.comment-input {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-top: 1px solid #262626;

  input {
    flex: 1;
    background: none;
    border: none;
    color: #fff;
    font-size: 14px;
    outline: none;

    &::placeholder {
      color: #a8a8a8;
    }
  }

  button {
    background: none;
    border: none;
    color: #0a66c2;
    cursor: pointer;
    font-weight: 600;
    font-size: 14px;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    &:not(:disabled):hover {
      color: #0a66c2;
    }
  }
}
</style>
