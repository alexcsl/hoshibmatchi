import { defineStore } from 'pinia'
import { feedAPI, postAPI, collectionAPI } from '@/services/api'

interface Post {
  id: string
  author_id: number
  caption: string
  author_username: string
  author_profile_url: string
  author_is_verified: boolean
  media_urls: string[]
  created_at: string
  is_reel: boolean
  like_count: number
  comment_count: number
  share_count?: number
  is_liked?: boolean
  is_saved?: boolean
}

interface FeedState {
  homeFeed: Post[]
  exploreFeed: Post[]
  reelsFeed: Post[]
  homePage: number
  explorePage: number
  reelsPage: number
  loading: boolean
  hasMore: boolean
}

export const useFeedStore = defineStore('feed', {
  state: (): FeedState => ({
    homeFeed: [],
    exploreFeed: [],
    reelsFeed: [],
    homePage: 1,
    explorePage: 1,
    reelsPage: 1,
    loading: false,
    hasMore: true
  }),

  actions: {
    async loadHomeFeed(page: number = 1, limit: number = 20) {
      this.loading = true
      try {
        console.log('Fetching home feed - page:', page, 'limit:', limit)
        const response = await feedAPI.getHomeFeed(page, limit)
        console.log('API response:', response)
        console.log('Response type:', typeof response)
        console.log('Is array?:', Array.isArray(response))
        
        // Handle different possible response structures
        let posts = []
        if (Array.isArray(response)) {
          // Response is directly an array
          posts = response
          console.log('Response is direct array with', posts.length, 'posts')
        } else if (response.posts && Array.isArray(response.posts)) {
          posts = response.posts
          console.log('Found posts in response.posts with', posts.length, 'posts')
        } else if (response.data && Array.isArray(response.data)) {
          posts = response.data
          console.log('Found posts in response.data with', posts.length, 'posts')
        } else if (response.data && response.data.posts && Array.isArray(response.data.posts)) {
          posts = response.data.posts
          console.log('Found posts in response.data.posts with', posts.length, 'posts')
        }
        
        console.log('Extracted posts:', posts.length, 'items')
        if (posts.length > 0) {
          console.log('First post sample:', posts[0])
        }
        
        // Ensure all posts have valid numeric values
        posts = posts.map((post: any) => ({
          ...post,
          like_count: post.like_count || 0,
          comment_count: post.comment_count || 0,
          share_count: post.share_count || 0,
          is_liked: post.is_liked || false,
          is_saved: post.is_saved || false
        }))
        
        if (page === 1) {
          this.homeFeed = posts
        } else {
          this.homeFeed.push(...posts)
        }
        this.homePage = page
        this.hasMore = posts.length === limit
        
        console.log('Home feed after load:', this.homeFeed.length, 'posts')
      } catch (error: any) {
        console.error('Failed to load home feed:', error)
        console.error('Error details:', error.response?.data || error.message)
      } finally {
        this.loading = false
      }
    },

    async loadExploreFeed(page: number = 1, limit: number = 20) {
      this.loading = true
      try {
        const response = await feedAPI.getExploreFeed(page, limit)
        
        // Handle different possible response structures
        let posts = []
        if (Array.isArray(response)) {
          posts = response
        } else if (response.posts && Array.isArray(response.posts)) {
          posts = response.posts
        } else if (response.data && Array.isArray(response.data)) {
          posts = response.data
        } else if (response.data && response.data.posts && Array.isArray(response.data.posts)) {
          posts = response.data.posts
        }
        
        // Ensure all posts have valid numeric values
        posts = posts.map((post: any) => ({
          ...post,
          like_count: post.like_count || 0,
          comment_count: post.comment_count || 0,
          share_count: post.share_count || 0,
          is_liked: post.is_liked || false,
          is_saved: post.is_saved || false
        }))
        
        if (page === 1) {
          this.exploreFeed = posts
        } else {
          this.exploreFeed.push(...posts)
        }
        this.explorePage = page
        this.hasMore = posts.length === limit
      } catch (error: any) {
        console.error('Failed to load explore feed:', error)
        console.error('Error details:', error.response?.data || error.message)
      } finally {
        this.loading = false
      }
    },

    async loadReelsFeed(page: number = 1, limit: number = 20) {
      this.loading = true
      try {
        const response = await feedAPI.getReelsFeed(page, limit)
        
        // Handle different possible response structures
        let posts = []
        if (Array.isArray(response)) {
          posts = response
        } else if (response.posts && Array.isArray(response.posts)) {
          posts = response.posts
        } else if (response.data && Array.isArray(response.data)) {
          posts = response.data
        } else if (response.data && response.data.posts && Array.isArray(response.data.posts)) {
          posts = response.data.posts
        }
        
        // Ensure all posts have valid numeric values
        posts = posts.map((post: any) => ({
          ...post,
          like_count: post.like_count || 0,
          comment_count: post.comment_count || 0,
          share_count: post.share_count || 0,
          is_liked: post.is_liked || false,
          is_saved: post.is_saved || false
        }))
        
        if (page === 1) {
          this.reelsFeed = posts
        } else {
          this.reelsFeed.push(...posts)
        }
        this.reelsPage = page
        this.hasMore = posts.length === limit
      } catch (error: any) {
        console.error('Failed to load reels feed:', error)
        console.error('Error details:', error.response?.data || error.message)
      } finally {
        this.loading = false
      }
    },

    async toggleLike(postId: string, feedType: 'home' | 'explore' | 'reels' = 'home') {
      const feed = feedType === 'home' ? this.homeFeed : feedType === 'explore' ? this.exploreFeed : this.reelsFeed
      const post = feed.find(p => p.id === postId)
      if (!post) return

      // Optimistic update
      const wasLiked = post.is_liked || false
      post.is_liked = !wasLiked
      
      // Ensure like_count is a number
      if (typeof post.like_count !== 'number' || isNaN(post.like_count)) {
        post.like_count = 0
      }
      
      post.like_count += post.is_liked ? 1 : -1

      try {
        if (wasLiked) {
          await postAPI.unlikePost(postId)
        } else {
          await postAPI.likePost(postId)
        }
      } catch (error) {
        // Rollback on error
        post.is_liked = wasLiked
        post.like_count += wasLiked ? 1 : -1
        console.error('Failed to toggle like:', error)
      }
    },

    async toggleSave(postId: string, collectionId: string = '1', feedType: 'home' | 'explore' | 'reels' = 'home') {
      const feed = feedType === 'home' ? this.homeFeed : feedType === 'explore' ? this.exploreFeed : this.reelsFeed
      const post = feed.find(p => p.id === postId)
      if (!post) return

      // Optimistic update
      const wasSaved = post.is_saved || false
      post.is_saved = !wasSaved

      try {
        if (wasSaved) {
          await collectionAPI.unsavePost(collectionId, postId)
        } else {
          // Use numeric post ID
          const numericPostId = parseInt(postId)
          if (isNaN(numericPostId)) {
            throw new Error('Invalid post ID')
          }
          await collectionAPI.savePost(collectionId, numericPostId)
        }
      } catch (error: any) {
        // Rollback on error
        post.is_saved = wasSaved
        console.error('Failed to toggle save:', error)
        console.error('Error details:', error.response?.data || error.message)
        
        // Show user-friendly message
        if (error.response?.status === 400) {
          console.warn('Collection not found or invalid. You may need to create a collection first.')
        }
      }
    },

    addPost(post: Post) {
      this.homeFeed.unshift(post)
    },

    updatePost(postId: string, updates: Partial<Post>) {
      const updateFeed = (feed: Post[]) => {
        const index = feed.findIndex(p => p.id === postId)
        if (index !== -1) {
          feed[index] = { ...feed[index], ...updates }
        }
      }

      updateFeed(this.homeFeed)
      updateFeed(this.exploreFeed)
      updateFeed(this.reelsFeed)
    },

    removePost(postId: string) {
      this.homeFeed = this.homeFeed.filter(p => p.id !== postId)
      this.exploreFeed = this.exploreFeed.filter(p => p.id !== postId)
      this.reelsFeed = this.reelsFeed.filter(p => p.id !== postId)
    }
  }
})
