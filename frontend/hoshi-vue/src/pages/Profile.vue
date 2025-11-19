<template>
  <div class="profile-page">
    <div class="profile-container">
      <!-- Profile Header -->
      <div class="profile-header">
        <img src="/placeholder.svg?height=150&width=150" alt="Profile" class="profile-pic" />
        <div class="profile-info">
          <h1>username</h1>
          <button class="edit-btn">Edit profile</button>
          <div class="bio">
            <p>Bio text goes here</p>
            <p>instagram.com</p>
          </div>
          <div class="stats">
            <div class="stat">
              <div class="number">150</div>
              <div class="label">posts</div>
            </div>
            <div class="stat">
              <div class="number">1.2K</div>
              <div class="label">followers</div>
            </div>
            <div class="stat">
              <div class="number">500</div>
              <div class="label">following</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="profile-tabs">
        <button class="tab" :class="{ active: activeTab === 'posts' }" @click="activeTab = 'posts'">
          <span class="icon">▦</span> Posts
        </button>
        <button class="tab" :class="{ active: activeTab === 'reels' }" @click="activeTab = 'reels'">
          <span class="icon">▶</span> Reels
        </button>
        <button class="tab" :class="{ active: activeTab === 'saved' }" @click="activeTab = 'saved'">
          <span class="icon">🔖</span> Saved
        </button>
        <button class="tab" :class="{ active: activeTab === 'tagged' }" @click="activeTab = 'tagged'">
          <span class="icon">📌</span> Tagged
        </button>
      </div>

      <!-- Content Grid -->
      <div class="content-grid">
        <div v-for="i in 9" :key="i" class="grid-item">
          <img :src="`/placeholder.svg?height=300&width=300&query=profile-post-${i}`" :alt="'Post ' + i" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const activeTab = ref('posts')
</script>

<style scoped lang="scss">
.profile-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.profile-container {
  max-width: 1000px;
}

.profile-header {
  display: flex;
  gap: 32px;
  margin-bottom: 40px;
  padding-bottom: 40px;
  border-bottom: 1px solid #262626;

  .profile-pic {
    width: 150px;
    height: 150px;
    border-radius: 50%;
    object-fit: cover;
  }

  .profile-info {
    flex: 1;

    h1 {
      font-size: 28px;
      font-weight: 300;
      margin-bottom: 12px;
    }

    .edit-btn {
      background-color: transparent;
      border: 1px solid #404040;
      color: #fff;
      padding: 6px 24px;
      border-radius: 6px;
      font-weight: 600;
      font-size: 14px;
      cursor: pointer;
      margin-bottom: 12px;

      &:hover {
        background-color: #262626;
      }
    }

    .bio {
      margin-bottom: 16px;

      p {
        font-size: 14px;
        line-height: 1.5;
      }
    }

    .stats {
      display: flex;
      gap: 40px;

      .stat {
        display: flex;
        flex-direction: column;

        .number {
          font-weight: 700;
          font-size: 18px;
        }

        .label {
          font-size: 14px;
          color: #a8a8a8;
        }
      }
    }
  }
}

.profile-tabs {
  display: flex;
  gap: 32px;
  margin-bottom: 32px;
  border-top: 1px solid #262626;
  padding-top: 16px;

  .tab {
    background: none;
    border: none;
    color: #a8a8a8;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    padding: 8px 0;
    border-bottom: 1px solid transparent;
    transition: all 0.2s;

    .icon {
      margin-right: 6px;
    }

    &:hover {
      color: #fff;
    }

    &.active {
      color: #fff;
      border-bottom-color: #fff;
    }
  }
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;

  .grid-item {
    aspect-ratio: 1;
    overflow: hidden;
    border-radius: 4px;
    background-color: #262626;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
      cursor: pointer;
      transition: transform 0.2s;

      &:hover {
        transform: scale(1.05);
      }
    }
  }
}

@media (max-width: 1024px) {
  .profile-page {
    padding-left: calc(72px + 20px);
  }
}

@media (max-width: 768px) {
  .profile-page {
    padding-left: 20px;
  }

  .profile-header {
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 16px;
  }

  .content-grid {
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  }
}
</style>
