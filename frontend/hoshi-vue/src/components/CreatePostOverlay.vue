<template>
  <div class="create-overlay" @click="$emit('close')">
    <div class="create-modal" @click.stop :class="{ 'has-files': selectedFiles.length > 0 }">
      <div class="modal-header">
        <button v-if="selectedFiles.length > 0" class="back-btn" @click="clearFiles">←</button>
        <h2>Create new post</h2>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>

      <div class="modal-content">
        <!-- Content Type Selection (shown first) -->
        <div v-if="!contentTypeSelected" class="content-type-selection">
          <h3>What do you want to create?</h3>
          <div class="type-options">
            <button class="type-option" @click="selectContentType('post')">
              <div class="type-icon">📷</div>
              <div class="type-label">Post</div>
              <div class="type-description">Share photos and videos to your feed</div>
            </button>
            <button class="type-option" @click="selectContentType('story')">
              <div class="type-icon">⭕</div>
              <div class="type-label">Story</div>
              <div class="type-description">Share a moment that disappears in 24h</div>
            </button>
          </div>
        </div>

        <!-- Upload Area (shown when no files selected but content type chosen) -->
        <div 
          v-else-if="selectedFiles.length === 0" 
          class="upload-area" 
          @dragover.prevent="isDragging = true" 
          @dragleave="isDragging = false" 
          @drop.prevent="handleDrop" 
          :class="{ dragging: isDragging }"
        >
          <div class="upload-icon">📷</div>
          <p class="upload-text">Drag photos and videos here</p>
          <input 
            ref="fileInput" 
            type="file" 
            multiple 
            accept="image/*,video/*" 
            style="display: none"
            @change="handleFileSelect"
          />
          <button class="select-btn" @click="fileInput?.click()">Select from computer</button>
        </div>

        <!-- Preview and Caption Area (shown when files selected) -->
        <div v-else class="post-creator">
          <div class="media-preview">
            <div class="media-carousel">
              <img 
                v-if="previewUrls[currentMediaIndex]" 
                :src="previewUrls[currentMediaIndex]" 
                alt="Preview" 
                class="preview-image"
              />
              
              <button 
                v-if="selectedFiles.length > 1 && currentMediaIndex > 0" 
                class="carousel-btn prev"
                @click="currentMediaIndex--"
              >
                ‹
              </button>
              <button 
                v-if="selectedFiles.length > 1 && currentMediaIndex < selectedFiles.length - 1" 
                class="carousel-btn next"
                @click="currentMediaIndex++"
              >
                ›
              </button>

              <div v-if="selectedFiles.length > 1" class="media-dots">
                <span 
                  v-for="(_, index) in selectedFiles" 
                  :key="index"
                  class="dot"
                  :class="{ active: index === currentMediaIndex }"
                  @click="currentMediaIndex = index"
                ></span>
              </div>
            </div>
          </div>

          <div class="post-details">
            <div class="user-info">
              <img 
                :src="currentUser?.profile_picture_url || '/placeholder.svg?height=28&width=28'" 
                alt="Your profile" 
                class="avatar"
              />
              <span class="username">{{ currentUser?.username || 'username' }}</span>
            </div>

            <textarea 
              v-model="caption"
              placeholder="Write a caption..."
              class="caption-input"
              maxlength="2200"
            ></textarea>

            <div class="char-count">{{ caption.length }}/2200</div>

            <div class="options">
              <div class="option-item">
                <input 
                  v-model="location" 
                  type="text" 
                  placeholder="Add location" 
                  class="text-input"
                />
                <span class="icon">📍</span>
              </div>
              
              <div class="option-item">
                <input 
                  v-model="collaboratorsText" 
                  type="text" 
                  placeholder="Add Collaborators (usernames: john, jane)" 
                  class="text-input"
                />
                <span class="icon">👥</span>
              </div>

              <div class="option-item">
                <label class="checkbox-label">
                  <input type="checkbox" v-model="isReel" />
                  <span>Share as Reel</span>
                </label>
              </div>

              <div class="option-item">
                <label class="checkbox-label">
                  <input type="checkbox" v-model="commentsDisabled" />
                  <span>Turn off commenting</span>
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="cancel-btn" @click="$emit('close')">Cancel</button>
        <button 
          class="post-btn" 
          :disabled="selectedFiles.length === 0 || isUploading"
          @click="handlePost"
        >
          {{ isUploading ? 'Posting...' : 'Share' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useFeedStore } from '@/stores/feed'
import { useAuthStore } from '@/stores/auth'
// We keep axios for the direct MinIO upload
import axios from 'axios' 
// We use your api service for the backend calls (it handles the token automatically)
import apiClient, { postAPI } from '@/services/api'

const emit = defineEmits<{
  close: []
  posted: []
}>()

const router = useRouter()
const feedStore = useFeedStore()
const authStore = useAuthStore()

const contentTypeSelected = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const isDragging = ref(false)
const selectedFiles = ref<File[]>([])
const previewUrls = ref<string[]>([])
const currentMediaIndex = ref(0)
const caption = ref('')
const isReel = ref(false)
const commentsDisabled = ref(false)
const isUploading = ref(false)
const location = ref('')
const collaboratorsText = ref('')

const currentUser = computed(() => authStore.user)

const selectContentType = (type: 'post' | 'story') => {
  if (type === 'story') {
    emit('close')
    router.push('/create-story')
  } else {
    contentTypeSelected.value = true
  }
}

const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.files) {
    addFiles(Array.from(target.files))
  }
}

