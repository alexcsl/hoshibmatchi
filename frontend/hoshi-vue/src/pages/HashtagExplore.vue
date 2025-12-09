<template>
  <div class="hashtag-explore-page">
    <div class="hashtag-header">
      <button
        class="back-btn"
        @click="$router.back()"
      >
        ←
      </button>
      <div class="hashtag-info">
        <h1>#{{ hashtagName }}</h1>
        <p class="post-count">
          {{ totalPosts }} posts
        </p>
      </div>
    </div>

    <div
      v-if="loading && posts.length === 0"
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
      v-else-if="posts.length > 0"
      class="posts-grid"
    >
      <div 
        v-for="post in posts" 
        :key="post.id" 
        class="post-item" 
        @click="handleOpenPost(post.id)"
      >
        <img 
          :src="getMediaUrl(post.media_urls?.[0]) || '/placeholder.svg?height=280&width=280'" 
          :alt="post.caption" 
          class="post-image" 
        />
        <div class="post-overlay">
          <div class="post-stats">
            <span class="stat">❤ {{ formatCount(post.like_count || 0) }}</span>
            <span class="stat">💬 {{ formatCount(post.comment_count || 0) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div
      v-else
      class="empty-state"
    >
      <p>No posts found with #{{ hashtagName }}</p>
      <p class="empty-subtitle">
        Be the first to post with this hashtag
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useRoute } from "vue-router";
import apiClient from "@/services/api";

const route = useRoute();
const hashtagName = ref("");
const posts = ref<any[]>([]);
const totalPosts = ref(0);
const loading = ref(false);

const fetchHashtagPosts = async () => {
  loading.value = true;
  try {
    const hashtag = route.params.hashtag as string;
    hashtagName.value = hashtag;
    
    const response = await apiClient.get(`/search/hashtags/${hashtag}`);
    posts.value = response.data.posts || [];
    totalPosts.value = response.data.total_post_count || 0;
  } catch (error) {
    console.error("Failed to fetch hashtag posts:", error);
    posts.value = [];
    totalPosts.value = 0;
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchHashtagPosts();
});

watch(() => route.params.hashtag, () => {
  if (route.params.hashtag) {
    fetchHashtagPosts();
  }
});

const handleOpenPost = (postId: string) => {
  if (window.openPostDetails) {
    window.openPostDetails(postId);
  }
};

const getMediaUrl = (url: string | undefined) => {
  if (!url) return "/placeholder.svg?height=280&width=280";
  if (url.startsWith("http")) return url;
  if (url.startsWith("/uploads/") || url.startsWith("uploads/")) {
    return `http://localhost:8000${url.startsWith("/") ? url : "/" + url}`;
  }
  return url;
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
.hashtag-explore-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.hashtag-header {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-bottom: 32px;
  padding-bottom: 20px;
  border-bottom: 1px solid #262626;

  .back-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 28px;
    cursor: pointer;
    padding: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: opacity 0.2s;

    &:hover {
      opacity: 0.7;
    }
  }

  .hashtag-info {
    flex: 1;

    h1 {
      font-size: 32px;
      font-weight: 700;
      margin-bottom: 4px;
      color: #fff;
    }

    .post-count {
      font-size: 14px;
      color: #a8a8a8;
    }
  }
}

.posts-grid {
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
  padding: 100px 20px;
  
  p {
    font-size: 18px;
    color: #fff;
    margin-bottom: 8px;
  }

  .empty-subtitle {
    font-size: 14px;
    color: #a8a8a8;
  }
}

.post-item {
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
        font-size: 16px;
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
  .hashtag-explore-page {
    padding-left: calc(72px + 20px);
  }
}

@media (max-width: 768px) {
  .hashtag-explore-page {
    padding-left: calc(60px + 20px);
  }

  .posts-grid {
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  }

  .hashtag-header {
    h1 {
      font-size: 24px;
    }
  }
}
</style>
