<template>
  <div class="create-story-page">
    <div class="story-editor-container">
      <!-- Left Panel: Upload Area -->
      <div class="upload-panel">
        <!-- Upload Area - shown when no media selected -->
        <div v-show="!storyMedia" class="upload-area" @dragover="handleDragOver" @dragleave="handleDragLeave" @drop="handleDrop">
          <div :class="['upload-content', { 'dragging': isDragging }]">
            <div class="upload-icon">📸</div>
            <h2>Create a story</h2>
            <p>Upload a photo or video to get started</p>
            <button class="upload-btn" @click="triggerFileUpload">Choose from device</button>
          </div>
          <input 
            ref="fileInput" 
            type="file" 
            accept="image/*,video/*" 
            @change="handleFileUpload"
            style="display: none"
          />
        </div>

        <!-- Story Preview - shown when media selected -->
        <div v-show="storyMedia" class="story-preview-container">
          <div class="story-preview" :style="{ filter: getFilterStyle(currentFilter) }">
            <img v-show="storyType === 'image'" :src="storyMedia" alt="Story preview" class="preview-media" />
            <video v-show="storyType === 'video'" :src="storyMedia" class="preview-media" @loadedmetadata="updateVideoDuration"></video>
            
            <!-- Text Overlay -->
            <div v-show="textOverlay.text" :class="['text-overlay', `text-${textOverlay.position}`]" :style="textOverlayStyle">
              {{ textOverlay.text }}
            </div>

            <!-- Stickers Overlay -->
            <div v-for="(sticker, idx) in stickers" :key="idx" class="sticker" :style="sticker.style">
              {{ sticker.emoji }}
            </div>
          </div>

          <!-- Change Media Button -->
          <button class="change-media-btn" @click="resetMedia">Change Media</button>
        </div>
      </div>

      <!-- Right Panel: Tools -->
      <div class="tools-panel">
        <div class="tools-header">
          <h3>Story Editor</h3>
          <button class="close-btn" @click="goBack">✕</button>
        </div>

        <div class="tools-section">
          <!-- Text Tool -->
          <div class="tool-group">
            <label class="tool-label">
              <span class="tool-icon">Aa</span>
              <span>Add Text</span>
            </label>
            <textarea 
              v-model="textOverlay.text"
              placeholder="Add text to your story"
              class="text-input"
              maxlength="200"
            ></textarea>
            <p class="char-count">{{ textOverlay.text.length }}/200</p>

            <!-- Text Position -->
            <div class="position-selector">
              <button 
                v-for="pos in positionOptions"
                :key="pos"
                :class="['pos-btn', { active: textOverlay.position === pos }]"
                @click="textOverlay.position = pos as any"
              >
                {{ pos }}
              </button>
            </div>

            <!-- Text Color -->
            <div class="color-picker">
              <label>Text Color</label>
              <div class="color-options">
                <button 
                  v-for="color in textColors"
                  :key="color"
                  :class="['color-btn', { active: textOverlay.color === color }]"
                  :style="{ backgroundColor: color }"
                  @click="textOverlay.color = color"
                ></button>
              </div>
            </div>
          </div>

          <!-- Stickers Tool -->
          <div class="tool-group">
            <label class="tool-label">
              <span class="tool-icon">✨</span>
              <span>Add Stickers</span>
            </label>
            <div class="sticker-grid">
              <button 
                v-for="emoji in stickersLibrary"
                :key="emoji"
                class="sticker-btn"
                @click="addSticker(emoji)"
              >
                {{ emoji }}
              </button>
            </div>
          </div>

          <!-- Filters Tool -->
          <div class="tool-group">
            <label class="tool-label">
              <span class="tool-icon">🎨</span>
              <span>Filters</span>
            </label>
            <div class="filter-grid">
              <button 
                v-for="filter in filters"
                :key="filter.name"
                :class="['filter-btn', { active: currentFilter === filter.name }]"
                @click="applyFilter(filter.name)"
              >
                {{ filter.name }}
              </button>
            </div>
          </div>

          <!-- Story Settings -->
          <div class="tool-group settings-group">
            <label class="checkbox-label">
              <input v-model="storySettings.allowReplies" type="checkbox" />
              <span>Allow replies</span>
            </label>
            <label class="checkbox-label">
              <input v-model="storySettings.hideViews" type="checkbox" />
              <span>Hide views</span>
            </label>
            <label class="checkbox-label">
              <input v-model="storySettings.saveToDrafts" type="checkbox" />
              <span>Save to drafts</span>
            </label>
          </div>

          <!-- Action Buttons -->
          <div class="action-buttons">
            <button class="discard-btn" @click="goBack">Discard</button>
            <button class="share-btn" @click="handleShareStory" :disabled="isLoading">
              {{ isLoading ? 'Sharing...' : 'Share Story' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<!-- eslint-disable-next-line vue/no-setup-props-destructure -->
<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const fileInput = ref<HTMLInputElement>()
const isDragging = ref(false)
const isLoading = ref(false)
const storyMedia = ref<string>('')
const storyType = ref<'image' | 'video'>('image')
const currentFilter = ref('None')
const videoDuration = ref(0)

const textOverlay = reactive({
  text: '',
  position: 'bottom',
  color: '#ffffff',
  fontSize: 24,
})

const stickers = ref<Array<{ emoji: string; style: Record<string, any> }>>([])

const storySettings = reactive({
  allowReplies: true,
  hideViews: false,
  saveToDrafts: false,
})

const textColors = ['#ffffff', '#000000', '#ff0000', '#00ff00', '#0000ff', '#ffff00', '#ff00ff', '#00ffff']
const positionOptions = ['top', 'middle', 'bottom']
const stickersLibrary = ['😀', '😂', '❤️', '👍', '🔥', '✨', '🎉', '🎊', '🌟', '💯', '👏', '🙌', '💕', '😍', '🤔', '😎']
const filters = [
  { name: 'None', css: 'filter: none' },
  { name: 'Grayscale', css: 'filter: grayscale(100%)' },
  { name: 'Sepia', css: 'filter: sepia(100%)' },
  { name: 'Bright', css: 'filter: brightness(1.3)' },
  { name: 'Contrast', css: 'filter: contrast(1.5)' },
  { name: 'Blur', css: 'filter: blur(5px)' },
]

const textOverlayStyle = computed(() => ({
  color: textOverlay.color,
  fontSize: `${textOverlay.fontSize}px`,
}))

const getFilterStyle = (filterName: string) => {
  const filter = filters.find(f => f.name === filterName)
  return filter?.css || 'filter: none'
}

const triggerFileUpload = () => {
  fileInput.value?.click()
}

const handleFileUpload = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  
  if (file) {
    const reader = new FileReader()
    reader.onload = (e) => {
      storyMedia.value = e.target?.result as string
      storyType.value = file.type.startsWith('video') ? 'video' : 'image'
    }
    reader.readAsDataURL(file)
  }
}

const handleDrop = (event: DragEvent) => {
  event.preventDefault()
  isDragging.value = false
  
  const file = event.dataTransfer?.files?.[0]
  if (file) {
    const reader = new FileReader()
    reader.onload = (e) => {
      storyMedia.value = e.target?.result as string
      storyType.value = file.type.startsWith('video') ? 'video' : 'image'
    }
    reader.readAsDataURL(file)
  }
}

const updateVideoDuration = (event: Event) => {
  const video = event.target as HTMLVideoElement
  videoDuration.value = Math.floor(video.duration)
}

const addSticker = (emoji: string) => {
  stickers.value.push({
    emoji,
    style: {
      top: `${Math.random() * 70 + 10}%`,
      left: `${Math.random() * 70 + 10}%`,
      fontSize: '48px',
      cursor: 'move',
    },
  })
}

const applyFilter = (filterName: string) => {
  currentFilter.value = filterName
}

const resetMedia = () => {
  storyMedia.value = ''
  storyType.value = 'image'
  textOverlay.text = ''
  stickers.value = []
  currentFilter.value = 'None'
}

const handleShareStory = async () => {
  if (!storyMedia.value) {
    alert('Please select an image or video first')
    return
  }

  isLoading.value = true
  try {
    console.log('Story data:', {
      media: storyMedia.value,
      type: storyType.value,
      textOverlay,
      stickers: stickers.value,
      filter: currentFilter.value,
      settings: storySettings,
    })

    await new Promise(resolve => setTimeout(resolve, 1500))

    alert('Story shared successfully!')
    router.push('/feed')
  } catch (error) {
    console.error('Error sharing story:', error)
    alert('Error sharing story. Please try again.')
  } finally {
    isLoading.value = false
  }
}

const goBack = () => {
  if (storyMedia.value) {
    if (confirm('Discard this story?')) {
      router.back()
    }
  } else {
    router.back()
  }
}

const handleDragOver = () => {
  isDragging.value = true
}

const handleDragLeave = () => {
  isDragging.value = false
}
</script>

<style scoped lang="scss">
.create-story-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.story-editor-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 32px;
  max-width: 1400px;
  margin: 0 auto;
  height: calc(100vh - 40px);
}

