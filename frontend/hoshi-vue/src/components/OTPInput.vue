<template>
  <div class="otp-container">
    <div class="otp-inputs">
      <input
        v-for="(digit, index) in digits"
        :key="index"
        type="text"
        maxlength="1"
        :value="digit"
        class="otp-digit"
        inputmode="numeric"
        @input="handleInput(index, $event)"
        @keydown="handleKeydown(index, $event)"
      />
    </div>
    <p
      v-if="errorMessage"
      class="error-message"
    >
      {{ errorMessage }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  modelValue?: string
  errorMessage?: string
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string]
}>();

const digits = computed(() => {
  const arr = props.modelValue?.split("") || [];
  while (arr.length < 6) arr.push("");
  return arr.slice(0, 6);
});

const handleInput = (index: number, event: Event) => {
  const input = event.target as HTMLInputElement;
  const value = input.value.replace(/[^0-9]/g, "");
  
  if (value) {
    const newValue = props.modelValue?.split("") || [];
    newValue[index] = value;
    emit("update:modelValue", newValue.slice(0, 6).join(""));
    
    if (index < 5) {
      const nextInput = document.querySelectorAll(".otp-digit")[index + 1] as HTMLInputElement;
      nextInput?.focus();
    }
  }
};

const handleKeydown = (index: number, event: KeyboardEvent) => {
  if (event.key === "Backspace") {
    const newValue = props.modelValue?.split("") || [];
    newValue[index] = "";
    emit("update:modelValue", newValue.slice(0, 6).join(""));
    
    if (index > 0) {
      const prevInput = document.querySelectorAll(".otp-digit")[index - 1] as HTMLInputElement;
      prevInput?.focus();
    }
  } else if (event.key === "ArrowLeft" && index > 0) {
    const prevInput = document.querySelectorAll(".otp-digit")[index - 1] as HTMLInputElement;
    prevInput?.focus();
  } else if (event.key === "ArrowRight" && index < 5) {
    const nextInput = document.querySelectorAll(".otp-digit")[index + 1] as HTMLInputElement;
    nextInput?.focus();
  }
};
</script>

<style scoped lang="scss">
.otp-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;

  .otp-inputs {
    display: flex;
    gap: 10px;
    justify-content: center;
  }

  .otp-digit {
    width: 50px;
    height: 50px;
    text-align: center;
    font-size: 24px;
    font-weight: 600;
    background-color: #262626;
    border: 2px solid #404040;
    border-radius: 8px;
    color: #fff;
    outline: none;
    transition: all 0.2s;

    &:focus {
      border-color: #0a66c2;
      background-color: #1a1a1a;
    }

    &:invalid {
      border-color: #f52424;
    }
  }

  .error-message {
    font-size: 12px;
    color: #f52424;
    text-align: center;
  }
}
</style>
