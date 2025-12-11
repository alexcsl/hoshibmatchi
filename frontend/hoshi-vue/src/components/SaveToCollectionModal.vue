<template>
  <div
    class="modal-overlay"
    @click.stop="$emit('close')"
  >
    <div
      class="modal-content"
      @click.stop
    >
      <div class="modal-header">
        <h3>Save to Collection</h3>
        <button
          class="close-btn"
          @click="$emit('close')"
        >
          ✕
        </button>
      </div>

      <div
        v-if="loading"
        class="loading-state"
      >
        Loading collections...
      </div>

      <div
        v-else
        class="collections-list"
      >
        <!-- Default Collection -->
        <div
          v-if="defaultCollection"
          class="collection-item default"
          @click="selectCollection(defaultCollection)"
        >
          <div class="collection-icon">
            🔖
          </div>
          <div class="collection-info">
            <div class="collection-name">
              {{ defaultCollection.name }}
            </div>
            <div class="collection-count">
              Default collection
            </div>
          </div>
          <div
            v-if="isPostInCollection(defaultCollection.id)"
            class="checkmark"
          >
            ✓
          </div>
        </div>

        <!-- User Collections -->
        <div
          v-for="collection in userCollections"
          :key="collection.id"
          class="collection-item"
          @click="selectCollection(collection)"
        >
          <div class="collection-icon">
            📁
          </div>
          <div class="collection-info">
            <div class="collection-name">
              {{ collection.name }}
            </div>
          </div>
          <div
            v-if="isPostInCollection(collection.id)"
            class="checkmark"
          >
            ✓
          </div>
        </div>

        <!-- Create New Collection -->
        <div
          v-if="!showCreateForm"
          class="collection-item create-new"
          @click="showCreateForm = true"
        >
          <div class="collection-icon">
            ➕
          </div>
          <div class="collection-info">
            <div class="collection-name">
              Create New Collection
            </div>
          </div>
        </div>

        <!-- Create Form -->
        <div
          v-if="showCreateForm"
          class="create-form"
        >
          <input
            v-model="newCollectionName"
            type="text"
            placeholder="Collection name"
            maxlength="100"
            @keyup.enter="createCollection"
          />
          <div class="form-actions">
            <button
              class="cancel-btn"
              @click="showCreateForm = false; newCollectionName = ''"
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onBeforeUnmount } from "vue";
import { collectionAPI } from "@/services/api";

interface Collection {
  id: string;
  user_id: string;
  name: string;
  is_default: boolean;
}

const props = defineProps<{
  postId: string;
  savedCollectionIds?: string[]; // IDs of collections this post is already in
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'saved', collectionId: string): void;
  (e: 'unsaved', collectionId: string): void;
}>();

// Log when component mounts/unmounts
console.warn(`🆕 SaveToCollectionModal CREATED for post ${props.postId}`);

// Reactive state
const loading = ref(false);
const collections = ref<Collection[]>([]);
const showCreateForm = ref(false);
const newCollectionName = ref("");
const creating = ref(false);
const localSavedCollectionIds = ref<string[]>([]);
const isTogglingCollection = ref(false);

onMounted(() => {
  console.warn(`⬆️ SaveToCollectionModal MOUNTED for post ${props.postId}`);
  console.warn(`   Initial savedCollectionIds from props: ${JSON.stringify(props.savedCollectionIds)}`);
  // Initialize local saved state from props - use reactive copy
  localSavedCollectionIds.value = props.savedCollectionIds ? [...props.savedCollectionIds] : [];
  console.warn(`   Initialized localSavedCollectionIds: ${JSON.stringify(localSavedCollectionIds.value)}`);
  loadCollections();
});

onBeforeUnmount(() => {
  console.warn(`⬇️ SaveToCollectionModal UNMOUNTING for post ${props.postId}`);
});

const defaultCollection = computed(() => {
  return collections.value.find(c => c.is_default);
});

const userCollections = computed(() => {
  return collections.value.filter(c => !c.is_default);
});

const isPostInCollection = (collectionId: string) => {
  const result = localSavedCollectionIds.value.includes(collectionId);
  console.log(`   📍 isPostInCollection(${collectionId}): ${result} | localIds: ${JSON.stringify(localSavedCollectionIds.value)}`);
  return result;
};

const loadCollections = async () => {
  try {
    loading.value = true;
    const response = await collectionAPI.getAll();
    collections.value = Array.isArray(response) ? response : (response.collections || []);
  } catch (error) {
    console.error("Failed to load collections:", error);
  } finally {
    loading.value = false;
  }
};

