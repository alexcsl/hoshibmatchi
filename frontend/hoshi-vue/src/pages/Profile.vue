<template>
  <div class="profile-page">
    <div v-if="loading" class="loading">Loading profile...</div>
    
    <div v-else-if="error" class="error">
      <div class="error-content">
        <h3>{{ error }}</h3>
        <p v-if="isOwnProfileError">We couldn't load your profile info.</p>
        <button v-if="isOwnProfileError" @click="retryAuth" class="retry-btn">Retry</button>
      </div>
    </div>
    
    <div v-else class="profile-container">
      <div class="profile-header">
        <img 
          :src="profile.profile_picture_url || '/default-avatar.svg'" 
          :alt="profile.username" 
          class="profile-pic" 
        />
        <div class="profile-info">
          <div class="profile-top">
            <h1>{{ profile.username }}</h1>
            
            <button v-if="isOwnProfile" class="edit-btn" @click="$router.push('/edit-profile')">
              Edit profile
            </button>
            
            <div v-else class="action-buttons">
              <button 
                class="follow-btn" 
                :class="{ following: profile.is_following }"
                @click="toggleFollow"
                :disabled="followLoading"
              >
                {{ profile.is_following ? 'Following' : 'Follow' }}
              </button>
              <button class="message-btn" @click="sendMessage">Message</button>
            </div>
          </div>

          <div class="stats">
            <div class="stat">
              <span class="number">{{ formatNumber((profile.posts_count || 0) + (profile.reel_count || 0)) }}</span>
              <span class="label">posts</span>
            </div>
            <button class="stat" @click="showFollowers">
              <span class="number">{{ formatNumber(profile.followers_count || 0) }}</span>
              <span class="label">followers</span>
            </button>
            <button class="stat" @click="showFollowing">
              <span class="number">{{ formatNumber(profile.following_count || 0) }}</span>
              <span class="label">following</span>
            </button>
          </div>

          <div class="bio">
            <h2 class="name">{{ profile.name || profile.username }}</h2>
            <p v-if="profile.bio" class="bio-text">{{ profile.bio }}</p>
            <a v-if="profile.website" :href="profile.website" target="_blank" class="website">
              {{ profile.website }}
            </a>
          </div>
        </div>
      </div>

      <div class="profile-tabs">
        <button class="tab" :class="{ active: activeTab === 'posts' }" @click="switchTab('posts')">
          <span class="icon">▦</span> POSTS
        </button>
        <button class="tab" :class="{ active: activeTab === 'reels' }" @click="switchTab('reels')">
          <span class="icon">▶</span> REELS
        </button>
        <button v-if="isOwnProfile" class="tab" :class="{ active: activeTab === 'saved' }" @click="switchTab('saved')">
          <span class="icon">🔖</span> SAVED
        </button>
        <button class="tab" :class="{ active: activeTab === 'tagged' }" @click="switchTab('tagged')">
          <span class="icon">📌</span> TAGGED
        </button>
      </div>

      <div v-if="postsLoading" class="loading">Loading posts...</div>
      
      <div v-else-if="posts.length === 0" class="empty-state">
        <div class="empty-icon">📷</div>
        <h3>{{ emptyStateMessage }}</h3>
      </div>

      <div v-else class="content-grid">
        <div v-for="post in posts" :key="post.id" class="grid-item" @click="openPost(post)">
          <img 
            v-if="post.media_urls && post.media_urls.length > 0"
            :src="getMediaUrl(post.media_urls[0])" 
            :alt="post.caption" 
            @error="handleImageError"
          />
          <div v-if="post.is_reel" class="type-badge">▶</div>
          
          <div class="post-overlay">
             <div class="overlay-stats">
               <span>❤️ {{ formatNumber(post.like_count || 0) }}</span>
               <span>💬 {{ formatNumber(post.comment_count || 0) }}</span>
             </div>
          </div>
          <div v-if="post.media_urls && post.media_urls.length > 1" class="multi-icon">▦</div>
        </div>
      </div>
    </div>

    <PostDetailsOverlay 
      v-if="showPostDetails && selectedPost" 
      :post-id="selectedPost.id" 
      :post-object="selectedPost" 
      @close="closePostDetails" 
      @like="handlePostLike"
      @save="handlePostSave"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import apiClient, { userAPI } from '@/services/api'
