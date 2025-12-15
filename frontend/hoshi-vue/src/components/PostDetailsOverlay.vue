<template>
  <div
    class="post-details-overlay"
    @click="$emit('close')"
  >
    <!-- Navigation buttons (only for Explore context) -->
    <button 
      v-if="props.context === 'explore' && canNavigatePrevious" 
      class="nav-btn nav-prev"
      @click.stop="navigateToPrevious"
    >
      ‹
    </button>
    
    <div
      class="post-details-modal"
      @click.stop
    >
      <button
        class="close-btn"
        @click="$emit('close')"
      >
        ✕
      </button>

      <div
        v-if="loading"
        class="loading-state"
      >
        <div class="spinner">
          Loading...
        </div>
      </div>

      <div
        v-else-if="postData"
        class="post-details-content"
      >
        <!-- Post Image -->
        <div class="post-image-container">
          <div
            v-if="postData.media_urls?.length > 0"
            class="media-carousel"
          >
            <div v-if="loadingMedia" class="media-loading">
              <div class="loading-spinner">Loading media...</div>
            </div>
            <template v-else-if="secureMediaUrls.length > 0">
              <video 
                v-if="isVideoUrl(secureMediaUrls[currentMediaIndex])"
                :src="secureMediaUrls[currentMediaIndex]" 
                class="post-image"
                controls
                playsinline
              ></video>
              <img 
                v-else
                :src="secureMediaUrls[currentMediaIndex]" 
                :alt="'Post by ' + postData.author_username" 
                class="post-image" 
              />
            </template>
            
            <button 
              v-if="postData.media_urls.length > 1 && currentMediaIndex > 0" 
              class="carousel-btn prev"
              @click="currentMediaIndex--"
            >
              ‹
            </button>
            <button 
              v-if="postData.media_urls.length > 1 && currentMediaIndex < postData.media_urls.length - 1" 
              class="carousel-btn next"
              @click="currentMediaIndex++"
            >
              ›
            </button>
          </div>
        </div>

        <!-- Post Info -->
        <div class="post-info">
          <!-- Header -->
          <div class="info-header">
            <div class="user-info">
              <SecureImage
                :src="displayAuthorProfileUrl"
                :alt="displayAuthorUsername"
                class-name="avatar"
                loading-placeholder="/placeholder.svg?height=32&width=32"
                error-placeholder="/default-avatar.svg"
              />
              <div>
                <div class="username">
                  {{ displayAuthorUsername }}
                  <span
                    v-if="postData.author_is_verified"
                    class="verified"
                  >✓</span>
                </div>
              </div>
            </div>
            <button
              class="options-btn"
              @click="handleOptions"
            >
              ⋯
            </button>
          </div>

          <!-- Caption & Comments -->
          <div class="comments-section">
            <!-- Original Caption -->
            <div
              v-if="postData.caption"
              class="comment original-caption"
            >
              <div class="comment-header">
                <SecureImage
                  :src="displayAuthorProfileUrl"
                  :alt="displayAuthorUsername"
                  class-name="comment-avatar"
                  loading-placeholder="/placeholder.svg?height=32&width=32"
                  error-placeholder="/default-avatar.svg"
                />
                <div class="comment-content">
                  <div class="comment-text">
                    <strong>{{ displayAuthorUsername }}</strong>
                    <span
                      @click="handleRichTextClick"
                      v-html="formattedCaption"
                    ></span>
                  </div>
                  <div class="comment-time">
                    {{ formatTimestamp(postData.created_at) }}
                  </div>
                </div>
              </div>
            </div>

            <!-- Comments -->
            <div
              v-for="comment in comments"
              :key="comment.id"
              class="comment"
            >
              <div class="comment-header">
                <SecureImage
                  :src="comment.author_profile_url"
                  :alt="comment.author_username"
                  class-name="comment-avatar"
                  loading-placeholder="/placeholder.svg?height=32&width=32"
                  error-placeholder="/default-avatar.svg"
                />
                <div class="comment-content">
                  <div class="comment-text">
                    <strong>{{ comment.author_username }}</strong>
                    <span v-if="isGifUrl(comment.content)">
                      <img
                        :src="comment.content"
                        alt="GIF"
                        class="comment-gif"
                      />
                    </span>
                    <span v-else>{{ comment.content }}</span>
                  </div>
                  <div class="comment-actions">
                    <span class="comment-time">{{ formatTimestamp(comment.created_at) }}</span>
                    <button 
                      class="reply-btn"
                      :class="{ liked: comment.is_liked }"
                      @click="toggleCommentLike(comment)"
                    >
                      {{ comment.is_liked ? '❤️' : '🤍' }} {{ comment.like_count || 0 }}
                    </button>
                    <button
                      class="reply-btn"
                      @click="startReply(comment)"
                    >
                      Reply
                    </button>
                    <button 
                      v-if="isOwnComment(comment) || isPostOwner" 
                      class="reply-btn delete-btn"
                      @click="handleDeleteComment(comment.id)"
                    >
                      Delete
                    </button>
                    <button 
                      v-if="(comment.reply_count || 0) > 0" 
                      class="reply-btn"
                      @click="toggleReplies(comment.id)"
                    >
                      View replies ({{ comment.reply_count || 0 }})
                    </button>
                  </div>
                </div>
              </div>
              
              <!-- Replies -->
              <div
                v-if="expandedReplies[comment.id]"
                class="replies"
              >
                <div
                  v-for="reply in commentReplies[comment.id]"
                  :key="reply.id"
                  class="comment reply"
                >
                  <div class="comment-header">
                    <SecureImage
                      :src="reply.author_profile_url"
                      :alt="reply.author_username"
                      class-name="comment-avatar"
                      loading-placeholder="/placeholder.svg?height=32&width=32"
                      error-placeholder="/default-avatar.svg"
                    />
                    <div class="comment-content">
                      <div class="comment-text">
                        <strong>{{ reply.author_username }}</strong>
                        <span v-if="isGifUrl(reply.content)">
                          <img
                            :src="reply.content"
                            alt="GIF"
                            class="comment-gif"
                          />
                        </span>
                        <span v-else>{{ reply.content }}</span>
                      </div>
                      <div class="comment-actions">
                        <span class="comment-time">{{ formatTimestamp(reply.created_at) }}</span>
                        <button 
                          class="reply-btn"
                          :class="{ liked: reply.is_liked }"
                          @click="toggleCommentLike(reply)"
                        >
                          {{ reply.is_liked ? '❤️' : '🤍' }} {{ reply.like_count || 0 }}
                        </button>
                        <button 
                          v-if="isOwnComment(reply) || isPostOwner" 
                          class="reply-btn delete-btn"
                          @click="handleDeleteComment(reply.id, true, comment.id)"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div
              v-if="loadingComments"
              class="loading-comments"
            >
              Loading comments...
            </div>
          </div>

          <!-- Actions -->
          <div class="post-actions">
            <button 
              class="action-btn" 
              :class="{ liked: postData.is_liked }"
              @click="handleLike"
            >
              {{ postData.is_liked ? '❤️' : '🤍' }}
            </button>
            <button class="action-btn">
              💬
            </button>
            <button
              class="action-btn"
              @click="handleShare"
            >
              📤
            </button>
            <button 
              class="action-btn" 
              :class="{ saved: postData.is_saved }"
              style="margin-left: auto;"
              @click="handleSave"
            >
              {{ postData.is_saved ? '🔖' : '🏷️' }}
            </button>
          </div>

          <!-- Likes -->
          <div class="likes-info">
            <strong>{{ formatLikes(postData.like_count) }}</strong>
          </div>

          <div class="timestamp-info">
            {{ formatTimestamp(postData.created_at) }}
          </div>

          <!-- Comment Input -->
          <div class="comment-input">
            <div
              v-if="replyingTo"
              class="replying-indicator"
            >
              <span>Replying to @{{ replyingTo.author_username }}</span>
              <button @click="cancelReply">
                ✕
              </button>
            </div>
            <div class="input-row">
              <button
                class="emoji-btn"
                @click="showGifPicker = !showGifPicker"
              >
                🎬
              </button>
              <input 
                v-model="newComment" 
                type="text"
                :placeholder="replyingTo ? `Reply to ${replyingTo.author_username}...` : 'Add a comment...'" 
                @keyup.enter="handleAddComment"
              />
              <button 
                v-if="newComment.trim()"
                :disabled="isSubmitting"
                @click="handleAddComment"
              >
                Post
              </button>
            </div>
            
            <!-- GIF Picker -->
            <div
              v-if="showGifPicker"
              class="gif-picker"
            >
              <div class="gif-search">
                <input 
                  v-model="gifSearchQuery" 
                  type="text" 
                  placeholder="Search GIFs..."
                  @input="searchGifs"
                />
              </div>
              <div class="gif-grid">
                <img 
                  v-for="gif in gifs" 
                  :key="gif.id" 
                  :src="gif.images.fixed_height_small.url" 
                  class="gif-item"
                  @click="selectGif(gif)"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <button 
      v-if="props.context === 'explore' && canNavigateNext" 
      class="nav-btn nav-next"
      @click.stop="navigateToNext"
    >
      ›
    </button>

    <!-- Save to Collection Modal -->
    <SaveToCollectionModal
      v-if="showSaveModal && postData"
      :post-id="postData.id"
      :saved-collection-ids="savedCollectionIds"
      @close="showSaveModal = false"
      @saved="handleCollectionSaved"
      @unsaved="handleCollectionUnsaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from "vue";
