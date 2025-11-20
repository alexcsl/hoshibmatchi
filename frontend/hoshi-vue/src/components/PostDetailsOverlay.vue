<template>
  <div class="post-details-overlay" @click="$emit('close')">
    <div class="post-details-modal" @click.stop>
      <button class="close-btn" @click="$emit('close')">✕</button>

      <div v-if="loading" class="loading-state">
        <div class="spinner">Loading...</div>
      </div>

      <div v-else-if="postData" class="post-details-content">
        <!-- Post Image -->
        <div class="post-image-container">
          <div v-if="postData.media_urls?.length > 0" class="media-carousel">
            <img 
              :src="postData.media_urls[currentMediaIndex]" 
              :alt="'Post by ' + postData.author_username" 
              class="post-image" 
            />
            
            <button 
              v-if="postData.media_urls.length > 1 && currentMediaIndex > 0" 
              class="carousel-btn prev"
              @click="currentMediaIndex--"
            >
              ‹
            </button>
            <button 
              v-if="postData.media_urls.length > 1 && currentMediaIndex < postData.media_urls.length - 1" 
              class="carousel-btn next"
              @click="currentMediaIndex++"
            >
              ›
            </button>
          </div>
        </div>

        <!-- Post Info -->
        <div class="post-info">
          <!-- Header -->
          <div class="info-header">
            <div class="user-info">
              <img 
                :src="postData.author_profile_url || '/placeholder.svg?height=32&width=32'" 
                :alt="postData.author_username" 
                class="avatar" 
              />
              <div>
                <div class="username">
                  {{ postData.author_username }}
                  <span v-if="postData.author_is_verified" class="verified">✓</span>
                </div>
              </div>
            </div>
            <button class="options-btn" @click="handleOptions">⋯</button>
          </div>

          <!-- Caption & Comments -->
          <div class="comments-section">
            <!-- Original Caption -->
            <div v-if="postData.caption" class="comment original-caption">
              <div class="comment-header">
                <img 
                  :src="postData.author_profile_url || '/placeholder.svg?height=32&width=32'" 
                  :alt="postData.author_username" 
                  class="comment-avatar" 
                />
                <div class="comment-content">
                  <div class="comment-text">
                    <strong>{{ postData.author_username }}</strong>
                    {{ postData.caption }}
                  </div>
                  <div class="comment-time">{{ formatTimestamp(postData.created_at) }}</div>
                </div>
              </div>
            </div>

            <!-- Comments -->
            <div v-for="comment in comments" :key="comment.id" class="comment">
              <div class="comment-header">
                <img 
                  :src="comment.author_profile_url || '/placeholder.svg?height=32&width=32'" 
                  :alt="comment.author_username" 
                  class="comment-avatar" 
                />
                <div class="comment-content">
                  <div class="comment-text">
                    <strong>{{ comment.author_username }}</strong>
                    {{ comment.content }}
                  </div>
                  <div class="comment-actions">
                    <span class="comment-time">{{ formatTimestamp(comment.created_at) }}</span>
                    <button class="like-btn" @click="handleLikeComment(comment.id)">
                      {{ comment.is_liked ? '❤️' : 'Like' }}
                    </button>
                    <button 
                      v-if="(comment.reply_count || 0) > 0" 
                      class="reply-btn"
                      @click="toggleReplies(comment.id)"
                    >
                      View replies ({{ comment.reply_count || 0 }})
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="loadingComments" class="loading-comments">
              Loading comments...
            </div>
          </div>

          <!-- Actions -->
          <div class="post-actions">
            <button 
              class="action-btn" 
              :class="{ liked: postData.is_liked }"
              @click="handleLike"
            >
              {{ postData.is_liked ? '❤️' : '🤍' }}
            </button>
            <button class="action-btn">💬</button>
            <button class="action-btn" @click="handleShare">📤</button>
            <button 
              class="action-btn" 
              :class="{ saved: postData.is_saved }"
              @click="handleSave"
              style="margin-left: auto;"
            >
              {{ postData.is_saved ? '🔖' : '🏷️' }}
            </button>
          </div>

          <!-- Likes -->
          <div class="likes-info">
            <strong>{{ formatLikes(postData.like_count) }}</strong>
          </div>

          <div class="timestamp-info">
            {{ formatTimestamp(postData.created_at) }}
          </div>

          <!-- Comment Input -->
          <div class="comment-input">
            <input 
              type="text" 
              v-model="newComment"
              placeholder="Add a comment..." 
              @keyup.enter="handleAddComment"
            />
            <button 
              v-if="newComment.trim()"
              @click="handleAddComment"
              :disabled="isSubmitting"
            >
              Post
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useFeedStore } from '@/stores/feed'
import { commentAPI } from '@/services/api'

