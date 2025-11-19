<template>
  <div class="archive-page">
    <div class="archive-container">
      <h1>Archive</h1>
      <p class="subtitle">Only you can see your archived posts</p>

      <div class="archive-grid">
        <div v-for="i in 12" :key="i" class="archive-item">
          <img :src="`/placeholder.svg?height=280&width=280&query=archive-${i}`" :alt="'Archived post ' + i" />
          <div class="archive-overlay">
            <button class="action-btn">Unarchive</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped lang="scss">
.archive-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.archive-container {
  max-width: 1200px;

  h1 {
    font-size: 32px;
    font-weight: 700;
    margin-bottom: 8px;
  }

  .subtitle {
    color: #a8a8a8;
    font-size: 14px;
    margin-bottom: 32px;
  }
}

.archive-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.archive-item {
  position: relative;
  aspect-ratio: 1;
  cursor: pointer;
  overflow: hidden;
  border-radius: 4px;
  background-color: #262626;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.2s;
  }

  .archive-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0);
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0;
    transition: all 0.2s;

    .action-btn {
      background-color: #0a66c2;
      border: none;
      color: #fff;
      padding: 8px 24px;
      border-radius: 24px;
      font-weight: 600;
      font-size: 14px;
      cursor: pointer;

      &:hover {
        background-color: #0958a3;
      }
    }
  }

  &:hover {
    img {
      transform: scale(1.05);
    }

    .archive-overlay {
      background-color: rgba(0, 0, 0, 0.5);
      opacity: 1;
    }
  }
}

@media (max-width: 1024px) {
  .archive-page {
    padding-left: calc(72px + 20px);
  }
}

@media (max-width: 768px) {
  .archive-page {
    padding-left: calc(60px + 20px);
  }

  .archive-grid {
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  }
}
</style>
