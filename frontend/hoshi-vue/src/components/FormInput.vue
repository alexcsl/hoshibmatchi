<template>
  <div class="form-input">
    <input
      :type="type"
      :placeholder="placeholder"
      :value="modelValue"
      class="input"
      :class="{ error: errorMessage }"
      @input="handleInput"
    />
    <p
      v-if="errorMessage"
      class="error-message"
    >
      {{ errorMessage }}
    </p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  type?: string
  placeholder?: string
  modelValue?: string
  errorMessage?: string
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string]
}>();

const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  emit("update:modelValue", target.value);
};
</script>

<style scoped lang="scss">
.form-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;

  .input {
    width: 100%;
    padding: 10px 16px;
    background-color: #262626;
    border: 1px solid #404040;
    border-radius: 5px;
    color: #fff;
    font-size: 14px;
    outline: none;
    transition: all 0.2s;

    &:focus {
      border-color: #818384;
      background-color: #1a1a1a;
    }

    &::placeholder {
      color: #a8a8a8;
    }

    &.error {
      border-color: #f52424;
      background-color: rgba(245, 36, 36, 0.1);
    }
  }

  .error-message {
    font-size: 12px;
    color: #f52424;
    margin-top: -4px;
  }
}
</style>
