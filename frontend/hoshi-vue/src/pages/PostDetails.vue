<template>
  <div class="post-details-page">
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
      class="post-details-container"
    >
      <!-- Post Media Section -->
      <div class="post-media-section">
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
              class="post-media"
              controls
              playsinline
            ></video>
            <img 
              v-else
              :src="secureMediaUrls[currentMediaIndex]" 
              :alt="'Post by ' + postData.author_username" 
              class="post-media" 
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

          <!-- Media indicators -->
          <div v-if="postData.media_urls.length > 1" class="media-indicators">
            <span
              v-for="(_, index) in postData.media_urls"
              :key="index"
              :class="['indicator', { active: index === currentMediaIndex }]"
            ></span>
          </div>
        </div>
      </div>

      <!-- Post Info Section -->
      <div class="post-info-section">
        <!-- Header -->
        <div class="post-header">
          <div class="user-info">
            <SecureImage
              :src="postData.author_profile_url"
              :alt="postData.author_username"
              class-name="avatar"
              loading-placeholder="/placeholder.svg?height=32&width=32"
              error-placeholder="/default-avatar.svg"
            />
            <div>
              <div class="username" @click="goToProfile(postData.author_username)">
                {{ postData.author_username }}
                <span
                  v-if="postData.author_is_verified"
                  class="verified"
                >✓</span>
              </div>
              <div class="location" v-if="postData.location">
                {{ postData.location }}
              </div>
            </div>
          </div>
          <button
            class="options-btn"
            @click="showOptionsModal = true"
            title="Options"
          >
            ⋯
          </button>
        </div>

        <!-- Caption with Comments -->
        <div class="content-section">
          <!-- Original caption as first "comment" -->
          <div class="caption-comment">
            <SecureImage
              :src="postData.author_profile_url"
              :alt="postData.author_username"
              class-name="comment-avatar"
              loading-placeholder="/placeholder.svg?height=32&width=32"
              error-placeholder="/default-avatar.svg"
            />
            <div class="comment-content">
              <div class="comment-username">
                {{ postData.author_username }}
                <span
                  v-if="postData.author_is_verified"
                  class="verified"
                >✓</span>
              </div>
              <div 
                class="caption-text"
                v-html="formattedCaption"
                @click="handleRichTextClick"
              ></div>
              <div v-if="showingSummary && aiSummary" class="ai-summary">
                <strong>AI Summary:</strong> {{ aiSummary }}
              </div>
              <div class="comment-meta">
                {{ formatTimestamp(postData.created_at) }}
                <button class="ai-btn" @click="toggleSummary">
                  {{ loadingAi ? 'Loading...' : showingSummary ? 'Hide Summary' : '✨ AI Summary' }}
                </button>
              </div>
            </div>
          </div>

          <!-- Comments -->
          <div class="comments-list">
            <div
              v-if="loadingComments"
              class="loading-comments"
            >
              Loading comments...
            </div>
            <div
              v-else-if="comments.length === 0"
              class="no-comments"
            >
              No comments yet. Be the first to comment!
            </div>
            <div
              v-for="comment in comments"
              :key="comment.id"
              class="comment-item"
            >
              <SecureImage
                :src="comment.author_profile_url"
                :alt="comment.author_username"
                class-name="comment-avatar"
                loading-placeholder="/placeholder.svg?height=32&width=32"
                error-placeholder="/default-avatar.svg"
              />
              <div class="comment-content">
                <div class="comment-header">
                  <span class="comment-username">
                    {{ comment.author_username }}
                    <span
                      v-if="comment.author_is_verified"
                      class="verified"
                    >✓</span>
                  </span>
                  <button
                    v-if="comment.user_id === authStore.user?.user_id"
                    class="delete-comment-btn"
                    @click="deleteComment(comment.id)"
                    title="Delete comment"
                  >
                    🗑️
                  </button>
                </div>
                <div class="comment-text">
                  <span v-if="isGifUrl(comment.content)">
                    <img
                      :src="comment.content"
                      alt="GIF"
                      class="comment-gif"
                    />
                  </span>
                  <span v-else v-html="formatRichText(comment.content)" @click="handleRichTextClick"></span>
                </div>
                <div class="comment-meta">
                  <span>{{ formatTimestamp(comment.created_at) }}</span>
                  <button 
                    :class="['like-btn', { liked: comment.is_liked }]"
                    @click="toggleCommentLike(comment)"
                  >
                    {{ comment.is_liked ? '❤️' : '🤍' }} {{ comment.like_count || 0 }}
                  </button>
                  <button @click="replyingTo = comment" class="reply-btn">
                    Reply
                  </button>
                </div>

                <!-- Replies -->
                <div v-if="commentReplies[comment.id]?.length > 0" class="replies">
                  <button
                    v-if="!expandedReplies[comment.id]"
                    @click="toggleReplies(comment.id)"
                    class="view-replies-btn"
                  >
                    View {{ commentReplies[comment.id].length }} {{ commentReplies[comment.id].length === 1 ? 'reply' : 'replies' }}
                  </button>

                  <div v-if="expandedReplies[comment.id]" class="replies-list">
                    <div
                      v-for="reply in commentReplies[comment.id]"
                      :key="reply.id"
                      class="reply-item"
                    >
                      <SecureImage
                        :src="reply.author_profile_url"
                        :alt="reply.author_username"
                        class-name="reply-avatar"
                        loading-placeholder="/placeholder.svg?height=24&width=24"
                        error-placeholder="/default-avatar.svg"
                      />
                      <div class="reply-content">
                        <div class="reply-header">
                          <span class="reply-username">
                            {{ reply.author_username }}
                            <span
                              v-if="reply.author_is_verified"
                              class="verified"
                            >✓</span>
                          </span>
                          <button
                            v-if="reply.user_id === authStore.user?.user_id"
                            class="delete-comment-btn"
                            @click="deleteComment(reply.id)"
                            title="Delete reply"
                          >
                            🗑️
                          </button>
                        </div>
                        <div class="reply-text">
                          <span v-if="isGifUrl(reply.content)">
                            <img
                              :src="reply.content"
                              alt="GIF"
                              class="comment-gif"
                            />
                          </span>
                          <span v-else v-html="formatRichText(reply.content)" @click="handleRichTextClick"></span>
                        </div>
                        <div class="reply-meta">
                          {{ formatTimestamp(reply.created_at) }}
                          <button 
                            :class="['like-btn', { liked: reply.is_liked }]"
                            @click="toggleCommentLike(reply)"
                          >
                            {{ reply.is_liked ? '❤️' : '🤍' }} {{ reply.like_count || 0 }}
                          </button>
                        </div>
                      </div>
                    </div>
                    <button
                      @click="toggleReplies(comment.id)"
                      class="hide-replies-btn"
                    >
                      Hide replies
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="post-actions">
          <div class="action-buttons">
            <button 
              :class="['action-btn', { active: postData.is_liked }]"
              @click="toggleLike"
            >
              {{ postData.is_liked ? '❤️' : '🤍' }}
            </button>
            <button class="action-btn" @click="focusCommentInput">
              💬
            </button>
            <button class="action-btn" @click="showShareModal = true">
              📤
            </button>
            <button 
              :class="['action-btn save-btn', { active: postData.is_saved }]"
              @click="handleSaveClick"
            >
              {{ postData.is_saved ? '🔖' : '📑' }}
            </button>
          </div>

          <div class="post-stats">
            <div class="likes-count clickable" @click="showPostLikes">
              {{ postData.like_count || 0 }} {{ (postData.like_count || 0) === 1 ? 'like' : 'likes' }}
            </div>
            <div class="timestamp">
              {{ formatTimestamp(postData.created_at) }}
            </div>
          </div>
        </div>

        <!-- Comment Input -->
        <div class="comment-input-section">
          <div v-if="replyingTo" class="replying-to">
            Replying to @{{ replyingTo.commenter_username }}
            <button @click="replyingTo = null" class="cancel-reply">✕</button>
          </div>
          <div class="comment-input-wrapper">
            <button
              class="emoji-btn"
              @click="showGifPicker = !showGifPicker"
              title="Add GIF"
            >
              GIF
            </button>
            <input
              ref="commentInputRef"
              v-model="newComment"
              type="text"
              placeholder="Add a comment..."
              @keyup.enter="submitComment"
            />
            <button
              :disabled="!newComment.trim() || isSubmitting"
              class="post-comment-btn"
              @click="submitComment"
            >
              {{ isSubmitting ? 'Posting...' : 'Post' }}
            </button>
          </div>

          <!-- GIF Picker -->
          <div v-if="showGifPicker" class="gif-picker">
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
                :alt="gif.title"
                class="gif-item"
                @click="selectGif(gif)"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="error-state">
      <p>Post not found</p>
      <button @click="$router.push('/feed')" class="back-btn">
        Go to Feed
      </button>
    </div>

    <!-- Save to Collection Modal -->
    <SaveToCollectionModal
      v-if="showSaveModal && postData"
      :post-id="postData.id"
      :saved-collection-ids="savedCollectionIds"
      @close="showSaveModal = false"
      @saved="handleCollectionSaved"
      @unsaved="handleCollectionUnsaved"
    />

    <!-- Share Modal -->
    <div v-if="showShareModal" class="modal-overlay" @click="showShareModal = false">
      <div class="share-modal" @click.stop>
        <div class="modal-header">
          <h3>Share Post</h3>
          <button class="close-btn-modal" @click="showShareModal = false">✕</button>
        </div>
        <div class="share-options">
          <button class="share-option" @click="copyLink">
            <span class="share-icon">🔗</span>
            <span>Copy Link</span>
          </button>
          <button class="share-option" @click="shareToFacebook">
            <span class="share-icon">📘</span>
            <span>Facebook</span>
          </button>
          <button class="share-option" @click="shareToTwitter">
            <span class="share-icon">🐦</span>
            <span>Twitter</span>
          </button>
          <button class="share-option" @click="shareViaEmail">
            <span class="share-icon">📧</span>
            <span>Email</span>
          </button>
          <button class="share-option" @click="sendToMessages">
            <span class="share-icon">💬</span>
            <span>Send to Friend</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Likes Modal -->
    <div v-if="showLikesModal" class="modal-overlay" @click="showLikesModal = false">
      <div class="likes-modal" @click.stop>
        <div class="modal-header">
          <h3>Likes</h3>
          <button class="close-btn-modal" @click="showLikesModal = false">✕</button>
        </div>
        <div class="likes-list">
          <div v-if="loadingLikers" class="loading-likers">Loading...</div>
          <div v-else-if="likers.length === 0" class="no-likers">No likes yet</div>
          <div v-else class="likers-container">
            <div v-for="liker in likers" :key="liker.user_id" class="liker-item">
              <SecureImage
                :src="liker.profile_picture_url"
                :alt="liker.username"
                class-name="liker-avatar"
                loading-placeholder="/placeholder.svg"
                error-placeholder="/default-avatar.svg"
              />
              <div class="liker-info">
                <div class="liker-username">{{ liker.username }}</div>
                <div class="liker-name">{{ liker.name }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Report Modal -->
    <div v-if="showReportModal" class="modal-overlay" @click="showReportModal = false">
      <div class="report-modal" @click.stop>
        <div class="modal-header">
          <h3>Report Post</h3>
          <button class="close-btn-modal" @click="showReportModal = false">✕</button>
        </div>
        <div class="report-reasons">
          <button
            v-for="reason in reportReasons"
            :key="reason"
            class="report-reason"
            @click="submitReport(reason)"
          >
            {{ reason }}
          </button>
          <input
            v-model="customReportReason"
            type="text"
            placeholder="Other reason..."
            class="custom-reason-input"
            @keyup.enter="submitReport(customReportReason)"
          />
          <button
            v-if="customReportReason.trim()"
            class="submit-report-btn"
            @click="submitReport(customReportReason)"
          >
            Submit Report
          </button>
        </div>
      </div>
    </div>

    <!-- Options Modal -->
    <div v-if="showOptionsModal" class="modal-overlay" @click="showOptionsModal = false">
      <div class="options-modal" @click.stop>
        <button v-if="isOwnPost" class="option-btn danger" @click="deletePost">
          Delete Post
        </button>
        <button v-else class="option-btn danger" @click="showReportModal = true; showOptionsModal = false">
          Report Post
        </button>
        <button class="option-btn" @click="sharePost">
          Share Post
        </button>
        <button class="option-btn" @click="copyLink">
          Copy Link
        </button>
        <button class="option-btn cancel" @click="showOptionsModal = false">
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useFeedStore } from "@/stores/feed";
import { postAPI, mediaAPI, commentAPI, collectionAPI } from "@/services/api";
import SecureImage from "@/components/SecureImage.vue";
import SaveToCollectionModal from "@/components/SaveToCollectionModal.vue";
import { useRichText } from "@/composables/useRichText";

