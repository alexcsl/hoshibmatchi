<template>
  <div
    class="notification-overlay"
    @click="$emit('close')"
  >
    <div
      class="notification-panel"
      @click.stop
    >
      <div class="notification-header">
        <h2>Notifications</h2>
        <div class="header-actions">
          <button 
            v-if="unreadCount > 0" 
            class="mark-all-btn" 
            :disabled="markingAll"
            @click="markAllAsRead"
          >
            Mark all as read
          </button>
          <button
            class="close-btn"
            @click="$emit('close')"
          >
            ✕
          </button>
        </div>
      </div>

      <div class="notifications-list">
        <div
          v-if="loading"
          class="loading"
        >
          Loading notifications...
        </div>
        <div
          v-else-if="notifications.length === 0"
          class="empty"
        >
          No notifications
        </div>
        <div 
          v-for="notification in notifications"
          v-else 
          :key="notification.id" 
          class="notification-item"
          :class="{ unread: !notification.is_read }"
          @click="notification.type !== 'follow.request' && handleNotificationClick(notification)"
        >
          <img 
            :src="notification.actor_profile_picture_url || '/default-avatar.png'" 
            :alt="notification.actor_username" 
            class="avatar" 
          />
          <div class="notification-content">
            <div class="notification-text">
              <strong>{{ notification.actor_username }}</strong>
              <span
                v-if="notification.actor_is_verified"
                class="verified-badge"
              >✓</span>
              {{ getNotificationText(notification.type) }}
            </div>
            <div class="notification-time">
              {{ formatTime(notification.created_at) }}
            </div>
          </div>
          
          <!-- Follow request action buttons -->
          <div 
            v-if="notification.type === 'follow.request'" 
            class="follow-request-actions"
            @click.stop
          >
            <button 
              class="accept-btn" 
              :disabled="processingRequest[notification.id]"
              @click="handleApproveRequest(notification)"
            >
              Accept
            </button>
            <button 
              class="reject-btn" 
              :disabled="processingRequest[notification.id]"
              @click="handleRejectRequest(notification)"
            >
              Reject
            </button>
          </div>
          
          <div
            v-if="!notification.is_read"
            class="unread-dot"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { notificationAPI, userAPI, type NotificationItem } from "../services/api";
import { useRouter } from "vue-router";

const router = useRouter();

const emit = defineEmits<{
  close: []
}>();

const notifications = ref<NotificationItem[]>([]);
const unreadCount = ref(0);
const loading = ref(false);
const markingAll = ref(false);
const processingRequest = ref<Record<number, boolean>>({});
let pollInterval: ReturnType<typeof setInterval> | null = null;

const loadNotifications = async (showLoading = true) => {
  if (showLoading) loading.value = true;
  try {
    const data = await notificationAPI.getNotifications(50);
    notifications.value = data.notifications;
    unreadCount.value = data.unread_count;
  } catch (error: any) {
    console.error("Failed to load notifications:", error);
    // Show user-friendly error message
    if (error.code === "ERR_NETWORK") {
      notifications.value = [];
      // Optionally show a toast or error message to user
    }
  } finally {
    if (showLoading) loading.value = false;
  }
};

const markAllAsRead = async () => {
  if (markingAll.value) return;
  
  markingAll.value = true;
  try {
    await notificationAPI.markAllAsRead();
    // Update local state
    notifications.value = notifications.value.map(n => ({ ...n, is_read: true }));
    unreadCount.value = 0;
  } catch (error) {
    console.error("Failed to mark all as read:", error);
  } finally {
    markingAll.value = false;
  }
};

const handleNotificationClick = async (notification: NotificationItem) => {
  // Mark as read if unread
  if (!notification.is_read) {
    try {
      await notificationAPI.markAsRead(notification.id);
      notification.is_read = true;
      unreadCount.value = Math.max(0, unreadCount.value - 1);
    } catch (error) {
      console.error("Failed to mark notification as read:", error);
    }
  }

  // Navigate based on notification type
  emit("close");
  
  if (notification.type === "post.liked" || notification.type === "post.commented" || notification.type === "comment.created") {
    // Navigate to post (would need to fetch post and navigate to it)
    console.log("Navigate to post:", notification.entity_id);
  } else if (notification.type === "user.followed" || notification.type === "follow.approved") {
    // Navigate to user profile
    router.push(`/profile/${notification.actor_username}`);
  }
};

