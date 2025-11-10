<template>
  <div class="main-layout">
    <LeftSidebar />
    
    <main class="main-content">
      <router-view />
    </main>
    
    </div>
</template>

<script setup lang="ts">
import LeftSidebar from '../components/LeftSidebar.vue'
</script>

<style scoped>
.main-layout {
  display: flex;
}

/* This is the key to the independent scrolling */
.main-content {
  /* It must be positioned to the *right* of the sidebar */
  margin-left: 250px; /* Must match sidebar width */
  
  /* It must take up the remaining width */
  width: calc(100% - 250px);
  
  /* It must have its *own* scrollbar */
  height: 100vh;
  overflow-y: auto; /*  */
  
  padding: 2rem; /* Example padding */
}
</style>