import { useFeedStore } from "@/stores/feed";
import { useAuthStore } from "@/stores/auth";
import { commentAPI, postAPI, collectionAPI } from "@/services/api";
import { useRichText } from "@/composables/useRichText";
import { getSecureMediaURL } from "@/services/media";
import SecureImage from "@/components/SecureImage.vue";
import SaveToCollectionModal from "@/components/SaveToCollectionModal.vue";

interface Comment {
  id: string
  post_id: number
  content: string
  author_username: string
  author_profile_url: string
  created_at: string
  parent_comment_id?: number
  is_liked?: boolean
  like_count?: number
  reply_count?: number
  user_id?: number
}

const props = defineProps<{
  postId: string
  postObject?: any
  context?: string
}>();

const emit = defineEmits<{
  close: []
  like: [postId: string]
  save: [postId: string]
}>();

const feedStore = useFeedStore();
const authStore = useAuthStore();
const { formatRichText, handleRichTextClick } = useRichText();

const loading = ref(false);
const loadingComments = ref(false);
const isSubmitting = ref(false);
const currentMediaIndex = ref(0);
const newComment = ref("");
const comments = ref<Comment[]>([]);
const replyingTo = ref<Comment | null>(null);
const expandedReplies = ref<Record<string, boolean>>({});

