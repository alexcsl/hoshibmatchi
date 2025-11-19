<template>
  <div class="messages-page">
    <div class="messages-container">
      <!-- Conversations List -->
      <div class="conversations-list">
        <div class="conversations-header">
          <h1>Messages</h1>
          <button class="new-message-btn">✉</button>
        </div>

        <div class="conversations-search">
          <input type="text" placeholder="Search Direct" class="search-input" />
        </div>

        <div class="conversations">
          <div v-for="i in 10" :key="i" class="conversation-item" :class="{ active: i === 1 }">
            <img :src="`/placeholder.svg?height=56&width=56&query=user-${i}`" :alt="'User ' + i" class="avatar" />
            <div class="conversation-info">
              <div class="username">user_{{ i }}</div>
              <div class="last-message">Last message here...</div>
            </div>
            <div class="timestamp">2h</div>
          </div>
        </div>
      </div>

      <!-- Chat Area -->
      <div class="chat-area">
        <div class="chat-header">
          <div class="chat-user">
            <img src="/placeholder.svg?height=40&width=40" alt="User" class="avatar" />
            <div>
              <div class="username">user_1</div>
              <div class="status">Active now</div>
            </div>
          </div>
          <div class="chat-actions">
            <button>📞</button>
            <button>📹</button>
            <button>ℹ</button>
          </div>
        </div>

        <div class="messages">
          <div v-for="i in 5" :key="i" class="message" :class="i % 2 === 0 ? 'sent' : 'received'">
            <div class="message-content">{{ i % 2 === 0 ? 'You: ' : 'User: ' }}Message {{ i }}</div>
          </div>
        </div>

        <div class="message-input-area">
          <input type="text" placeholder="Aa" class="message-input" />
          <button class="send-btn">❤</button>
          <button class="send-btn">😊</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped lang="scss">
.messages-page {
  width: 100%;
  padding-left: calc(244px);
  height: 100vh;
  background-color: #000;
  display: flex;
}

.messages-container {
  display: flex;
  width: 100%;
  height: 100%;
}

.conversations-list {
  width: 360px;
  border-right: 1px solid #262626;
  display: flex;
  flex-direction: column;

  .conversations-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #262626;

    h1 {
      font-size: 24px;
      font-weight: 700;
    }

    .new-message-btn {
      background: none;
      border: none;
      color: #fff;
      font-size: 20px;
      cursor: pointer;
      padding: 0;
    }
  }

  .conversations-search {
    padding: 8px 16px;

    .search-input {
      width: 100%;
      background-color: #262626;
      border: none;
      border-radius: 20px;
      padding: 10px 16px;
      color: #fff;
      font-size: 14px;
      outline: none;

      &::placeholder {
        color: #a8a8a8;
      }
    }
  }

  .conversations {
    flex: 1;
    overflow-y: auto;

    .conversation-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 16px;
      cursor: pointer;
      transition: background-color 0.2s;
      border-left: 3px solid transparent;

      &:hover {
        background-color: #262626;
      }

      &.active {
        background-color: #262626;
        border-left-color: #0a66c2;
      }

      .avatar {
        width: 56px;
        height: 56px;
        border-radius: 50%;
        object-fit: cover;
        flex-shrink: 0;
      }

      .conversation-info {
        flex: 1;

        .username {
          font-weight: 600;
          font-size: 14px;
        }

        .last-message {
          font-size: 12px;
          color: #a8a8a8;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
      }

      .timestamp {
        font-size: 12px;
        color: #a8a8a8;
        flex-shrink: 0;
      }
    }
  }
}

.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;

  .chat-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 20px;
    border-bottom: 1px solid #262626;

    .chat-user {
      display: flex;
      align-items: center;
      gap: 12px;

      .avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        object-fit: cover;
      }

      .username {
        font-weight: 600;
        font-size: 14px;
      }

      .status {
        font-size: 12px;
        color: #a8a8a8;
      }
    }

    .chat-actions {
      display: flex;
      gap: 16px;

      button {
        background: none;
        border: none;
        color: #0a66c2;
        font-size: 18px;
        cursor: pointer;
        padding: 0;
      }
    }
  }

  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 16px 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;

    .message {
      display: flex;

      .message-content {
        max-width: 70%;
        padding: 12px 16px;
        border-radius: 18px;
        font-size: 14px;
        word-wrap: break-word;
      }

      &.sent {
        justify-content: flex-end;

        .message-content {
          background-color: #0a66c2;
          color: #fff;
        }
      }

      &.received {
        justify-content: flex-start;

        .message-content {
          background-color: #262626;
          color: #fff;
        }
      }
    }
  }

  .message-input-area {
    display: flex;
    gap: 12px;
    padding: 12px 20px;
    border-top: 1px solid #262626;

    .message-input {
      flex: 1;
      background-color: #262626;
      border: none;
      border-radius: 20px;
      padding: 10px 16px;
      color: #fff;
      font-size: 14px;
      outline: none;

      &::placeholder {
        color: #a8a8a8;
      }
    }

    .send-btn {
      background: none;
      border: none;
      color: #0a66c2;
      font-size: 18px;
      cursor: pointer;
      padding: 0;
    }
  }
}

@media (max-width: 1024px) {
  .messages-page {
    padding-left: calc(72px);
  }

  .conversations-list {
    width: 300px;
  }
}

@media (max-width: 768px) {
  .messages-page {
    padding-left: 0;
  }

  .messages-container {
    flex-direction: column;
  }

  .conversations-list {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #262626;
    height: auto;
    max-height: 200px;
  }

  .chat-area {
    flex: 1;
  }
}
</style>