const handleDrop = (e: DragEvent) => {
  isDragging.value = false
  if (e.dataTransfer?.files) {
    addFiles(Array.from(e.dataTransfer.files))
  }
}

const addFiles = (files: File[]) => {
  // Filter for images and videos only
  const validFiles = files.filter(file => 
    file.type.startsWith('image/') || file.type.startsWith('video/')
  )

  if (validFiles.length === 0) {
    alert('Please select valid image or video files')
    return
  }

  selectedFiles.value = validFiles
  
  // Create preview URLs
  previewUrls.value = validFiles.map(file => URL.createObjectURL(file))
  currentMediaIndex.value = 0
}

const clearFiles = () => {
  // Clean up preview URLs
  previewUrls.value.forEach(url => URL.revokeObjectURL(url))
  
  selectedFiles.value = []
  previewUrls.value = []
  currentMediaIndex.value = 0
  caption.value = ''
  isReel.value = false
  commentsDisabled.value = false
  contentTypeSelected.value = false
}

const handlePost = async () => {
  if (selectedFiles.value.length === 0 || isUploading.value) return

  isUploading.value = true

  try {
    const finalMediaUrls: string[] = []

    // 1. Loop through each selected file
    for (const file of selectedFiles.value) {
      
      // A. Request Upload URL from Backend
      // We use 'apiClient' here because it automatically attaches 'Authorization: Bearer <jwt_token>'
      const { data: uploadData } = await apiClient.get('/media/upload-url', {
        params: {
          filename: file.name,
          type: file.type
        }
      })

      // B. Upload Binary to MinIO
      // We use raw 'axios' here because we do NOT want the Authorization header (MinIO uses the URL signature)
      await axios.put(uploadData.upload_url, file, {
        headers: { 'Content-Type': file.type }
      })

      // C. Keep the final URL to save in the DB
      finalMediaUrls.push(uploadData.final_media_url)
    }

    // Post Collaborators
    const collaboratorIds: number[] = []
    if (collaboratorsText.value.trim()) {
      const usernames = collaboratorsText.value.split(',').map(u => u.trim())
      
      for (const username of usernames) {
        if (!username) continue
        try {
          // We try to find the user to get their ID
          const res = await apiClient.get(`/users/${username}`)
          // Handle different response structures depending on your API Gateway
          const userObj = res.data.user || res.data
          if (userObj && (userObj.id || userObj.user_id)) {
             collaboratorIds.push(Number(userObj.id || userObj.user_id))
          }
        } catch (err) {
          console.warn(`Could not find user: ${username}`)
        }
      }
    }

    // 2. Create Post in Backend
    // We use postAPI which also uses apiClient internally
    const response = await postAPI.createPost({
      caption: caption.value,
      media_urls: finalMediaUrls,
      comments_disabled: commentsDisabled.value,
      is_reel: isReel.value,
      location: location.value,
      collaborator_ids: collaboratorIds
    })

    // 3. Update Feed
    if (response.post) {
      feedStore.addPost(response.post)
    }

    emit('posted')
    emit('close')
    clearFiles()

  } catch (error: any) {
    console.error('Failed to create post:', error)
    const msg = error.response?.data?.error || error.message || 'Failed to create post.'
    alert(msg)
  } finally {
    isUploading.value = false
  }
}

onUnmounted(() => {
  previewUrls.value.forEach(url => URL.revokeObjectURL(url))
})
</script>

