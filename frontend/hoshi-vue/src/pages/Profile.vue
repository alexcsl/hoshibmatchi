<template>
  <div class="profile-page">
    <div
      v-if="loading"
      class="loading"
    >
      Loading profile...
    </div>
    
    <div
      v-else-if="error"
      class="error"
    >
      <div class="error-content">
        <h3>{{ error }}</h3>
        <p v-if="isOwnProfileError">
          We couldn't load your profile info.
        </p>
        <button
          v-if="isOwnProfileError"
          class="retry-btn"
          @click="retryAuth"
        >
          Retry
        </button>
      </div>
    </div>
    
    <div
      v-else
      class="profile-container"
    >
      <div class="profile-header">
        <SecureImage
          :src="profile.profile_picture_url"
          :alt="profile.username"
          class-name="profile-pic"
          loading-placeholder="/default-avatar.svg"
          error-placeholder="/default-avatar.svg"
        />
        <div class="profile-info">
          <div class="profile-top">
            <h1>
              {{ profile.username }}
              <span
                v-if="profile.is_verified"
                class="verified-badge"
                title="Verified Account"
              >✓</span>
            </h1>
            
            <button
              v-if="isOwnProfile"
              class="edit-btn"
              @click="$router.push('/edit-profile')"
            >
              Edit profile
            </button>
            
            <button
              v-if="isOwnProfile && !profile.is_verified"
              class="verify-btn"
              @click="showVerificationForm = true"
            >
              Request Verification
            </button>
            
            <div
              v-else
              class="action-buttons"
            >
              <button 
                class="follow-btn" 
                :class="{ following: profile.is_following }"
                :disabled="followLoading"
                @click="toggleFollow"
              >
                {{ profile.is_following ? 'Following' : 'Follow' }}
              </button>
              <button
                class="message-btn"
                @click="sendMessage"
              >
                Message
              </button>
              <button
                class="report-btn"
                @click="reportUser"
              >
                Report
              </button>
              <button
                v-if="isAdmin"
                class="ban-btn"
                :disabled="banLoading"
                @click="toggleBanUser"
              >
                {{ profile.is_banned ? 'Unban' : 'Ban' }} User
              </button>
            </div>
          </div>

          <div class="stats">
            <div class="stat">
              <span class="number">{{ formatNumber((profile.posts_count || 0) + (profile.reel_count || 0)) }}</span>
              <span class="label">posts</span>
            </div>
            <button
              class="stat"
              @click="showFollowers"
            >
              <span class="number">{{ formatNumber(profile.followers_count || 0) }}</span>
              <span class="label">followers</span>
            </button>
            <button
              class="stat"
              @click="showFollowing"
            >
              <span class="number">{{ formatNumber(profile.following_count || 0) }}</span>
              <span class="label">following</span>
            </button>
          </div>

          <div class="bio">
            <h2 class="name">
              {{ profile.name || profile.username }}
            </h2>
            <p
              v-if="profile.bio"
              class="bio-text"
            >
              {{ profile.bio }}
            </p>
            <a
              v-if="profile.website"
              :href="profile.website"
              target="_blank"
              class="website"
            >
              {{ profile.website }}
            </a>
          </div>
        </div>
      </div>

      <div class="profile-tabs">
        <button
          class="tab"
          :class="{ active: activeTab === 'posts' }"
          @click="switchTab('posts')"
        >
          <span class="icon">▦</span> POSTS
        </button>
        <button
          class="tab"
          :class="{ active: activeTab === 'reels' }"
          @click="switchTab('reels')"
        >
          <span class="icon">▶</span> REELS
        </button>
        <button
          v-if="isOwnProfile"
          class="tab"
          :class="{ active: activeTab === 'saved' }"
          @click="switchTab('saved')"
        >
          <span class="icon">🔖</span> SAVED
        </button>
        <button
          class="tab"
          :class="{ active: activeTab === 'tagged' }"
          @click="switchTab('tagged')"
        >
          <span class="icon">📌</span> TAGGED
        </button>
      </div>

      <div
        v-if="postsLoading"
        class="loading"
      >
        Loading posts...
      </div>
      
      <!-- Collections Grid for Saved Tab -->
      <div
        v-else-if="activeTab === 'saved'"
        class="collections-section"
      >
        <div class="collections-header">
          <h3>My Collections</h3>
          <button
            class="create-collection-btn"
            @click="showCreateCollectionModal = true"
          >
            ➕ New
          </button>
        </div>

        <div
          v-if="collections.length === 0"
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
              <div class="collection-name">
                {{ collection.name }}
                <span
                  v-if="collection.is_default"
                  class="default-badge"
                >Default</span>
              </div>
              <p class="collection-count">
                {{ collectionPosts[collection.id]?.length || 0 }} posts
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Posts Grid for other tabs -->
      <div
        v-else-if="posts.length === 0"
        class="empty-state"
      >
        <div class="empty-icon">
          📷
        </div>
        <h3>{{ emptyStateMessage }}</h3>
      </div>

      <div
        v-else
        class="content-grid"
      >
        <div
          v-for="post in posts"
          :key="post.id"
          class="grid-item"
          @click="openPost(post)"
        >
          <MediaThumbnail
            v-if="post.media_urls && post.media_urls.length > 0"
            :thumbnail-url="post.thumbnail_url"
            :media-urls="post.media_urls"
            :is-video="post.is_reel"
            :alt="post.caption"
            class-name=""
          />
          
          <div class="post-overlay">
            <div class="overlay-stats">
              <span>❤️ {{ formatNumber(post.like_count || 0) }}</span>
              <span>💬 {{ formatNumber(post.comment_count || 0) }}</span>
            </div>
          </div>
          <div
            v-if="post.media_urls && post.media_urls.length > 1"
            class="multi-icon"
          >
            ▦
          </div>
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

    <!-- Followers Modal -->
    <div
      v-if="showFollowersModal"
      class="modal-overlay"
      @click="showFollowersModal = false"
    >
      <div
        class="followers-modal"
        @click.stop
      >
        <div class="modal-header">
          <h3>Followers</h3>
          <button
            class="close-btn"
            @click="showFollowersModal = false"
          >
            ✕
          </button>
        </div>
        <div class="users-list">
          <div
            v-if="loadingFollowers"
            class="loading-users"
          >
            Loading...
          </div>
          <div
            v-else-if="followers.length === 0"
            class="no-users"
          >
            No followers yet
          </div>
          <div
            v-for="user in followers"
            v-else
            :key="user.user_id"
            class="user-item"
            @click="navigateToProfile(user.username)"
          >
            <SecureImage
              :src="user.profile_picture_url"
              :alt="user.username"
              class-name="user-avatar"
              loading-placeholder="/placeholder.svg?height=40&width=40"
              error-placeholder="/placeholder.svg?height=40&width=40"
            />
            <div class="user-info">
              <div class="user-username">
                {{ user.username }}
                <span
                  v-if="user.is_verified"
                  class="verified"
                >✓</span>
              </div>
              <div
                v-if="user.full_name"
                class="user-fullname"
              >
                {{ user.full_name }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Following Modal -->
    <div
      v-if="showFollowingModal"
      class="modal-overlay"
      @click="showFollowingModal = false"
    >
      <div
        class="followers-modal"
        @click.stop
      >
        <div class="modal-header">
          <h3>Following</h3>
          <button
            class="close-btn"
            @click="showFollowingModal = false"
          >
            ✕
          </button>
        </div>
        <div class="users-list">
          <div
            v-if="loadingFollowing"
            class="loading-users"
          >
            Loading...
          </div>
          <div
            v-else-if="following.length === 0"
            class="no-users"
          >
            Not following anyone yet
          </div>
          <div
            v-for="user in following"
            v-else
            :key="user.user_id"
            class="user-item"
            @click="navigateToProfile(user.username)"
          >
            <SecureImage
              :src="user.profile_picture_url"
              :alt="user.username"
              class-name="user-avatar"
              loading-placeholder="/placeholder.svg?height=40&width=40"
              error-placeholder="/placeholder.svg?height=40&width=40"
            />
            <div class="user-info">
              <div class="user-username">
                {{ user.username }}
                <span
                  v-if="user.is_verified"
                  class="verified"
                >✓</span>
              </div>
              <div
                v-if="user.full_name"
                class="user-fullname"
              >
                {{ user.full_name }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Verification Request Modal -->
    <div
      v-if="showVerificationForm"
      class="modal-overlay"
      @click.self="showVerificationForm = false"
    >
      <div class="modal-content verification-modal">
        <div class="modal-header">
          <h2>Request Verification</h2>
          <button
            class="close-btn"
            @click="showVerificationForm = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>ID Card Number</label>
            <input
              v-model="verificationForm.id_card_number"
              type="text"
              placeholder="Enter your ID card number"
              required
            />
          </div>
          <div class="form-group">
            <label>Face Picture URL</label>
            <input
              v-model="verificationForm.face_picture_url"
              type="text"
              placeholder="Enter URL to your face picture with ID"
              required
            />
          </div>
          <div class="form-group">
            <label>Reason for Verification</label>
            <textarea
              v-model="verificationForm.reason"
              placeholder="Why should your account be verified?"
              rows="4"
              required
            ></textarea>
          </div>
          <div
            v-if="verificationError"
            class="error-message"
          >
            {{ verificationError }}
          </div>
          <div
            v-if="verificationSuccess"
            class="success-message"
          >
            {{ verificationSuccess }}
          </div>
        </div>
        <div class="modal-footer">
          <button
            class="cancel-btn"
            @click="showVerificationForm = false"
          >
            Cancel
          </button>
          <button
            class="submit-btn"
            :disabled="verificationLoading"
            @click="submitVerificationRequest"
          >
            {{ verificationLoading ? 'Submitting...' : 'Submit Request' }}
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Create Collection Modal -->
  <div
    v-if="showCreateCollectionModal"
    class="modal-overlay"
    @click.self="showCreateCollectionModal = false"
  >
    <div class="modal-content small-modal">
      <div class="modal-header">
        <h2>New Collection</h2>
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
          @keyup.enter="createCollection"
          autofocus
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
          class="submit-btn"
          :disabled="!newCollectionName.trim()"
          @click="createCollection"
        >
          Create
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import apiClient, { userAPI, collectionAPI } from "@/services/api";
import PostDetailsOverlay from "@/components/PostDetailsOverlay.vue";
import SecureImage from "@/components/SecureImage.vue";
import MediaThumbnail from "@/components/MediaThumbnail.vue";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const profile = ref<any>({});
const posts = ref<any[]>([]);
const collections = ref<any[]>([]);
const collectionPosts = ref<Record<string, any[]>>({});
const loading = ref(true);
const postsLoading = ref(false);
const error = ref("");
const activeTab = ref("posts");
const followLoading = ref(false);
const showPostDetails = ref(false);
const selectedPost = ref<any>(null);
const showCreateCollectionModal = ref(false);
const newCollectionName = ref("");

// Verification
const showVerificationForm = ref(false);
const verificationLoading = ref(false);
const verificationError = ref("");
const verificationSuccess = ref("");
const verificationForm = ref({
  id_card_number: "",
  face_picture_url: "",
  reason: ""
});

// Ban user
const banLoading = ref(false);

// Admin check
const isAdmin = computed(() => {
  return authStore.user?.role === "admin";
});

const isOwnProfile = computed(() => {
  if (!authStore.user) return false;
  if (!route.params.username) return true;
  return route.params.username === authStore.user.username;
});

const isOwnProfileError = computed(() => error.value && !route.params.username);

const emptyStateMessage = computed(() => {
  if (activeTab.value === "posts") return "No posts yet";
  if (activeTab.value === "reels") return "No reels yet";
  if (activeTab.value === "saved") return "No saved posts";
  return "No tagged posts";
});

const openCollection = (collection: any) => {
  // Navigate to the Collections page with the collection ID
  router.push(`/collections/${collection.id}`);
};

const createCollection = async () => {
  if (!newCollectionName.value.trim()) {
    return;
  }

  try {
    await collectionAPI.create(newCollectionName.value.trim());
    newCollectionName.value = "";
    showCreateCollectionModal.value = false;
    
    // Refresh collections
    if (activeTab.value === "saved") {
      const username = getTargetUsername();
      if (username) {
        fetchPosts(username);
      }
    }
  } catch (error) {
    console.error("Failed to create collection:", error);
  }
};

const getTargetUsername = (): string | null => {
  const routeParam = route.params.username as string;
  if (routeParam && routeParam !== "undefined") return routeParam;
  if (authStore.user?.username) return authStore.user.username;
  return null;
};

const fetchProfile = async () => {
  loading.value = true;
  error.value = "";
  try {
    const username = getTargetUsername();
    if (!username) {
      if (!route.params.username) return; 
      error.value = "User not found";
      loading.value = false;
      return;
    }
    
    const response = await apiClient.get(`/users/${username}`);
    const data = response.data;
    if (data.user) {
        profile.value = { ...data.user, posts_count: data.post_count, reel_count: data.reel_count };
    } else {
        profile.value = data;
    }

    if (!route.params.username && username) router.replace(`/profile/${username}`);
  } catch (err: any) {
    console.error("Profile Error:", err);
    error.value = "Failed to load profile";
  } finally {
    loading.value = false;
  }
};

const fetchPosts = async (tab: string) => {
  postsLoading.value = true;
  try {
    const username = getTargetUsername();
    if (!username) {
        if (!route.params.username && loading.value) return;
        posts.value = [];
        postsLoading.value = false;
        return;
    }

    if (tab === "tagged") {
        const response = await userAPI.getUserTagged(username); // Use new API
        posts.value = response || [];
    }

    if (tab === "saved") {
        // --- Fetch Collections ---
        const collectionsRes = await collectionAPI.getAll();
        collections.value = Array.isArray(collectionsRes) ? collectionsRes : (collectionsRes?.collections || []);
        console.log("Profile: Loaded collections:", collections.value);
        
        // Load all posts for each collection to get accurate count and show preview
        for (const collection of collections.value) {
            try {
                const postsResponse = await collectionAPI.getPosts(collection.id, 1, 100);
                const posts = Array.isArray(postsResponse) 
                    ? postsResponse 
                    : (postsResponse?.posts || []);
                collectionPosts.value[collection.id] = posts;
                console.log(`Profile: Collection ${collection.id} (${collection.name}) has ${posts.length} posts`);
            } catch (err) {
                console.error(`Failed to load posts for collection ${collection.id}:`, err);
                collectionPosts.value[collection.id] = [];
            }
        }
        
        console.log("Profile: All collection posts:", collectionPosts.value);
        // No need to set posts.value for saved tab
        posts.value = [];
    } else {
        // Standard endpoints
        let endpoint = `/users/${username}/posts`;
        if (tab === "reels") endpoint = `/users/${username}/reels`;
        else if (tab === "tagged") endpoint = `/users/${username}/tagged`;

        const response = await apiClient.get(endpoint);
        const rawData = response.data;
        if (rawData.posts) posts.value = rawData.posts;
        else if (rawData.reels) posts.value = rawData.reels;
        else if (Array.isArray(rawData)) posts.value = rawData;
        else posts.value = [];
    }
  } catch (err) {
    console.error("Posts Error:", err);
    posts.value = [];
  } finally {
    postsLoading.value = false;
  }
};

const switchTab = (tab: string) => {
  activeTab.value = tab;
  fetchPosts(tab);
};

const openPost = (post: any) => {
  selectedPost.value = post;
  showPostDetails.value = true;
};

const closePostDetails = () => {
  showPostDetails.value = false;
  selectedPost.value = null;
};

const handlePostLike = async (postId: string) => {
  if (!selectedPost.value) return;
  
  // Optimistic update
  const wasLiked = selectedPost.value.is_liked || false;
  selectedPost.value.is_liked = !wasLiked;
  selectedPost.value.like_count = (selectedPost.value.like_count || 0) + (wasLiked ? -1 : 1);
  
  // Also update in posts array
  const postInList = posts.value.find(p => p.id === postId);
  if (postInList) {
    postInList.is_liked = !wasLiked;
    postInList.like_count = (postInList.like_count || 0) + (wasLiked ? -1 : 1);
  }
  
  try {
    if (wasLiked) {
      await apiClient.delete(`/posts/${postId}/like`);
    } else {
      await apiClient.post(`/posts/${postId}/like`);
    }
  } catch (err) {
    // Rollback on error
    selectedPost.value.is_liked = wasLiked;
    selectedPost.value.like_count = (selectedPost.value.like_count || 0) + (wasLiked ? 1 : -1);
    if (postInList) {
      postInList.is_liked = wasLiked;
      postInList.like_count = (postInList.like_count || 0) + (wasLiked ? 1 : -1);
    }
    console.error("Failed to toggle like:", err);
  }
};

const handlePostSave = async (postId: string) => {
  if (!selectedPost.value) return;
  
  // Optimistic update
  const wasSaved = selectedPost.value.is_saved || false;
  selectedPost.value.is_saved = !wasSaved;
  
  // Also update in posts array
  const postInList = posts.value.find(p => p.id === postId);
  if (postInList) {
    postInList.is_saved = !wasSaved;
  }
  
  try {
    const numericPostId = parseInt(postId);
    if (isNaN(numericPostId)) {
      throw new Error("Invalid post ID");
    }
    
    if (wasSaved) {
      // Unsave: Fetch collections and try to unsave
      const collectionsRes = await apiClient.get("/collections");
      const collections = Array.isArray(collectionsRes.data) ? collectionsRes.data : (collectionsRes.data.collections || []);
      
      if (collections.length > 0) {
        await apiClient.delete(`/collections/${collections[0].id}/posts/${postId}`);
      }
    } else {
      // Save: Use collection ID 1 - backend will auto-create if needed
      await apiClient.post("/collections/1/posts", { post_id: numericPostId });
    }
  } catch (err) {
    // Rollback on error
    selectedPost.value.is_saved = wasSaved;
    if (postInList) {
      postInList.is_saved = wasSaved;
    }
    console.error("Failed to toggle save:", err);
  }
};

const toggleFollow = async () => {
  if (followLoading.value) return;
  followLoading.value = true;

  const targetId = profile.value.user_id || profile.value.id;
  if (!targetId) {
    console.error("Cannot follow: User ID is missing on profile object", profile.value);
    alert("Error: User ID not found");
    followLoading.value = false;
    return;
  }

  try {
    if (profile.value.is_following) {
      await apiClient.delete(`/users/${targetId}/follow`);
      profile.value.is_following = false;
      profile.value.followers_count--;
    } else {
      await apiClient.post(`/users/${targetId}/follow`);
      profile.value.is_following = true;
      profile.value.followers_count++;
    }
  } catch (err) { 
    console.error(err); 
  } finally { 
    followLoading.value = false; 
  }
};

const sendMessage = () => router.push({ name: "Messages", query: { user: profile.value.username } });

const showFollowersModal = ref(false);
const showFollowingModal = ref(false);
const followers = ref<any[]>([]);
const following = ref<any[]>([]);
const loadingFollowers = ref(false);
const loadingFollowing = ref(false);

const showFollowers = async () => {
  showFollowersModal.value = true;
  loadingFollowers.value = true;
  
  try {
    const userId = profile.value?.user_id || authStore.user?.user_id;
    if (userId) {
      const response = await userAPI.getFollowers(userId);
      followers.value = response;
    }
  } catch (err) {
    console.error("Failed to load followers:", err);
    followers.value = [];
  } finally {
    loadingFollowers.value = false;
  }
};

const showFollowing = async () => {
  showFollowingModal.value = true;
  loadingFollowing.value = true;
  
  try {
    const userId = profile.value?.user_id || authStore.user?.user_id;
    if (userId) {
      const response = await userAPI.getFollowing(userId);
      following.value = response;
    }
  } catch (err) {
    console.error("Failed to load following:", err);
    following.value = [];
  } finally {
    loadingFollowing.value = false;
  }
};

const navigateToProfile = (username: string) => {
  showFollowersModal.value = false;
  showFollowingModal.value = false;
  router.push(`/${username}`);
};

const formatNumber = (num: number) => {
  if (!num) return "0";
  if (num >= 1000000) return `${(num/1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num/1000).toFixed(1)}K`;
  return num.toString();
};

const getMediaUrl = (url: string) => {
  if (!url) return "/placeholder.svg";
  if (url.startsWith("http")) return url;
  return `http://localhost:8000${url}`;
};

const handleImageError = (e: Event) => {
  (e.target as HTMLImageElement).src = "/placeholder.svg";
};

const submitVerificationRequest = async () => {
  if (!verificationForm.value.id_card_number || !verificationForm.value.face_picture_url || !verificationForm.value.reason) {
    verificationError.value = "All fields are required";
    return;
  }

  verificationLoading.value = true;
  verificationError.value = "";
  verificationSuccess.value = "";

  try {
    await userAPI.submitVerification(verificationForm.value);
    verificationSuccess.value = "Verification request submitted successfully! We'll review it soon.";
    setTimeout(() => {
      showVerificationForm.value = false;
      verificationForm.value = { id_card_number: "", face_picture_url: "", reason: "" };
      verificationSuccess.value = "";
    }, 2000);
  } catch (err: any) {
    verificationError.value = err.response?.data?.error || "Failed to submit verification request";
  } finally {
    verificationLoading.value = false;
  }
};

const toggleBanUser = async () => {
  if (!profile.value?.user_id) return;
  
  const action = profile.value.is_banned ? "unban" : "ban";
  if (!confirm(`Are you sure you want to ${action} this user?`)) return;

  banLoading.value = true;
  try {
    const endpoint = profile.value.is_banned 
      ? `/admin/users/${profile.value.user_id}/unban`
      : `/admin/users/${profile.value.user_id}/ban`;
    
    await apiClient.post(endpoint);
    profile.value.is_banned = !profile.value.is_banned;
    alert(`User ${action}ned successfully`);
  } catch (err: any) {
    alert(err.response?.data?.error || `Failed to ${action} user`);
  } finally {
    banLoading.value = false;
  }
};

const reportUser = async () => {
  if (!profile.value?.user_id) return;

  const reason = prompt("Why are you reporting this user?");
  if (!reason || !reason.trim()) return;

  try {
    await userAPI.reportUser(profile.value.user_id, reason.trim());
    alert("User reported successfully. Thank you for helping keep our community safe.");
  } catch (err: any) {
    alert(err.response?.data?.error || "Failed to report user");
  }
};

const retryAuth = () => {
    fetchProfile();
    fetchPosts(activeTab.value);
};

onMounted(() => {
  fetchProfile();
  fetchPosts("posts");
});

watch(() => authStore.user, (newUser) => {
    if (newUser && !route.params.username) {
        fetchProfile();
        fetchPosts(activeTab.value);
    }
}, { deep: true });

watch(() => route.params.username, () => {
  fetchProfile();
  fetchPosts("posts");
});
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

      h1 { 
        font-size: 28px; 
        font-weight: 300; 
        display: flex;
        align-items: center;
        gap: 8px;
      }
      
      .verified-badge {
        color: #4a9eff;
        font-size: 20px;
      }
      
      .edit-btn, .follow-btn, .message-btn, .more-btn, .verify-btn, .ban-btn, .report-btn {
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
      
      .verify-btn {
        background-color: #4a9eff;
        &:hover { background-color: #3a8eef; }
      }
      
      .ban-btn {
        background-color: #ed4956;
        &:hover { background-color: #d63447; }
      }
      
      .report-btn {
        background-color: #f09433;
        &:hover { background-color: #e0842a; }
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

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.65);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.followers-modal {
  background-color: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  max-height: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #404040;

    h3 {
      font-size: 16px;
      font-weight: 600;
      margin: 0;
    }

    .close-btn {
      background: none;
      border: none;
      color: #8e8e8e;
      font-size: 24px;
      cursor: pointer;
      padding: 0;
      width: 32px;
      height: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s;

      &:hover {
        color: #fff;
        background-color: rgba(255, 255, 255, 0.1);
        border-radius: 50%;
      }
    }
  }

  .users-list {
    overflow-y: auto;
    padding: 12px 20px;
    flex: 1;

    .loading-users,
    .no-users {
      text-align: center;
      padding: 20px;
      color: #8e8e8e;
    }

    .user-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 0;
      border-bottom: 1px solid #404040;
      cursor: pointer;

      &:last-child {
        border-bottom: none;
      }

      &:hover {
        background-color: rgba(255, 255, 255, 0.03);
      }

      .user-avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        object-fit: cover;
      }

      .user-info {
        flex: 1;

        .user-username {
          font-size: 14px;
          font-weight: 600;
          color: #fff;

          .verified {
            color: #4a9eff;
            margin-left: 4px;
          }
        }

        .user-fullname {
          font-size: 12px;
          color: #8e8e8e;
          margin-top: 2px;
        }
      }
    }
  }
}

.verification-modal {
  max-width: 500px;
  
  .form-group {
    margin-bottom: 16px;
    
    label {
      display: block;
      margin-bottom: 8px;
      font-weight: 600;
      color: #f1f1f1;
    }
    
    input, textarea {
      width: 100%;
      padding: 12px;
      background: #262626;
      border: 1px solid #363636;
      border-radius: 8px;
      color: #fff;
      font-size: 14px;
      
      &:focus {
        outline: none;
        border-color: #4a9eff;
      }
    }
    
    textarea {
      resize: vertical;
      font-family: inherit;
    }
  }
  
  .error-message {
    background: #ed49561a;
    color: #ed4956;
    padding: 12px;
    border-radius: 8px;
    margin-top: 12px;
    font-size: 14px;
  }
  
  .success-message {
    background: #00ba7c1a;
    color: #00ba7c;
    padding: 12px;
    border-radius: 8px;
    margin-top: 12px;
    font-size: 14px;
  }
  
  .modal-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
    
    button {
      padding: 10px 20px;
      border: none;
      border-radius: 8px;
      font-weight: 600;
      cursor: pointer;
      font-size: 14px;
      
      &.cancel-btn {
        background: #363636;
        color: #fff;
        
        &:hover {
          background: #262626;
        }
      }
      
      &.submit-btn {
        background: #4a9eff;
        color: #fff;
        
        &:hover {
          background: #3a8eef;
        }
        
        &:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      }
    }
  }
}

/* Collections Section Styles */
.collections-section {
  width: 100%;
}

.collections-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  padding: 0 12px;

  h3 {
    font-size: 20px;
    font-weight: 700;
    color: #fff;
    margin: 0;
  }

  .create-collection-btn {
    background: #4a9eff;
    color: #fff;
    border: none;
    padding: 8px 16px;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    font-size: 14px;
    transition: background 0.2s;

    &:hover {
      background: #3a8eef;
    }
  }
}

.collections-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  padding: 0 12px;
}