// Secure media URLs
const secureMediaUrls = ref<string[]>([]);
const loadingMedia = ref(true);
const commentReplies = ref<Record<string, Comment[]>>({});
const showGifPicker = ref(false);
const gifSearchQuery = ref("");
const gifs = ref<any[]>([]);
let gifSearchTimeout: any = null;

const postData = computed(() => {
  // If passed directly (from Profile page), use it
  if (props.postObject) return props.postObject;

  // Otherwise look in stores (from Feed page)
  return feedStore.homeFeed.find(p => p.id === props.postId) ||
         feedStore.exploreFeed.find(p => p.id === props.postId) ||
         feedStore.reelsFeed.find(p => p.id === props.postId);
});

const formattedCaption = computed(() => {
  if (!postData.value?.caption) return "";
  return formatRichText(postData.value.caption);
});

const currentPostIndex = computed(() => {
  if (props.context === "explore") {
    return feedStore.exploreFeed.findIndex(p => p.id === props.postId);
  }
  return -1;
});

const canNavigatePrevious = computed(() => {
  return props.context === "explore" && currentPostIndex.value > 0;
});

const canNavigateNext = computed(() => {
  return props.context === "explore" && 
         currentPostIndex.value >= 0 && 
         currentPostIndex.value < feedStore.exploreFeed.length - 1;
});