interface Comment {
  id: string;
  post_id: number;
  user_id?: number;
  content: string;
  created_at: string;
  author_username: string;
  author_profile_url: string;
  author_is_verified?: boolean;
  parent_comment_id?: number;
  like_count?: number;
  is_liked?: boolean;
  reply_count?: number;
}

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const feedStore = useFeedStore();
const { formatRichText, handleRichTextClick } = useRichText();

const loading = ref(true);
const loadingComments = ref(false);
const loadingMedia = ref(true);
const isSubmitting = ref(false);
const currentMediaIndex = ref(0);
const newComment = ref("");
const comments = ref<Comment[]>([]);
const replyingTo = ref<Comment | null>(null);
const expandedReplies = ref<Record<number, boolean>>({});
const secureMediaUrls = ref<string[]>([]);
const commentReplies = ref<Record<number, Comment[]>>({});
const showGifPicker = ref(false);
const gifSearchQuery = ref("");
const gifs = ref<any[]>([]);
const commentInputRef = ref<HTMLInputElement | null>(null);

// Modal states
const showSaveModal = ref(false);
const showShareModal = ref(false);
const showLikesModal = ref(false);
const showReportModal = ref(false);
const showOptionsModal = ref(false);

// Save to collection
const savedCollectionIds = ref<string[]>([]);

