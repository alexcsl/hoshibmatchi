<template>
  <div
    class="reels-viewer-overlay"
    @click="$emit('close')"
  >
    <div
      class="reels-viewer"
      @click.stop
    >
      <button
        class="close-btn"
        @click="$emit('close')"
      >
        ✕
      </button>

      <div 
        v-if="currentReel" 
        class="reel-container"
        @wheel="handleScroll"
      >
        <!-- Reel Content -->
        <div class="reel-media">
          <div v-if="loadingMedia" class="media-loading">
            <div class="loading-spinner">Loading...</div>
          </div>
          <video 
            v-else-if="currentMediaUrl && isVideoUrl(currentMediaUrl)"
            :src="currentMediaUrl" 
            class="reel-video"
            autoplay
            loop
            playsinline
            muted
          ></video>
          <img 
            v-else-if="currentMediaUrl"
            :src="currentMediaUrl" 
            :alt="'Reel by ' + currentReel.author_username" 
            class="reel-image" 
          />
        </div>

        <!-- Reel Info Sidebar -->
        <div class="reel-sidebar">
          <!-- Author Info -->
          <div
            class="author-section"
            @click="navigateToProfile"
          >
            <SecureImage
              :src="currentReel.author_profile_url"
              :alt="currentReel.author_username"
              class-name="author-avatar"
              loading-placeholder="/placeholder.svg?height=48&width=48"
              error-placeholder="/default-avatar.svg"
            />
            <div class="author-name">
              @{{ currentReel.author_username }}
            </div>
          </div>

          <!-- Actions -->
          <div class="actions-section">
            <button 
              class="action-btn" 
              :class="{ liked: currentReel.is_liked }"
              @click="handleLike"
            >
              <span class="icon">{{ currentReel.is_liked ? '❤️' : '🤍' }}</span>
              <span class="count">{{ formatCount(currentReel.like_count) }}</span>
            </button>

            <button
              class="action-btn"
              @click="showComments = !showComments"
            >
              <span class="icon">💬</span>
              <span class="count">{{ formatCount(currentReel.comment_count) }}</span>
            </button>

            <button
              class="action-btn"
              @click="handleShare"
            >
              <span class="icon">📤</span>
            </button>

            <button 
              class="action-btn" 
              :class="{ saved: currentReel.is_saved }"
              @click="handleSave"
            >
              <span class="icon">{{ currentReel.is_saved ? '🔖' : '🏷️' }}</span>
            </button>
          </div>
        </div>

        <!-- Caption - Centered at bottom -->
        <div
          v-if="currentReel.caption"
          class="caption-section"
        >
          <p
            class="caption-text"
            @click="handleRichTextClick"
            v-html="formattedCaption"
          ></p>
        </div>

        <!-- Comments Overlay -->
        <div
          v-if="showComments"
          class="comments-overlay"
          @click="showComments = false"
        >
          <div
            class="comments-panel"
            @click.stop
          >
            <div class="comments-header">
              <h3>Comments</h3>
              <button @click="showComments = false">
                ✕
              </button>
            </div>
            <div class="comments-list">
              <div
                v-for="comment in comments"
                :key="comment.id"
                class="comment-item"
              >
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
                  <div class="comment-time">
                    {{ formatTimestamp(comment.created_at) }}
                  </div>
                </div>
              </div>
            </div>
            <div class="comment-input">
              <input 
                v-model="newComment" 
                type="text"
                placeholder="Add a comment..." 
                @keyup.enter="handleAddComment"
              />
              <button 
                v-if="newComment.trim()"
                :disabled="isSubmitting"
                @click="handleAddComment"
              >
                Post
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Navigation Indicators -->
      <div class="navigation-indicators">
        <div 
          v-if="currentIndex > 0" 
          class="nav-hint top"
          @click="goToPrevious"
        >
          <span>↑</span>
          <span class="hint-text">Previous</span>
        </div>
        <div 
          v-if="currentIndex < reels.length - 1" 
          class="nav-hint bottom"
          @click="goToNext"
        >
          <span class="hint-text">Next</span>
          <span>↓</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { useFeedStore } from "@/stores/feed";
import { commentAPI } from "@/services/api";
import { useRichText } from "@/composables/useRichText";
import { getSecureMediaURL } from "@/services/media";

interface Comment {
  id: string
  post_id: number
  content: string
  author_username: string
  author_profile_url: string
  created_at: string
}

const props = defineProps<{
  initialIndex: number
}>();

const emit = defineEmits<{
  close: []
}>();

const feedStore = useFeedStore();
const currentIndex = ref(props.initialIndex);
const showComments = ref(false);
const comments = ref<Comment[]>([]);
const newComment = ref("");
const isSubmitting = ref(false)
const { formatRichText, handleRichTextClick } = useRichText()

