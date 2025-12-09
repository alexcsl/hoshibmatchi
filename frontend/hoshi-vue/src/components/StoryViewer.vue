<template>
  <div
    class="story-viewer"
    @click="handleClick"
    @mousedown="handlePauseStart"
    @mouseup="handlePauseEnd"
    @mouseleave="handlePauseEnd"
  >
    <button
      class="close-btn"
      @click.stop="$emit('close')"
    >
      ✕
    </button>

    <div class="story-container">
      <div class="story-image-wrapper">
        <div v-if="loadingMedia" class="media-loading">
          <div class="loading-spinner">Loading story...</div>
        </div>
        <img 
          v-else-if="secureMediaUrl"
          :src="secureMediaUrl" 
          :alt="story.author_username" 
          class="story-image" 
          :style="{ filter: getFilterStyle(story.filter_name) }"
        />
        
        <!-- Pause Indicator -->
        <div
          v-if="isPaused"
          class="pause-indicator"
        >
          <div class="pause-icon">
            ⏸
          </div>
        </div>
        
        <!-- Text Overlay -->
        <div
          v-if="story.caption"
          class="text-overlay"
        >
          {{ story.caption }}
        </div>
        
        <!-- Stickers Overlay -->
        <div 
          v-for="(sticker, idx) in parsedStickers" 
          :key="idx" 
          class="sticker" 
          :style="sticker.style"
        >
          {{ sticker.emoji }}
        </div>
        
        <div class="story-progress">
          <div
            class="progress-bar"
            :style="{ width: progressPercentage + '%' }"
          ></div>
        </div>
      </div>

      <div class="story-header">
        <div class="user-info">
          <img
            :src="story.author_profile_url || '/default-avatar.svg'"
            :alt="story.author_username"
            class="avatar"
          />
          <div>
            <div class="username">
              {{ story.author_username }}
            </div>
            <div class="timestamp">
              {{ formatTime(story.created_at) }}
            </div>
          </div>
        </div>
        <button
          class="more-btn"
          @click.stop
        >
          ⋯
        </button>
      </div>

      <button
        v-if="canGoPrev"
        class="nav-btn prev"
        @click.stop="goToPrev"
      >
        ◀
      </button>
      <button
        v-if="canGoNext"
        class="nav-btn next"
        @click.stop="goToNext"
      >
        ▶
      </button>

      <!-- Action Buttons -->
      <div
        class="story-actions"
        @click.stop
      >
        <button
          class="action-btn"
          :class="{ liked: isLiked }"
          @click="toggleLike"
        >
          <span class="icon">{{ isLiked ? '❤️' : '🤍' }}</span>
        </button>
        <button
          class="action-btn"
          @click="handleShare"
        >
          <span class="icon">📤</span>
        </button>
      </div>

      <div
        class="story-reply"
        @click.stop
      >
        <input 
          v-model="replyMessage" 
          type="text" 
          placeholder="Reply..." 
          class="reply-input"
          @keyup.enter="sendReply"
        />
        <button
          class="send-btn"
          :disabled="!replyMessage.trim()"
          @click="sendReply"
        >
          📤
        </button>
      </div>
    </div>

    <!-- Share Modal -->
    <div
      v-if="showShareModal"
      class="modal-overlay"
      @click.stop="closeShareModal"
    >
      <div
        class="modal-content"
        @click.stop
      >
        <h3>Share Story</h3>
        <p>Choose a recipient:</p>
        <div class="share-options">
          <div
            class="share-option"
            @click="shareToMessages"
          >
            <span class="icon">💬</span>
            <span>Send in Message</span>
          </div>
        </div>
        <button
          class="modal-close-btn"
          @click="closeShareModal"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { storyAPI } from "@/services/api";
import { getSecureMediaURL } from "@/services/media";

const router = useRouter();

// Interface matching Backend Protobuf/JSON
export interface Story {
  id: string
  author_username: string
  author_profile_url: string 
  media_url: string         
  created_at: string
  caption?: string
  filter_name?: string
  stickers_json?: string
  is_liked?: boolean
}

const props = defineProps<{
  stories: Story[]
  initialIndex?: number
}>();

const emit = defineEmits<{
  (e: "close"): void
  (e: "prev"): void
  (e: "next"): void
}>();

const currentIndex = ref(props.initialIndex || 0);
const progress = ref(0);
const storyDuration = 5000; 
const isPaused = ref(false);
const isLiked = ref(false);
const replyMessage = ref("");
const showShareModal = ref(false);

// Secure media URL
const secureMediaUrl = ref<string>("");
const loadingMedia = ref(true);

const story = computed(() => props.stories[currentIndex.value]);
const progressPercentage = computed(() => Math.min((progress.value / storyDuration) * 100, 100));
const canGoPrev = computed(() => currentIndex.value > 0);
const canGoNext = computed(() => currentIndex.value < props.stories.length - 1);

