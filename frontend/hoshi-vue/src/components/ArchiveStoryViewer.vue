<template>
  <div
    class="story-viewer"
    @click="handleClick"
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
        <video
          v-else-if="isVideoStory && secureMediaUrl"
          ref="videoElement"
          :src="secureMediaUrl"
          class="story-video"
          :style="{ filter: getFilterStyle(story.filter_name) }"
          @loadedmetadata="onVideoLoaded"
          @ended="onVideoEnded"
          playsinline
          muted
        />
        <img 
          v-else-if="secureMediaUrl"
          :src="secureMediaUrl" 
          :alt="story.author_username" 
          class="story-image" 
          :style="{ filter: getFilterStyle(story.filter_name) }"
        />
        
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
            <!-- Show full date for archive instead of relative time -->
            <div class="timestamp">
              {{ formatFullDate(story.created_at) }}
            </div>
          </div>
        </div>
        <button class="more-btn">
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

      <!-- Archive-specific info banner -->
      <div class="archive-banner">
        <span class="archive-icon">📦</span>
        <span>Archived Story</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from "vue";
import { getSecureMediaURL } from "@/services/media";

// Interface matching Backend Protobuf/JSON
export interface Story {
  id: string
  author_username: string
  author_profile_url: string 
  media_url: string
  media_type: string
  created_at: string
  caption?: string
  filter_name?: string
  stickers_json?: string
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
const storyDuration = ref(5000);
const videoElement = ref<HTMLVideoElement | null>(null);
const isVideoStory = ref(false); 

// Secure media URL
const secureMediaUrl = ref<string>("");
const loadingMedia = ref(true);

const story = computed(() => props.stories[currentIndex.value]);
const progressPercentage = computed(() => Math.min((progress.value / storyDuration.value) * 100, 100));
const canGoPrev = computed(() => currentIndex.value > 0);
const canGoNext = computed(() => currentIndex.value < props.stories.length - 1);

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

// Check if media is video
const checkIfVideo = () => {
  const mediaType = story.value.media_type || '';
  return mediaType.includes('video') || mediaType === 'mp4' || mediaType === 'webm';
};

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

// Video handlers
const onVideoLoaded = () => {
  if (videoElement.value) {
    const duration = videoElement.value.duration;
    if (duration && !isNaN(duration)) {
      storyDuration.value = duration * 1000;
    }
    videoElement.value.play();
  }
};

const onVideoEnded = () => {
  if (canGoNext.value) {
    goToNext();
  } else {
    emit('close');
  }
};

// Format full date for archive (e.g., "Jan 15, 2024 at 3:45 PM")
const formatFullDate = (dateStr: string) => {
  const date = new Date(dateStr);
  const dateOptions: Intl.DateTimeFormatOptions = { 
    month: "short", 
    day: "numeric", 
    year: "numeric" 
  };
  const timeOptions: Intl.DateTimeFormatOptions = { 
    hour: "numeric", 
    minute: "2-digit",
    hour12: true
  };
  const datePart = date.toLocaleDateString("en-US", dateOptions);
  const timePart = date.toLocaleTimeString("en-US", timeOptions);
  return `${datePart} at ${timePart}`;
};

let interval: number | null = null;

const startProgress = () => {
  stopProgress(); // Ensure no duplicate intervals
  progress.value = 0;
  interval = setInterval(() => {
    progress.value += 100;
    if (progress.value >= storyDuration.value) {
      if (canGoNext.value) {
        goToNext();
      } else {
        emit("close");
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

const goToNext = () => {
  if (videoElement.value && isVideoStory.value) {
    videoElement.value.pause();
    videoElement.value.currentTime = 0;
  }
  
  if (canGoNext.value) {
    currentIndex.value++;
    emit("next");
    startProgress();
  } else {
    emit("close");
  }
};

const goToPrev = () => {
  if (videoElement.value && isVideoStory.value) {
    videoElement.value.pause();
    videoElement.value.currentTime = 0;
  }
  
  if (canGoPrev.value) {
    currentIndex.value--;
    emit("prev");
    startProgress();
  } else {
      // Restart current story if at beginning
      startProgress();
  }
};

const handleClick = (event: MouseEvent) => {
  const element = event.currentTarget as HTMLElement;
  const rect = element.getBoundingClientRect();
  const x = event.clientX - rect.left;

  if (x > rect.width / 2) {
    goToNext();
  } else {
    goToPrev();
  }
};

// Load secure media URL when story changes
const loadSecureMedia = async () => {
  loadingMedia.value = true;
  secureMediaUrl.value = "";
  
  // Check if current story is video
  isVideoStory.value = checkIfVideo();
  
  // Reset duration for images
  if (!isVideoStory.value) {
    storyDuration.value = 5000;
  }
  
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
  startProgress();
}, { immediate: true });

onMounted(() => {
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

  .story-video {
    width: 100%;
    height: 100%;
    object-fit: cover;
    
    &::-webkit-media-controls {
      display: none !important;
    }
    
    &::-webkit-media-controls-enclosure {
      display: none !important;
    }
    
    pointer-events: none;
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
      color: rgba(255, 255, 255, 0.8);
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

.archive-banner {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  background-color: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(10px);
  padding: 8px 20px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  z-index: 5;

  .archive-icon {
    font-size: 16px;
  }
}
</style>
