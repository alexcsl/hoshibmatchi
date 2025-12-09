<template>
  <div class="media-thumbnail">
    <SecureImage
      v-if="thumbnailSrc"
      :src="thumbnailSrc"
      :alt="alt"
      :class-name="className"
      :loading-placeholder="loadingPlaceholder"
      :error-placeholder="videoPlaceholder"
    />
    <img
      v-else
      :src="videoPlaceholder"
      :alt="alt"
      :class="className"
    />
    <div
      v-if="isVideo"
      class="video-indicator"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="currentColor"
        class="play-icon"
      >
        <path d="M8 5v14l11-7z" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SecureImage from './SecureImage.vue'

interface Props {
  thumbnailUrl?: string
  mediaUrls?: string[]
  isVideo?: boolean
  alt?: string
  className?: string
  loadingPlaceholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  alt: '',
  className: '',
  isVideo: false,
  loadingPlaceholder: 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="300" height="300"%3E%3Crect fill="%23f0f0f0" width="300" height="300"/%3E%3Ctext x="50%25" y="50%25" text-anchor="middle" dy=".3em" fill="%23999" font-family="sans-serif" font-size="16"%3ELoading...%3C/text%3E%3C/svg%3E'
})

// SVG placeholder for videos without thumbnails
const videoPlaceholder = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="300" height="300"%3E%3Crect fill="%23000" width="300" height="300"/%3E%3Cg transform="translate(150,150)"%3E%3Ccircle fill="rgba(255,255,255,0.9)" r="40"/%3E%3Cpath fill="%23000" d="M-8,-12 L-8,12 L16,0 Z"/%3E%3C/g%3E%3C/svg%3E'

const thumbnailSrc = computed(() => {
  console.log('MediaThumbnail - Props:', {
    thumbnailUrl: props.thumbnailUrl,
    mediaUrls: props.mediaUrls,
    isVideo: props.isVideo
  });
  
  // If thumbnail exists and is not empty, use it
  if (props.thumbnailUrl && props.thumbnailUrl.trim() !== '') {
    console.log('✅ Using thumbnail_url:', props.thumbnailUrl);
    return props.thumbnailUrl
  }
  
  // If no thumbnail but have media_urls, check if first media is an image
  if (props.mediaUrls && props.mediaUrls.length > 0) {
    const firstMedia = props.mediaUrls[0]
    // Check if it's an image (not a video)
    if (firstMedia && !firstMedia.match(/\.(mp4|mov|avi|webm|mkv)$/i)) {
      console.log('✅ Using first media_url (image):', firstMedia);
      return firstMedia
    }
    console.log('⚠️ First media is a video, no thumbnail available');
  }
  
  // No valid image thumbnail available
  console.log('⚠️ No thumbnail source available - using placeholder');
  return undefined
})
</script>

<style scoped>
.media-thumbnail {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.media-thumbnail img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.video-indicator {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 48px;
  height: 48px;
  background: rgba(255, 255, 255, 0.9);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.play-icon {
  width: 24px;
  height: 24px;
  color: #000;
  margin-left: 3px;
}
</style>
