<template>
  <div
    class="create-overlay"
    @click="$emit('close')"
  >
    <div
      class="create-modal"
      :class="{ 'has-files': selectedFiles.length > 0 }"
      @click.stop
    >
      <div class="modal-header">
        <button
          v-if="selectedFiles.length > 0"
          class="back-btn"
          @click="clearFiles"
        >
          ←
        </button>
        <h2>Create new post</h2>
        <button
          class="close-btn"
          @click="$emit('close')"
        >
          ✕
        </button>
      </div>

      <div class="modal-content">
        <!-- Content Type Selection (shown first) -->
        <div
          v-if="!contentTypeSelected"
          class="content-type-selection"
        >
          <h3>What do you want to create?</h3>
          <div class="type-options">
            <button
              class="type-option"
              @click="selectContentType('post')"
            >
              <div class="type-icon">
                📷
              </div>
              <div class="type-label">
                Post
              </div>
              <div class="type-description">
                Share photos and videos to your feed
              </div>
            </button>
            <button
              class="type-option"
              @click="selectContentType('story')"
            >
              <div class="type-icon">
                ⭕
              </div>
              <div class="type-label">
                Story
              </div>
              <div class="type-description">
                Share a moment that disappears in 24h
              </div>
            </button>
          </div>
        </div>

        <!-- Upload Area (shown when no files selected but content type chosen) -->
        <div 
          v-else-if="selectedFiles.length === 0" 
          class="upload-area" 
          :class="{ dragging: isDragging }" 
          @dragover.prevent="isDragging = true" 
          @dragleave="isDragging = false" 
          @drop.prevent="handleDrop"
        >
          <div class="upload-icon">
            📷
          </div>
          <p class="upload-text">
            Drag photos and videos here
          </p>
          <input 
            ref="fileInput" 
            type="file" 
            multiple 
            accept="image/*,video/*" 
            style="display: none"
            @change="handleFileSelect"
          />
          <button
            class="select-btn"
            @click="fileInput?.click()"
          >
            Select from computer
          </button>
        </div>

        <!-- Preview and Caption Area (shown when files selected) -->
        <div
          v-else
          class="post-creator"
        >
          <div class="media-preview">
            <div class="media-carousel">
              <template v-if="previewUrls[currentMediaIndex]">
                <video 
                  v-if="isVideoFile(selectedFiles[currentMediaIndex])"
                  :key="currentMediaIndex"
                  ref="videoPreviewRef"
                  :src="previewUrls[currentMediaIndex]" 
                  class="preview-image"
                  controls
                  playsinline
                  @loadedmetadata="onVideoLoaded"
                  @timeupdate="onVideoTimeUpdate"
                  @seeked="captureFrame"
                ></video>
                <img 
                  v-else
                  :src="previewUrls[currentMediaIndex]" 
                  alt="Preview" 
                  class="preview-image"
                />
              </template>
              
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

              <div
                v-if="selectedFiles.length > 1"
                class="media-dots"
              >
                <span 
                  v-for="(_, index) in selectedFiles" 
                  :key="index"
                  class="dot"
                  :class="{ active: index === currentMediaIndex }"
                  @click="currentMediaIndex = index"
                ></span>
              </div>
            </div>

            <!-- Video Thumbnail Frame Selector -->
            <div
              v-if="isVideoFile(selectedFiles[currentMediaIndex]) && videoDuration > 0"
              class="frame-selector"
            >
              <div class="frame-selector-header">
                <span class="frame-icon">🎬</span>
                <span class="frame-label">Choose Thumbnail Frame</span>
              </div>
              <div class="frame-slider-container">
                <input
                  v-model.number="thumbnailTimestamp"
                  type="range"
                  min="0"
                  :max="videoDuration"
                  step="0.1"
                  class="frame-slider"
                  @input="seekToTimestamp"
                />
                <div class="frame-time">
                  {{ formatTime(thumbnailTimestamp) }} / {{ formatTime(videoDuration) }}
                </div>
              </div>
              <div class="frame-preview">
                <canvas
                  ref="thumbnailCanvas"
                  class="thumbnail-preview"
                  width="120"
                  height="120"
                ></canvas>
                <span class="preview-label">Thumbnail Preview ({{ thumbnailTimestamp.toFixed(1) }}s)</span>
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

            <div class="char-count">
              {{ caption.length }}/2200
            </div>

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
                  <input
                    v-model="isReel"
                    type="checkbox"
                  />
                  <span>Share as Reel</span>
                </label>
              </div>

              <div class="option-item">
                <label class="checkbox-label">
                  <input
                    v-model="commentsDisabled"
                    type="checkbox"
                  />
                  <span>Turn off commenting</span>
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button
          class="cancel-btn"
          @click="$emit('close')"
        >
          Cancel
        </button>
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
import { ref, computed, onUnmounted, watch } from "vue";
import { useRouter } from "vue-router";
import { useFeedStore } from "@/stores/feed";
import { useAuthStore } from "@/stores/auth";
// We keep axios for the direct MinIO upload
import axios from "axios"; 
// We use your api service for the backend calls (it handles the token automatically)
import apiClient, { postAPI } from "@/services/api";

