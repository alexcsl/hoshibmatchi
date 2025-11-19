<template>
  <div v-if="message" class="error-alert">
    <span class="error-icon">⚠</span>
    <span class="error-text">{{ message }}</span>
    <button @click="$emit('close')" class="close-btn">✕</button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  message?: string
}>()

defineEmits<{
  close: []
}>()
</script>

<style scoped lang="scss">
.error-alert {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background-color: rgba(245, 36, 36, 0.1);
  border: 1px solid #f52424;
  border-radius: 6px;
  color: #f52424;
  font-size: 13px;
  animation: slideDown 0.3s ease-out;

  @keyframes slideDown {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .error-icon {
    font-size: 16px;
    flex-shrink: 0;
  }

  .error-text {
    flex: 1;
  }

  .close-btn {
    background: none;
    border: none;
    color: #f52424;
    cursor: pointer;
    font-size: 18px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: opacity 0.2s;

    &:hover {
      opacity: 0.7;
    }
  }
}
</style>
