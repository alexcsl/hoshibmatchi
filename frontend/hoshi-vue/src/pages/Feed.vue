<template>
  <div class="feed-container">
    <div class="feed-wrapper">
      <main class="main-feed">
        <!-- Stories -->
        <div class="stories">
          <div class="story-item your-story" @click="handleCreateStory">
            <img 
              :src="authStore.user?.profile_picture_url || '/default-avatar.svg'" 
              alt="Your story" 
              class="story-image" 
            />
            <div class="story-label">Your story</div>
          </div>

            <div 
              v-for="group in feedStore.storyFeed" 
              :key="group.user_id" 
              class="story-item"
              :class="{ 'seen': group.all_seen }"
              @click="openStoryViewer(group)"
            >
              <img 
                  :src="group.user_profile_url || '/default-avatar.svg'" 
                  :alt="group.username" 
                  class="story-image" 
              />
              <div class="story-label">{{ group.username }}</div>
            </div>
          </div>

        <StoryViewer 
            v-if="showStoryViewer && selectedStoryGroup"
            :stories="selectedStoryGroup.stories"
            @close="showStoryViewer = false"
        />

        <!-- Posts -->
        <div class="posts" ref="postsContainer">
          <div v-if="feedStore.loading && feedStore.homeFeed.length === 0" class="loading-skeleton">
            <div v-for="i in 3" :key="`skeleton-${i}`" class="skeleton-post">
              <div class="skeleton-header"></div>
              <div class="skeleton-image"></div>
              <div class="skeleton-actions"></div>
            </div>
          </div>

          <PostCard
            v-for="post in feedStore.homeFeed"
            :key="post.id"
            :post="post"
            @like="handleLike"
            @save="handleSave"
            @comment="handleComment"
            @share="handleShare"
            @open-details="handleOpenPostDetails"
            @open-options="handleOpenOptions"
          />

          <div v-if="feedStore.loading && feedStore.homeFeed.length > 0" class="loading-more">
            Loading more posts...
          </div>

          <div v-if="!feedStore.loading && !feedStore.hasMore && feedStore.homeFeed.length > 0" class="end-message">
            You're all caught up!
          </div>

          <div v-if="!feedStore.loading && feedStore.homeFeed.length === 0" class="empty-feed">
            <p>No posts to show yet</p>
            <p class="empty-subtitle">Follow people to see their posts here</p>
          </div>
        </div>
      </main>

      <!-- Sidebar -->
      <aside class="sidebar">
        <div class="user-card">
          <img src="/placeholder.svg?height=56&width=56" alt="User" class="profile-pic" />
          <div class="user-details">
            <div class="username">username</div>
            <div class="fullname">Full Name</div>
          </div>
          <button class="switch-btn">Switch</button>
        </div>

        <div class="suggestions">
          <div class="suggestions-header">
            <h3>Suggestions For You</h3>
            <a href="#">See All</a>
          </div>
          <div v-for="i in 5" :key="`suggestion-${i}`" class="suggestion-item">
            <img :src="`/placeholder.svg?height=32&width=32&query=suggestion-${i}`" alt="Suggestion" />
            <div class="suggestion-info">
              <div class="username">user_{{ i }}</div>
              <div class="mutual">Suggested for you</div>
            </div>
            <button class="follow-btn">Follow</button>
          </div>
        </div>

        <footer class="sidebar-footer">
          <a href="#">About</a>
          <a href="#">Help</a>
          <a href="#">Press</a>
          <a href="#">API</a>
          <a href="#">Jobs</a>
          <a href="#">Privacy</a>
          <a href="#">Terms</a>
          <a href="#">Locations</a>
          <a href="#">Language</a>
        </footer>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router' // Import Router
import { useFeedStore } from '@/stores/feed'
import { useAuthStore } from '@/stores/auth'
import { commentAPI } from '@/services/api'
import PostCard from '@/components/PostCard.vue'
import StoryViewer from '@/components/StoryViewer.vue'

const router = useRouter()
const feedStore = useFeedStore()
const authStore = useAuthStore()
const postsContainer = ref<HTMLElement | null>(null)
const showStoryViewer = ref(false)
const selectedStoryGroup = ref<any>(null)

onMounted(async () => {
  // Load initial feed
  if (feedStore.homeFeed.length === 0) {
    console.log('Loading initial home feed...')
    await feedStore.loadHomeFeed(1)
    console.log('Home feed loaded:', feedStore.homeFeed.length, 'posts')
    console.log('Feed data:', feedStore.homeFeed)
  }

  await feedStore.loadStoryFeed()
  // Setup infinite scroll
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})

const handleScroll = () => {
  if (feedStore.loading || !feedStore.hasMore) return

  const scrollPosition = window.innerHeight + window.scrollY
  const threshold = document.body.offsetHeight - 500

  if (scrollPosition >= threshold) {
    feedStore.loadHomeFeed(feedStore.homePage + 1)
  }
}

const handleLike = async (postId: string) => {
  await feedStore.toggleLike(postId, 'home')
}

const handleSave = async (postId: string) => {
  await feedStore.toggleSave(postId, '1', 'home')
}

const handleComment = async (postId: string, content: string) => {
  try {
    const numericPostId = parseInt(postId)
    if (isNaN(numericPostId)) {
      console.error('Invalid post ID:', postId)
      return
    }
    
    await commentAPI.createComment({
      post_id: numericPostId,
      content
    })
    
    // Increment comment count locally
    const post = feedStore.homeFeed.find(p => p.id === postId)
    if (post) {
      feedStore.updatePost(postId, {
        comment_count: (post.comment_count || 0) + 1
      } as any)
    }
  } catch (error: any) {
    console.error('Failed to add comment:', error)
    console.error('Error details:', error.response?.data || error.message)
    
    // Show user-friendly error
    if (error.response?.status === 500) {
      console.warn('Server error when creating comment. The backend may need to be checked.')
    }
  }
}