// Likes modal
const loadingLikers = ref(false);
const likers = ref<any[]>([]);

// Report modal
const reportReasons = [
  "Spam or misleading",
  "Inappropriate content",
  "Harassment or hate speech",
  "Violence or dangerous organizations",
  "Intellectual property violation",
  "Sale of illegal or regulated goods"
];
const customReportReason = ref("");

// AI Summary
const showingSummary = ref(false);
const aiSummary = ref("");
const loadingAi = ref(false);

let gifSearchTimeout: any = null;

const postData = ref<any>(null);

const postId = computed(() => route.params.id as string);

const isOwnPost = computed(() => {
  return postData.value?.author_id === authStore.user?.user_id;
});

const formattedCaption = computed(() => {
  if (!postData.value?.caption) return "";
  return formatRichText(postData.value.caption);
});

const isVideoUrl = (url: string): boolean => {
  return /\.(mp4|webm|ogg|mov)(\?|$)/i.test(url);
};

const formatTimestamp = (timestamp: string): string => {
  if (!timestamp) return "";
  const date = new Date(timestamp);
  const now = new Date();
  const diffInMs = now.getTime() - date.getTime();
  const diffInMins = Math.floor(diffInMs / 60000);
  const diffInHours = Math.floor(diffInMins / 60);
  const diffInDays = Math.floor(diffInHours / 24);

  if (diffInMins < 1) return "Just now";
  if (diffInMins < 60) return `${diffInMins}m ago`;
  if (diffInHours < 24) return `${diffInHours}h ago`;
  if (diffInDays < 7) return `${diffInDays}d ago`;
  
  return date.toLocaleDateString("en-US", { 
    month: "short", 
    day: "numeric",
    year: date.getFullYear() !== now.getFullYear() ? "numeric" : undefined
  });
};

