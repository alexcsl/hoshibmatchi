<template>
  <div
    class="search-overlay"
    @click="$emit('close')"
  >
    <div
      class="search-panel"
      @click.stop
    >
      <div class="search-header">
        <input 
          v-model="searchQuery"
          type="text" 
          placeholder="Search"
          class="search-input"
          @input="handleSearch"
        />
        <button
          class="close-btn"
          @click="$emit('close')"
        >
          ✕
        </button>
      </div>

      <!-- Recent Searches -->
      <div
        v-if="!searchQuery"
        class="search-content"
      >
        <div class="section">
          <div class="section-header">
            <h3>Recent</h3>
            <button
              class="clear-btn"
              @click="clearRecent"
            >
              Clear all
            </button>
          </div>
          <div
            v-if="recentSearches.length"
            class="search-list"
          >
            <div
              v-for="search in recentSearches"
              :key="search.id"
              class="search-item"
            >
              <span class="icon">🕐</span>
              <span class="text">{{ search.query }}</span>
              <button
                class="remove-btn"
                @click.stop="removeRecent(search.id)"
              >
                ✕
              </button>
            </div>
          </div>
          <div
            v-else
            class="empty"
          >
            No recent searches
          </div>
        </div>
      </div>

      <!-- Search Results -->
      <div
        v-else
        class="search-results"
      >
        <div class="results-list">
          <div
            v-for="user in filteredUsers"
            :key="user.id"
            class="result-item user-result"
            @click="selectUser(user)"
          >
            <img
              :src="user.avatar"
              :alt="user.username"
              class="avatar"
            />
            <div class="result-info">
              <div class="username">
                {{ user.username }}
              </div>
              <div class="fullname">
                {{ user.fullname }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

defineEmits<{
  close: []
}>();

const searchQuery = ref("");
const recentSearches = ref([
  { id: 1, query: "#genshin" },
  { id: 2, query: "@selyucormer" },
  { id: 3, query: "#anime" }
]);

const allUsers = [
  { id: 1, username: "user_1", fullname: "User One", avatar: "/placeholder.svg?height=40&width=40" },
  { id: 2, username: "user_2", fullname: "User Two", avatar: "/placeholder.svg?height=40&width=40" },
  { id: 3, username: "user_3", fullname: "User Three", avatar: "/placeholder.svg?height=40&width=40" },
  { id: 4, username: "user_4", fullname: "User Four", avatar: "/placeholder.svg?height=40&width=40" },
  { id: 5, username: "user_5", fullname: "User Five", avatar: "/placeholder.svg?height=40&width=40" }
];

const filteredUsers = computed(() => {
  if (!searchQuery.value) return [];
  return allUsers.filter(user => 
    user.username.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
    user.fullname.toLowerCase().includes(searchQuery.value.toLowerCase())
  );
});

const handleSearch = () => {
  // Implement debounce for real API calls
};

const clearRecent = () => {
  recentSearches.value = [];
};

const removeRecent = (id: number) => {
  recentSearches.value = recentSearches.value.filter(s => s.id !== id);
};

const selectUser = (user: any) => {
  console.log("Selected user:", user);
};
</script>

<style scoped lang="scss">
.search-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.8);
  display: flex;
  z-index: 90;
}

.search-panel {
  width: 360px;
  height: 100vh;
  background-color: #000;
  border-right: 1px solid #262626;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.search-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px;
  border-bottom: 1px solid #262626;
  position: sticky;
  top: 0;
  background-color: #000;
  z-index: 10;

  .search-input {
    flex: 1;
    background-color: #262626;
    border: none;
    border-radius: 20px;
    padding: 10px 16px;
    color: #fff;
    font-size: 14px;
    outline: none;

    &::placeholder {
      color: #a8a8a8;
    }

    &:focus {
      background-color: #404040;
    }
  }

  .close-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
    padding: 0;
  }
}

.search-content {
  flex: 1;
  padding: 16px;
}

.section {
  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;

    h3 {
      font-size: 14px;
      font-weight: 600;
      color: #a8a8a8;
    }

    .clear-btn {
      background: none;
      border: none;
      color: #0a66c2;
      font-size: 12px;
      cursor: pointer;
      font-weight: 600;
    }
  }

  .search-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .search-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 12px;
    border-radius: 8px;
    cursor: pointer;
    transition: background-color 0.2s;

    &:hover {
      background-color: #262626;

      .remove-btn {
        opacity: 1;
      }
    }

    .icon {
      font-size: 16px;
    }

    .text {
      flex: 1;
      font-size: 14px;
    }

    .remove-btn {
      background: none;
      border: none;
      color: #a8a8a8;
      font-size: 16px;
      cursor: pointer;
      opacity: 0;
      transition: opacity 0.2s;
    }
  }

  .empty {
    color: #a8a8a8;
    font-size: 14px;
    text-align: center;
    padding: 24px;
  }
}

.search-results {
  flex: 1;
  padding: 16px;

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .result-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    border-radius: 8px;
    cursor: pointer;
    transition: background-color 0.2s;

    &:hover {
      background-color: #262626;
    }

    .avatar {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      object-fit: cover;
    }

    .result-info {
      flex: 1;

      .username {
        font-weight: 600;
        font-size: 14px;
      }

      .fullname {
        font-size: 12px;
        color: #a8a8a8;
      }
    }
  }
}
</style>
