<template>
  <div 
    class="mini-message" 
    :style="messageStyle" 
    @mousedown="startDrag"
    @click="handleClick"
  >
    <div class="message-header">
      <span class="icon">💬</span>
      <span class="title">Messages</span>
      <div class="message-avatars">
        <img
          v-for="msg in messages.slice(0, 3)"
          :key="msg.id"
          :src="msg.avatar"
          :alt="msg.username"
          class="avatar"
        />
      </div>
    </div>
    
    <div
      v-if="hasUnread"
      class="unread-badge"
    >
      {{ totalUnread }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";

interface Message {
  id: number
  username: string
  avatar: string
  unreadCount: number
}

const props = defineProps<{
  messages: Message[]
}>();

const emit = defineEmits<{
  click: []
}>();

const isDragging = ref(false);
const dragStartTime = ref(0);
const offsetX = ref(0);
const offsetY = ref(0);
const posX = ref(0);
const posY = ref(0);

const messageStyle = computed(() => ({
  left: `${posX.value}px`,
  top: `${posY.value}px`
}));

const hasUnread = computed(() => props.messages.some(m => m.unreadCount > 0));
const totalUnread = computed(() => props.messages.reduce((sum, m) => sum + m.unreadCount, 0));

// Initialize position from localStorage or default to bottom-right
onMounted(() => {
  const savedPos = localStorage.getItem("miniMessagePosition");
  if (savedPos) {
    try {
      const { x, y } = JSON.parse(savedPos);
      posX.value = x;
      posY.value = y;
    } catch {
      // Use default position
      setDefaultPosition();
    }
  } else {
    setDefaultPosition();
  }
});

const setDefaultPosition = () => {
  posX.value = window.innerWidth - 320;
  posY.value = window.innerHeight - 120;
};

const startDrag = (e: MouseEvent) => {
  isDragging.value = false;
  dragStartTime.value = Date.now();
  offsetX.value = e.clientX - posX.value;
  offsetY.value = e.clientY - posY.value;

  const handleMouseMove = (moveEvent: MouseEvent) => {
    // If mouse moved more than 5 pixels, it's a drag
    const dx = moveEvent.clientX - (offsetX.value + posX.value);
    const dy = moveEvent.clientY - (offsetY.value + posY.value);
    if (Math.abs(dx) > 5 || Math.abs(dy) > 5) {
      isDragging.value = true;
    }

    if (isDragging.value) {
      let newX = moveEvent.clientX - offsetX.value;
      let newY = moveEvent.clientY - offsetY.value;

      // Constrain to window bounds
      const maxX = window.innerWidth - 280;  // component width
      const maxY = window.innerHeight - 80;  // component height
      
      newX = Math.max(0, Math.min(newX, maxX));
      newY = Math.max(0, Math.min(newY, maxY));

      posX.value = newX;
      posY.value = newY;
    }
  };

  const handleMouseUp = () => {
    // Save position to localStorage
    if (isDragging.value) {
      localStorage.setItem("miniMessagePosition", JSON.stringify({
        x: posX.value,
        y: posY.value
      }));
    }

    // Reset after a short delay to allow click detection
    setTimeout(() => {
      isDragging.value = false;
    }, 10);

    document.removeEventListener("mousemove", handleMouseMove);
    document.removeEventListener("mouseup", handleMouseUp);
  };

  document.addEventListener("mousemove", handleMouseMove);
  document.addEventListener("mouseup", handleMouseUp);
};

const handleClick = () => {
  // Only emit click if it wasn't a drag
  if (!isDragging.value && Date.now() - dragStartTime.value < 300) {
    emit("click");
  }
};
</script>

<style scoped lang="scss">
.mini-message {
  position: fixed;
  width: 280px;
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  border: 1px solid #404040;
  border-radius: 20px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: grab;
  z-index: 40;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  transition: background 0.2s;
  user-select: none;

  &:hover {
    background: linear-gradient(135deg, #262626 0%, #333 100%);
  }

  &:active {
    cursor: grabbing;
  }

  .message-header {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;

    .icon {
      font-size: 20px;
    }

    .title {
      font-weight: 600;
      font-size: 14px;
    }

    .message-avatars {
      display: flex;
      margin-left: auto;
      gap: -8px;

      .avatar {
        width: 28px;
        height: 28px;
        border-radius: 50%;
        border: 2px solid #1a1a1a;
        object-fit: cover;
        margin-left: -8px;

        &:first-child {
          margin-left: 0;
        }
      }
    }
  }

  .unread-badge {
    position: absolute;
    top: -8px;
    left: 20px;
    background-color: #e0245e;
    color: #fff;
    border-radius: 50%;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 700;
    border: 2px solid #1a1a1a;
  }
}
</style>
