<template>
  <router-link 
    v-if="route"
    :to="`/${route}`"
    class="nav-item"
    :class="{ active }"
    @click.prevent="$emit('click')"
  >
    <span
      class="icon"
      v-html="getIcon(icon)"
    ></span>
    <span class="label">{{ label }}</span>
    <span
      v-if="badge && badge > 0"
      class="badge"
    >{{ badge > 99 ? '99+' : badge }}</span>
  </router-link>
  <button 
    v-else
    class="nav-item"
    @click="$emit('click')"
  >
    <span
      class="icon"
      v-html="getIcon(icon)"
    ></span>
    <span class="label">{{ label }}</span>
    <span
      v-if="badge && badge > 0"
      class="badge"
    >{{ badge > 99 ? '99+' : badge }}</span>
  </button>
</template>

<script setup lang="ts">
defineProps<{
  icon: string
  label: string
  active?: boolean
  route?: string
  badge?: number
}>();

defineEmits<{
  click: []
}>();

const getIcon = (icon: string) => {
  const icons: Record<string, string> = {
    home: "⌂",
    search: "🔍",
    compass: "◉",
    "play-circle": "▶",
    send: "✉",
    heart: "❤",
    "plus-square": "➕",
    user: "👤"
  };
  return icons[icon] || icon;
};
</script>

<style scoped lang="scss">
.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: none;
  border: none;
  color: #fff;
  font-size: 16px;
  cursor: pointer;
  border-radius: 24px;
  transition: all 0.2s;
  text-decoration: none;
  font-weight: 400;
  position: relative;

  .icon {
    font-size: 20px;
  }

  .label {
    font-weight: 500;
  }

  .badge {
    background-color: #ff3040;
    color: #fff;
    font-size: 11px;
    font-weight: 700;
    padding: 2px 6px;
    border-radius: 10px;
    min-width: 18px;
    text-align: center;
    margin-left: auto;
  }

  &:hover {
    background-color: #262626;
  }

  &.active {
    background-color: #262626;
    font-weight: 700;
  }
}
</style>
