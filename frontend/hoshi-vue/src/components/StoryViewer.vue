<template>
  <div class="story-viewer" @click="handleClick">
    <button class="close-btn" @click.stop="$emit('close')">✕</button>

    <div class="story-container">
      <!-- Story Image -->
      <div class="story-image-wrapper">
        <img :src="story.image" :alt="story.username" class="story-image" />
        <div class="story-progress">
          <div class="progress-bar" :style="{ width: progressPercentage + '%' }"></div>
        </div>
      </div>

      <!-- Story Header -->
      <div class="story-header">
        <div class="user-info">
          <img :src="story.avatar" :alt="story.username" class="avatar" />
          <div>
            <div class="username">{{ story.username }}</div>
            <div class="timestamp">{{ story.timestamp }}</div>
          </div>
        </div>
        <button class="more-btn">⋯</button>
      </div>

      <!-- Navigation -->
      <button v-if="canGoPrev" class="nav-btn prev" @click.stop="goToPrev">◀</button>
      <button v-if="canGoNext" class="nav-btn next" @click.stop="goToNext">▶</button>

      <!-- Story Reply -->
      <div class="story-reply">
        <input type="text" placeholder="Reply..." class="reply-input" />
        <button class="send-btn">📤</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'

interface Story {
  id: number
  username: string
  avatar: string
  image: string
  timestamp: string
}

const props = defineProps<{
  stories: Story[]
  initialIndex?: number
}>()

const emit = defineEmits<{
  close: []
  prev: []
  next: []
}>()

const currentIndex = ref(props.initialIndex || 0)
const progress = ref(0)
const storyDuration = 5000 // 5 seconds per story

const story = computed(() => props.stories[currentIndex.value])
const progressPercentage = computed(() => Math.min((progress.value / storyDuration) * 100, 100))
const canGoPrev = computed(() => currentIndex.value > 0)
const canGoNext = computed(() => currentIndex.value < props.stories.length - 1)

let interval: number | null = null

const startProgress = () => {
  progress.value = 0
  interval = setInterval(() => {
    progress.value += 100
    if (progress.value >= storyDuration) {
      if (canGoNext.value) {
        goToNext()
      } else {
        emit('close')
      }
    }
  }, 100)
}

const stopProgress = () => {
  if (interval) {
    clearInterval(interval)
  }
}

const goToNext = () => {
  if (canGoNext.value) {
    currentIndex.value++
    emit('next')
    progress.value = 0
    startProgress()
  }
}

const goToPrev = () => {
  if (canGoPrev.value) {
    currentIndex.value--
    emit('prev')
    progress.value = 0
    startProgress()
  }
}

const handleClick = () => {
  // Tap right side to go next
  const element = event?.target as HTMLElement
  const rect = element.getBoundingClientRect()
  const x = (event as MouseEvent).clientX - rect.left

  if (x > rect.width / 2) {
    goToNext()
  } else {
    goToPrev()
  }
}

onMounted(() => {
  startProgress()
})

onUnmounted(() => {
  stopProgress()
})
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

  .story-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
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
  }
}
</style>
