<template>
  <div class="main-layout">
    <Sidebar 
      :current-route="currentRoute"
      @navigate="handleNavigation"
      @open-search="isSearchOpen = true"
      @open-notifications="isNotificationsOpen = true"
      @open-create="isCreatePostOpen = true"
      @open-settings="handleSettingsClick"
      @open-saved="handleSavedClick"
      @logout="handleLogout"
    />
    
    <main class="main-content">
      <router-view />
    </main>

    <!-- Mini Message Component (not on Messages page) -->
    <MiniMessage
      v-if="!isMessagesPage"
      :messages="recentMessages"
      @click="navigateToMessages"
    />

    <!-- Overlays -->
    <SearchPanel
      v-if="isSearchOpen"
      @close="isSearchOpen = false"
    />
    <NotificationOverlay
      v-if="isNotificationsOpen"
      @close="isNotificationsOpen = false"
    />
    <CreatePostOverlay
      v-if="isCreatePostOpen"
      @close="isCreatePostOpen = false"
      @posted="handlePostCreated"
    />
    <PostDetailsOverlay 
      v-if="isPostDetailsOpen && selectedPostId" 
      :post-id="selectedPostId" 
      :context="postContext"
      @close="isPostDetailsOpen = false"
      @like="handlePostDetailsLike"
      @save="handlePostDetailsSave"
    />
    <ReelsViewer 
      v-if="isReelsViewerOpen" 
      :initial-index="currentReelIndex" 
      @close="isReelsViewerOpen = false"
    />
    <StoryViewer
      v-if="isStoryViewerOpen"
      :stories="stories"
      :initial-index="currentStoryIndex"
      @close="isStoryViewerOpen = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useFeedStore } from "@/stores/feed";
import Sidebar from "../components/Sidebar.vue";
import MiniMessage from "../components/MiniMessage.vue";
import SearchPanel from "../components/SearchPanel.vue";
import NotificationOverlay from "../components/NotificationOverlay.vue";
import CreatePostOverlay from "../components/CreatePostOverlay.vue";
import PostDetailsOverlay from "../components/PostDetailsOverlay.vue";
import StoryViewer from "../components/StoryViewer.vue";
import ReelsViewer from "../components/ReelsViewer.vue";

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const feedStore = useFeedStore();

const isSearchOpen = ref(false);
const isNotificationsOpen = ref(false);
const isCreatePostOpen = ref(false);
const isPostDetailsOpen = ref(false);
const isStoryViewerOpen = ref(false);
const isReelsViewerOpen = ref(false);
const currentStoryIndex = ref(0);
const currentReelIndex = ref(0);
const selectedPostId = ref<string | null>(null);
const postContext = ref<string>("feed");

const currentRoute = computed(() => {
  const name = route.path.split("/")[1] || "feed";
  return name;
});

const isMessagesPage = computed(() => currentRoute.value === "messages");

const recentMessages = ref([
  { id: 1, username: "user_1", avatar: "/placeholder.svg?height=40&width=40", unreadCount: 2 },
  { id: 2, username: "user_2", avatar: "/placeholder.svg?height=40&width=40", unreadCount: 0 },
  { id: 3, username: "user_3", avatar: "/placeholder.svg?height=40&width=40", unreadCount: 1 }
]);

const stories = ref([
  { 
    id: "1", 
    author_username: "user_1", 
    author_profile_url: "/placeholder.svg?height=48&width=48", 
    media_url: "/placeholder.svg?height=800&width=500", 
    created_at: new Date().toISOString() 
  },
  { 
    id: "2", 
    author_username: "user_2", 
    author_profile_url: "/placeholder.svg?height=48&width=48", 
    media_url: "/placeholder.svg?height=800&width=500", 
    created_at: new Date().toISOString() 
  }
]);

const handlePostCreated = () => {
  feedStore.loadHomeFeed(1);
};

const handleNavigation = (path: string) => {
  if (path === "profile") {
    router.push(`/profile/${authStore.user?.username || ""}`);
  } else {
    router.push(`/${path}`);
  }
};

const handleSettingsClick = () => {
  router.push("/settings");
};

const handleSavedClick = () => {
  router.push("/saved");
};

const navigateToMessages = () => {
  router.push("/messages");
};

const handleLogout = () => {
  authStore.logout();
  router.push("/login");
};

const handlePostDetailsLike = () => {
  // Refresh feed
};

const handlePostDetailsSave = () => {
  // Handle save
};

// Expose global functions for overlays
if (typeof window !== "undefined") {
  window.openPostDetails = (postId: string, context = "feed") => {
    selectedPostId.value = postId;
    postContext.value = context;
    isPostDetailsOpen.value = true;
  };
  
  window.openStoryViewer = (index = 0) => {
    currentStoryIndex.value = index;
    isStoryViewerOpen.value = true;
  };
  
  window.openReelsViewer = (index: number) => {
    currentReelIndex.value = index;
    isReelsViewerOpen.value = true;
  };
}

declare global {
  interface Window {
    openPostDetails: (postId: string, context?: string) => void;
    openStoryViewer: (index?: number) => void;
    openReelsViewer: (index: number) => void;
  }
}
</script>

<style scoped>
.main-layout {
  display: flex;
  background-color: #000; /* Example background */
  color: #fff;
}

.main-content {
  /* It must be positioned to the *right* of the sidebar */
  /* This value MUST match the width of LeftSidebar */
  margin-left: 244px; 
  
  /* It must take up the remaining width */
  width: calc(100% - 244px);
  
  /* It must have its *own* scrollbar */
  height: 100vh;
  overflow-y: auto;
  
  padding: 2rem; /* Example padding */
}

/* Responsive adjustments for sidebar width changes */
@media (max-width: 1264px) {
  .main-content {
    margin-left: 244px;
    width: calc(100% - 244px);
    padding: 1.5rem;
  }
}

@media (max-width: 1024px) {
  .main-content {
    margin-left: 72px;
    width: calc(100% - 72px);
    padding: 1rem;
  }
}

@media (max-width: 768px) {
  .main-content {
    margin-left: 60px;
    width: calc(100% - 60px);
    padding: 0.5rem;
  }
}
</style>