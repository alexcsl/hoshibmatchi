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
          <div v-if="post.location" class="location">{{ post.location }}</div>
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
      <div class="likes" @click.stop="handleShowLikes" style="cursor: pointer;">
        <strong>{{ formatLikes(post.like_count) }}</strong>
      </div>
      <div class="caption">
          <strong>{{ post.author_username }}</strong>
          <span v-if="!showingSummary" v-html="formattedCaption" @click="handleRichTextClick"></span>
          <span v-else>{{ aiSummary }}</span>
      </div>
      
      <button class="ai-btn" @click.stop="toggleSummary">
        {{ showingSummary ? 'Show Original' : '✨ Summarize with AI' }}
      </button>
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

    <!-- Share Modal -->
    <div v-if="showShareModal" class="share-modal-overlay" @click.stop="showShareModal = false">
      <div class="share-modal" @click.stop>
        <div class="share-header">
          <h3>Share</h3>
          <button class="close-share-btn" @click="showShareModal = false">✕</button>
        </div>
        <div class="share-options">
          <button class="share-option" @click="copyLink">
            <span class="share-icon">🔗</span>
            <span>Copy Link</span>
          </button>
          <button class="share-option" @click="shareToFacebook">
            <span class="share-icon">📘</span>
            <span>Facebook</span>
          </button>
          <button class="share-option" @click="shareToTwitter">
            <span class="share-icon">🐦</span>
            <span>Twitter</span>
          </button>
          <button class="share-option" @click="shareViaEmail">
            <span class="share-icon">✉️</span>
            <span>Email</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Options Modal -->
    <div v-if="showOptionsModal" class="share-modal-overlay" @click.stop="showOptionsModal = false">
      <div class="options-modal" @click.stop>
        <button v-if="isOwnPost" class="option-btn danger" @click="handleDeletePost">
          <span>🗑️</span>
          <span>Delete Post</span>
        </button>
        <button v-if="!isOwnPost" class="option-btn" @click="showOptionsModal = false">
          <span>🚫</span>
          <span>Report</span>
        </button>
        <button class="option-btn" @click="showOptionsModal = false">
          <span>✕</span>
          <span>Cancel</span>
        </button>
      </div>
    </div>

    <!-- Likes Modal -->
    <div v-if="showLikesModal" class="share-modal-overlay" @click.stop="showLikesModal = false">
      <div class="likes-modal" @click.stop>
        <div class="likes-header">
          <h3>Likes</h3>
          <button class="close-likes-btn" @click="showLikesModal = false">✕</button>
        </div>
        <div class="likes-list">
          <div v-if="loadingLikers" class="loading-likers">Loading...</div>
          <div v-else-if="likers.length === 0" class="no-likers">No likes yet</div>
          <div v-else v-for="liker in likers" :key="liker.user_id" class="liker-item">
            <img 
              :src="liker.profile_picture_url || '/placeholder.svg?height=40&width=40'" 
              :alt="liker.username" 
              class="liker-avatar" 
            />
            <div class="liker-info">
              <div class="liker-username">
                {{ liker.username }}
                <span v-if="liker.is_verified" class="verified">✓</span>
              </div>
              <div v-if="liker.full_name" class="liker-fullname">{{ liker.full_name }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { postAPI } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useRichText } from '@/composables/useRichText'

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
  location?: string
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
  deleted: [postId: string]
  archived: [postId: string]
}>()

const commentText = ref('')
const isSubmitting = ref(false)
const currentMediaIndex = ref(0)
const authStore = useAuthStore()
const { formatRichText, handleRichTextClick } = useRichText()