const navigateToPrevious = () => {
  if (canNavigatePrevious.value) {
    const previousPost = feedStore.exploreFeed[currentPostIndex.value - 1];
    if (previousPost && window.openPostDetails) {
      emit("close");
      setTimeout(() => {
        window.openPostDetails(previousPost.id, "explore");
      }, 100);
    }
  }
};

const navigateToNext = () => {
  if (canNavigateNext.value) {
    const nextPost = feedStore.exploreFeed[currentPostIndex.value + 1];
    if (nextPost && window.openPostDetails) {
      emit("close");
      setTimeout(() => {
        window.openPostDetails(nextPost.id, "explore");
      }, 100);
    }
  }
};

onMounted(async () => {
  loadingComments.value = true;
  try {
    const postIdNum = parseInt(props.postId);
    if (isNaN(postIdNum)) {
      console.error("Invalid post ID:", props.postId);
      return;
    }

    // Load secure URLs for media
    if (postData.value?.media_urls && postData.value.media_urls.length > 0) {
      loadingMedia.value = true;
      try {
        secureMediaUrls.value = await Promise.all(
          postData.value.media_urls.map((url: string) => getSecureMediaURL(url))
        );
      } catch (error) {
        console.error('Failed to load secure media URLs:', error);
        secureMediaUrls.value = postData.value.media_urls; // Fallback
      } finally {
        loadingMedia.value = false;
      }
    } else {
      loadingMedia.value = false;
    }
    
    const response = await commentAPI.getCommentsByPost(postIdNum);
    comments.value = response || [];
    console.log("Loaded comments:", comments.value.length);
  } catch (error) {
    console.error("Failed to load comments:", error);
  } finally {
    loadingComments.value = false;
  }
});

// Load trending GIFs when picker opens
watch(showGifPicker, (isOpen) => {
  if (isOpen && gifs.value.length === 0) {
    searchGifs();
  }
});

const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp);
  const now = new Date();
  const diffInMs = now.getTime() - date.getTime();
  const diffInSecs = Math.floor(diffInMs / 1000);
  const diffInMins = Math.floor(diffInSecs / 60);
  const diffInHours = Math.floor(diffInMins / 60);
  const diffInDays = Math.floor(diffInHours / 24);

  if (diffInDays > 7) {
    return date.toLocaleDateString("en-US", { month: "long", day: "numeric", year: "numeric" });
  } else if (diffInDays > 0) {
    return `${diffInDays} day${diffInDays > 1 ? "s" : ""} ago`;
  } else if (diffInHours > 0) {
    return `${diffInHours} hour${diffInHours > 1 ? "s" : ""} ago`;
  } else if (diffInMins > 0) {
    return `${diffInMins} minute${diffInMins > 1 ? "s" : ""} ago`;
  } else {
    return "Just now";
  }
};

const formatLikes = (count: number | undefined) => {
  const likeCount = count || 0;
  if (likeCount >= 1000000) {
    return `${(likeCount / 1000000).toFixed(1)}M likes`;
  } else if (likeCount >= 1000) {
    return `${(likeCount / 1000).toFixed(1)}K likes`;
  } else {
    return `${likeCount} like${likeCount !== 1 ? "s" : ""}`;
  }
};

const isVideoUrl = (url: string) => {
  if (!url) return false;
  const videoExtensions = [".mp4", ".webm", ".ogg", ".mov"];
  const lowerUrl = url.toLowerCase();
  return videoExtensions.some(ext => lowerUrl.includes(ext));
};

const handleLike = () => {
  emit("like", props.postId);
};

// Save Modal State
const showSaveModal = ref(false);
const savedCollectionIds = ref<string[]>([]);