const loadPost = async () => {
  try {
    loading.value = true;
    const response = await postAPI.getPost(parseInt(postId.value));
    postData.value = response;
    
    // Load secure media URLs
    if (postData.value.media_urls?.length > 0) {
      await loadSecureMediaUrls();
    } else {
      loadingMedia.value = false;
    }
    
    await loadComments();
  } catch (error) {
    console.error("Failed to load post:", error);
  } finally {
    loading.value = false;
  }
};

const loadSecureMediaUrls = async () => {
  if (!postData.value?.media_urls || postData.value.media_urls.length === 0) {
    loadingMedia.value = false;
    return;
  }

  try {
    loadingMedia.value = true;
    const urls = await Promise.all(
      postData.value.media_urls.map(async (url: string) => {
        try {
          // If it's already a full URL, return it
          if (url.startsWith("http://") || url.startsWith("https://")) {
            return url;
          }
          // Get secure URL from backend
          const cleanUrl = url.startsWith("/") ? url.substring(1) : url;
          return await mediaAPI.getSecureMediaURL(cleanUrl);
        } catch (error) {
          console.error("Failed to load media URL:", url, error);
          return "/placeholder.svg";
        }
      })
    );
    secureMediaUrls.value = urls;
  } catch (error) {
    console.error("Failed to load secure media URLs:", error);
  } finally {
    loadingMedia.value = false;
  }
};

const loadComments = async () => {
  try {
    loadingComments.value = true;
    const response = await commentAPI.getCommentsByPost(parseInt(postId.value));
    
    console.log("📝 Comments Response:", response);
    console.log("📝 Total comments received:", response?.length || 0);
    
    // Separate top-level comments and replies
    const topLevel: Comment[] = [];
    const repliesMap: Record<number, Comment[]> = {};
    
    response.forEach((comment: Comment) => {
      console.log(`📝 Processing comment ${comment.id}:`, {
        id: comment.id,
        parent_id: comment.parent_comment_id,
        content: comment.content?.substring(0, 50)
      });
      
      if (comment.parent_comment_id) {
        const parentId = comment.parent_comment_id;
        if (!repliesMap[parentId]) {
          repliesMap[parentId] = [];
        }
        repliesMap[parentId].push(comment);
        console.log(`  ↳ Added as reply to comment ${parentId}`);
      } else {
        topLevel.push(comment);
        console.log(`  ↳ Added as top-level comment`);
      }
    });
    
    console.log("📝 Top-level comments:", topLevel.length);
    console.log("📝 Replies map:", Object.keys(repliesMap).map(k => `${k}: ${repliesMap[Number(k)].length} replies`));
    
    comments.value = topLevel;
    commentReplies.value = repliesMap;
  } catch (error) {
    console.error("Failed to load comments:", error);
  } finally {
    loadingComments.value = false;
  }
};

