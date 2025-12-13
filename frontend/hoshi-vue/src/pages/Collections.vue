<template>
  <div class="collections-page">
    <div class="collections-container">
      <div class="page-header">
        <h1>Saved Collections</h1>
        <button
          class="create-collection-btn"
          @click="showCreateModal = true"
        >
          ➕ New Collection
        </button>
      </div>

      <div
        v-if="loading"
        class="loading-state"
      >
        <p>Loading your collections...</p>
      </div>

      <div
        v-else-if="collections.length === 0"
        class="empty-state"
      >
        <div class="empty-icon">
          🔖
        </div>
        <h2>Save your favorite posts</h2>
        <p>Organize posts you've saved into collections</p>
      </div>

      <div
        v-else
        class="collections-grid"
      >
        <!-- Default Collection -->
        <div
          v-if="defaultCollection"
          class="collection-card default"
          @click="openCollection(defaultCollection)"
        >
          <div class="collection-cover">
            <div
              v-if="collectionPosts[defaultCollection.id]?.length > 0"
              class="cover-grid"
            >
              <SecureImage
                v-for="(post, idx) in collectionPosts[defaultCollection.id].slice(0, 4)"
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
              <span>🔖</span>
            </div>
          </div>
          <div class="collection-info">
            <div class="collection-header">
              <h3>{{ defaultCollection.name }}</h3>
              <span class="default-badge">Default</span>
            </div>
            <p class="collection-count">
              {{ collectionPosts[defaultCollection.id]?.length || 0 }} posts
            </p>
          </div>
        </div>

        <!-- User Collections -->
        <div
          v-for="collection in userCollections"
          :key="collection.id"
          class="collection-card"
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
              <span>📁</span>
            </div>
          </div>
          <div class="collection-info">
            <div class="collection-header">
              <h3>{{ collection.name }}</h3>
              <button
                class="options-btn"
                @click.stop="openOptionsMenu(collection)"
              >
                ⋯
              </button>
            </div>
            <p class="collection-count">
              {{ collectionPosts[collection.id]?.length || 0 }} posts
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Collection Modal -->
    <div
      v-if="showCreateModal"
      class="modal-overlay"
      @click="showCreateModal = false"
    >
      <div
        class="modal-content"
        @click.stop
      >
        <div class="modal-header">
          <h3>New Collection</h3>
          <button
            class="close-btn"
            @click="showCreateModal = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <input
            v-model="newCollectionName"
            type="text"
            placeholder="Collection name"
            maxlength="100"
            @keyup.enter="createCollection"
          />
        </div>
        <div class="modal-footer">
          <button
            class="cancel-btn"
            @click="showCreateModal = false"
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

    <!-- Options Menu Modal -->
    <div
      v-if="showOptionsMenu && selectedCollection"
      class="modal-overlay"
      @click="showOptionsMenu = false"
    >
      <div
        class="options-modal"
        @click.stop
      >
        <button
          class="option-btn"
          @click="startRename"
        >
          <span>✏️</span>
          <span>Rename</span>
        </button>
        <button
          class="option-btn danger"
          @click="deleteCollection"
        >
          <span>🗑️</span>
          <span>Delete Collection</span>
        </button>
        <button
          class="option-btn"
          @click="showOptionsMenu = false"
        >
          <span>✕</span>
          <span>Cancel</span>
        </button>
      </div>
    </div>

    <!-- Rename Modal -->
    <div
      v-if="showRenameModal && selectedCollection"
      class="modal-overlay"
      @click="showRenameModal = false"
    >
      <div
        class="modal-content"
        @click.stop
      >
        <div class="modal-header">
          <h3>Rename Collection</h3>
          <button
            class="close-btn"
            @click="showRenameModal = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <input
            v-model="renameValue"
            type="text"
            placeholder="Collection name"
            maxlength="100"
            @keyup.enter="confirmRename"
          />
        </div>
        <div class="modal-footer">
          <button
            class="cancel-btn"
            @click="showRenameModal = false"
          >
            Cancel
          </button>
          <button
            class="create-btn"
            :disabled="!renameValue.trim() || renaming"
            @click="confirmRename"
          >
            {{ renaming ? 'Renaming...' : 'Rename' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Collection Details View -->
    <div
      v-if="showCollectionDetails && selectedCollection"
      class="collection-details-overlay"
      @click="closeCollectionDetails"
    >
      <div
        class="collection-details-modal"
        @click.stop
      >
        <div class="details-header">
          <button
            class="back-btn"
            @click="closeCollectionDetails"
          >
            ← Back
          </button>
          <h2>{{ selectedCollection.name }}</h2>
          <button
            v-if="!selectedCollection.is_default"
            class="options-btn"
            @click="openOptionsMenu(selectedCollection)"
          >
            ⋯
          </button>
          <div v-else />
        </div>

        <div
          v-if="loadingPosts"
          class="loading-posts"
        >
          Loading posts...
        </div>

        <div
          v-else-if="selectedCollectionPosts.length === 0"
          class="empty-collection"
        >
          <div class="empty-icon">
            📷
          </div>
          <h3>No saved posts yet</h3>
          <p>Posts you save will appear here</p>
        </div>

        <div
          v-else
          class="posts-grid"
        >
          <div
            v-for="post in selectedCollectionPosts"
            :key="post.id"
            class="grid-item"
            @click="openPost(post)"
          >
            <SecureImage
              :src="post.media_urls?.[0]"
              :alt="post.caption"
              loading-placeholder="/placeholder.svg"
              error-placeholder="/placeholder.svg"
            />
            <div class="post-overlay">
              <span class="stat">❤️ {{ post.like_count || 0 }}</span>
              <span class="stat">💬 {{ post.comment_count || 0 }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { collectionAPI } from "@/services/api";
import SecureImage from "@/components/SecureImage.vue";

const route = useRoute();
const router = useRouter();

interface Collection {
  id: string;
  user_id: string;
  name: string;
  is_default: boolean;
}

interface Post {
  id: string;
  author_username: string;
  author_profile_url: string;
  media_urls: string[];
  caption: string;
  like_count: number;
  comment_count: number;
  is_saved: boolean;
  is_liked?: boolean;
}

const loading = ref(true);
const collections = ref<Collection[]>([]);
const collectionPosts = ref<Record<string, Post[]>>({});
const showCreateModal = ref(false);
const newCollectionName = ref("");
const creating = ref(false);
const showOptionsMenu = ref(false);
const selectedCollection = ref<Collection | null>(null);
const showRenameModal = ref(false);
const renameValue = ref("");
const renaming = ref(false);
const showCollectionDetails = ref(false);
const selectedCollectionPosts = ref<Post[]>([]);
const loadingPosts = ref(false);

const defaultCollection = computed(() => {
  return collections.value.find(c => c.is_default);
});

const userCollections = computed(() => {
  return collections.value.filter(c => !c.is_default).sort((a, b) => a.name.localeCompare(b.name));
});

const loadCollections = async () => {
  try {
    loading.value = true;
    const response = await collectionAPI.getAll();
    collections.value = Array.isArray(response) ? response : (response.collections || []);

    // Load preview posts for each collection
    for (const collection of collections.value) {
      try {
        const postsResponse = await collectionAPI.getPosts(collection.id, 1, 4);
        collectionPosts.value[collection.id] = Array.isArray(postsResponse) 
          ? postsResponse 
          : (postsResponse?.posts || []);
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

  try {
    creating.value = true;
    await collectionAPI.create(newCollectionName.value.trim());
    showCreateModal.value = false;
    newCollectionName.value = "";
    await loadCollections();
  } catch (error: any) {
    console.error("Failed to create collection:", error);
    alert(error.response?.data?.error || "Failed to create collection");
  } finally {
    creating.value = false;
  }
};

const openOptionsMenu = (collection: Collection) => {
  selectedCollection.value = collection;
  showOptionsMenu.value = true;
  showCollectionDetails.value = false;
};

const startRename = () => {
  if (!selectedCollection.value) return;
  renameValue.value = selectedCollection.value.name;
  showOptionsMenu.value = false;
  showRenameModal.value = true;
};

const confirmRename = async () => {
  if (!selectedCollection.value || !renameValue.value.trim() || renaming.value) return;

  try {
    renaming.value = true;
    await collectionAPI.rename(selectedCollection.value.id, renameValue.value.trim());
    showRenameModal.value = false;
    selectedCollection.value = null;
    renameValue.value = "";
    await loadCollections();
  } catch (error: any) {
    console.error("Failed to rename collection:", error);
    alert(error.response?.data?.error || "Failed to rename collection");
  } finally {
    renaming.value = false;
  }
};

const deleteCollection = async () => {
  if (!selectedCollection.value) return;

  const confirmed = confirm(`Are you sure you want to delete "${selectedCollection.value.name}"? This action cannot be undone.`);
  if (!confirmed) return;

  try {
    await collectionAPI.delete(selectedCollection.value.id);
    showOptionsMenu.value = false;
    selectedCollection.value = null;
    await loadCollections();
  } catch (error: any) {
    console.error("Failed to delete collection:", error);
    alert(error.response?.data?.error || "Failed to delete collection");
  }
};

const openCollection = async (collection: Collection) => {
  selectedCollection.value = collection;
  showCollectionDetails.value = true;
  
  try {
    loadingPosts.value = true;
    const response = await collectionAPI.getPosts(collection.id, 1, 50);
    selectedCollectionPosts.value = Array.isArray(response) 
      ? response 
      : (response?.posts || []);
  } catch (error) {
    console.error("Failed to load collection posts:", error);
    selectedCollectionPosts.value = [];
  } finally {
    loadingPosts.value = false;
  }
};

const closeCollectionDetails = () => {
  // Check if we came from Profile by checking if route has id param
  if (route.params.id) {
    // Navigate back to Profile saved tab
    router.back();
  } else {
    // Just close the modal
    showCollectionDetails.value = false;
    selectedCollection.value = null;
    selectedCollectionPosts.value = [];
  }
};

const openPost = (post: Post) => {
  router.push(`/p/${post.id}`);
};

onMounted(async () => {
  await loadCollections();
  
  // If there's a collection ID in the route, open that collection
  const collectionId = route.params.id as string;
  if (collectionId) {
    const collection = collections.value.find(c => c.id === collectionId);
    if (collection) {
      await openCollection(collection);
    }
  }
});
</script>

<style scoped lang="scss">
.collections-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.collections-container {
  max-width: 1200px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 32px;

  h1 {
    font-size: 32px;
    font-weight: 700;
  }

  .create-collection-btn {
    padding: 10px 20px;
    background: #0095f6;
    border: none;
    border-radius: 8px;
    color: #fff;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;

    &:hover {
      background: #1877f2;
    }
  }
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #a8a8a8;

  .empty-icon {
    font-size: 64px;
    margin-bottom: 16px;
  }

  h2 {
    font-size: 24px;
    margin-bottom: 8px;
    color: #fff;
  }

  p {
    font-size: 16px;
    color: #a8a8a8;
  }
}

.collections-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
}

.collection-card {
  background: #262626;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  &.default {
    border: 2px solid #0095f6;
  }

  .collection-cover {
    position: relative;
    width: 100%;
    padding-top: 100%;
    background: #1a1a1a;
    overflow: hidden;

    .cover-grid {
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      grid-template-rows: repeat(2, 1fr);
      gap: 2px;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }

    .empty-cover {
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 48px;
    }
  }

  .collection-info {
    padding: 16px;

    .collection-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 4px;

      h3 {
        font-size: 16px;
        font-weight: 600;
        margin: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        flex: 1;
      }

      .default-badge {
        background: #0095f6;
        color: #fff;
        font-size: 10px;
        font-weight: 600;
        padding: 2px 8px;
        border-radius: 4px;
        margin-left: 8px;
      }

      .options-btn {
        background: none;
        border: none;
        color: #fff;
        font-size: 20px;
        cursor: pointer;
        padding: 4px 8px;
        margin-left: 8px;
        border-radius: 4px;
        transition: background 0.2s;

        &:hover {
          background: rgba(255, 255, 255, 0.1);
        }
      }
    }

    .collection-count {
      font-size: 14px;
      color: #a8a8a8;
      margin: 0;
    }
  }
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.modal-content {
  background: #262626;
  border-radius: 12px;
  width: 400px;
  max-width: 90vw;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #363636;

  h3 {
    font-size: 16px;
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
    border-radius: 4px;
    transition: background 0.2s;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
    }
  }
}

.modal-body {
  padding: 20px;

  input {
    width: 100%;
    padding: 12px;
    background: #000;
    border: 1px solid #363636;
    border-radius: 8px;
    color: #fff;
    font-size: 14px;

    &:focus {
      outline: none;
      border-color: #0095f6;
    }

    &::placeholder {
      color: #737373;
    }
  }
}

.modal-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding: 12px 20px;
  border-top: 1px solid #363636;

  button {
    padding: 8px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  .cancel-btn {
    background: transparent;
    border: 1px solid #363636;
    color: #fff;

    &:hover:not(:disabled) {
      background: rgba(255, 255, 255, 0.05);
    }
  }

  .create-btn {
    background: #0095f6;
    border: none;
    color: #fff;

    &:hover:not(:disabled) {
      background: #1877f2;
    }
  }
}

.options-modal {
  background: #262626;
  border-radius: 12px;
  overflow: hidden;
  min-width: 300px;

  .option-btn {
    width: 100%;
    padding: 16px 20px;
    background: none;
    border: none;
    border-bottom: 1px solid #363636;
    color: #fff;
    font-size: 14px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 12px;
    transition: background 0.2s;

    &:last-child {
      border-bottom: none;
    }

    &:hover {
      background: rgba(255, 255, 255, 0.05);
    }

    &.danger {
      color: #ed4956;
    }

    span:first-child {
      font-size: 20px;
    }
  }
}

.collection-details-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.95);
  z-index: 9999;
  overflow-y: auto;
}

.collection-details-modal {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  min-height: 100vh;

  .details-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 32px;
    padding-bottom: 20px;
    border-bottom: 1px solid #363636;

    .back-btn {
      background: none;
      border: none;
      color: #fff;
      font-size: 16px;
      cursor: pointer;
      padding: 8px 16px;
      border-radius: 8px;
      transition: background 0.2s;

      &:hover {
        background: rgba(255, 255, 255, 0.1);
      }
    }

    h2 {
      flex: 1;
      text-align: center;
      font-size: 24px;
      margin: 0;
    }

    .options-btn {
      background: none;
      border: none;
      color: #fff;
      font-size: 24px;
      cursor: pointer;
      padding: 8px 16px;
      border-radius: 8px;
      transition: background 0.2s;

      &:hover {
        background: rgba(255, 255, 255, 0.1);
      }
    }
  }

  .loading-posts,
  .empty-collection {
    text-align: center;
    padding: 60px 20px;
    color: #a8a8a8;

    .empty-icon {
      font-size: 64px;
      margin-bottom: 16px;
    }

    h3 {
      font-size: 20px;
      margin-bottom: 8px;
      color: #fff;
    }

    p {
      font-size: 14px;
      color: #a8a8a8;
    }
  }

  .posts-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;

    .grid-item {
      position: relative;
      aspect-ratio: 1;
      cursor: pointer;
      overflow: hidden;
      border-radius: 4px;
      background: #262626;

      img {
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
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 20px;
        opacity: 0;
        transition: opacity 0.2s;

        .stat {
          color: #fff;
          font-size: 16px;
          font-weight: 600;
        }
      }

      &:hover {
        img {
          transform: scale(1.05);
        }

        .post-overlay {
          opacity: 1;
        }
      }
    }
  }
}

@media (max-width: 1024px) {
  .collections-page {
    padding-left: calc(72px + 20px);
  }
}

@media (max-width: 768px) {
  .collections-page {
    padding-left: calc(60px + 20px);
  }

  .collections-grid,
  .posts-grid {
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  }
}
</style>