const handleSave = async () => {
  if (!postData.value) return;
  
  // Use the new efficient endpoint to get which collections contain this post
  try {
    const response = await collectionAPI.getCollectionsForPost(postData.value.id);
    savedCollectionIds.value = response.collection_ids || [];
    showSaveModal.value = true;
  } catch (error) {
    console.error("Failed to load saved collections:", error);
    // On error, show modal anyway with empty saved state
    savedCollectionIds.value = [];
    showSaveModal.value = true;
  }
};

const handleCollectionSaved = (collectionId: string) => {
  // Update post saved state
  if (postData.value) {
    postData.value.is_saved = true;
  }
  emit("save", props.postId);
};

const handleCollectionUnsaved = async (collectionId: string) => {
  // Check if post is still in any collection
  try {
    const response = await collectionAPI.getAll();
    const collections = Array.isArray(response) ? response : (response.collections || []);
    
    let stillSaved = false;
    for (const collection of collections) {
      try {
        const postsResponse = await collectionAPI.getPosts(collection.id, 1, 100);
        const posts = Array.isArray(postsResponse) ? postsResponse : (postsResponse.posts || []);
        if (posts.some((p: any) => p.id === postData.value?.id)) {
          stillSaved = true;
          break;
        }
      } catch (err) {
        console.error(`Failed to check collection ${collection.id}:`, err);
      }
    }
    
    if (postData.value) {
      postData.value.is_saved = stillSaved;
    }
    emit("save", props.postId);
  } catch (error) {
    console.error("Failed to check saved status:", error);
  }
};

const handleShare = () => {
  // TODO: Implement share functionality
  console.log("Share post:", props.postId);
};

const handleOptions = () => {
  if (isPostOwner.value) {
    handleDeletePost();
  } else {
    handleReportPost();
  }
};

const handleReportPost = () => {
  const reason = prompt("Why are you reporting this post?");
  if (!reason || !reason.trim()) return;

  const numericPostId = parseInt(props.postId);
  if (isNaN(numericPostId)) {
    alert("Invalid post ID");
    return;
  }

  postAPI.reportPost(numericPostId, reason.trim())
    .then(() => {
      alert("Post reported successfully. Thank you for helping keep our community safe.");
    })
    .catch((err: any) => {
      alert(err.response?.data?.error || "Failed to report post");
    });
};

const handleAddComment = async () => {
  if (!newComment.value.trim() || isSubmitting.value) return;
  
  isSubmitting.value = true;
  try {
    const numericPostId = parseInt(props.postId);
    if (isNaN(numericPostId)) {
      console.error("Invalid post ID:", props.postId);
      alert("Invalid post ID");
      return;
    }
    
    const response = await commentAPI.createComment({
      post_id: numericPostId,
      content: newComment.value.trim(),
      parent_comment_id: replyingTo.value ? parseInt(replyingTo.value.id) : undefined
    });

    // Add comment to local list
    if (response) {
      if (replyingTo.value) {
        // Add to replies
        if (!commentReplies.value[replyingTo.value.id]) {
          commentReplies.value[replyingTo.value.id] = [];
        }
        commentReplies.value[replyingTo.value.id].unshift(response);
        
        // Update reply count
        const parentComment = comments.value.find(c => c.id === replyingTo.value!.id);
        if (parentComment) {
          parentComment.reply_count = (parentComment.reply_count || 0) + 1;
        }
      } else {
        // Add to main comments
        comments.value.unshift(response);
      }
    }

    // Update comment count in feed
    if (postData.value) {
      feedStore.updatePost(props.postId, {
        comment_count: (postData.value.comment_count || 0) + 1
      } as any);
    }

    newComment.value = "";
    replyingTo.value = null;
  } catch (error: any) {
    console.error("Failed to add comment:", error);
    console.error("Error details:", error.response?.data || error.message);
    
    // Show user-friendly error
    if (error.response?.status === 500) {
      alert("Failed to post comment. The server encountered an error. Please try again later.");
    } else {
      alert("Failed to post comment. Please try again.");
    }
  } finally {
    isSubmitting.value = false;
  }
};

const isGifUrl = (url: string) => {
  return url && (url.includes("giphy.com") || url.endsWith(".gif"));
};