// Update liked status when story changes
const updateLikedStatus = () => {
  isLiked.value = story.value.is_liked || false;
};

// Parse stickers from JSON
const parsedStickers = computed(() => {
  try {
    if (story.value.stickers_json) {
      return JSON.parse(story.value.stickers_json);
    }
  } catch (e) {
    console.error("Failed to parse stickers:", e);
  }
  return [];
});

// Get filter CSS
const getFilterStyle = (filterName?: string) => {
  if (!filterName || filterName === "None") return "none";
  const filters: Record<string, string> = {
    "Grayscale": "grayscale(100%)",
    "Sepia": "sepia(100%)",
    "Bright": "brightness(1.3)",
    "Contrast": "contrast(1.5)",
    "Blur": "blur(5px)"
  };
  return filters[filterName] || "none";
};

// Helper for media URLs
const getMediaUrl = (url: string) => {
  if (!url) return "";
  if (url.startsWith("http")) return url;
  if (url.startsWith("/uploads/") || url.startsWith("uploads/")) {
    return `http://localhost:8000${url.startsWith("/") ? url : "/" + url}`;
  }
  return url;
};

// Helper for timestamp
const formatTime = (dateStr: string) => {
  const date = new Date(dateStr);
  const now = new Date();
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000); // seconds

  if (diff < 60) return "Just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
  return `${Math.floor(diff / 86400)}d`;
};

let interval: number | null = null;

const startProgress = () => {
  if (isPaused.value) return;
  stopProgress(); // Ensure no duplicate intervals
  progress.value = 0;
  interval = setInterval(() => {
    if (!isPaused.value) {
      progress.value += 100;
      if (progress.value >= storyDuration) {
        if (canGoNext.value) {
          goToNext();
        } else {
          emit("close");
        }
      }
    }
  }, 100);
};

const stopProgress = () => {
  if (interval) {
    clearInterval(interval);
    interval = null;
  }
};

const handlePauseStart = () => {
  isPaused.value = true;
};

const handlePauseEnd = () => {
  isPaused.value = false;
};

const goToNext = () => {
  if (canGoNext.value) {
    currentIndex.value++;
    emit("next");
    updateLikedStatus();
    startProgress();
  } else {
    emit("close");
  }
};

const goToPrev = () => {
  if (canGoPrev.value) {
    currentIndex.value--;
    emit("prev");
    updateLikedStatus();
    startProgress();
  } else {
      // Restart current story if at beginning
      startProgress();
  }
};

const handleClick = (event: MouseEvent) => {
  // Don't navigate if clicking on buttons or inputs
  const target = event.target as HTMLElement;
  if (target.closest(".action-btn, .reply-input, .send-btn, .nav-btn, .close-btn, .more-btn")) {
    return;
  }

  const element = event.currentTarget as HTMLElement;
  const rect = element.getBoundingClientRect();
  const x = event.clientX - rect.left;

  if (x > rect.width / 2) {
    goToNext();
  } else {
    goToPrev();
  }
};

// Like functionality
const toggleLike = async () => {
  try {
    if (isLiked.value) {
      await storyAPI.unlikeStory(story.value.id);
      isLiked.value = false;
    } else {
      await storyAPI.likeStory(story.value.id);
      isLiked.value = true;
    }
  } catch (error) {
    console.error("Failed to toggle like:", error);
  }
};

// Reply functionality
const sendReply = () => {
  if (!replyMessage.value.trim()) return;
  
  // Navigate to messages with the story author
  router.push({
    name: "Messages",
    query: {
      user: story.value.author_username,
      message: `Story reply: ${replyMessage.value}`
    }
  });
  
  replyMessage.value = "";
  emit("close");
};

// Share functionality
const handleShare = () => {
  showShareModal.value = true;
};

const closeShareModal = () => {
  showShareModal.value = false;
};

const shareToMessages = () => {
  // Navigate to messages to share the story
  router.push({
    name: "Messages",
    query: {
      shareStory: story.value.id,
      shareUrl: getMediaUrl(story.value.media_url)
    }
  });
  
  closeShareModal();
  emit("close");
};

// Load secure media URL when story changes
const loadSecureMedia = async () => {
  loadingMedia.value = true;
  secureMediaUrl.value = "";
  
  try {
    if (story.value.media_url) {
      secureMediaUrl.value = await getSecureMediaURL(story.value.media_url);
    }
  } catch (error) {
    console.error("Failed to load story media:", error);
  } finally {
    loadingMedia.value = false;
  }
};

// Watch for story changes
watch(story, () => {
  loadSecureMedia();
  updateLikedStatus();
  startProgress();
}, { immediate: true });

onMounted(() => {
  updateLikedStatus();
  startProgress();
});