const handleShare = async (postId: string) => {
  // TODO: Implement share functionality
  console.log('Share post:', postId)
}

const handleOpenPostDetails = (postId: string) => {
  if (window.openPostDetails) {
    window.openPostDetails(postId)
  }
}

const handleCreateStory = () => {
    // We can reuse CreatePostOverlay, but pass a prop or mode?
    // Or strictly speaking, "Create" button in sidebar usually handles both.
    // For now, trigger the global create event or router push
    router.push('/create-story')   // If Feed emits this, or use a global bus/store
    // Actually, sidebar handles 'Create'. Let's assume we open the overlay
    // but set `isReel` to false and maybe we need an `isStory` flag in CreatePostOverlay later.
    // For MVP: Just alert or log
    console.log("Open Create Overlay in Story Mode")
}

const openStoryViewer = (group: any) => {
    selectedStoryGroup.value = group
    showStoryViewer.value = true
}

const handleOpenOptions = (postId: string) => {
  // TODO: Implement post options menu
  console.log('Open options for post:', postId)
}
</script>

<style scoped lang="scss">
.feed-container {
  width: 100%;
  background-color: #000;
  min-height: 100vh;
  padding-left: 244px;
}

.feed-wrapper {
  display: flex;
  gap: 32px;
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
}

.main-feed {
  flex: 1;
  max-width: 620px;
}

.stories {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  overflow-x: auto;
  padding-bottom: 8px;

  &::-webkit-scrollbar {
    height: 4px;
  }

  .story-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    flex-shrink: 0;
    position: relative;

    .story-image {
      width: 56px;
      height: 56px;
      border-radius: 50%;
      border: 2px solid #404040;
      object-fit: cover;
      transition: border-color 0.2s;
    }

    .story-label {
      font-size: 12px;
      color: #a8a8a8;
      text-align: center;
      max-width: 56px;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    &:hover .story-image {
      border-color: #0a66c2;
    }

    &.your-story .story-image {
      border: 2px solid #404040;
    }

    &.your-story::after {
      content: '➕';
      position: absolute;
      bottom: 0;
      right: 0;
      background-color: #0a66c2;
      border: 2px solid #000;
      border-radius: 50%;
      width: 20px;
      height: 20px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 10px;
      color: #fff;
    }
  }
}

.posts {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.loading-skeleton {
  display: flex;
  flex-direction: column;
  gap: 20px;

  .skeleton-post {
    border: 1px solid #262626;
    border-radius: 1px;
    overflow: hidden;

    .skeleton-header {
      height: 60px;
      background: linear-gradient(90deg, #1a1a1a 25%, #2a2a2a 50%, #1a1a1a 75%);
      background-size: 200% 100%;
      animation: shimmer 1.5s infinite;
    }

    .skeleton-image {
      height: 500px;
      background: linear-gradient(90deg, #1a1a1a 25%, #2a2a2a 50%, #1a1a1a 75%);
      background-size: 200% 100%;
      animation: shimmer 1.5s infinite;
    }

    .skeleton-actions {
      height: 100px;
      background: linear-gradient(90deg, #1a1a1a 25%, #2a2a2a 50%, #1a1a1a 75%);
      background-size: 200% 100%;
      animation: shimmer 1.5s infinite;
    }
  }
}

@keyframes shimmer {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

.loading-more {
  text-align: center;
  padding: 20px;
  color: #a8a8a8;
  font-size: 14px;
}

.end-message {
  text-align: center;
  padding: 40px 20px;
  color: #a8a8a8;
  font-size: 14px;
}

.empty-feed {
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

.sidebar {
  width: 280px;
  position: sticky;
  top: 80px;
  height: fit-content;

  @media (max-width: 1024px) {
    display: none;
  }
}

.user-card {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  padding: 8px 0;

  .profile-pic {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    object-fit: cover;
  }

  .user-details {
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

  .switch-btn {
    background: none;
    border: none;
    color: #0a66c2;
    cursor: pointer;
    font-weight: 600;
    font-size: 12px;
    padding: 0;
  }
}

.suggestions {
  margin-bottom: 24px;

  .suggestions-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    padding: 0 4px;

    h3 {
      font-size: 14px;
      font-weight: 600;
      color: #a8a8a8;
    }

    a {
      font-size: 12px;
      color: #fff;
      text-decoration: none;
      font-weight: 600;

      &:hover {
        color: #a8a8a8;
      }
    }
  }

  .suggestion-item {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
    padding: 8px 0;

    img {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      object-fit: cover;
    }

    .suggestion-info {
      flex: 1;

      .username {
        font-weight: 600;
        font-size: 14px;
      }

      .mutual {
        font-size: 12px;
        color: #a8a8a8;
      }
    }

    .follow-btn {
      background: none;
      border: none;
      color: #0a66c2;
      cursor: pointer;
      font-weight: 600;
      font-size: 12px;
      padding: 0;
    }
  }
}

.sidebar-footer {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 16px;
  border-top: 1px solid #262626;

  a {
    font-size: 11px;
    color: #666;
    text-decoration: none;

    &:hover {
      color: #a8a8a8;
    }
  }
}

@media (max-width: 1024px) {
  .feed-container {
    padding-left: calc(72px);
  }

  .feed-wrapper {
    gap: 16px;
  }
}

@media (max-width: 768px) {
  .feed-container {
    padding-left: calc(60px);
  }

  .feed-wrapper {
    flex-direction: column;
    padding: 12px;
  }

  .main-feed {
    max-width: 100%;
  }

  .stories {
    margin-bottom: 16px;
  }
}
</style>