const toggleCommentLike = async (comment: Comment) => {
  try {
    if (comment.is_liked) {
      await commentAPI.unlikeComment(comment.id);
      comment.is_liked = false;
      comment.like_count = Math.max(0, (comment.like_count || 0) - 1);
    } else {
      await commentAPI.likeComment(comment.id);
      comment.is_liked = true;
      comment.like_count = (comment.like_count || 0) + 1;
    }
  } catch (error) {
    console.error("Failed to toggle comment like:", error);
  }
};

const handleDeleteComment = async (commentId: string, isReply: boolean = false, parentCommentId?: string) => {
  if (!confirm("Delete this comment?")) return;
  
  try {
    await commentAPI.deleteComment(commentId);
    
    if (isReply && parentCommentId) {
      // Remove from replies
      commentReplies.value[parentCommentId] = commentReplies.value[parentCommentId].filter(
        (c: Comment) => c.id !== commentId
      );
      // Update reply count
      const parentComment = comments.value.find(c => c.id === parentCommentId);
      if (parentComment && parentComment.reply_count) {
        parentComment.reply_count -= 1;
      }
    } else {
      // Remove from main comments
      comments.value = comments.value.filter(c => c.id !== commentId);
    }
    
    // Update comment count in feed
    if (postData.value) {
      feedStore.updatePost(props.postId, {
        comment_count: Math.max(0, (postData.value.comment_count || 0) - 1)
      } as any);
    }
  } catch (error) {
    console.error("Failed to delete comment:", error);
    alert("Failed to delete comment");
  }
};

const isOwnComment = (comment: Comment) => {
  return comment.user_id === authStore.user?.user_id;
};

const isPostOwner = computed(() => {
  return postData.value?.author_id === authStore.user?.user_id;
});

// Use authStore data for current user's profile to ensure real-time updates
const displayAuthorProfileUrl = computed(() => {
  if (isPostOwner.value && authStore.user?.profile_picture_url) {
    return authStore.user.profile_picture_url;
  }
  return postData.value?.author_profile_url || '';
});

const displayAuthorUsername = computed(() => {
  if (isPostOwner.value && authStore.user?.username) {
    return authStore.user.username;
  }
  return postData.value?.author_username || '';
});

const handleDeletePost = async () => {
  if (!confirm("Delete this post?")) return;
  
  try {
    await postAPI.deletePost(props.postId);
    emit("close");
    // Optionally refresh feed
    feedStore.removePost(props.postId);
  } catch (error) {
    console.error("Failed to delete post:", error);
    alert("Failed to delete post");
  }
};

const toggleReplies = async (commentId: string) => {
  if (expandedReplies.value[commentId]) {
    expandedReplies.value[commentId] = false;
    return;
  }
  
  try {
    // Always reload replies to get latest data including newly added replies
    const numericPostId = parseInt(props.postId);
    const response = await commentAPI.getCommentsByPost(numericPostId);
    
    // Filter replies for this comment
    const replies = response.filter((c: Comment) => 
      c.parent_comment_id === parseInt(commentId)
    );
    commentReplies.value[commentId] = replies;
    
    expandedReplies.value[commentId] = true;
  } catch (error) {
    console.error("Failed to load replies:", error);
  }
};

const startReply = (comment: Comment) => {
  replyingTo.value = comment;
  newComment.value = "";
};

const cancelReply = () => {
  replyingTo.value = null;
  newComment.value = "";
};

// GIF search using Giphy API
const searchGifs = () => {
  clearTimeout(gifSearchTimeout);
  gifSearchTimeout = setTimeout(async () => {
    if (!gifSearchQuery.value.trim()) {
      // Load trending GIFs
      try {
        const response = await fetch(
          "https://api.giphy.com/v1/gifs/trending?api_key=sXpGFDGZs0Dv1mmNFvYaGUvYwKX0PWIh&limit=20"
        );
        const data = await response.json();
        gifs.value = data.data || [];
      } catch (error) {
        console.error("Failed to load trending GIFs:", error);
      }
      return;
    }
    
    try {
      const response = await fetch(
        `https://api.giphy.com/v1/gifs/search?api_key=sXpGFDGZs0Dv1mmNFvYaGUvYwKX0PWIh&q=${encodeURIComponent(gifSearchQuery.value)}&limit=20`
      );
      const data = await response.json();
      gifs.value = data.data || [];
    } catch (error) {
      console.error("Failed to search GIFs:", error);
    }
  }, 500);
};

