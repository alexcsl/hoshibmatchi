<template>
  <div class="main-layout">
    <LeftSidebar />
    
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
// Create this component next
import LeftSidebar from "../components/LeftSidebar.vue";
</script>

<style scoped>
.main-layout {
  display: flex;
  background-color: #000; /* Example background */
  color: #fff;
}

/* This is the key to the PDF requirement [cite: 279, 280] */
.main-content {
  /* It must be positioned to the *right* of the sidebar */
  /* This value MUST match the width of LeftSidebar */
  margin-left: 244px; 
  
  /* It must take up the remaining width */
  width: calc(100% - 244px);
  
  /* It must have its *own* scrollbar */
  height: 100vh;
  overflow-y: auto;
  
  padding: 2rem; /* Example padding */
}

/* Responsive adjustments for sidebar width changes */
@media (max-width: 1264px) {
  .main-content {
    margin-left: 244px;
    width: calc(100% - 244px);
    padding: 1.5rem;
  }
}

@media (max-width: 1024px) {
  .main-content {
    margin-left: 72px;
    width: calc(100% - 72px);
    padding: 1rem;
  }
}

@media (max-width: 768px) {
  .main-content {
    margin-left: 60px;
    width: calc(100% - 60px);
    padding: 0.5rem;
  }
}
</style>