.collection-card {
  cursor: pointer;
  border-radius: 12px;
  overflow: hidden;
  background: #262626;
  transition: transform 0.2s, box-shadow 0.2s;

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3);
  }

  &.default {
    border: 2px solid #4a9eff;
  }

  .collection-cover {
    aspect-ratio: 1;
    background: #1a1a1a;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;

    .cover-grid {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      grid-template-rows: repeat(2, 1fr);
      width: 100%;
      height: 100%;
      gap: 2px;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }

    .empty-cover {
      font-size: 48px;
    }
  }

  .collection-info {
    padding: 12px;

    .collection-name {
      font-weight: 700;
      font-size: 16px;
      color: #fff;
      margin-bottom: 4px;
      display: flex;
      align-items: center;
      gap: 8px;

      .default-badge {
        background: #4a9eff;
        color: #fff;
        font-size: 10px;
        padding: 2px 6px;
        border-radius: 4px;
        font-weight: 600;
      }
    }

    .collection-count {
      font-size: 14px;
      color: #999;
      margin: 0;
    }
  }
}

/* Small Modal for Create Collection */
.small-modal {
  max-width: 400px !important;
}

.collection-name-input {
  width: 100%;
  padding: 12px;
  background: #262626;
  border: 1px solid #363636;
  border-radius: 8px;
  color: #fff;
  font-size: 14px;
  outline: none;

  &:focus {
    border-color: #4a9eff;
  }

  &::placeholder {
    color: #999;
  }
}

/* Collection Details Overlay */
.collection-details-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.9);
  z-index: 1000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  overflow-y: auto;
  padding: 20px;
}

.collection-details-content {
  background: #000;
  border-radius: 12px;
  width: 100%;
  max-width: 935px;
  margin: auto;

  .details-header {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 20px;
    border-bottom: 1px solid #262626;

    .back-btn {
      background: none;
      border: none;
      color: #4a9eff;
      font-size: 16px;
      font-weight: 600;
      cursor: pointer;
      padding: 8px 12px;
      border-radius: 6px;
      transition: background 0.2s;

      &:hover {
        background: #1a1a1a;
      }
    }

    h2 {
      margin: 0;
      font-size: 24px;
      font-weight: 700;
      color: #fff;
    }
  }

  .details-body {
    padding: 20px;
    min-height: 400px;
  }
}
</style>