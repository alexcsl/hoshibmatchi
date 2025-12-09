<template>
  <div class="reels-page">
    <div class="reels-header">
      <h1>Reels</h1>
      <p>Watch short, engaging videos from creators you follow</p>
    </div>

    <div
      v-if="feedStore.loading && feedStore.reelsFeed.length === 0"
      class="loading-state"
    >
      <div class="loading-grid">
        <div
          v-for="i in 6"
          :key="`skeleton-${i}`"
          class="skeleton-reel"
        ></div>
      </div>
    </div>

    <div
      v-else-if="feedStore.reelsFeed.length > 0"
      class="reels-container"
    >
      <div 
        v-for="reel in feedStore.reelsFeed" 
        :key="reel.id" 
        class="reel-item" 
        @click="handleOpenReel(reel.id)"
      >
        <MediaThumbnail
          :thumbnail-url="reel.thumbnail_url"
          :media-urls="reel.media_urls"
          :is-video="true"
          :alt="reel.caption"
          class-name="reel-thumbnail"
        />
        <div class="reel-overlay">
          <div class="reel-info">
            <div class="creator">
              @{{ reel.author_username }}
            </div>
            <div class="title">
              {{ reel.caption || 'Watch this reel' }}
            </div>
            <div class="stats">
              <span>❤ {{ formatCount(reel.like_count) }}</span>
              <span>💬 {{ formatCount(reel.comment_count) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div
      v-else
      class="empty-state"
    >
      <p>No reels to watch yet</p>
      <p class="empty-subtitle">
        Check back later for new reels
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { useFeedStore } from "@/stores/feed";
import SecureImage from "@/components/SecureImage.vue";
import MediaThumbnail from "@/components/MediaThumbnail.vue";

const feedStore = useFeedStore();

onMounted(async () => {
  if (feedStore.reelsFeed.length === 0) {
    await feedStore.loadReelsFeed(1, 20);
  }
});

const handleOpenReel = (reelId: string) => {
  const reelIndex = feedStore.reelsFeed.findIndex(r => r.id === reelId);
  if (reelIndex !== -1 && window.openReelsViewer) {
    window.openReelsViewer(reelIndex);
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
</script>

<style scoped lang="scss">
.reels-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.reels-header {
  margin-bottom: 32px;

  h1 {
    font-size: 32px;
    font-weight: 700;
    margin-bottom: 8px;
  }

  p {
    font-size: 14px;
    color: #a8a8a8;
  }
}

.reels-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.loading-state {
  width: 100%;
}

.loading-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.skeleton-reel {
  aspect-ratio: 9 / 16;
  background: linear-gradient(90deg, #1a1a1a 25%, #2a2a2a 50%, #1a1a1a 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 8px;
}

@keyframes shimmer {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  
  p {
    font-size: 16px;
    color: #fff;
    margin-bottom: 8px;
  }

  .empty-subtitle {
    font-size: 14px;
    color: #a8a8a8;
  }
}

.reel-item {
  position: relative;
  aspect-ratio: 9 / 16;
  cursor: pointer;
  border-radius: 8px;
  overflow: hidden;
  background-color: #262626;

  .reel-thumbnail {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.2s;
  }

  .reel-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: linear-gradient(180deg, rgba(0,0,0,0) 0%, rgba(0,0,0,0.7) 100%);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    opacity: 0;
    transition: opacity 0.2s;

    .play-button {
      font-size: 48px;
      color: #fff;
    }

    .reel-info {
      align-self: flex-start;
      color: #fff;

      .creator {
        font-size: 12px;
        color: #a8a8a8;
      }

      .title {
        font-size: 16px;
        font-weight: 600;
        margin-top: 4px;
      }

      .stats {
        display: flex;
        gap: 16px;
        margin-top: 8px;
        font-size: 14px;
      }
    }
  }

  &:hover {
    .reel-thumbnail {
      transform: scale(1.05);
    }

    .reel-overlay {
      opacity: 1;
    }
  }
}

@media (max-width: 1024px) {
  .reels-page {
    padding-left: calc(72px + 20px);
  }

  .reels-container {
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  }
}

@media (max-width: 768px) {
  .reels-page {
    padding-left: calc(60px + 20px);
  }

  .reels-container {
    grid-template-columns: 1fr;
  }
}
</style>