const selectGif = (gif: any) => {
  newComment.value = gif.images.fixed_height.url;
  showGifPicker.value = false;
  gifSearchQuery.value = "";
};
</script>

<style scoped lang="scss">
.post-details-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 95;
}

.nav-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(255, 255, 255, 0.9);
  border: none;
  color: #000;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  font-size: 32px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 96;
  transition: background-color 0.2s, transform 0.2s;

  &:hover {
    background: rgba(255, 255, 255, 1);
    transform: translateY(-50%) scale(1.1);
  }

  &:active {
    transform: translateY(-50%) scale(0.95);
  }

  &.nav-prev {
    left: 20px;
  }

  &.nav-next {
    right: 20px;
  }
}

.post-details-modal {
  background-color: #262626;
  border-radius: 8px;
  width: 90%;
  max-width: 1100px;
  max-height: 90vh;
  display: flex;
  overflow: hidden;
  position: relative;
}

.close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  background: rgba(0, 0, 0, 0.7);
  border: none;
  color: #fff;
  font-size: 24px;
  cursor: pointer;
  z-index: 10;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    background: rgba(0, 0, 0, 0.9);
  }
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 500px;

  .spinner {
    color: #a8a8a8;
    font-size: 16px;
  }
}

.post-details-content {
  display: flex;
  width: 100%;
  height: 100%;
}

.post-image-container {
  width: 65%;
  background-color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;

  .media-carousel {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;

    .post-image {
      max-width: 100%;
      max-height: 90vh;
      object-fit: contain;
    }

    .carousel-btn {
      position: absolute;
      top: 50%;
      transform: translateY(-50%);
      background: rgba(0, 0, 0, 0.5);
      border: none;
      color: #fff;
      width: 40px;
      height: 40px;
      border-radius: 50%;
      font-size: 24px;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;

      &:hover {
        background: rgba(0, 0, 0, 0.7);
      }

      &.prev {
        left: 12px;
      }

      &.next {
        right: 12px;
      }
    }
  }
}

.post-info {
  width: 35%;
  display: flex;
  flex-direction: column;
  background-color: #000;
  max-height: 90vh;
}

.info-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #262626;

  .user-info {
    display: flex;
    align-items: center;
    gap: 12px;

    .avatar {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      object-fit: cover;
    }

    .username {
      font-weight: 600;
      font-size: 14px;
      display: flex;
      align-items: center;
      gap: 4px;

      .verified {
        color: #0a66c2;
        font-size: 12px;
      }
    }
  }

  .options-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
    padding: 0;

    &:hover {
      opacity: 0.7;
    }
  }
}

.comments-section {
  flex: 1;
  overflow-y: auto;
  padding: 16px;

  .comment {
    margin-bottom: 16px;

    &.original-caption {
      padding-bottom: 16px;
      border-bottom: 1px solid #262626;
      margin-bottom: 16px;
    }

    .comment-header {
      display: flex;
      gap: 12px;

      .comment-avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        object-fit: cover;
        flex-shrink: 0;
      }

      .comment-content {
        flex: 1;

        .comment-text {
          font-size: 14px;
          line-height: 1.5;
          margin-bottom: 8px;
          word-wrap: break-word;

          strong {
            margin-right: 4px;
          }

          .comment-gif {
            display: block;
            max-width: 200px;
            max-height: 200px;
            margin-top: 8px;
            border-radius: 8px;
            object-fit: contain;
          }
        }

        .comment-actions {
          display: flex;
          gap: 16px;
          align-items: center;

          .comment-time {
            font-size: 12px;
            color: #a8a8a8;
          }

          .like-btn,
          .reply-btn {
            background: none;
            border: none;
            color: #a8a8a8;
            cursor: pointer;
            padding: 0;
            font-size: 12px;
            font-weight: 600;

            &:hover {
              color: #fff;
            }

            &.delete-btn {
              color: #ed4956;

              &:hover {
                color: #ff6b7a;
              }
            }

            &.liked {
              color: #ed4956;
            }
          }
        }
      }
    }
  }

  .loading-comments {
    text-align: center;
    color: #a8a8a8;
    font-size: 14px;
    padding: 20px;
  }
}