interface Comment {
  id: string
  post_id: number
  content: string
  author_username: string
  author_profile_url: string
  created_at: string
  parent_comment_id?: number
  is_liked?: boolean
  reply_count?: number
}

const props = defineProps<{
  postId: string
  postObject?: any
}>()

const emit = defineEmits<{
  close: []
  like: [postId: string]
  save: [postId: string]
}>()

const feedStore = useFeedStore()

const loading = ref(false)
const loadingComments = ref(false)
const isSubmitting = ref(false)
const currentMediaIndex = ref(0)
const newComment = ref('')
const comments = ref<Comment[]>([])

const postData = computed(() => {
  // If passed directly (from Profile page), use it
  if (props.postObject) return props.postObject

  // Otherwise look in stores (from Feed page)
  return feedStore.homeFeed.find(p => p.id === props.postId) ||
         feedStore.exploreFeed.find(p => p.id === props.postId) ||
         feedStore.reelsFeed.find(p => p.id === props.postId)
})

onMounted(async () => {
  loadingComments.value = true
  try {
    const postIdNum = parseInt(props.postId)
    if (isNaN(postIdNum)) {
      console.error('Invalid post ID:', props.postId)
      return
    }
    
    const response = await commentAPI.getCommentsByPost(postIdNum)
    comments.value = response || []
    console.log('Loaded comments:', comments.value.length)
  } catch (error) {
    console.error('Failed to load comments:', error)
  } finally {
    loadingComments.value = false
  }
})

const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diffInMs = now.getTime() - date.getTime()
  const diffInSecs = Math.floor(diffInMs / 1000)
  const diffInMins = Math.floor(diffInSecs / 60)
  const diffInHours = Math.floor(diffInMins / 60)
  const diffInDays = Math.floor(diffInHours / 24)

  if (diffInDays > 7) {
    return date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
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
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(1)}M likes`
  } else if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}K likes`
  } else {
    return `${count} like${count !== 1 ? 's' : ''}`
  }
}

const handleLike = () => {
  emit('like', props.postId)
}

const handleSave = () => {
  emit('save', props.postId)
}

const handleShare = () => {
  // TODO: Implement share functionality
  console.log('Share post:', props.postId)
}

const handleOptions = () => {
  // TODO: Implement options menu
  console.log('Open options for post:', props.postId)
}

const handleAddComment = async () => {
  if (!newComment.value.trim() || isSubmitting.value) return
  
  isSubmitting.value = true
  try {
    const numericPostId = parseInt(props.postId)
    if (isNaN(numericPostId)) {
      console.error('Invalid post ID:', props.postId)
      alert('Invalid post ID')
      return
    }
    
    const response = await commentAPI.createComment({
      post_id: numericPostId,
      content: newComment.value.trim()
    })

    // Add comment to local list (the response IS the comment, not wrapped)
    if (response) {
      comments.value.unshift(response) // Add to top
    }

    // Update comment count in feed
    if (postData.value) {
      feedStore.updatePost(props.postId, {
        comment_count: (postData.value.comment_count || 0) + 1
      } as any)
    }

    newComment.value = ''
  } catch (error: any) {
    console.error('Failed to add comment:', error)
    console.error('Error details:', error.response?.data || error.message)
    
    // Show user-friendly error
    if (error.response?.status === 500) {
      alert('Failed to post comment. The server encountered an error. Please try again later.')
    } else {
      alert('Failed to post comment. Please try again.')
    }
  } finally {
    isSubmitting.value = false
  }
}

const handleLikeComment = async (commentId: string) => {
  // TODO: Implement comment like functionality
  console.log('Like comment:', commentId)
}