// ===== Upload Panel =====
.upload-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.upload-area {
  width: 100%;
  height: 100%;
  border: 2px dashed #404040;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #1a1a1a;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: #606060;
  }

  &.dragging {
    border-color: #0a66c2;
    background-color: rgba(10, 102, 194, 0.1);
  }
}

.upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  text-align: center;

  &.dragging {
    transform: scale(1.05);
  }

  .upload-icon {
    font-size: 64px;
  }

  h2 {
    font-size: 24px;
    font-weight: 600;
    color: #fff;
  }

  p {
    color: #a8a8a8;
    font-size: 14px;
  }
}

.upload-btn {
  background-color: #0a66c2;
  border: none;
  color: #fff;
  padding: 10px 32px;
  border-radius: 24px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background-color: #0857a6;
  }

  &:active {
    transform: scale(0.98);
  }
}

// ===== Story Preview =====
.story-preview-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.story-preview {
  position: relative;
  width: 100%;
  aspect-ratio: 9/16;
  background-color: #1a1a1a;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #262626;
}

.preview-media {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.text-overlay {
  position: absolute;
  left: 16px;
  right: 16px;
  text-align: center;
  font-weight: 600;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
  word-wrap: break-word;
  padding: 12px;

  &.text-top {
    top: 32px;
  }

  &.text-middle {
    top: 50%;
    transform: translateY(-50%);
  }

  &.text-bottom {
    bottom: 32px;
  }
}

.sticker {
  position: absolute;
  user-select: none;
  cursor: grab;

  &:active {
    cursor: grabbing;
  }
}

.change-media-btn {
  width: 100%;
  background-color: #262626;
  border: 1px solid #404040;
  color: #fff;
  padding: 10px 16px;
  border-radius: 24px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background-color: #404040;
  }
}