const handleApproveRequest = async (notification: NotificationItem) => {
  processingRequest.value[notification.id] = true;
  try {
    await userAPI.approveFollowRequest(Number(notification.entity_id));
    // Remove notification from list
    const index = notifications.value.findIndex(n => n.id === notification.id);
    if (index !== -1) {
      notifications.value.splice(index, 1);
    }
    if (!notification.is_read) {
      unreadCount.value = Math.max(0, unreadCount.value - 1);
    }
  } catch (error) {
    console.error("Failed to approve follow request:", error);
  } finally {
    delete processingRequest.value[notification.id];
  }
};

const handleRejectRequest = async (notification: NotificationItem) => {
  processingRequest.value[notification.id] = true;
  try {
    await userAPI.rejectFollowRequest(Number(notification.entity_id));
    // Remove notification from list
    const index = notifications.value.findIndex(n => n.id === notification.id);
    if (index !== -1) {
      notifications.value.splice(index, 1);
    }
    if (!notification.is_read) {
      unreadCount.value = Math.max(0, unreadCount.value - 1);
    }
  } catch (error) {
    console.error("Failed to reject follow request:", error);
  } finally {
    delete processingRequest.value[notification.id];
  }
};

const getNotificationText = (type: string): string => {
  const texts: Record<string, string> = {
    "post.liked": "liked your post",
    "user.followed": "started following you",
    "follow.request": "requested to follow you",
    "follow.approved": "accepted your follow request",
    "post.commented": "commented on your post",
    "comment.created": "commented on your post", // Alias
    "post.shared": "shared your post",
    "comment.liked": "liked your comment",
    "story.liked": "liked your story"
  };
  return texts[type] || "interacted with your content";
};

const formatTime = (timestamp: string): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);
  
  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
};

onMounted(() => {
  loadNotifications();
  // Poll for new notifications every 5 seconds
  pollInterval = setInterval(() => {
    loadNotifications(false); // Don't show loading spinner on polls
  }, 5000);
});

onUnmounted(() => {
  // Cleanup interval when component is destroyed
  if (pollInterval) {
    clearInterval(pollInterval);
  }
});

</script>

<style scoped lang="scss">
.notification-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.8);
  display: flex;
  z-index: 90;
}

.notification-panel {
  width: 360px;
  height: 100vh;
  background-color: var(--bg-primary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  background-color: var(--bg-primary);
  z-index: 10;

  h2 {
    font-size: 24px;
    font-weight: 700;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .mark-all-btn {
    background: none;
    border: none;
    color: var(--accent-primary);
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    padding: 4px 8px;
    transition: opacity 0.2s;

    &:hover {
      opacity: 0.7;
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  .close-btn {
    background: none;
    border: none;
    color: var(--text-primary);
    font-size: 20px;
    cursor: pointer;
    padding: 0;
  }
}

.notifications-list {
  flex: 1;
  padding: 16px;
  overflow-y: auto;

  .loading {
    text-align: center;
    color: var(--text-secondary);
    padding: 40px 16px;
  }

  .empty {
    text-align: center;
    color: var(--text-secondary);
    padding: 40px 16px;
  }

  .notification-item {
    display: flex;
    gap: 12px;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: background-color 0.2s;
    position: relative;

    &:hover {
      background-color: var(--bg-hover);
    }

    &.unread {
      background-color: rgba(10, 102, 194, 0.1);
    }

    .avatar {
      width: 48px;
      height: 48px;
      border-radius: 50%;
      object-fit: cover;
      flex-shrink: 0;
    }

    .notification-content {
      flex: 1;

      .notification-text {
        font-size: 14px;
        line-height: 1.4;
        color: var(--text-primary);

        .verified-badge {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          width: 14px;
          height: 14px;
          background-color: var(--accent-primary);
          color: var(--text-primary);
          border-radius: 50%;
          font-size: 10px;
          margin-left: 4px;
        }
      }

      .notification-time {
        font-size: 12px;
        color: var(--text-secondary);
        margin-top: 4px;
      }
    }

    .unread-dot {
      width: 8px;
      height: 8px;
      background-color: var(--accent-primary);
      border-radius: 50%;
      flex-shrink: 0;
      margin-top: 8px;
    }

    .follow-request-actions {
      display: flex;
      gap: 8px;
      margin-top: 8px;
      width: 100%;

      button {
        flex: 1;
        padding: 6px 16px;
        border-radius: 6px;
        border: none;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: opacity 0.2s;

        &:hover:not(:disabled) {
          opacity: 0.8;
        }

        &:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      }

      .accept-btn {
        background-color: var(--accent-primary);
        color: var(--text-primary);
      }

      .reject-btn {
        background-color: transparent;
        color: var(--text-primary);
        border: 1px solid var(--text-secondary);
      }
    }
  }
}
</style>
