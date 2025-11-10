<template>
  <router-view />
</template>

<script setup lang="ts">
// No logic needed here!
</script>

<style>
/* We remove 'scoped' so these are global resets.
   This is critical for the layout. */
body {
  margin: 0;
  padding: 0;
  font-family: system-ui, Avenir, Helvetica, Arial, sans-serif;
  background-color: #000;
  color: #fff;
  overflow: hidden; /* Prevents the BODY from scrolling */
}

#app {
  /* Remove all old centering styles */
  max-width: none;
  margin: 0;
  padding: 0;
  text-align: left;
}

/* Universal box-sizing is good practice */
*,
*::before,
*::after {
  box-sizing: border-box;
}
</style>