// ===== Tools Panel =====
.tools-panel {
  display: flex;
  flex-direction: column;
  background-color: #1a1a1a;
  border-radius: 12px;
  border: 1px solid #262626;
  padding: 24px;
  overflow-y: auto;
  max-height: calc(100vh - 40px);
}

.tools-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid #262626;

  h3 {
    font-size: 18px;
    font-weight: 600;
    color: #fff;
  }

  .close-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
    padding: 0;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;

    &:hover {
      color: #a8a8a8;
    }
  }
}

.tools-section {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.tool-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tool-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
  color: #fff;

  .tool-icon {
    font-size: 18px;
  }
}

.text-input {
  width: 100%;
  padding: 10px 16px;
  background-color: #262626;
  border: 1px solid #404040;
  border-radius: 8px;
  color: #fff;
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  min-height: 80px;
  outline: none;
  transition: all 0.2s;

  &:focus {
    border-color: #818384;
    background-color: #1a1a1a;
  }

  &::placeholder {
    color: #a8a8a8;
  }
}

.char-count {
  font-size: 12px;
  color: #a8a8a8;
  text-align: right;
}

// ===== Position Selector =====
.position-selector {
  display: flex;
  gap: 8px;

  .pos-btn {
    flex: 1;
    padding: 8px 12px;
    background-color: #262626;
    border: 1px solid #404040;
    color: #fff;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    text-transform: capitalize;

    &:hover {
      background-color: #404040;
    }

    &.active {
      background-color: #0a66c2;
      border-color: #0a66c2;
    }
  }
}

// ===== Color Picker =====
.color-picker {
  display: flex;
  flex-direction: column;
  gap: 8px;

  label {
    font-size: 13px;
    color: #a8a8a8;
  }
}

.color-options {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.color-btn {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    transform: scale(1.1);
  }

  &.active {
    border-color: #fff;
    box-shadow: 0 0 0 2px #1a1a1a, 0 0 0 4px #fff;
  }
}

// ===== Sticker Grid =====
.sticker-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.sticker-btn {
  aspect-ratio: 1;
  background-color: #262626;
  border: 1px solid #404040;
  border-radius: 8px;
  font-size: 24px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    background-color: #404040;
    transform: scale(1.1);
  }

  &:active {
    transform: scale(0.95);
  }
}

// ===== Filter Grid =====
.filter-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.filter-btn {
  padding: 10px 16px;
  background-color: #262626;
  border: 1px solid #404040;
  color: #fff;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background-color: #404040;
  }

  &.active {
    background-color: #0a66c2;
    border-color: #0a66c2;
  }
}

// ===== Settings Group =====
.settings-group {
  gap: 16px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  color: #fff;
  cursor: pointer;

  input {
    width: 18px;
    height: 18px;
    cursor: pointer;
  }
}

// ===== Action Buttons =====
.action-buttons {
  display: flex;
  gap: 12px;
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid #262626;
}

.discard-btn {
  flex: 1;
  background-color: transparent;
  border: 1px solid #404040;
  color: #fff;
  padding: 10px 24px;
  border-radius: 24px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background-color: #262626;
  }

  &:active {
    transform: scale(0.98);
  }
}

.share-btn {
  flex: 1;
  background-color: #0a66c2;
  border: none;
  color: #fff;
  padding: 10px 24px;
  border-radius: 24px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover:not(:disabled) {
    background-color: #0857a6;
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  &:active:not(:disabled) {
    transform: scale(0.98);
  }
}

// ===== Responsive Design =====
@media (max-width: 1200px) {
  .create-story-page {
    padding-left: calc(72px + 20px);
  }

  .story-editor-container {
    grid-template-columns: 1fr;
    gap: 24px;
    height: auto;
  }

  .tools-panel {
    max-height: auto;
  }

  .story-preview {
    aspect-ratio: 16/9;
  }
}

@media (max-width: 768px) {
  .create-story-page {
    padding-left: 20px;
    padding-right: 20px;
  }

  .tools-panel {
    max-height: 60vh;
  }

  .sticker-grid,
  .filter-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  .action-buttons {
    flex-direction: column;
  }
}
</style>