// Secure media URL state
const currentMediaUrl = ref<string>("");
const loadingMedia = ref(true);

const reels = computed(() => feedStore.reelsFeed)
const currentReel = computed(() => reels.value[currentIndex.value])

const isVideoUrl = (url: string) => {
  if (!url) return false
  const videoExtensions = ['.mp4', '.mov', '.avi', '.mkv', '.webm', '.flv', '.wmv']
  return videoExtensions.some(ext => url.toLowerCase().includes(ext))
}

const formattedCaption = computed(() => {
  if (!currentReel.value?.caption) return "";
  return formatRichText(currentReel.value.caption);
});

// Load secure URL for current reel
const loadSecureUrl = async () => {
  if (!currentReel.value?.media_urls || currentReel.value.media_urls.length === 0) {
    loadingMedia.value = false;
    return;
  }

  loadingMedia.value = true;
  try {
    currentMediaUrl.value = await getSecureMediaURL(currentReel.value.media_urls[0]);
  } catch (error) {
    console.error('Failed to load secure media URL:', error);
    currentMediaUrl.value = currentReel.value.media_urls[0]; // Fallback
  } finally {
    loadingMedia.value = false;
  }
};

// Load comments when reel changes
watch(currentIndex, async (newIndex) => {
  if (reels.value[newIndex]) {
    await Promise.all([loadComments(), loadSecureUrl()]);
  }
}, { immediate: true });

// Load on mount
onMounted(() => {
  loadSecureUrl();
});

const loadComments = async () => {
  if (!currentReel.value) return;
  
  try {
    const postIdNum = parseInt(currentReel.value.id);
    if (isNaN(postIdNum)) return;
    
    const response = await commentAPI.getCommentsByPost(postIdNum);
    comments.value = response || [];
  } catch (error) {
    console.error("Failed to load comments:", error);
  }
};

const goToNext = () => {
  if (currentIndex.value < reels.value.length - 1) {
    currentIndex.value++;
    showComments.value = false;
  }
};

const goToPrevious = () => {
  if (currentIndex.value > 0) {
    currentIndex.value--;
    showComments.value = false;
  }
};

const handleScroll = (event: WheelEvent) => {
  if (showComments.value) return; // Don't scroll reels when comments are open
  
  if (event.deltaY > 0) {
    // Scrolling down
    goToNext();
  } else if (event.deltaY < 0) {
    // Scrolling up
    goToPrevious();
  }
};

const handleLike = async () => {
  if (!currentReel.value) return;
  await feedStore.toggleLike(currentReel.value.id, "reels");
};

const handleSave = async () => {
  if (!currentReel.value) return;
  await feedStore.toggleSave(currentReel.value.id, "reels");
};

const handleShare = async () => {
  if (!currentReel.value) return;
  
  const url = `${window.location.origin}/reel/${currentReel.value.id}`;
  try {
    if (navigator.share) {
      await navigator.share({
        title: `Reel by ${currentReel.value.author_username}`,
        text: currentReel.value.caption || "Check out this reel!",
        url: url
      });
    } else {
      await navigator.clipboard.writeText(url);
      alert("Link copied to clipboard!");
    }
  } catch (error) {
    console.error("Share failed:", error);
  }
};

const navigateToProfile = () => {
  if (!currentReel.value) return;
  emit("close");
  // Navigate to profile page
  window.location.href = `/profile/${currentReel.value.author_username}`;
};

const handleAddComment = async () => {
  if (!newComment.value.trim() || isSubmitting.value || !currentReel.value) return;
  
  isSubmitting.value = true;
  try {
    const numericPostId = parseInt(currentReel.value.id);
    if (isNaN(numericPostId)) return;
    
    const response = await commentAPI.createComment({
      post_id: numericPostId,
      content: newComment.value.trim()
    });

    if (response) {
      comments.value.unshift(response);
    }

    if (currentReel.value) {
      feedStore.updatePost(currentReel.value.id, {
        comment_count: (currentReel.value.comment_count || 0) + 1
      } as any);
    }

    newComment.value = "";
  } catch (error) {
    console.error("Failed to add comment:", error);
  } finally {
    isSubmitting.value = false;
  }
};

const formatCount = (count: number) => {
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(1)}M`;
  } else if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}K`;
  }
  return count.toString();
};

const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp);
  const now = new Date();
  const diffInMs = now.getTime() - date.getTime();
  const diffInSecs = Math.floor(diffInMs / 1000);
  const diffInMins = Math.floor(diffInSecs / 60);
  const diffInHours = Math.floor(diffInMins / 60);
  const diffInDays = Math.floor(diffInHours / 24);

  if (diffInDays > 7) {
    return date.toLocaleDateString("en-US", { month: "long", day: "numeric", year: "numeric" });
  } else if (diffInDays > 0) {
    return `${diffInDays} day${diffInDays > 1 ? "s" : ""} ago`;
  } else if (diffInHours > 0) {
    return `${diffInHours} hour${diffInHours > 1 ? "s" : ""} ago`;
  } else if (diffInMins > 0) {
    return `${diffInMins} minute${diffInMins > 1 ? "s" : ""} ago`;
  } else {
    return "Just now";
  }
};
</script>

