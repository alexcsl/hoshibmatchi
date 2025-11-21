<template>
  <div class="app-wrapper">
    <!-- Main layout for authenticated pages -->
    <template v-if="isAuthPage">
      <div class="app-layout">
        <!-- Sidebar Navigation -->
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

        <!-- Main Content -->
        <main class="main-content">
          <router-view />
        </main>

        <!-- Mini Message Component (not on Messages page) -->
        <MiniMessage v-if="!isMessagesPage" :messages="recentMessages" @click="navigateToMessages" />
      </div>

      <!-- Overlays -->
      <SearchPanel v-if="isSearchOpen" @close="isSearchOpen = false" />
      <NotificationOverlay v-if="isNotificationsOpen" @close="isNotificationsOpen = false" />
      <CreatePostOverlay v-if="isCreatePostOpen" @close="isCreatePostOpen = false" @posted="handlePostCreated" />
      <PostDetailsOverlay 
        v-if="isPostDetailsOpen && selectedPostId" 
        @close="isPostDetailsOpen = false" 
        :post-id="selectedPostId"
        :context="postContext"
        @like="handlePostDetailsLike"
        @save="handlePostDetailsSave"
      />
      <ReelsViewer 
        v-if="isReelsViewerOpen" 
        @close="isReelsViewerOpen = false" 
        :initial-index="currentReelIndex"
      />
      <StoryViewer v-if="isStoryViewerOpen" @close="isStoryViewerOpen = false" :stories="stories" :initial-index="currentStoryIndex" />
    </template>

    <!-- Auth pages without sidebar -->
    <template v-else>
      <router-view />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useFeedStore } from '@/stores/feed'
import Sidebar from './components/Sidebar.vue'
import MiniMessage from './components/MiniMessage.vue'
import SearchOverlay from './components/SearchOverlay.vue'
import SearchPanel from './components/SearchPanel.vue'
import NotificationOverlay from './components/NotificationOverlay.vue'
import CreatePostOverlay from './components/CreatePostOverlay.vue'
import PostDetailsOverlay from './components/PostDetailsOverlay.vue'
import StoryViewer from './components/StoryViewer.vue'
import ReelsViewer from './components/ReelsViewer.vue'

// Extend Window interface for global functions
declare global {
  interface Window {
    openPostDetails: (postId: string, context?: string) => void
    openStoryViewer: (index?: number) => void
    openReelsViewer: (index: number) => void
  }
}

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const feedStore = useFeedStore()

const isSearchOpen = ref(false)
const isNotificationsOpen = ref(false)
const isCreatePostOpen = ref(false)
const isPostDetailsOpen = ref(false)
const isStoryViewerOpen = ref(false)
const isReelsViewerOpen = ref(false)
const currentStoryIndex = ref(0)
const currentReelIndex = ref(0)
const selectedPostId = ref<string | null>(null)
const postContext = ref<string>('feed')

const authPages = ['Login', 'SignUp', 'LoginOTP', 'VerifyOTP', 'ResetPassword', 'ForgotPassword', 'GoogleCallback', 'Register']
const isAuthPage = computed(() => !authPages.includes(route.name as string) && route.path !== '/')

const currentRoute = computed(() => {
  const name = route.path.split('/')[1] || 'feed'
  return name
})

const isMessagesPage = computed(() => currentRoute.value === 'messages')

// Mock data - replace with real data from backend
const recentMessages = ref([
  { id: 1, username: 'user_1', avatar: '/placeholder.svg?height=40&width=40', unreadCount: 2 },
  { id: 2, username: 'user_2', avatar: '/placeholder.svg?height=40&width=40', unreadCount: 0 },
  { id: 3, username: 'user_3', avatar: '/placeholder.svg?height=40&width=40', unreadCount: 1 }
])

const stories = ref([
  { 
    id: '1', 
    author_username: 'user_1', 
    author_profile_url: '/placeholder.svg?height=48&width=48', 
    media_url: '/placeholder.svg?height=800&width=500', 
    created_at: new Date().toISOString() 
  },
  { 
    id: '2', 
    author_username: 'user_2', 
    author_profile_url: '/placeholder.svg?height=48&width=48', 
    media_url: '/placeholder.svg?height=800&width=500', 
    created_at: new Date().toISOString() 
  }
])

const handlePostCreated = () => {
  // Refresh feed after post is created
  feedStore.loadHomeFeed(1)
}

const handlePostDetailsLike = async (postId: string) => {
  await feedStore.toggleLike(postId, 'home')
}

const handlePostDetailsSave = async (postId: string) => {
  await feedStore.toggleSave(postId, '1', 'home')
}

const handleNavigation = (path: string) => {
  router.push(`/${path}`)
}

const navigateToMessages = () => {
  router.push('/messages')
}

const handleSettingsClick = () => {
  router.push('/settings')
}

const handleSavedClick = () => {
  router.push('/archive')
}

const handleLogout = () => {
  // Clear auth state using store
  authStore.logout()
  // Redirect to login page
  router.push('/login')
}

window.openPostDetails = (postId: string, context: string = 'feed') => {
  selectedPostId.value = postId
  postContext.value = context
  isPostDetailsOpen.value = true
}

window.openReelsViewer = (index: number) => {
  currentReelIndex.value = index
  isReelsViewerOpen.value = true
}

window.openStoryViewer = (index: number = 0) => {
  currentStoryIndex.value = index
  isStoryViewerOpen.value = true
}
</script>

<style scoped lang="scss">
.app-wrapper {
  width: 100%;
  min-height: 100vh;
  background-color: #000;
}

.app-layout {
  display: flex;
  width: 100%;
  min-height: 100vh;
  position: relative;
}

.main-content {
  flex: 1;
  overflow-y: auto;
  background-color: #000;
}
</style>