const emit = defineEmits<{
  close: []
  posted: []
}>();

const router = useRouter();
const feedStore = useFeedStore();
const authStore = useAuthStore();

const contentTypeSelected = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);
const isDragging = ref(false);
const selectedFiles = ref<File[]>([]);
const previewUrls = ref<string[]>([]);
const currentMediaIndex = ref(0);
const caption = ref("");
const isReel = ref(false);
const commentsDisabled = ref(false);
const isUploading = ref(false);
const location = ref("");
const collaboratorsText = ref("");

// Video thumbnail frame selector state
const videoPreviewRef = ref<HTMLVideoElement | null>(null);
const thumbnailCanvas = ref<HTMLCanvasElement | null>(null);
const videoDuration = ref(0);
const thumbnailTimestamp = ref(1.0); // Default to 1 second

const currentUser = computed(() => authStore.user);

const isVideoFile = (file: File) => {
  return file && file.type.startsWith("video/");
};

const selectContentType = (type: "post" | "story") => {
  if (type === "story") {
    emit("close");
    router.push("/create-story");
  } else {
    contentTypeSelected.value = true;
  }
};

const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files) {
    addFiles(Array.from(target.files));
  }
};

const handleDrop = (e: DragEvent) => {
  isDragging.value = false;
  if (e.dataTransfer?.files) {
    addFiles(Array.from(e.dataTransfer.files));
  }
};

const addFiles = (files: File[]) => {
  // Filter for images and videos only
  const validFiles = files.filter(file => 
    file.type.startsWith("image/") || file.type.startsWith("video/")
  );

  if (validFiles.length === 0) {
    alert("Please select valid image or video files");
    return;
  }

  selectedFiles.value = validFiles;
  
  // Create preview URLs
  previewUrls.value = validFiles.map(file => URL.createObjectURL(file));
  currentMediaIndex.value = 0;
};

const clearFiles = () => {
  // Clean up preview URLs
  previewUrls.value.forEach(url => URL.revokeObjectURL(url));
  
  selectedFiles.value = [];
  previewUrls.value = [];
  currentMediaIndex.value = 0;
  caption.value = "";
  isReel.value = false;
  commentsDisabled.value = false;
  contentTypeSelected.value = false;
  videoDuration.value = 0;
  thumbnailTimestamp.value = 1.0;
};

// Watch for media index changes to reset video state
watch(() => currentMediaIndex.value, () => {
  videoDuration.value = 0;
  thumbnailTimestamp.value = 1.0;
});

// Video frame selector functions
const onVideoLoaded = () => {
  if (videoPreviewRef.value && videoPreviewRef.value.duration) {
    videoDuration.value = videoPreviewRef.value.duration;
    thumbnailTimestamp.value = Math.min(1.0, videoDuration.value);
    console.log('Video loaded - Duration:', videoDuration.value, 'seconds');
    // Seek to initial timestamp to capture frame
    videoPreviewRef.value.currentTime = thumbnailTimestamp.value;
  }
};

const onVideoTimeUpdate = () => {
  captureFrame();
};

const seekToTimestamp = () => {
  if (videoPreviewRef.value) {
    console.log('Seeking to:', thumbnailTimestamp.value, 'seconds');
    videoPreviewRef.value.currentTime = thumbnailTimestamp.value;
  }
};

const captureFrame = () => {
  if (!videoPreviewRef.value || !thumbnailCanvas.value) return;
  
  const video = videoPreviewRef.value;
  const canvas = thumbnailCanvas.value;
  const ctx = canvas.getContext('2d');
  
  if (ctx && video.readyState >= 2) {
    const aspectRatio = video.videoWidth / video.videoHeight;
    const canvasSize = 120;
    
    let drawWidth = canvasSize;
    let drawHeight = canvasSize;
    let offsetX = 0;
    let offsetY = 0;
    
    if (aspectRatio > 1) {
      drawHeight = canvasSize / aspectRatio;
      offsetY = (canvasSize - drawHeight) / 2;
    } else {
      drawWidth = canvasSize * aspectRatio;
      offsetX = (canvasSize - drawWidth) / 2;
    }
    
    ctx.fillStyle = '#000';
    ctx.fillRect(0, 0, canvasSize, canvasSize);
    ctx.drawImage(video, offsetX, offsetY, drawWidth, drawHeight);
  }
};

const formatTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};

const handlePost = async () => {
  if (selectedFiles.value.length === 0 || isUploading.value) return;

  console.log('=== STARTING POST CREATION ===');
  console.log('Selected files:', selectedFiles.value.map(f => ({ name: f.name, type: f.type })));
  console.log('Thumbnail timestamp:', thumbnailTimestamp.value);

  isUploading.value = true;

  try {
    const finalMediaUrls: string[] = [];
    const thumbnailUrls: { [key: string]: string } = {};

    // 1. Loop through each selected file
    for (let i = 0; i < selectedFiles.value.length; i++) {
      const file = selectedFiles.value[i];
      console.log(`\n--- Processing file ${i + 1}/${selectedFiles.value.length}: ${file.name} ---`);
      
      // A. Request Upload URL from Backend
      console.log('Step 1: Requesting upload URL...');
      // We use 'apiClient' here because it automatically attaches 'Authorization: Bearer <jwt_token>'
      const { data: uploadData } = await apiClient.get("/media/upload-url", {
        params: {
          filename: file.name,
          type: file.type
        }
      });
      console.log('Upload URL received:', uploadData.upload_url.substring(0, 50) + '...');
      console.log('Final media URL:', uploadData.final_media_url);

      // B. Upload Binary to MinIO
      console.log('Step 2: Uploading file to MinIO...');
      // We use raw 'axios' here because we do NOT want the Authorization header (MinIO uses the URL signature)
      await axios.put(uploadData.upload_url, file, {
        headers: { "Content-Type": file.type }
      });
      console.log('✅ File uploaded successfully');

      // C. Keep the final URL to save in the DB
      const mediaPath = uploadData.final_media_url;
      finalMediaUrls.push(mediaPath);

      // D. Generate thumbnail for videos
      if (file.type.startsWith("video/")) {
        console.log('Step 3: Generating thumbnail for video...');
        console.log('  - Video path:', mediaPath);
        console.log('  - Timestamp:', thumbnailTimestamp.value, 'seconds');
        try {
          const { data: thumbnailData } = await apiClient.post("/media/generate-thumbnail", {
            object_name: mediaPath,
            timestamp_seconds: thumbnailTimestamp.value
          });
          console.log('✅ Thumbnail API response:', thumbnailData);
          if (thumbnailData.thumbnail_url) {
            thumbnailUrls[mediaPath] = thumbnailData.thumbnail_url;
            console.log('✅ Thumbnail URL saved:', thumbnailData.thumbnail_url);
          } else {
            console.warn('⚠️ No thumbnail_url in response');
          }
        } catch (error: any) {
          console.error("❌ Failed to generate thumbnail:", error);
          console.error("Error details:", error.response?.data || error.message);
        }
      } else {
        console.log('Step 3: Skipping thumbnail (not a video)');
      }
    }

    console.log('\n=== UPLOAD COMPLETE ===');
    console.log('Final media URLs:', finalMediaUrls);
    console.log('Thumbnail URLs:', thumbnailUrls);

    // Determine thumbnail URL (use first video thumbnail or first image)
    let thumbnailUrl = "";
    if (Object.keys(thumbnailUrls).length > 0) {
      thumbnailUrl = Object.values(thumbnailUrls)[0];
      console.log('✅ Using video thumbnail:', thumbnailUrl);
    } else if (finalMediaUrls.length > 0 && !selectedFiles.value[0].type.startsWith("video/")) {
      thumbnailUrl = finalMediaUrls[0]; // Use first image as thumbnail
      console.log('✅ Using first image as thumbnail:', thumbnailUrl);
    } else {
      console.log('⚠️ No thumbnail available - video will show placeholder');
    }

    // Post Collaborators
    console.log('\n--- Processing Collaborators ---');
    const collaboratorIds: number[] = [];
    if (collaboratorsText.value.trim()) {
      const usernames = collaboratorsText.value.split(",").map(u => u.trim());
      console.log('Looking up collaborators:', usernames);
      
      for (const username of usernames) {
        if (!username) continue;
        try {
          // We try to find the user to get their ID
          const res = await apiClient.get(`/users/${username}`);
          // Handle different response structures depending on your API Gateway
          const userObj = res.data.user || res.data;
          if (userObj && (userObj.id || userObj.user_id)) {
             collaboratorIds.push(Number(userObj.id || userObj.user_id));
             console.log('  ✅', username, '-> ID:', userObj.id || userObj.user_id);
          }
        } catch {
          console.warn(`  ❌ Could not find user: ${username}`);
        }
      }
    } else {
      console.log('No collaborators specified');
    }

    // 2. Create Post in Backend
    console.log('\n--- Creating Post ---');
    const postData = {
      caption: caption.value,
      media_urls: finalMediaUrls,
      comments_disabled: commentsDisabled.value,
      is_reel: isReel.value,
      location: location.value,
      collaborator_ids: collaboratorIds,
      thumbnail_url: thumbnailUrl
    };
    console.log('Post data:', postData);
    
    // We use postAPI which also uses apiClient internally
    const response = await postAPI.createPost(postData);
    console.log('✅ Post created:', response);

    // 3. Update Feed
    if (response.post) {
      feedStore.addPost(response.post);
      console.log('✅ Post added to feed store');
    }

    emit("posted");
    emit("close");
    clearFiles();
    console.log('=== POST CREATION COMPLETE ===\n');

  } catch (error: any) {
    console.error("❌ POST CREATION FAILED");
    console.error("Error:", error);
    console.error("Response:", error.response?.data);
    const msg = error.response?.data?.error || error.message || "Failed to create post.";
    alert(msg);
  } finally {
    isUploading.value = false;
  }
};

