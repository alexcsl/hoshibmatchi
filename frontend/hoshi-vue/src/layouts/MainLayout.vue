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
</style>