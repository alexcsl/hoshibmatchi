<template>
  <div class="archive-page">
    <div class="archive-container">
      <div class="header-with-back">
        <button
          class="back-button"
          @click="$router.back()"
        >
          ← Back
        </button>
        <h1>Archive</h1>
      </div>
      <p class="subtitle">
        Only you can see your archived stories
      </p>

      <div
        v-if="loading"
        class="loading-state"
      >
        <p>Loading your archive...</p>
      </div>

      <div
        v-else-if="stories.length === 0"
        class="empty-state"
      >
        <p>No stories in your archive yet</p>
        <p class="empty-subtitle">
          Your stories will be saved here after 24 hours
        </p>
      </div>

      <div
        v-else
        class="archive-grid"
      >
        <div 
          v-for="story in stories" 
          :key="story.id" 
          class="archive-item"
          @click="openStory(story)"
        >
          <SecureImage
            v-if="!isVideo(story.media_type)"
            :src="story.media_url" 
            :alt="'Archived story ' + story.id" 
            class-name="archive-media"
            :style="{ filter: getFilterStyle(story.filter_name) }"
            loading-placeholder="/placeholder.svg"
            error-placeholder="/placeholder.svg"
          />
          <video
            v-else
            :src="story.media_url"
            class="archive-media"
            :style="{ filter: getFilterStyle(story.filter_name) }"
            muted
            playsinline
          />
          <div class="archive-overlay">
            <div class="story-date">
              {{ formatDate(story.created_at) }}
            </div>
            <div v-if="isVideo(story.media_type)" class="video-indicator">▶</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Story Viewer for Archive -->
    <ArchiveStoryViewer 
      v-if="showStoryViewer && selectedStory"
      :stories="stories"
      :initial-index="selectedStoryIndex"
      @close="closeStoryViewer"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { storyAPI, type Story } from "@/services/api";
import ArchiveStoryViewer from "@/components/ArchiveStoryViewer.vue";
import SecureImage from "@/components/SecureImage.vue";

const stories = ref<Story[]>([]);
const loading = ref(true);
const showStoryViewer = ref(false);
const selectedStory = ref<Story | null>(null);
const selectedStoryIndex = ref(0);

const isVideo = (mediaType: string) => {
  return mediaType && (mediaType.includes('video') || mediaType === 'mp4' || mediaType === 'webm');
};

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

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr);
  const options: Intl.DateTimeFormatOptions = { 
    month: "short", 
    day: "numeric", 
    year: "numeric" 
  };
  return date.toLocaleDateString("en-US", options);
};

const openStory = (story: Story) => {
  selectedStory.value = story;
  selectedStoryIndex.value = stories.value.findIndex(s => s.id === story.id);
  showStoryViewer.value = true;
};

const closeStoryViewer = () => {
  showStoryViewer.value = false;
  selectedStory.value = null;
};

const fetchArchive = async () => {
  try {
    loading.value = true;
    const data = await storyAPI.getArchive();
    stories.value = data;
  } catch (error) {
    console.error("Failed to fetch archive:", error);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchArchive();
});
</script>

<style scoped lang="scss">
.archive-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.archive-container {
  max-width: 1200px;

  .header-with-back {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 8px;

    .back-button {
      background: none;
      border: none;
      color: #fff;
      font-size: 16px;
      cursor: pointer;
      padding: 8px 16px;
      border-radius: 8px;
      transition: background-color 0.2s;

      &:hover {
        background-color: rgba(255, 255, 255, 0.1);
      }
    }
  }

  h1 {
    font-size: 32px;
    font-weight: 700;
    margin-bottom: 0;
  }

  .subtitle {
    color: #a8a8a8;
    font-size: 14px;
    margin-bottom: 32px;
  }
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #a8a8a8;

  p {
    font-size: 18px;
    margin-bottom: 8px;
  }

  .empty-subtitle {
    font-size: 14px;
    color: #666;
  }
}

.archive-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.archive-item {
  position: relative;
  aspect-ratio: 1;
  cursor: pointer;
  overflow: hidden;
  border-radius: 4px;
  background-color: #262626;

  .archive-media {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.2s;
    display: block;
  }

  .archive-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: linear-gradient(to bottom, rgba(0, 0, 0, 0) 60%, rgba(0, 0, 0, 0.8) 100%);
    display: flex;
    align-items: flex-end;
    justify-content: center;
    padding: 16px;
    opacity: 0;
    transition: all 0.2s;

    .story-date {
      color: #fff;
      font-size: 14px;
      font-weight: 600;
      text-shadow: 0 1px 4px rgba(0, 0, 0, 0.5);
    }

    .video-indicator {
      position: absolute;
      top: 12px;
      right: 12px;
      color: #fff;
      font-size: 20px;
      text-shadow: 0 1px 4px rgba(0, 0, 0, 0.5);
    }
  }

  &:hover {
    .archive-media {
      transform: scale(1.05);
    }

    .archive-overlay {
      opacity: 1;
    }
  }
}

@media (max-width: 1024px) {
  .archive-page {
    padding-left: calc(72px + 20px);
  }
}

@media (max-width: 768px) {
  .archive-page {
    padding-left: calc(60px + 20px);
  }

  .archive-grid {
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  }
}
</style>
