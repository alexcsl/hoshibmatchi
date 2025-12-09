<template>
  <div class="password-validator">
    <div
      class="validator-item"
      :class="{ active: hasMinLength }"
    >
      <span class="indicator">✓</span>
      <span class="text">At least 8 characters</span>
    </div>
    <div
      class="validator-item"
      :class="{ active: hasUppercase }"
    >
      <span class="indicator">✓</span>
      <span class="text">One uppercase letter</span>
    </div>
    <div
      class="validator-item"
      :class="{ active: hasLowercase }"
    >
      <span class="indicator">✓</span>
      <span class="text">One lowercase letter</span>
    </div>
    <div
      class="validator-item"
      :class="{ active: hasNumber }"
    >
      <span class="indicator">✓</span>
      <span class="text">One number</span>
    </div>
    <div
      class="validator-item"
      :class="{ active: hasSpecialChar }"
    >
      <span class="indicator">✓</span>
      <span class="text">One special character</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  password?: string
}>();

const hasMinLength = computed(() => (props.password?.length || 0) >= 8);
const hasUppercase = computed(() => /[A-Z]/.test(props.password || ""));
const hasLowercase = computed(() => /[a-z]/.test(props.password || ""));
const hasNumber = computed(() => /[0-9]/.test(props.password || ""));
const hasSpecialChar = computed(() => /[!@#$%^&*(),.?":{}|<>]/.test(props.password || ""));
</script>

<style scoped lang="scss">
.password-validator {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 15px;
  background-color: #262626;
  border-radius: 8px;
  border: 1px solid #404040;

  .validator-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #a8a8a8;
    transition: color 0.2s;

    &.active {
      color: #31a24c;

      .indicator {
        color: #31a24c;
      }
    }

    .indicator {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 18px;
      height: 18px;
      border: 1.5px solid #a8a8a8;
      border-radius: 50%;
      font-size: 11px;
      color: transparent;
      transition: all 0.2s;
    }
  }
}
</style>