onUnmounted(() => {
  previewUrls.value.forEach(url => URL.revokeObjectURL(url));
});
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
  background-color: var(--bg-elevated);
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
  border-bottom: 1px solid var(--border-color);

  .back-btn {
    background: none;
    border: none;
    color: var(--text-primary);
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
    color: var(--text-primary);
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
    color: var(--text-primary);
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
    background-color: var(--bg-secondary);
    border: 2px solid var(--border-color);
    border-radius: 12px;
    padding: 32px 20px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: var(--text-primary);

    &:hover {
      border-color: var(--accent-primary);
      background-color: var(--bg-elevated);
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
      color: var(--text-secondary);
      line-height: 1.4;
    }
  }
}

.upload-area {
  border: 2px dashed var(--border-color);
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
    border-color: var(--accent-primary);
    background-color: rgba(10, 102, 194, 0.1);
  }

  .upload-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  .upload-text {
    font-size: 16px;
    margin-bottom: 12px;
    color: var(--text-primary);
  }

  .select-btn {
    background-color: var(--accent-primary);
    border: none;
    color: var(--text-primary);
    padding: 8px 24px;
    border-radius: 24px;
    font-weight: 600;
    font-size: 14px;
    cursor: pointer;
    transition: background-color 0.2s;

    &:hover {
      background-color: var(--accent-hover);
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
  color: var(--text-primary);
  font-size: 14px;
  padding: 8px 0;
  outline: none;
  
  &::placeholder {
    color: var(--text-secondary);
  }
}

.icon {
  font-size: 16px;
  margin-left: 8px;
}

.media-preview {
  background-color: var(--bg-primary);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;

  .media-carousel {
    position: relative;
    width: 100%;
    flex: 1;
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
      color: var(--text-primary);
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
          background: var(--text-primary);
        }

        &:hover {
          background: rgba(255, 255, 255, 0.8);
        }
      }
    }
  }

  .frame-selector {
    width: 100%;
    background: rgba(38, 38, 38, 0.95);
    padding: 16px;
    border-top: 1px solid rgba(255, 255, 255, 0.1);

    .frame-selector-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;
      color: var(--text-primary);
      font-size: 14px;
      font-weight: 600;

      .frame-icon {
        font-size: 18px;
      }
    }

    .frame-slider-container {
      margin-bottom: 12px;

      .frame-slider {
        width: 100%;
        height: 4px;
        border-radius: 2px;
        background: rgba(255, 255, 255, 0.2);
        outline: none;
        cursor: pointer;
        -webkit-appearance: none;

        &::-webkit-slider-thumb {
          -webkit-appearance: none;
          appearance: none;
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: #0095f6;
          cursor: pointer;
          transition: all 0.2s;

          &:hover {
            background: #1da1f2;
            transform: scale(1.1);
          }
        }

        &::-moz-range-thumb {
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: #0095f6;
          cursor: pointer;
          border: none;
          transition: all 0.2s;

          &:hover {
            background: #1da1f2;
            transform: scale(1.1);
          }
        }
      }

      .frame-time {
        color: rgba(255, 255, 255, 0.7);
        font-size: 12px;
        text-align: center;
        margin-top: 8px;
      }
    }

    .frame-preview {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 8px;

      .thumbnail-preview {
        border: 2px solid rgba(255, 255, 255, 0.2);
        border-radius: 8px;
        background: #000;
      }

      .preview-label {
        color: rgba(255, 255, 255, 0.5);
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
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