const submitComment = async () => {
  if (!newComment.value.trim() || isSubmitting.value) return;

  try {
    isSubmitting.value = true;
    const parentId = replyingTo.value?.id;
    
    await commentAPI.createComment({
      post_id: parseInt(postId.value),
      content: newComment.value,
      parent_comment_id: parentId
    });
    
    newComment.value = "";
    replyingTo.value = null;
    showGifPicker.value = false;
    
    await loadComments();
  } catch (error) {
    console.error("Failed to submit comment:", error);
    alert("Failed to post comment. Please try again.");
  } finally {
    isSubmitting.value = false;
  }
};

const deleteComment = async (commentId: number) => {
  if (!confirm("Are you sure you want to delete this comment?")) return;

  try {
    await commentAPI.deleteComment(commentId.toString());
    await loadComments();
    
    // Update comment count
    if (postData.value) {
      postData.value.comment_count = Math.max(0, (postData.value.comment_count || 0) - 1);
      feedStore.updatePost(postId.value, {
        comment_count: postData.value.comment_count
      } as any);
    }
  } catch (error) {
    console.error("Failed to delete comment:", error);
    alert("Failed to delete comment. Please try again.");
  }
};

const toggleCommentLike = async (comment: Comment) => {
  try {
    if (comment.is_liked) {
      await commentAPI.unlikeComment(comment.id.toString());
      comment.is_liked = false;
      comment.like_count = Math.max(0, (comment.like_count || 0) - 1);
    } else {
      await commentAPI.likeComment(comment.id.toString());
      comment.is_liked = true;
      comment.like_count = (comment.like_count || 0) + 1;
    }
  } catch (error) {
    console.error("Failed to toggle comment like:", error);
  }
};

const isGifUrl = (url: string) => {
  return url && (url.includes("giphy.com") || url.endsWith(".gif"));
};

const toggleReplies = (commentId: number) => {
  console.log(`🔄 toggleReplies called for comment ${commentId}`);
  console.log(`   Current state:`, expandedReplies.value[commentId]);
  console.log(`   Replies available:`, commentReplies.value[commentId]?.length || 0);
  // Create new object to ensure reactivity
  expandedReplies.value = {
    ...expandedReplies.value,
    [commentId]: !expandedReplies.value[commentId]
  };
  console.log(`   New state:`, expandedReplies.value[commentId]);
};

const showPostLikes = async () => {
  if (!postData.value || (postData.value.like_count || 0) === 0) return;
  
  loadingLikers.value = true;
  showLikesModal.value = true;
  
  try {
    const response = await postAPI.getPostLikers(postId.value);
    likers.value = response || [];
  } catch (error) {
    console.error("Failed to load likers:", error);
    likers.value = [];
  } finally {
    loadingLikers.value = false;
  }
};

const copyLink = () => {
  const url = window.location.href;
  try {
    // Try modern clipboard API first
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(() => {
        alert("Link copied to clipboard!");
        showShareModal.value = false;
        showOptionsModal.value = false;
      }).catch(err => {
        console.error("Failed to copy link:", err);
        alert("Failed to copy link. Please try again.");
      });
    } else {
      // Fallback for browsers without clipboard API
      const textArea = document.createElement('textarea');
      textArea.value = url;
      textArea.style.position = 'fixed';
      textArea.style.left = '-999999px';
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand('copy');
      document.body.removeChild(textArea);
      alert("Link copied to clipboard!");
      showShareModal.value = false;
      showOptionsModal.value = false;
    }
  } catch (err) {
    console.error("Failed to copy link:", err);
    alert("Failed to copy link. Please try again.");
  }
};

const shareToFacebook = () => {
  const url = window.location.href;
  window.open(`https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(url)}`, "_blank");
  showShareModal.value = false;
};

const shareToTwitter = () => {
  const url = window.location.href;
  const text = postData.value?.caption ? postData.value.caption.substring(0, 200) : "Check out this post!";
  window.open(`https://twitter.com/intent/tweet?url=${encodeURIComponent(url)}&text=${encodeURIComponent(text)}`, "_blank");
  showShareModal.value = false;
};

const shareViaEmail = () => {
  const url = window.location.href;
  const subject = "Check out this post!";
  const body = `I thought you might like this: ${url}`;
  window.location.href = `mailto:?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`;
  showShareModal.value = false;
};

const sendToMessages = () => {
  // Navigate to messages with post link
  const url = window.location.href;
  router.push({ path: '/messages', query: { share: url } });
  showShareModal.value = false;
};

const submitReport = async (reason: string) => {
  if (!reason || !reason.trim()) {
    alert("Please select or enter a reason for reporting");
    return;
  }
  
  try {
    await postAPI.reportPost(parseInt(postId.value), reason.trim());
    showReportModal.value = false;
    customReportReason.value = "";
    alert("Post reported successfully. Thank you for helping keep our community safe.");
  } catch (error) {
    console.error("Failed to report post:", error);
    alert("Failed to report post. Please try again.");
  }
};