import PostDetailsOverlay from '@/components/PostDetailsOverlay.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const profile = ref<any>({})
const posts = ref<any[]>([])
const loading = ref(true)
const postsLoading = ref(false)
const error = ref('')
const activeTab = ref('posts')
const followLoading = ref(false)
const showPostDetails = ref(false)
const selectedPost = ref<any>(null)

const isOwnProfile = computed(() => {
  if (!authStore.user) return false
  if (!route.params.username) return true
  return route.params.username === authStore.user.username
})

const isOwnProfileError = computed(() => error.value && !route.params.username)

const emptyStateMessage = computed(() => {
  if (activeTab.value === 'posts') return 'No posts yet'
  if (activeTab.value === 'reels') return 'No reels yet'
  if (activeTab.value === 'saved') return 'No saved posts'
  return 'No tagged posts'
})

const getTargetUsername = (): string | null => {
  const routeParam = route.params.username as string
  if (routeParam && routeParam !== 'undefined') return routeParam
  if (authStore.user?.username) return authStore.user.username
  return null
}

const fetchProfile = async () => {
  loading.value = true
  error.value = ''
  try {
    const username = getTargetUsername()
    if (!username) {
      if (!route.params.username) return 
      error.value = 'User not found'
      loading.value = false
      return
    }
    
    const response = await apiClient.get(`/users/${username}`)
    const data = response.data
    if (data.user) {
        profile.value = { ...data.user, posts_count: data.post_count, reel_count: data.reel_count }
    } else {
        profile.value = data
    }

    if (!route.params.username && username) router.replace(`/profile/${username}`)
  } catch (err: any) {
    console.error('Profile Error:', err)
    error.value = 'Failed to load profile'
  } finally {
    loading.value = false
  }
}

const fetchPosts = async (tab: string) => {
  postsLoading.value = true
  try {
    const username = getTargetUsername()
    if (!username) {
        if (!route.params.username && loading.value) return
        posts.value = []
        postsLoading.value = false
        return
    }

    if (tab === 'tagged') {
        const response = await userAPI.getUserTagged(username) // Use new API
        posts.value = response || []
    }

    if (tab === 'saved') {
        // --- Fetch Saved Posts (Collections) ---
        // Get all collections
        const collectionsRes = await apiClient.get('/collections')
        const collections = Array.isArray(collectionsRes.data) ? collectionsRes.data : (collectionsRes.data.collections || [])
        
        // Fetch posts from all collections and combine them
        const allSavedPosts: any[] = []
        
        for (const collection of collections) {
            try {
                const savedPostsRes = await apiClient.get(`/collections/${collection.id}`)
                // Backend returns posts array directly, not wrapped in {posts: [...]}
                const collectionPosts = Array.isArray(savedPostsRes.data) ? savedPostsRes.data : (savedPostsRes.data.posts || [])
                allSavedPosts.push(...collectionPosts)
            } catch (err) {
                console.error(`Failed to fetch posts from collection ${collection.id}:`, err)
            }
        }
        
        // Remove duplicates based on post ID
        const uniquePosts = Array.from(
            new Map(allSavedPosts.map(post => [post.id, post])).values()
        )
        
        posts.value = uniquePosts
    } else {
        // Standard endpoints
        let endpoint = `/users/${username}/posts`
        if (tab === 'reels') endpoint = `/users/${username}/reels`
        else if (tab === 'tagged') endpoint = `/users/${username}/tagged`

        const response = await apiClient.get(endpoint)
        const rawData = response.data
        if (rawData.posts) posts.value = rawData.posts
        else if (rawData.reels) posts.value = rawData.reels
        else if (Array.isArray(rawData)) posts.value = rawData
        else posts.value = []
    }
  } catch (err) {
    console.error('Posts Error:', err)
    posts.value = []
  } finally {
    postsLoading.value = false
  }
}

const switchTab = (tab: string) => {
  activeTab.value = tab
  fetchPosts(tab)
}

const openPost = (post: any) => {
  selectedPost.value = post
  showPostDetails.value = true
}

