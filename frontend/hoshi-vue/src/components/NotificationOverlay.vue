<template>
  <div class="notification-overlay" @click="$emit('close')">
    <div class="notification-panel" @click.stop>
      <div class="notification-header">
        <h2>Notifications</h2>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>

      <div class="notifications-list">
        <div v-if="notifications.length === 0" class="empty">No notifications</div>
        <div v-for="notification in notifications" :key="notification.id" class="notification-item">
          <img :src="notification.avatar" :alt="notification.username" class="avatar" />
          <div class="notification-content">
            <div class="notification-text">
              <strong>{{ notification.username }}</strong>
              {{ notification.text }}
            </div>
            <div class="notification-time">{{ notification.timestamp }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Notification {
  id: number
  type: string
  username: string
  avatar: string
  text: string
  timestamp: string
}

defineProps<{
  notifications: Notification[]
}>()

defineEmits<{
  close: []
}>()
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
  background-color: #000;
  border-right: 1px solid #262626;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #262626;
  position: sticky;
  top: 0;
  background-color: #000;
  z-index: 10;

  h2 {
    font-size: 24px;
    font-weight: 700;
  }

  .close-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 20px;
    cursor: pointer;
    padding: 0;
  }
}

.notifications-list {
  flex: 1;
  padding: 16px;

  .empty {
    text-align: center;
    color: #a8a8a8;
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

    &:hover {
      background-color: #262626;
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
        font-size: 13px;
        line-height: 1.4;
        color: #fff;
      }

      .notification-time {
        font-size: 12px;
        color: #a8a8a8;
        margin-top: 4px;
      }
    }
  }
}
</style>
