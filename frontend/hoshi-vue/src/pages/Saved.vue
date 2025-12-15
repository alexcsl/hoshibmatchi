<template>
  <div class="saved-page">
    <div class="saved-header">
      <button
        class="back-btn"
        @click="goBack"
      >
        <span class="icon">←</span>
      </button>
      <h1>Saved Collections</h1>
      <button
        class="create-collection-btn"
        @click="showCreateCollectionModal = true"
      >
        ➕ New
      </button>
    </div>

    <div
      v-if="loading"
      class="loading"
    >
      Loading collections...
    </div>

    <div
      v-else-if="collections.length === 0"
      class="empty-state"
    >
      <div class="empty-icon">
        🔖
      </div>
      <h3>No collections yet</h3>
      <p>Create a collection to organize your saved posts</p>
    </div>

    <div
      v-else
      class="collections-grid"
    >
      <div
        v-for="collection in collections"
        :key="collection.id"
        class="collection-card"
        :class="{ default: collection.is_default }"
        @click="openCollection(collection)"
      >
        <div class="collection-cover">
          <div
            v-if="collectionPosts[collection.id]?.length > 0"
            class="cover-grid"
          >
            <SecureImage
              v-for="(post, idx) in collectionPosts[collection.id].slice(0, 4)"
              :key="post.id"
              :src="post.media_urls?.[0]"
              :alt="`Saved post ${idx + 1}`"
              loading-placeholder="/placeholder.svg"
              error-placeholder="/placeholder.svg"
            />
          </div>
          <div
            v-else
            class="empty-cover"
          >
            <span>{{ collection.is_default ? '🔖' : '📁' }}</span>
          </div>
        </div>
        <div class="collection-info">
          <h4>{{ collection.name }}</h4>
          <p class="post-count">
            {{ collectionPosts[collection.id]?.length || 0 }} posts
          </p>
        </div>
      </div>
    </div>

    <!-- Create Collection Modal -->
    <div
      v-if="showCreateCollectionModal"
      class="modal-overlay"
      @click="showCreateCollectionModal = false"
    >
      <div
        class="modal"
        @click.stop
      >
        <div class="modal-header">
          <h3>New Collection</h3>
          <button
            class="close-btn"
            @click="showCreateCollectionModal = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <input
            v-model="newCollectionName"
            type="text"
            placeholder="Collection name"
            class="collection-name-input"
            maxlength="50"
            @keyup.enter="createCollection"
          />
        </div>
        <div class="modal-footer">
          <button
            class="cancel-btn"
            @click="showCreateCollectionModal = false"
          >
            Cancel
          </button>
          <button
            class="create-btn"
            :disabled="!newCollectionName.trim() || creating"
            @click="createCollection"
          >
            {{ creating ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { collectionAPI } from "../services/api";
import SecureImage from "../components/SecureImage.vue";

interface Collection {
  id: number;
  name: string;
  is_default: boolean;
}

interface Post {
  id: string;
  media_urls?: string[];
}

const router = useRouter();
const collections = ref<Collection[]>([]);
const collectionPosts = ref<Record<string, Post[]>>({});
const loading = ref(true);
const showCreateCollectionModal = ref(false);
const newCollectionName = ref("");
const creating = ref(false);

const goBack = () => {
  router.back();
};

const openCollection = (collection: Collection) => {
  router.push(`/collections/${collection.id}`);
};

const loadCollections = async () => {
  loading.value = true;
  try {
    const collectionsRes = await collectionAPI.getAll();
    collections.value = Array.isArray(collectionsRes) ? collectionsRes : (collectionsRes?.collections || []);
    
    // Load posts for each collection (first 4 for preview)
    for (const collection of collections.value) {
      try {
        const postsRes = await collectionAPI.getPosts(String(collection.id), 1, 4);
        collectionPosts.value[collection.id] = Array.isArray(postsRes) ? postsRes : (postsRes?.posts || []);
      } catch (error) {
        console.error(`Failed to load posts for collection ${collection.id}:`, error);
        collectionPosts.value[collection.id] = [];
      }
    }
  } catch (error) {
    console.error("Failed to load collections:", error);
    collections.value = [];
  } finally {
    loading.value = false;
  }
};

const createCollection = async () => {
  if (!newCollectionName.value.trim() || creating.value) return;
  
  creating.value = true;
  try {
    await collectionAPI.create(newCollectionName.value.trim());
    newCollectionName.value = "";
    showCreateCollectionModal.value = false;
    await loadCollections();
  } catch (error) {
    console.error("Failed to create collection:", error);
    alert("Failed to create collection");
  } finally {
    creating.value = false;
  }
};

onMounted(() => {
  loadCollections();
});
</script>

<style scoped>
.saved-page {
  min-height: 100vh;
  background: #000;
  color: #fff;
  padding: 60px 20px 20px;
}

.saved-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 32px;
  max-width: 1200px;
  margin-left: auto;
  margin-right: auto;
}

.back-btn {
  background: none;
  border: none;
  color: #fff;
  font-size: 24px;
  cursor: pointer;
  padding: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  transition: background 0.2s;
}

.back-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

.saved-header h1 {
  flex: 1;
  font-size: 24px;
  font-weight: 600;
  margin: 0;
}

.create-collection-btn {
  padding: 8px 16px;
  background: #0095f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  transition: background 0.2s;
}

.create-collection-btn:hover {
  background: #0084e0;
}

.loading {
  text-align: center;
  padding: 60px 20px;
  color: #8e8e8e;
}

.empty-state {
  text-align: center;
  padding: 80px 20px;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.empty-state h3 {
  font-size: 22px;
  margin-bottom: 8px;
}

.empty-state p {
  color: #8e8e8e;
  font-size: 14px;
}

.collections-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.collection-card {
  cursor: pointer;
  transition: transform 0.2s;
}

.collection-card:hover {
  transform: translateY(-4px);
}

.collection-cover {
  aspect-ratio: 1;
  background: #121212;
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 12px;
}

.cover-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 2px;
  height: 100%;
}

.cover-grid img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.empty-cover {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 48px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.collection-card.default .empty-cover {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.collection-info {
  padding: 0 4px;
}

.collection-info h4 {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 4px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-count {
  font-size: 14px;
  color: #8e8e8e;
  margin: 0;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #363636;
}

.modal-header h3 {
  font-size: 16px;
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
}

.modal-body {
  padding: 20px;
}

.collection-name-input {
  width: 100%;
  padding: 12px;
  background: #121212;
  border: 1px solid #363636;
  border-radius: 8px;
  color: #fff;
  font-size: 14px;
}

.collection-name-input:focus {
  outline: none;
  border-color: #0095f6;
}

.modal-footer {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid #363636;
}

.cancel-btn,
.create-btn {
  flex: 1;
  padding: 10px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.cancel-btn {
  background: #363636;
  color: #fff;
}

.cancel-btn:hover {
  background: #404040;
}

.create-btn {
  background: #0095f6;
  color: #fff;
}

.create-btn:hover:not(:disabled) {
  background: #0084e0;
}

.create-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .saved-page {
    padding: 60px 16px 16px;
  }

  .collections-grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 16px;
  }

  .saved-header h1 {
    font-size: 20px;
  }
}
</style>