const closePostDetails = () => {
  showPostDetails.value = false
  selectedPost.value = null
}

const handlePostLike = async (postId: string) => {
  if (!selectedPost.value) return
  
  // Optimistic update
  const wasLiked = selectedPost.value.is_liked || false
  selectedPost.value.is_liked = !wasLiked
  selectedPost.value.like_count = (selectedPost.value.like_count || 0) + (wasLiked ? -1 : 1)
  
  // Also update in posts array
  const postInList = posts.value.find(p => p.id === postId)
  if (postInList) {
    postInList.is_liked = !wasLiked
    postInList.like_count = (postInList.like_count || 0) + (wasLiked ? -1 : 1)
  }
  
  try {
    if (wasLiked) {
      await apiClient.delete(`/posts/${postId}/like`)
    } else {
      await apiClient.post(`/posts/${postId}/like`)
    }
  } catch (err) {
    // Rollback on error
    selectedPost.value.is_liked = wasLiked
    selectedPost.value.like_count = (selectedPost.value.like_count || 0) + (wasLiked ? 1 : -1)
    if (postInList) {
      postInList.is_liked = wasLiked
      postInList.like_count = (postInList.like_count || 0) + (wasLiked ? 1 : -1)
    }
    console.error('Failed to toggle like:', err)
  }
}

const handlePostSave = async (postId: string) => {
  if (!selectedPost.value) return
  
  // Optimistic update
  const wasSaved = selectedPost.value.is_saved || false
  selectedPost.value.is_saved = !wasSaved
  
  // Also update in posts array
  const postInList = posts.value.find(p => p.id === postId)
  if (postInList) {
    postInList.is_saved = !wasSaved
  }
  
  try {
    const numericPostId = parseInt(postId)
    if (isNaN(numericPostId)) {
      throw new Error('Invalid post ID')
    }
    
    if (wasSaved) {
      // Unsave: Fetch collections and try to unsave
      const collectionsRes = await apiClient.get('/collections')
      const collections = Array.isArray(collectionsRes.data) ? collectionsRes.data : (collectionsRes.data.collections || [])
      
      if (collections.length > 0) {
        await apiClient.delete(`/collections/${collections[0].id}/posts/${postId}`)
      }
    } else {
      // Save: Use collection ID 1 - backend will auto-create if needed
      await apiClient.post(`/collections/1/posts`, { post_id: numericPostId })
    }
  } catch (err) {
    // Rollback on error
    selectedPost.value.is_saved = wasSaved
    if (postInList) {
      postInList.is_saved = wasSaved
    }
    console.error('Failed to toggle save:', err)
  }
}

const toggleFollow = async () => {
  if (followLoading.value) return
  followLoading.value = true

  const targetId = profile.value.user_id || profile.value.id
  if (!targetId) {
    console.error("Cannot follow: User ID is missing on profile object", profile.value)
    alert("Error: User ID not found")
    followLoading.value = false
    return
  }

  try {
    if (profile.value.is_following) {
      await apiClient.delete(`/users/${targetId}/follow`)
      profile.value.is_following = false
      profile.value.followers_count--
    } else {
      await apiClient.post(`/users/${targetId}/follow`)
      profile.value.is_following = true
      profile.value.followers_count++
    }
  } catch (err) { 
    console.error(err) 
  } finally { 
    followLoading.value = false 
  }
}

const sendMessage = () => router.push({ name: 'Messages', query: { user: profile.value.username } })
const showFollowers = () => console.log('Show followers')
const showFollowing = () => console.log('Show following')

const formatNumber = (num: number) => {
  if (!num) return '0'
  if (num >= 1000000) return `${(num/1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num/1000).toFixed(1)}K`
  return num.toString()
}

const getMediaUrl = (url: string) => {
  if (!url) return '/placeholder.svg'
  if (url.startsWith('http')) return url
  return `http://localhost:8000${url}`
}

const handleImageError = (e: Event) => {
  (e.target as HTMLImageElement).src = '/placeholder.svg'
}

const retryAuth = () => {
    fetchProfile()
    fetchPosts(activeTab.value)
}

onMounted(() => {
  fetchProfile()
  fetchPosts('posts')
})