const selectCollection = async (collection: Collection) => {
  console.warn(`🔵 selectCollection CALLED: ${collection.name} (ID: ${collection.id})`);
  console.warn(`   Current lock status: ${isTogglingCollection.value}`);
  console.warn(`   Local saved IDs: ${JSON.stringify(localSavedCollectionIds.value)}`);
  
  // Prevent multiple simultaneous toggles
  if (isTogglingCollection.value) {
    console.warn(`   ⛔ BLOCKED: Already toggling`);
    return;
  }

  const isCurrentlySaved = isPostInCollection(collection.id);
  const numericPostId = parseInt(props.postId);

  console.warn(`   isCurrentlySaved: ${isCurrentlySaved}`);
  console.warn(`   Will ${isCurrentlySaved ? 'UNSAVE' : 'SAVE'}`);

  if (isNaN(numericPostId)) {
    console.error("Invalid post ID");
    return;
  }

  isTogglingCollection.value = true;
  console.warn(`   🔒 LOCK SET`);
  let actionCompleted = false;
  let wasSaved = false;
  
  try {
    if (isCurrentlySaved) {
      // Unsave from this collection
      await collectionAPI.unsavePost(collection.id, props.postId);
      // Update local state immediately
      localSavedCollectionIds.value = localSavedCollectionIds.value.filter(id => id !== collection.id);
      actionCompleted = true;
      wasSaved = false;
    } else {
      // Save to this collection
      await collectionAPI.savePost(collection.id, numericPostId);
      // Update local state immediately - CREATE NEW ARRAY for reactivity
      localSavedCollectionIds.value = [...localSavedCollectionIds.value, collection.id];
      actionCompleted = true;
      wasSaved = true;
    }
  } catch (error: unknown) {
    console.error("Failed to toggle save:", error);
    const errorObj = error as { response?: { data?: { error?: string } } };
    const errorMessage = errorObj?.response?.data?.error || "Failed to save post";
    alert(errorMessage);
  } finally {
    isTogglingCollection.value = false;
    console.warn(`   🔓 LOCK CLEARED`);
    // Emit AFTER clearing the lock to prevent race conditions
    if (actionCompleted) {
      if (wasSaved) {
        console.warn(`   📤 EMITTING "saved" for collection ${collection.id}`);
        emit("saved", collection.id);
      } else {
        console.warn(`   📤 EMITTING "unsaved" for collection ${collection.id}`);
        emit("unsaved", collection.id);
      }
    }
    console.warn(`🔵 selectCollection COMPLETED for ${collection.name}`);
  }
};

const createCollection = async () => {
  if (!newCollectionName.value.trim() || creating.value) return;

  try {
    creating.value = true;
    const newCollection = await collectionAPI.create(newCollectionName.value.trim());
    
    // Add to collections list
    await loadCollections();
    
    // Automatically save post to new collection
    const numericPostId = parseInt(props.postId);
    if (!isNaN(numericPostId)) {
      await collectionAPI.savePost(newCollection.id, numericPostId);
      // Update local state immediately
      localSavedCollectionIds.value.push(newCollection.id);
      emit("saved", newCollection.id);
    }
    
    // Reset form
    showCreateForm.value = false;
    newCollectionName.value = "";
  } catch (error: unknown) {
    console.error("Failed to create collection:", error);
    const errorObj = error as { response?: { data?: { error?: string } } };
    const errorMessage = errorObj?.response?.data?.error || "Failed to create collection";
    alert(errorMessage);
  } finally {
    creating.value = false;
  }
};

// Initialize local saved state from props on mount only
// Don't watch for prop changes to avoid race conditions with our own updates

onMounted(() => {
  // Initialize local saved state from props
  localSavedCollectionIds.value = [...(props.savedCollectionIds || [])];
  loadCollections();
});
</script>

<style scoped lang="scss">
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
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
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

.loading-state {
  padding: 40px 20px;
  text-align: center;
  color: #a8a8a8;
}

.collections-list {
  overflow-y: auto;
  padding: 8px 0;
  max-height: calc(80vh - 60px);
}

.collection-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  cursor: pointer;
  transition: background 0.2s;

  &:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  &.default {
    background: rgba(255, 255, 255, 0.03);
  }

  &.create-new {
    color: #0095f6;
    font-weight: 600;

    .collection-icon {
      font-size: 20px;
    }
  }

  .collection-icon {
    font-size: 24px;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 8px;
  }

  .collection-info {
    flex: 1;

    .collection-name {
      font-size: 14px;
      font-weight: 500;
      margin-bottom: 2px;
    }

    .collection-count {
      font-size: 12px;
      color: #a8a8a8;
    }
  }

  .checkmark {
    color: #0095f6;
    font-size: 20px;
    font-weight: bold;
  }
}

.create-form {
  padding: 12px 20px;
  border-top: 1px solid #363636;
  background: rgba(255, 255, 255, 0.02);

  input {
    width: 100%;
    padding: 10px 12px;
    background: #000;
    border: 1px solid #363636;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    margin-bottom: 12px;

    &:focus {
      outline: none;
      border-color: #0095f6;
    }

    &::placeholder {
      color: #737373;
    }
  }

  .form-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;

    button {
      padding: 8px 16px;
      border-radius: 6px;
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
}
</style>