const toggleSummary = async () => {
  if (showingSummary.value) {
    showingSummary.value = false;
    return;
  }
  
  if (aiSummary.value) {
    showingSummary.value = true;
    return;
  }
  
  loadingAi.value = true;
  try {
    const res = await postAPI.summarizeCaption(postId.value);
    aiSummary.value = res.summary;
    showingSummary.value = true;
  } catch (error) {
    console.error("AI Summarization failed:", error);
    alert("AI Summarization failed");
  } finally {
    loadingAi.value = false;
  }
};

const toggleLike = async () => {
  try {
    if (postData.value.is_liked) {
      await postAPI.unlikePost(postId.value);
      postData.value.is_liked = false;
      postData.value.like_count = Math.max(0, (postData.value.like_count || 0) - 1);
    } else {
      await postAPI.likePost(postId.value);
      postData.value.is_liked = true;
      postData.value.like_count = (postData.value.like_count || 0) + 1;
    }
    // Update feed store
    feedStore.updatePost(postId.value, {
      is_liked: postData.value.is_liked,
      like_count: postData.value.like_count
    } as any);
  } catch (error) {
    console.error("Failed to toggle like:", error);
  }
};

const handleSaveClick = async () => {
  if (!postData.value) return;
  
  // Load saved collections first
  try {
    const response = await collectionAPI.getCollectionsForPost(postData.value.id);
    savedCollectionIds.value = response.collection_ids || [];
    showSaveModal.value = true;
  } catch (error) {
    console.error("Failed to load saved collections:", error);
    savedCollectionIds.value = [];
    showSaveModal.value = true;
  }
};

const handleCollectionSaved = (collectionId: string) => {
  // Update post saved state
  if (postData.value) {
    postData.value.is_saved = true;
    // Update feed store
    feedStore.updatePost(postId.value, { is_saved: true } as any);
  }
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
      // Update feed store
      feedStore.updatePost(postId.value, { is_saved: stillSaved } as any);
    }
  } catch (error) {
    console.error("Failed to check saved status:", error);
  }
};



const deletePost = async () => {
  if (!confirm("Are you sure you want to delete this post?")) return;

  try {
    await postAPI.deletePost(postId.value);
    showOptionsModal.value = false;
    router.push("/feed");
  } catch (error) {
    console.error("Failed to delete post:", error);
    alert("Failed to delete post. Please try again.");
  }
};

const sharePost = () => {
  const url = window.location.href;
  if (navigator.share) {
    navigator.share({
      title: `Post by ${postData.value.author_username}`,
      url: url
    }).catch(err => {
      // User cancelled or error occurred
      console.log("Share cancelled", err);
    });
  } else {
    // Fallback to copy
    copyLink();
  }
};

const goToProfile = (username: string) => {
  router.push(`/profile/${username}`);
};

const focusCommentInput = () => {
  nextTick(() => {
    commentInputRef.value?.focus();
  });
};

const searchGifs = () => {
  clearTimeout(gifSearchTimeout);
  gifSearchTimeout = setTimeout(async () => {
    try {
      const apiKey = 'sXpGFDGZs0Dv1mmNFvYaGUvYwKX0PWIh';
      const query = gifSearchQuery.value.trim();
      
      const endpoint = query
        ? `https://api.giphy.com/v1/gifs/search?api_key=${apiKey}&q=${encodeURIComponent(query)}&limit=20`
        : `https://api.giphy.com/v1/gifs/trending?api_key=${apiKey}&limit=20`;
      
      const response = await fetch(endpoint);
      const data = await response.json();
      gifs.value = data.data || [];
    } catch (error) {
      console.error("Failed to search GIFs:", error);
    }
  }, 300);
};

const selectGif = (gif: any) => {
  newComment.value = gif.images.original.url;
  showGifPicker.value = false;
};

onMounted(() => {
  loadPost();
  searchGifs(); // Load trending GIFs
});

// Load trending GIFs when picker opens
watch(showGifPicker, (isOpen) => {
  if (isOpen && gifs.value.length === 0) {
    searchGifs();
  }
});
</script>

<style scoped>
.post-details-page {
  min-height: 100vh;
  background: #000;
  color: #fff;
  padding: 20px;
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  gap: 20px;
}

.spinner {
  font-size: 18px;
  color: #888;
}

.post-details-container {
  max-width: 1200px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr 500px;
  gap: 20px;
  background: #000;
  border: 1px solid #333;
  border-radius: 8px;
  overflow: hidden;
}

.post-media-section {
  background: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  min-height: 500px;
}

.media-carousel {
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.post-media {
  max-width: 100%;
  max-height: 80vh;
  object-fit: contain;
}

.carousel-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(0, 0, 0, 0.7);
  color: white;
  border: none;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.carousel-btn:hover {
  background: rgba(0, 0, 0, 0.9);
}

.carousel-btn.prev {
  left: 10px;
}

.carousel-btn.next {
  right: 10px;
}

.media-indicators {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 6px;
  z-index: 10;
}

.indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.5);
  transition: background 0.3s;
}