const toggleReplies = (commentId: string) => {
  // TODO: Implement replies loading
  console.log('Toggle replies for comment:', commentId)
}
</script>

<style scoped lang="scss">
.post-details-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 95;
}

.post-details-modal {
  background-color: #262626;
  border-radius: 8px;
  width: 90%;
  max-width: 1100px;
  max-height: 90vh;
  display: flex;
  overflow: hidden;
  position: relative;
}

.close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  background: rgba(0, 0, 0, 0.7);
  border: none;
  color: #fff;
  font-size: 24px;
  cursor: pointer;
  z-index: 10;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    background: rgba(0, 0, 0, 0.9);
  }
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 500px;

  .spinner {
    color: #a8a8a8;
    font-size: 16px;
  }
}

.post-details-content {
  display: flex;
  width: 100%;
  height: 100%;
}

.post-image-container {
  width: 65%;
  background-color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;

  .media-carousel {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;

    .post-image {
      max-width: 100%;
      max-height: 90vh;
      object-fit: contain;
    }

    .carousel-btn {
      position: absolute;
      top: 50%;
      transform: translateY(-50%);
      background: rgba(0, 0, 0, 0.5);
      border: none;
      color: #fff;
      width: 40px;
      height: 40px;
      border-radius: 50%;
      font-size: 24px;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;

      &:hover {
        background: rgba(0, 0, 0, 0.7);
      }

      &.prev {
        left: 12px;
      }

      &.next {
        right: 12px;
      }
    }
  }
}

.post-info {
  width: 35%;
  display: flex;
  flex-direction: column;
  background-color: #000;
  max-height: 90vh;
}

.info-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #262626;

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

.comments-section {
  flex: 1;
  overflow-y: auto;
  padding: 16px;

  .comment {
    margin-bottom: 16px;

    &.original-caption {
      padding-bottom: 16px;
      border-bottom: 1px solid #262626;
      margin-bottom: 16px;
    }

    .comment-header {
      display: flex;
      gap: 12px;

      .comment-avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        object-fit: cover;
        flex-shrink: 0;
      }

      .comment-content {
        flex: 1;

        .comment-text {
          font-size: 14px;
          line-height: 1.5;
          margin-bottom: 8px;
          word-wrap: break-word;

          strong {
            margin-right: 4px;
          }
        }

        .comment-actions {
          display: flex;
          gap: 16px;
          align-items: center;

          .comment-time {
            font-size: 12px;
            color: #a8a8a8;
          }

          .like-btn,
          .reply-btn {
            background: none;
            border: none;
            color: #a8a8a8;
            cursor: pointer;
            padding: 0;
            font-size: 12px;
            font-weight: 600;

            &:hover {
              color: #fff;
            }
          }
        }
      }
    }
  }

  .loading-comments {
    text-align: center;
    color: #a8a8a8;
    font-size: 14px;
    padding: 20px;
  }
}

.post-actions {
  display: flex;
  gap: 12px;
  padding: 8px 16px;
  border-top: 1px solid #262626;

  .action-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 24px;
    cursor: pointer;
    padding: 8px;
    transition: transform 0.1s;

    &:hover {
      opacity: 0.7;
    }

    &:active {
      transform: scale(0.9);
    }

    &.liked,
    &.saved {
      animation: pop 0.3s ease;
    }
  }
}

@keyframes pop {
  0% { transform: scale(1); }
  50% { transform: scale(1.2); }
  100% { transform: scale(1); }
}

.likes-info {
  padding: 8px 16px;
  font-size: 14px;
}

.timestamp-info {
  padding: 0 16px 12px;
  font-size: 12px;
  color: #a8a8a8;
  text-transform: uppercase;
}

.comment-input {
  display: flex;
  gap: 12px;
  padding: 16px;
  border-top: 1px solid #262626;
  align-items: center;

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
      color: #0958a3;
    }
  }
}

@media (max-width: 768px) {
  .post-details-modal {
    max-width: 100%;
    max-height: 100vh;
    border-radius: 0;
  }

  .post-details-content {
    flex-direction: column;
  }

  .post-image-container {
    width: 100%;
    height: 50%;
  }

  .post-info {
    width: 100%;
    height: 50%;
  }
}
</style>