const isOwnPost = computed(() => {
  return authStore.user?.user_id === props.post.author_id
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

// Share Modal State
const showShareModal = ref(false)
const showOptionsModal = ref(false)
const showLikesModal = ref(false)
const likers = ref<any[]>([])
const loadingLikers = ref(false)

const handleShare = () => {
  showShareModal.value = true
}

const handleShowLikes = async () => {
  if (props.post.like_count === 0) return
  
  showLikesModal.value = true
  loadingLikers.value = true
  
  try {
    const response = await postAPI.getPostLikers(props.post.id)
    likers.value = response
  } catch (error) {
    console.error('Failed to load likers:', error)
    likers.value = []
  } finally {
    loadingLikers.value = false
  }
}

const copyLink = async () => {
  const url = `${window.location.origin}/post/${props.post.id}`
  try {
    await navigator.clipboard.writeText(url)
    alert('Link copied to clipboard!')
    showShareModal.value = false
  } catch (err) {
    console.error('Failed to copy', err)
  }
}

const shareToFacebook = () => {
  const url = `${window.location.origin}/post/${props.post.id}`
  window.open(`https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(url)}`, '_blank')
  showShareModal.value = false
}

const shareToTwitter = () => {
  const url = `${window.location.origin}/post/${props.post.id}`
  const text = props.post.caption ? props.post.caption.substring(0, 200) : 'Check out this post!'
  window.open(`https://twitter.com/intent/tweet?url=${encodeURIComponent(url)}&text=${encodeURIComponent(text)}`, '_blank')
  showShareModal.value = false
}

const shareViaEmail = () => {
  const url = `${window.location.origin}/post/${props.post.id}`
  const subject = 'Check out this post!'
  const body = `I thought you might like this: ${url}`
  window.location.href = `mailto:?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`
  showShareModal.value = false
}

const handlePostClick = () => {
  emit('openDetails', props.post.id)
}

const handleViewComments = () => {
  emit('openDetails', props.post.id)
}

const handleOptions = () => {
  showOptionsModal.value = true
}

const handleDeletePost = async () => {
  if (!confirm('Are you sure you want to delete this post? This action cannot be undone.')) {
    return
  }
  
  try {
    await postAPI.deletePost(props.post.id)
    showOptionsModal.value = false
    // Emit event to parent to remove from feed
    emit('deleted', props.post.id)
    alert('Post deleted successfully')
  } catch (error) {
    console.error('Failed to delete post:', error)
    alert('Failed to delete post. Please try again.')
  }
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

const showingSummary = ref(false)
const aiSummary = ref('')
const loadingAi = ref(false)

// Rich Text Formatter for hashtags and mentions
const formattedCaption = computed(() => {
  if (!props.post.caption) return ''
  return formatRichText(props.post.caption)
})

const toggleSummary = async () => {
    if (showingSummary.value) {
        showingSummary.value = false
        return
    }
    
    if (aiSummary.value) {
        showingSummary.value = true
        return
    }
    
    loadingAi.value = true
    try {
        const res = await postAPI.summarizeCaption(props.post.id)
        aiSummary.value = res.summary
        showingSummary.value = true
    } catch(e) {
        alert("AI Summarization failed")
    } finally {
        loadingAi.value = false
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

.location {
  font-size: 12px;
  color: #fff;
}

/* Rich text styles for hashtags and mentions */
:deep(.rich-text-hashtag),
:deep(.rich-text-mention) {
  color: #0095f6;
  font-weight: 500;
  cursor: pointer;
  &:hover {
    text-decoration: underline;
  }
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

.ai-btn {
    background: none; border: none; color: #0095f6; 
    font-size: 12px; font-weight: 600; cursor: pointer;
    margin-top: 4px; display: block;
}

.share-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.share-modal {
  background-color: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;

  .share-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #404040;

    h3 {
      font-size: 16px;
      font-weight: 600;
      margin: 0;
    }

    .close-share-btn {
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

  .share-options {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
    padding: 20px;

    .share-option {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 8px;
      padding: 16px;
      background-color: #1a1a1a;
      border: 1px solid #404040;
      border-radius: 8px;
      color: #fff;
      cursor: pointer;
      transition: all 0.2s;

      &:hover {
        background-color: #404040;
        transform: translateY(-2px);
      }

      .share-icon {
        font-size: 32px;
      }

      span:last-child {
        font-size: 14px;
        font-weight: 500;
      }
    }
  }
}

.options-modal {
  background-color: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;

  .option-btn {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 16px;
    background: none;
    border: none;
    border-bottom: 1px solid #404040;
    color: #fff;
    font-size: 14px;
    cursor: pointer;
    transition: background-color 0.2s;

    &:last-child {
      border-bottom: none;
    }

    &:hover {
      background-color: #404040;
    }

    &.danger {
      color: #ff4458;
      font-weight: 600;
    }

    span:first-child {
      font-size: 20px;
    }
  }
}

.likes-modal {
  background-color: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  max-height: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;

  .likes-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #404040;

    h3 {
      font-size: 16px;
      font-weight: 600;
      margin: 0;
    }

    .close-likes-btn {
      background: none;
      border: none;
      color: #8e8e8e;
      font-size: 24px;
      cursor: pointer;
      padding: 0;
      width: 32px;
      height: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s;

      &:hover {
        color: #fff;
        background-color: rgba(255, 255, 255, 0.1);
        border-radius: 50%;
      }
    }
  }

  .likes-list {
    overflow-y: auto;
    padding: 12px 20px;
    flex: 1;

    .loading-likers,
    .no-likers {
      text-align: center;
      padding: 20px;
      color: #8e8e8e;
    }

    .liker-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 0;
      border-bottom: 1px solid #404040;

      &:last-child {
        border-bottom: none;
      }

      .liker-avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        object-fit: cover;
      }

      .liker-info {
        flex: 1;

        .liker-username {
          font-size: 14px;
          font-weight: 600;
          color: #fff;

          .verified {
            color: #4a9eff;
            margin-left: 4px;
          }
        }

        .liker-fullname {
          font-size: 12px;
          color: #8e8e8e;
          margin-top: 2px;
        }
      }
    }
  }
}
</style>

