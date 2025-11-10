<template>
  <router-view />
</template>

<script setup lang="ts">
// No logic needed here, it's just a router outlet.
</script>

<style>
/* We removed the 'scoped' attribute from <style scoped>
   so these styles are global, or we can move them to style.css */
   
/* We need to reset the default browser styles */
body {
  margin: 0;
  font-family: system-ui, Avenir, Helvetica, Arial, sans-serif;
  background-color: #242424;
  color: rgba(255, 255, 255, 0.87);
}

/* Remove the default centering from the original style.css! */
#app {
  /* DO NOT center text or content here. */
  /* Let the layout components handle it. */
}

/* Universal box-sizing is good practice */
*,
*::before,
*::after {
  box-sizing: border-box;
}
</style>