.indicator.active {
  background: #fff;
}

.post-info-section {
  display: flex;
  flex-direction: column;
  height: 80vh;
  max-height: 900px;
  background: #000;
}

.post-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #333;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

:deep(.avatar) {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
}

:deep(.comment-avatar) {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

:deep(.reply-avatar) {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

:deep(.liker-avatar) {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.username {
  font-weight: 600;
  cursor: pointer;
}

.username:hover {
  text-decoration: underline;
}

.verified {
  color: #0095f6;
  margin-left: 4px;
}

.location {
  font-size: 12px;
  color: #888;
}

.delete-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 18px;
  opacity: 0.7;
  transition: opacity 0.2s;
}

.delete-btn:hover {
  opacity: 1;
}

.content-section {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.caption-comment {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #333;
}

.comment-content {
  flex: 1;
}

.comment-username {
  font-weight: 600;
  margin-bottom: 4px;
}

.caption-text {
  margin-bottom: 8px;
  line-height: 1.5;
  word-wrap: break-word;
  overflow-wrap: break-word;
}

.caption-text :deep(.rich-text-hashtag),
.caption-text :deep(.rich-text-mention),
.comment-text :deep(.rich-text-hashtag),
.comment-text :deep(.rich-text-mention) {
  color: #0095f6;
  font-weight: 500;
  cursor: pointer;
}

.caption-text :deep(.rich-text-hashtag):hover,
.caption-text :deep(.rich-text-mention):hover,
.comment-text :deep(.rich-text-hashtag):hover,
.comment-text :deep(.rich-text-mention):hover {
  text-decoration: underline;
}

.comment-meta {
  font-size: 12px;
  color: #888;
  display: flex;
  align-items: center;
  gap: 12px;
}

.ai-btn {
  background: none;
  border: none;
  color: #0095f6;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
}

.ai-btn:hover {
  opacity: 0.7;
}

.ai-summary {
  margin: 8px 0;
  padding: 12px;
  background: #1a1a1a;
  border-left: 3px solid #0095f6;
  border-radius: 4px;
  font-size: 14px;
  line-height: 1.5;
}

.comments-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.comment-item {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.comment-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  flex-wrap: wrap;
}

.delete-comment-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  opacity: 0.5;
  transition: opacity 0.2s;
}

.delete-comment-btn:hover {
  opacity: 1;
}

.comment-text {
  margin-bottom: 4px;
  line-height: 1.4;
}

.comment-gif {
  max-width: 100%;
  max-height: 300px;
  border-radius: 8px;
  margin-top: 8px;
  display: block;
}

.reply-btn {
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  margin-left: 12px;
}

.reply-btn:hover {
  color: #fff;
}

.like-btn {
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  margin-left: 8px;
  transition: transform 0.2s;
}

.like-btn:hover {
  transform: scale(1.1);
}

.like-btn.liked {
  color: #ed4956;
  animation: like-animation 0.3s ease;
}

.replies {
  margin-top: 12px;
  padding-left: 12px;
  border-left: 2px solid #333;
}

.view-replies-btn,
.hide-replies-btn {
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 12px;
}

.view-replies-btn:hover,
.hide-replies-btn:hover {
  color: #fff;
}

.replies-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 8px;
}

.reply-item {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.reply-content {
  flex: 1;
  min-width: 0;
}

.reply-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.reply-username {
  font-weight: 600;
  font-size: 14px;
}

.reply-text {
  margin-bottom: 4px;
  line-height: 1.4;
  font-size: 14px;
  word-wrap: break-word;
  overflow-wrap: break-word;
}

.reply-meta {
  font-size: 11px;
  color: #888;
  display: flex;
  align-items: center;
  gap: 8px;
}

.post-actions {
  padding: 12px 16px;
  border-top: 1px solid #333;
  border-bottom: 1px solid #333;
}

.action-buttons {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}

.action-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 24px;
  transition: transform 0.2s;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-btn.active {
  animation: like-animation 0.3s ease;
}

.save-btn {
  margin-left: auto;
}

.post-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.likes-count {
  font-weight: 600;
  font-size: 14px;
}

.likes-count.clickable {
  cursor: pointer;
}

.likes-count.clickable:hover {
  opacity: 0.7;
}

.timestamp {
  font-size: 10px;
  color: #888;
  text-transform: uppercase;
}

.comment-input-section {
  padding: 16px;
}

.replying-to {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #1a1a1a;
  border-radius: 8px;
  margin-bottom: 8px;
  font-size: 14px;
  color: #888;
}

.cancel-reply {
  background: none;
  border: none;
  color: #fff;
  cursor: pointer;
  font-size: 16px;
}

.comment-input-wrapper {
  display: flex;
  gap: 12px;
  align-items: center;
}

.emoji-btn {
  background: none;
  border: 1px solid #333;
  color: #888;
  cursor: pointer;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.emoji-btn:hover {
  color: #fff;
  border-color: #555;
}

.comment-input-wrapper input {
  flex: 1;
  background: none;
  border: none;
  color: #fff;
  font-size: 14px;
  outline: none;
}

.post-comment-btn {
  background: none;
  border: none;
  color: #0095f6;
  cursor: pointer;
  font-weight: 600;
  font-size: 14px;
}

.post-comment-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.gif-picker {
  margin-top: 12px;
  background: #1a1a1a;
  border-radius: 8px;
  padding: 12px;
}

.gif-search input {
  width: 100%;
  padding: 8px 12px;
  background: #000;
  border: 1px solid #333;
  border-radius: 6px;
  color: #fff;
  margin-bottom: 12px;
}

.gif-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  max-height: 300px;
  overflow-y: auto;
}

.gif-item {
  width: 100%;
  height: 100px;
  object-fit: cover;
  cursor: pointer;
  border-radius: 4px;
}

.gif-item:hover {
  opacity: 0.8;
}

.no-comments,
.loading-comments {
  text-align: center;
  padding: 40px 20px;
  color: #888;
}

.back-btn {
  padding: 10px 20px;
  background: #0095f6;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 600;
}

.back-btn:hover {
  background: #007acc;
}

@keyframes like-animation {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.3);
  }
  100% {
    transform: scale(1);
  }
}