<style scoped lang="scss">
.reels-viewer-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.95);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.reels-viewer {
  width: 100%;
  max-width: 500px;
  height: 100vh;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-btn {
  position: absolute;
  top: 20px;
  right: 20px;
  background: rgba(0, 0, 0, 0.7);
  border: none;
  color: #fff;
  font-size: 28px;
  cursor: pointer;
  z-index: 10;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    background: rgba(0, 0, 0, 0.9);
  }
}

.reel-container {
  width: 100%;
  height: 100%;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.reel-media {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;

  .reel-image {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }

  .reel-video {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
}

.reel-sidebar {
  position: absolute;
  right: 20px;
  bottom: 80px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  align-items: center;
  z-index: 5;
}

.author-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition: transform 0.2s;

  &:hover {
    transform: scale(1.05);
  }

  .author-avatar {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    border: 2px solid #fff;
    object-fit: cover;
  }

  .author-name {
    font-size: 12px;
    font-weight: 600;
    color: #fff;
    text-align: center;
  }
}

.actions-section {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .action-btn {
    background: none;
    border: none;
    color: #fff;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    transition: transform 0.1s;

    .icon {
      font-size: 28px;
    }

    .count {
      font-size: 12px;
      font-weight: 600;
    }

    &:hover {
      transform: scale(1.1);
    }

    &:active {
      transform: scale(0.95);
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

.caption-section {
  position: absolute;
  bottom: 80px;
  left: 50%;
  transform: translateX(-50%);
  width: 80%;
  max-width: 400px;
  color: #fff;
  text-align: center;

  .caption-text {
    font-size: 14px;
    line-height: 1.4;
    background: rgba(0, 0, 0, 0.7);
    padding: 12px 16px;
    border-radius: 8px;
    backdrop-filter: blur(10px);
  }
}

/* Rich text styles for hashtags and mentions */
:deep(.rich-text-hashtag),
:deep(.rich-text-mention) {
  color: #0095f6;
  font-weight: 600;
  cursor: pointer;
  &:hover {
    text-decoration: underline;
  }
}

.comments-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: flex-end;
  z-index: 20;
}

.comments-panel {
  width: 100%;
  max-height: 70vh;
  background-color: #000;
  border-radius: 16px 16px 0 0;
  display: flex;
  flex-direction: column;
}

.comments-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #262626;

  h3 {
    font-size: 16px;
    font-weight: 600;
  }

  button {
    background: none;
    border: none;
    color: #fff;
    font-size: 24px;
    cursor: pointer;
  }
}

.comments-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;

  .comment-item {
    display: flex;
    gap: 12px;
    margin-bottom: 16px;

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
        margin-bottom: 4px;

        strong {
          margin-right: 4px;
        }
      }

      .comment-time {
        font-size: 12px;
        color: #a8a8a8;
      }
    }
  }
}

.comment-input {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid #262626;
  align-items: center;

  input {
    flex: 1;
    background: #262626;
    border: 1px solid #404040;
    border-radius: 20px;
    color: #fff;
    font-size: 14px;
    padding: 10px 16px;
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

.navigation-indicators {
  position: absolute;
  right: 80px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  gap: 20px;
  z-index: 10;

  .nav-hint {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    color: #fff;
    cursor: pointer;
    opacity: 0.7;
    transition: opacity 0.2s;

    &:hover {
      opacity: 1;
    }

    span {
      font-size: 24px;
    }

    .hint-text {
      font-size: 10px;
      font-weight: 600;
      text-transform: uppercase;
    }

    &.top {
      animation: bounce-up 2s infinite;
    }

    &.bottom {
      animation: bounce-down 2s infinite;
    }
  }
}

@keyframes bounce-up {
  0%, 20%, 50%, 80%, 100% {
    transform: translateY(0);
  }
  40% {
    transform: translateY(-10px);
  }
  60% {
    transform: translateY(-5px);
  }
}

@keyframes bounce-down {
  0%, 20%, 50%, 80%, 100% {
    transform: translateY(0);
  }
  40% {
    transform: translateY(10px);
  }
  60% {
    transform: translateY(5px);
  }
}

@media (max-width: 768px) {
  .reel-sidebar {
    right: 10px;
    bottom: 60px;
  }

  .caption-section {
    right: 80px;
    left: 10px;
    bottom: 10px;
  }

  .navigation-indicators {
    right: 60px;
  }
}
</style>