onUnmounted(() => {
  stopProgress();
});
</script>

<style scoped lang="scss">
.story-viewer {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.95);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 101;
}

.close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  background: none;
  border: none;
  color: #fff;
  font-size: 24px;
  cursor: pointer;
  z-index: 10;
}

.story-container {
  position: relative;
  width: 100%;
  max-width: 500px;
  height: 100vh;
  max-height: 800px;
  display: flex;
  flex-direction: column;
}

.story-image-wrapper {
  position: relative;
  flex: 1;
  overflow: hidden;

  .media-loading {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.8);

    .loading-spinner {
      color: #fff;
      font-size: 16px;
    }
  }

  .story-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .text-overlay {
    position: absolute;
    bottom: 80px;
    left: 16px;
    right: 16px;
    text-align: center;
    color: #fff;
    font-size: 24px;
    font-weight: 600;
    text-shadow: 0 2px 8px rgba(0, 0, 0, 0.8);
    word-wrap: break-word;
    padding: 12px;
    z-index: 2;
  }

  .sticker {
    position: absolute;
    user-select: none;
    z-index: 2;
    pointer-events: none;
  }

  .story-progress {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 3px;
    background-color: rgba(255, 255, 255, 0.3);

    .progress-bar {
      height: 100%;
      background-color: #fff;
      transition: width 0.1s linear;
    }
  }

  .pause-indicator {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 10;

    .pause-icon {
      font-size: 48px;
      color: rgba(255, 255, 255, 0.9);
      text-shadow: 0 2px 8px rgba(0, 0, 0, 0.8);
      animation: pulse 1s ease-in-out infinite;
    }
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 1;
      transform: scale(1);
    }
    50% {
      opacity: 0.7;
      transform: scale(1.1);
    }
  }
}

.story-header {
  position: absolute;
  top: 12px;
  left: 12px;
  right: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  z-index: 5;

  .user-info {
    display: flex;
    align-items: center;
    gap: 12px;

    .avatar {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      border: 2px solid #fff;
      object-fit: cover;
    }

    .username {
      font-weight: 600;
      font-size: 14px;
      color: #fff;
    }

    .timestamp {
      font-size: 12px;
      color: rgba(255, 255, 255, 0.7);
    }
  }

  .more-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
  }
}

.nav-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.8);
  font-size: 24px;
  cursor: pointer;
  padding: 12px;
  z-index: 5;
  transition: color 0.2s;

  &:hover {
    color: #fff;
  }

  &.prev {
    left: 12px;
  }

  &.next {
    right: 12px;
  }
}

.story-actions {
  position: absolute;
  right: 16px;
  bottom: 90px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  z-index: 5;

  .action-btn {
    background: rgba(0, 0, 0, 0.5);
    border: none;
    border-radius: 50%;
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: transform 0.2s, background 0.2s;

    .icon {
      font-size: 24px;
    }

    &:hover {
      transform: scale(1.1);
      background: rgba(0, 0, 0, 0.7);
    }

    &.liked {
      animation: heartBeat 0.3s ease;
    }
  }

  @keyframes heartBeat {
    0%, 100% {
      transform: scale(1);
    }
    25% {
      transform: scale(1.3);
    }
    50% {
      transform: scale(1.1);
    }
  }
}

.story-reply {
  position: absolute;
  bottom: 20px;
  left: 12px;
  right: 12px;
  display: flex;
  gap: 12px;
  z-index: 5;

  .reply-input {
    flex: 1;
    background-color: rgba(255, 255, 255, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.3);
    border-radius: 20px;
    padding: 10px 16px;
    color: #fff;
    font-size: 14px;
    outline: none;

    &::placeholder {
      color: rgba(255, 255, 255, 0.6);
    }

    &:focus {
      background-color: rgba(255, 255, 255, 0.3);
    }
  }

  .send-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 18px;
    cursor: pointer;
    transition: opacity 0.2s;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    &:not(:disabled):hover {
      transform: scale(1.1);
    }
  }
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: #262626;
  border-radius: 12px;
  padding: 24px;
  max-width: 400px;
  width: 90%;
  color: #fff;

  h3 {
    margin: 0 0 12px 0;
    font-size: 20px;
    font-weight: 600;
  }

  p {
    margin: 0 0 16px 0;
    color: #a8a8a8;
    font-size: 14px;
  }

  .share-options {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 16px;

    .share-option {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px;
      background: #363636;
      border-radius: 8px;
      cursor: pointer;
      transition: background 0.2s;

      .icon {
        font-size: 24px;
      }

      &:hover {
        background: #404040;
      }
    }
  }

  .modal-close-btn {
    width: 100%;
    padding: 12px;
    background: #363636;
    border: none;
    border-radius: 8px;
    color: #fff;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;

    &:hover {
      background: #404040;
    }
  }
}
</style>
