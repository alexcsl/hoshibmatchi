<template>
  <div
    class="search-panel-overlay"
    @click="$emit('close')"
  >
    <div
      class="search-panel"
      @click.stop
    >
      <div class="search-header">
        <h2>Search</h2>
        <button
          class="close-btn"
          @click="$emit('close')"
        >
          ✕
        </button>
      </div>

      <div class="search-input-wrapper">
        <input 
          ref="searchInput"
          v-model="searchQuery" 
          type="text" 
          placeholder="Search users, hashtags..."
          class="search-input"
          @input="handleSearch"
        />
        <button
          v-if="searchQuery"
          class="clear-btn"
          @click="clearSearch"
        >
          ✕
        </button>
      </div>

      <!-- Recent Searches -->
      <div
        v-if="!searchQuery && recentSearches.length > 0"
        class="recent-section"
      >
        <div class="section-header">
          <h3>Recent</h3>
          <button
            class="clear-all-btn"
            @click="clearAllRecent"
          >
            Clear all
          </button>
        </div>
        <div class="recent-list">
          <div 
            v-for="(item, index) in recentSearches" 
            :key="index" 
            class="recent-item"
            @click="handleRecentClick(item)"
          >
            <img 
              v-if="item.type === 'user'" 
              :src="getMediaUrl(item.profile_picture_url) || '/default-avatar.svg'" 
              class="recent-avatar" 
            />
            <div
              v-else
              class="hashtag-icon"
            >
              #
            </div>
            <div class="recent-info">
              <div class="recent-name">
                {{ item.name }}
              </div>
              <div class="recent-username">
                {{ item.username }}
              </div>
            </div>
            <button
              class="remove-btn"
              @click.stop="removeRecent(index)"
            >
              ✕
            </button>
          </div>
        </div>
      </div>

      <!-- Search Results -->
      <div
        v-if="searchQuery"
        class="results-section"
      >
        <!-- Loading State -->
        <div
          v-if="isSearching"
          class="loading-results"
        >
          <div class="spinner"></div>
          <p>Searching...</p>
        </div>

        <!-- No Results -->
        <div
          v-else-if="searchResults.length === 0 && !isSearching"
          class="no-results"
        >
          <p>No results found for "{{ searchQuery }}"</p>
        </div>

        <!-- User Results -->
        <div
          v-else
          class="results-list"
        >
          <div 
            v-for="result in searchResults" 
            :key="result.id" 
            class="result-item"
            @click="handleResultClick(result)"
          >
            <img 
              v-if="result.type === 'user'" 
              :src="getMediaUrl(result.profile_picture_url) || '/default-avatar.svg'" 
              class="result-avatar" 
            />
            <div
              v-else
              class="hashtag-icon large"
            >
              #
            </div>
            <div class="result-info">
              <div class="result-name">
                {{ result.name }}
              </div>
              <div class="result-username">
                {{ result.username }}
              </div>
              <div
                v-if="result.followers_count !== undefined"
                class="result-meta"
              >
                {{ formatNumber(result.followers_count) }} followers
              </div>
              <div
                v-else-if="result.post_count !== undefined"
                class="result-meta"
              >
                {{ formatNumber(result.post_count) }} posts
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import apiClient from "@/services/api";

const emit = defineEmits<{
  close: []
}>();

const router = useRouter();
const searchInput = ref<HTMLInputElement>();
const searchQuery = ref("");
const isSearching = ref(false);
const searchResults = ref<any[]>([]);
const recentSearches = ref<any[]>([]);
let searchTimeout: number | null = null;

onMounted(() => {
  searchInput.value?.focus();
  loadRecentSearches();
});

const loadRecentSearches = () => {
  const stored = localStorage.getItem("recentSearches");
  if (stored) {
    try {
      recentSearches.value = JSON.parse(stored);
    } catch {
      recentSearches.value = [];
    }
  }
};

const saveRecentSearches = () => {
  localStorage.setItem("recentSearches", JSON.stringify(recentSearches.value));
};

const handleSearch = () => {
  if (searchTimeout) {
    clearTimeout(searchTimeout);
  }

  if (!searchQuery.value.trim()) {
    searchResults.value = [];
    return;
  }

  searchTimeout = window.setTimeout(async () => {
    isSearching.value = true;
    try {
      const query = searchQuery.value.trim();
      const isHashtagSearch = query.startsWith("#");
      
      if (isHashtagSearch) {
        // Search for hashtags
        const hashtagName = query.substring(1); // Remove the # prefix
        if (!hashtagName) {
          searchResults.value = [];
          isSearching.value = false;
          return;
        }
        
        const response = await apiClient.get(`/search/hashtags/${hashtagName}`);
        
        // Create a hashtag result item
        searchResults.value = [{
          id: `hashtag-${hashtagName}`,
          type: "hashtag",
          name: `#${hashtagName}`,
          username: `${response.data.total_post_count || 0} posts`,
          post_count: response.data.total_post_count || 0,
          hashtag_name: hashtagName
        }];
      } else {
        // Search for users
        const response = await apiClient.get("/search/users", {
          params: { q: query }
        });
        
        // The backend returns an array directly, not wrapped in a users property
        const users = Array.isArray(response.data) ? response.data : (response.data.users || []);
        
        searchResults.value = users.map((user: any) => ({
          id: user.user_id || user.id,
          type: "user",
          name: user.name || user.username,
          username: user.username,
          profile_picture_url: user.profile_picture_url,
          followers_count: user.followers_count
        }));
      }
    } catch (error: any) {
      console.error("Search failed:", error);
      console.error("Error response:", error.response?.data);
      searchResults.value = [];
    } finally {
      isSearching.value = false;
    }
  }, 300);
};

