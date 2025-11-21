<template>
  <div class="archive-page">
    <div class="archive-container">
      <h1>Archive</h1>
      <p class="subtitle">Only you can see your archived stories</p>

      <div v-if="loading" class="loading-state">
        <p>Loading your archive...</p>
      </div>

      <div v-else-if="stories.length === 0" class="empty-state">
        <p>No stories in your archive yet</p>
        <p class="empty-subtitle">Your stories will be saved here after 24 hours</p>
      </div>

      <div v-else class="archive-grid">
        <div 
          v-for="story in stories" 
          :key="story.id" 
          class="archive-item"
          @click="openStory(story)"
        >
          <img 
            :src="getMediaUrl(story.media_url)" 
            :alt="'Archived story ' + story.id" 
            :style="{ filter: getFilterStyle(story.filter_name) }"
          />
          <div class="archive-overlay">
            <div class="story-date">{{ formatDate(story.created_at) }}</div>
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
import { ref, onMounted } from 'vue'
import { storyAPI, type Story } from '@/services/api'
import ArchiveStoryViewer from '@/components/ArchiveStoryViewer.vue'

const stories = ref<Story[]>([])
const loading = ref(true)
const showStoryViewer = ref(false)
const selectedStory = ref<Story | null>(null)
const selectedStoryIndex = ref(0)

const getMediaUrl = (url: string) => {
  if (!url) return '/placeholder.svg'
  if (url.startsWith('http')) return url
  if (url.startsWith('/uploads/') || url.startsWith('uploads/')) {
    return `http://localhost:8000${url.startsWith('/') ? url : '/' + url}`
  }
  return url
}

const getFilterStyle = (filterName?: string) => {
  if (!filterName || filterName === 'None') return 'none'
  const filters: Record<string, string> = {
    'Grayscale': 'grayscale(100%)',
    'Sepia': 'sepia(100%)',
    'Bright': 'brightness(1.3)',
    'Contrast': 'contrast(1.5)',
    'Blur': 'blur(5px)'
  }
  return filters[filterName] || 'none'
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  const options: Intl.DateTimeFormatOptions = { 
    month: 'short', 
    day: 'numeric', 
    year: 'numeric' 
  }
  return date.toLocaleDateString('en-US', options)
}

const openStory = (story: Story) => {
  selectedStory.value = story
  selectedStoryIndex.value = stories.value.findIndex(s => s.id === story.id)
  showStoryViewer.value = true
}

const closeStoryViewer = () => {
  showStoryViewer.value = false
  selectedStory.value = null
}

const fetchArchive = async () => {
  try {
    loading.value = true
    const data = await storyAPI.getArchive()
    stories.value = data
  } catch (error) {
    console.error('Failed to fetch archive:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchArchive()
})
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

  h1 {
    font-size: 32px;
    font-weight: 700;
    margin-bottom: 8px;
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

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.2s;
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
  }

  &:hover {
    img {
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