.post-actions {
  display: flex;
  gap: 12px;
  padding: 8px 16px;
  border-top: 1px solid #262626;

  .action-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 24px;
    cursor: pointer;
    padding: 8px;
    transition: transform 0.1s;

    &:hover {
      opacity: 0.7;
    }

    &:active {
      transform: scale(0.9);
    }

    &.liked,
    &.saved {
      animation: pop 0.3s ease;
    }
  }
}

@keyframes pop {
  0% { transform: scale(1); }
  50% { transform: scale(1.2); }
  100% { transform: scale(1); }
}

.likes-info {
  padding: 8px 16px;
  font-size: 14px;
}

.timestamp-info {
  padding: 0 16px 12px;
  font-size: 12px;
  color: #a8a8a8;
  text-transform: uppercase;
}

.comment-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border-top: 1px solid #262626;

  .replying-indicator {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background-color: #1a1a1a;
    border-radius: 8px;
    font-size: 12px;
    color: #a8a8a8;

    button {
      background: none;
      border: none;
      color: #fff;
      cursor: pointer;
      font-size: 16px;
      padding: 0;

      &:hover {
        opacity: 0.7;
      }
    }
  }

  .input-row {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .emoji-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
    padding: 0;

    &:hover {
      opacity: 0.7;
    }
  }

  input {
    flex: 1;
    background: none;
    border: none;
    color: #fff;
    font-size: 14px;
    outline: none;

    &::placeholder {
      color: #a8a8a8;
    }
  }

  button:not(.emoji-btn) {
    background: none;
    border: none;
    color: #0a66c2;
    cursor: pointer;
    font-weight: 600;
    font-size: 14px;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    &:not(:disabled):hover {
      color: #0958a3;
    }
  }

  .gif-picker {
    background-color: #1a1a1a;
    border-radius: 8px;
    padding: 12px;
    max-height: 300px;
    overflow-y: auto;

    .gif-search {
      margin-bottom: 12px;

      input {
        width: 100%;
        background-color: #262626;
        border: 1px solid #404040;
        border-radius: 8px;
        padding: 8px 12px;
        color: #fff;
        font-size: 14px;
        outline: none;

        &::placeholder {
          color: #a8a8a8;
        }
      }
    }

    .gif-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 8px;

      .gif-item {
        width: 100%;
        height: 100px;
        object-fit: cover;
        border-radius: 4px;
        cursor: pointer;
        transition: transform 0.2s;

        &:hover {
          transform: scale(1.05);
        }
      }
    }
  }
}

.replies {
  margin-left: 44px;
  margin-top: 8px;
  padding-left: 12px;
  border-left: 2px solid #262626;

  .comment.reply {
    margin-bottom: 12px;
  }
}

.like-btn.liked {
  color: #ff4458;
}

/* Rich text styles for hashtags and mentions */
:deep(.rich-text-hashtag),
:deep(.rich-text-mention) {
  color: #0095f6;
  font-weight: 500;
  cursor: pointer;
  &:hover {
    text-decoration: underline;
  }
}

@media (max-width: 768px) {
  .post-details-modal {
    max-width: 100%;
    max-height: 100vh;
    border-radius: 0;
  }

  .post-details-content {
    flex-direction: column;
  }

  .post-image-container {
    width: 100%;
    height: 50%;
  }

  .post-info {
    width: 100%;
    height: 50%;
  }
}
</style>