watch(() => authStore.user, (newUser) => {
    if (newUser && !route.params.username) {
        fetchProfile()
        fetchPosts(activeTab.value)
    }
}, { deep: true })

watch(() => route.params.username, () => {
  fetchProfile()
  fetchPosts('posts')
})
</script>

<style scoped lang="scss">
.profile-page {
  width: 100%;
  padding: 30px 20px;
  padding-left: calc(244px + 40px);
  background-color: #000;
  min-height: 100vh;
  color: white;
}

.profile-container {
  max-width: 935px;
  margin: 0 auto;
}

.error {
    text-align: center;
    padding: 40px;
    color: #ff4444;
    .retry-btn { margin-top: 10px; background: #0095f6; color: white; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; }
}

.profile-header {
  display: flex;
  gap: 80px;
  margin-bottom: 44px;
  
  .profile-pic {
    width: 150px;
    height: 150px;
    border-radius: 50%;
    object-fit: cover;
    border: 1px solid #363636;
  }

  .profile-info {
    flex: 1;
    .profile-top {
      display: flex;
      align-items: center;
      gap: 20px;
      margin-bottom: 20px;

      h1 { font-size: 28px; font-weight: 300; }
      
      .edit-btn, .follow-btn, .message-btn, .more-btn {
        background-color: #363636;
        color: #fff;
        border: none;
        padding: 7px 16px;
        border-radius: 8px;
        font-weight: 600;
        font-size: 14px;
        cursor: pointer;
        &:hover { background-color: #262626; }
      }

      .follow-btn {
        background-color: #0095f6;
        &:hover { background-color: #1877f2; }
        &.following { background-color: #363636; color: #fff; }
      }
      
      .more-btn { padding: 0 10px; font-size: 18px; }
      .action-buttons { display: flex; gap: 8px; }
    }

    .stats {
      display: flex;
      gap: 40px;
      margin-bottom: 20px;

      .stat {
        display: flex;
        gap: 5px;
        background: transparent;
        border: none;
        color: #fff;
        padding: 0;
        font-size: 16px;
        cursor: pointer;
        &[disabled] { cursor: default; }
        .number { font-weight: 600; }
      }
    }

    .bio {
      font-size: 14px;
      .name { font-weight: 600; }
      .bio-text { white-space: pre-wrap; margin-top: 4px; }
      .website { color: #e0f1ff; text-decoration: none; font-weight: 600; display: block; margin-top: 4px; }
    }
  }
}

.profile-tabs {
  display: flex;
  justify-content: center;
  border-top: 1px solid #262626;
  gap: 60px;
  
  .tab {
    background: none;
    border: none;
    border-top: 1px solid transparent;
    color: #8e8e8e;
    padding: 12px 0;
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 1px;
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: -1px;
    &.active {
      border-top-color: white;
      color: white;
    }
  }
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 4px;
  margin-top: 10px;

  .grid-item {
    aspect-ratio: 1;
    position: relative;
    cursor: pointer;
    background-color: #262626;
    
    img { width: 100%; height: 100%; object-fit: cover; }
    
    .type-badge {
        position: absolute; top: 8px; right: 8px;
        color: white; text-shadow: 0 0 4px rgba(0,0,0,0.5);
    }

    .post-overlay {
      position: absolute; inset: 0;
      background: rgba(0,0,0,0.3);
      display: flex; justify-content: center; align-items: center;
      opacity: 0; transition: opacity 0.2s;
      .overlay-stats {
        color: white; font-weight: bold; display: flex; gap: 20px; font-size: 16px;
      }
    }
    &:hover .post-overlay { opacity: 1; }
  }
}

@media (max-width: 768px) {
  .profile-page { padding: 20px 0; padding-bottom: 60px; }
  .profile-header {
    flex-direction: column; padding: 0 20px; gap: 24px;
    .profile-pic { width: 77px; height: 77px; margin-right: 20px; }
    display: grid; grid-template-columns: auto 1fr;
    .profile-info {
        .profile-top { grid-column: 1 / -1; display: block; margin-top: 12px; }
        .stats { justify-content: space-around; border-top: 1px solid #262626; padding: 12px 0; margin: 0; grid-column: 1 / -1; }
    }
  }
}
</style>