const handleResultClick = (result: any) => {
  // Add to recent searches
  const existingIndex = recentSearches.value.findIndex(r => r.id === result.id);
  if (existingIndex !== -1) {
    recentSearches.value.splice(existingIndex, 1);
  }
  recentSearches.value.unshift(result);
  if (recentSearches.value.length > 10) {
    recentSearches.value = recentSearches.value.slice(0, 10);
  }
  saveRecentSearches();

  // Navigate based on type
  if (result.type === "user") {
    router.push(`/profile/${result.username}`);
    emit("close");
  } else if (result.type === "hashtag") {
    router.push(`/explore/tags/${result.hashtag_name}`);
    emit("close");
  }
};

const handleRecentClick = (item: any) => {
  if (item.type === "user") {
    router.push(`/profile/${item.username}`);
    emit("close");
  } else if (item.type === "hashtag") {
    router.push(`/explore/tags/${item.hashtag_name}`);
    emit("close");
  }
};

const removeRecent = (index: number) => {
  recentSearches.value.splice(index, 1);
  saveRecentSearches();
};

const clearAllRecent = () => {
  recentSearches.value = [];
  saveRecentSearches();
};

const clearSearch = () => {
  searchQuery.value = "";
  searchResults.value = [];
};

const getMediaUrl = (url: string | undefined) => {
  if (!url) return "/default-avatar.svg";
  if (url.startsWith("http")) return url;
  if (url.startsWith("/uploads/") || url.startsWith("uploads/")) {
    return `http://localhost:8000${url.startsWith("/") ? url : "/" + url}`;
  }
  return url;
};

const formatNumber = (num: number) => {
  if (!num) return "0";
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toString();
};
</script>

<style scoped lang="scss">
.search-panel-overlay {
  position: fixed;
  top: 0;
  left: 244px;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: 60;
}

.search-panel {
  position: fixed;
  left: 244px;
  top: 0;
  bottom: 0;
  width: 400px;
  background-color: #000;
  border-right: 1px solid #262626;
  display: flex;
  flex-direction: column;
  z-index: 61;
}

.search-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px 20px;
  border-bottom: 1px solid #262626;

  h2 {
    font-size: 24px;
    font-weight: 600;
    margin: 0;
  }

  .close-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      opacity: 0.7;
    }
  }
}

.search-input-wrapper {
  position: relative;
  padding: 16px 20px;
  border-bottom: 1px solid #262626;

  .search-input {
    width: 100%;
    padding: 12px 40px 12px 16px;
    background-color: #262626;
    border: none;
    border-radius: 8px;
    color: #fff;
    font-size: 14px;
    outline: none;

    &::placeholder {
      color: #a8a8a8;
    }

    &:focus {
      background-color: #1a1a1a;
    }
  }

  .clear-btn {
    position: absolute;
    right: 28px;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: #a8a8a8;
    font-size: 16px;
    cursor: pointer;
    padding: 4px;

    &:hover {
      color: #fff;
    }
  }
}

.recent-section {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;

  h3 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }

  .clear-all-btn {
    background: none;
    border: none;
    color: #0095f6;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;

    &:hover {
      color: #1877f2;
    }
  }
}

.recent-list,
.results-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.recent-item,
.result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.2s;

  &:hover {
    background-color: #262626;
  }

  .recent-avatar,
  .result-avatar {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    object-fit: cover;
  }

  .hashtag-icon {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    background-color: #262626;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: 600;
    color: #a8a8a8;

    &.large {
      font-size: 24px;
    }
  }

  .recent-info,
  .result-info {
    flex: 1;
    min-width: 0;

    .recent-name,
    .result-name {
      font-weight: 600;
      font-size: 14px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .recent-username,
    .result-username {
      font-size: 14px;
      color: #a8a8a8;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .result-meta {
      font-size: 12px;
      color: #a8a8a8;
      margin-top: 2px;
    }
  }

  .remove-btn {
    background: none;
    border: none;
    color: #a8a8a8;
    font-size: 16px;
    cursor: pointer;
    padding: 4px;
    opacity: 0;
    transition: opacity 0.2s;

    &:hover {
      color: #fff;
    }
  }

  &:hover .remove-btn {
    opacity: 1;
  }
}

.results-section {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.loading-results {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  gap: 16px;

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #262626;
    border-top-color: #0095f6;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  p {
    color: #a8a8a8;
    font-size: 14px;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.no-results {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;

  p {
    color: #a8a8a8;
    font-size: 14px;
  }
}

@media (max-width: 1024px) {
  .search-panel-overlay {
    left: 72px;
  }

  .search-panel {
    left: 72px;
    width: 350px;
  }
}

@media (max-width: 768px) {
  .search-panel-overlay {
    left: 0;
  }

  .search-panel {
    left: 0;
    width: 100%;
  }
}
</style>