.options-btn {
  background: none;
  border: none;
  color: #fff;
  font-size: 24px;
  cursor: pointer;
  padding: 8px;
  opacity: 0.7;
  transition: opacity 0.2s;
}

.options-btn:hover {
  opacity: 1;
}

/* Modal Overlay */
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

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #333;
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.close-btn-modal {
  background: none;
  border: none;
  color: #fff;
  font-size: 20px;
  cursor: pointer;
  padding: 0;
  opacity: 0.7;
  transition: opacity 0.2s;
}

.close-btn-modal:hover {
  opacity: 1;
}

/* Share Modal */
.share-modal {
  background: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;
}

.share-options {
  padding: 16px;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.share-option {
  background: #1a1a1a;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #fff;
  transition: background 0.2s;
}

.share-option:hover {
  background: #333;
}

.share-icon {
  font-size: 24px;
}

/* Likes Modal */
.likes-modal {
  background: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  max-height: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.likes-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.loading-likers,
.no-likers {
  text-align: center;
  padding: 40px 20px;
  color: #888;
}

.likers-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.liker-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: 8px;
  transition: background 0.2s;
}

.liker-item:hover {
  background: #1a1a1a;
}

.liker-info {
  flex: 1;
}

.liker-username {
  font-weight: 600;
  font-size: 14px;
}

.liker-name {
  font-size: 12px;
  color: #888;
}

/* Report Modal */
.report-modal {
  background: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;
}

.report-reasons {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.report-reason {
  background: #1a1a1a;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  color: #fff;
  text-align: left;
  transition: background 0.2s;
}

.report-reason:hover {
  background: #333;
}

.custom-reason-input {
  background: #1a1a1a;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 12px;
  color: #fff;
  outline: none;
}

.custom-reason-input:focus {
  border-color: #0095f6;
}

.submit-report-btn {
  background: #ed4956;
  border: none;
  border-radius: 8px;
  padding: 12px;
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.submit-report-btn:hover {
  background: #c13942;
}

/* Options Modal */
.options-modal {
  background: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.option-btn {
  background: none;
  border: none;
  border-bottom: 1px solid #333;
  padding: 16px;
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  transition: background 0.2s;
}

.option-btn:hover {
  background: #1a1a1a;
}

.option-btn:last-child {
  border-bottom: none;
}

.option-btn.danger {
  color: #ed4956;
  font-weight: 600;
}

.option-btn.cancel {
  font-weight: 600;
}

/* Responsive Design */
@media (max-width: 1264px) {
  .post-details-page {
    padding: 12px;
  }

  .post-details-container {
    gap: 12px;
  }
}

@media (max-width: 968px) {
  .post-details-page {
    padding: 8px;
  }

  .post-details-container {
    grid-template-columns: 1fr;
    max-width: 100%;
    gap: 0;
  }

  .post-media-section {
    min-height: 400px;
  }

  .post-info-section {
    height: auto;
    max-height: none;
  }

  .content-section {
    max-height: 400px;
  }
}

@media (max-width: 768px) {
  .post-details-page {
    padding: 0;
  }

  .post-details-container {
    border-radius: 0;
    border-left: none;
    border-right: none;
  }

  .gif-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