<style scoped lang="scss">
.create-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.create-modal {
  background-color: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  &.has-files {
    max-width: 900px;
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #404040;

  .back-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    margin-right: 12px;

    &:hover {
      opacity: 0.7;
    }
  }

  h2 {
    font-size: 18px;
    font-weight: 700;
    flex: 1;
    text-align: center;
  }

  .close-btn {
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

.modal-content {
  padding: 40px 20px;
  min-height: 300px;
}

.content-type-selection {
  text-align: center;
  padding: 20px;

  h3 {
    font-size: 20px;
    font-weight: 600;
    margin-bottom: 32px;
    color: #fff;
  }

  .type-options {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
    max-width: 600px;
    margin: 0 auto;

    @media (max-width: 640px) {
      grid-template-columns: 1fr;
    }
  }

  .type-option {
    background-color: #1a1a1a;
    border: 2px solid #404040;
    border-radius: 12px;
    padding: 32px 20px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: #fff;

    &:hover {
      border-color: #0a66c2;
      background-color: #262626;
      transform: translateY(-2px);
    }

    .type-icon {
      font-size: 48px;
      margin-bottom: 8px;
    }

    .type-label {
      font-size: 18px;
      font-weight: 600;
    }

    .type-description {
      font-size: 13px;
      color: #a8a8a8;
      line-height: 1.4;
    }
  }
}

.upload-area {
  border: 2px dashed #404040;
  border-radius: 8px;
  padding: 40px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;

  &.dragging {
    border-color: #0a66c2;
    background-color: rgba(10, 102, 194, 0.1);
  }

  .upload-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  .upload-text {
    font-size: 16px;
    margin-bottom: 12px;
    color: #fff;
  }

  .select-btn {
    background-color: #0a66c2;
    border: none;
    color: #fff;
    padding: 8px 24px;
    border-radius: 24px;
    font-weight: 600;
    font-size: 14px;
    cursor: pointer;
    transition: background-color 0.2s;

    &:hover {
      background-color: #0958a3;
    }
  }
}

.post-creator {
  display: grid;
  grid-template-columns: 1fr 400px;
  gap: 0;
  padding: 0;
  margin: -40px -20px;
  min-height: 500px;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
}

.text-input {
  width: 100%;
  background: transparent;
  border: none;
  color: white;
  font-size: 14px;
  padding: 8px 0;
  outline: none;
  
  &::placeholder {
    color: #a8a8a8;
  }
}

.icon {
  font-size: 16px;
  margin-left: 8px;
}

.media-preview {
  background-color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;

  .media-carousel {
    position: relative;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;

    .preview-image {
      max-width: 100%;
      max-height: 500px;
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

    .media-dots {
      position: absolute;
      bottom: 12px;
      left: 50%;
      transform: translateX(-50%);
      display: flex;
      gap: 6px;

      .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.5);
        cursor: pointer;
        transition: background 0.2s;

        &.active {
          background: #fff;
        }

        &:hover {
          background: rgba(255, 255, 255, 0.8);
        }
      }
    }
  }
}

.post-details {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  background-color: #262626;
  overflow-y: auto;

  .user-info {
    display: flex;
    align-items: center;
    gap: 12px;

    .avatar {
      width: 28px;
      height: 28px;
      border-radius: 50%;
      object-fit: cover;
    }

    .username {
      font-weight: 600;
      font-size: 14px;
    }
  }

  .caption-input {
    width: 100%;
    min-height: 150px;
    background: none;
    border: none;
    color: #fff;
    font-size: 14px;
    resize: none;
    outline: none;
    font-family: inherit;

    &::placeholder {
      color: #a8a8a8;
    }
  }

  .char-count {
    text-align: right;
    font-size: 12px;
    color: #a8a8a8;
  }

  .options {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-top: 12px;
    border-top: 1px solid #404040;

    .option-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 0;

      span {
        font-size: 14px;
      }

      .option-btn {
        background: none;
        border: none;
        color: #a8a8a8;
        font-size: 18px;
        cursor: pointer;
        padding: 0;

        &:hover {
          color: #fff;
        }
      }

      .checkbox-label {
        display: flex;
        align-items: center;
        gap: 12px;
        cursor: pointer;

        input[type="checkbox"] {
          width: 18px;
          height: 18px;
          cursor: pointer;
        }
      }
    }
  }
}

.modal-footer {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid #404040;
  justify-content: flex-end;

  .cancel-btn {
    background: none;
    border: 1px solid #404040;
    color: #fff;
    padding: 8px 24px;
    border-radius: 24px;
    font-weight: 600;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background-color: #262626;
    }
  }

  .post-btn {
    background-color: #0a66c2;
    border: none;
    color: #fff;
    padding: 8px 24px;
    border-radius: 24px;
    font-weight: 600;
    font-size: 14px;
    cursor: pointer;

    &:disabled {
      background-color: #404040;
      cursor: not-allowed;
      opacity: 0.6;
    }

    &:not(:disabled):hover {
      background-color: #0958a3;
    }
  }
}
</style>
