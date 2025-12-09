<template>
  <div class="explore-page">
    <div class="explore-header">
      <h1>Explore</h1>
    </div>

    <div
      v-if="feedStore.loading && feedStore.exploreFeed.length === 0"
      class="loading-state"
    >
      <div class="loading-grid">
        <div
          v-for="i in 12"
          :key="`skeleton-${i}`"
          class="skeleton-item"
        ></div>
      </div>
    </div>

    <div
      v-else-if="feedStore.exploreFeed.length > 0"
      class="explore-grid"
    >
      <div 
        v-for="post in feedStore.exploreFeed" 
        :key="post.id" 
        class="explore-item" 
        @click="handleOpenPost(post.id)"
      >
        <MediaThumbnail
          :thumbnail-url="post.thumbnail_url"
          :media-urls="post.media_urls"
          :is-video="post.is_reel"
          :alt="post.caption"
          class-name="post-image"
        />
        <div class="post-overlay">
          <div class="post-stats">
            <span class="stat">❤ {{ formatCount(post.like_count) }}</span>
            <span class="stat">💬 {{ formatCount(post.comment_count) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div
      v-else
      class="empty-state"
    >
      <p>No posts to explore yet</p>
      <p class="empty-subtitle">
        Check back later for new content
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
  if (feedStore.exploreFeed.length === 0) {
    await feedStore.loadExploreFeed(1, 30);
  }
});

const handleOpenPost = (postId: string) => {
  if (window.openPostDetails) {
    window.openPostDetails(postId, "explore");
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
.explore-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
}

.explore-header {
  margin-bottom: 20px;

  h1 {
    font-size: 32px;
    font-weight: 700;
    margin-bottom: 8px;
  }
}

.explore-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.loading-state {
  width: 100%;
}

.loading-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.skeleton-item {
  aspect-ratio: 1;
  background: linear-gradient(90deg, #1a1a1a 25%, #2a2a2a 50%, #1a1a1a 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
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

.explore-item {
  position: relative;
  aspect-ratio: 1;
  cursor: pointer;
  overflow: hidden;
  border-radius: 4px;
  background-color: #262626;

  .post-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.2s;
  }

  .post-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0);
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0;
    transition: all 0.2s;

    .post-stats {
      display: flex;
      gap: 24px;
      color: #fff;
      font-weight: 700;

      .stat {
        display: flex;
        align-items: center;
        gap: 6px;
      }
    }
  }

  &:hover {
    .post-image {
      transform: scale(1.05);
    }

    .post-overlay {
      background-color: rgba(0, 0, 0, 0.5);
      opacity: 1;
    }
  }
}

@media (max-width: 1024px) {
  .explore-page {
    padding-left: calc(72px + 20px);
  }
}

@media (max-width: 768px) {
  .explore-page {
    padding-left: calc(60px + 20px);
  }

  .explore-grid {
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  }
}
</style>
