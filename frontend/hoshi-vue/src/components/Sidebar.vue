<template>
  <aside class="sidebar">
    <div class="sidebar-content">
      <!-- Logo -->
      <router-link
        to="/feed"
        class="logo"
      >
        <img
          class="logo-image"
          src="../../public/instagram-logo.png"
          alt="Hoshi Logo"
        />
        <span class="logo-text">hoshiBmaTchi</span>
      </router-link>

      <!-- Navigation Items -->
      <nav class="nav-items">
        <NavItem 
          icon="home"
          label="Home"
          route="feed"
          :active="currentRoute === 'feed'"
          @click="$emit('navigate', 'feed')"
        />
        <NavItem 
          icon="search"
          label="Search"
          @click="$emit('open-search')"
        />
        <NavItem 
          icon="compass"
          label="Explore"
          route="explore"
          :active="currentRoute === 'explore'"
          @click="$emit('navigate', 'explore')"
        />
        <NavItem 
          icon="play-circle"
          label="Reels"
          route="reels"
          :active="currentRoute === 'reels'"
          @click="$emit('navigate', 'reels')"
        />
        <NavItem 
          icon="send"
          label="Messages"
          route="messages"
          :active="currentRoute === 'messages'"
          @click="$emit('navigate', 'messages')"
        />
        <NavItem 
          icon="heart"
          label="Notifications"
          :badge="unreadNotificationCount"
          @click="$emit('open-notifications')"
        />
        <NavItem 
          icon="plus-square"
          label="Create"
          @click="$emit('open-create')"
        />
        <NavItem 
          icon="user"
          label="Profile"
          route="profile"
          :active="currentRoute === 'profile'"
          @click="$emit('navigate', 'profile')"
        />
      </nav>

      <!-- More Menu -->
      <div class="more-menu">
        <button
          class="more-btn"
          @click="showMoreMenu = !showMoreMenu"
        >
          <span class="icon">☰</span>
          <span class="label">More</span>
        </button>
        
        <div
          v-if="showMoreMenu"
          class="more-dropdown"
        >
          <button
            class="dropdown-item"
            @click="handleSettings"
          >
            <span class="icon">⚙</span>
            <span>Settings</span>
          </button>
          <button
            class="dropdown-item"
            @click="handleSaved"
          >
            <span class="icon">📦</span>
            <span>Archive</span>
          </button>
          <button
            v-if="isAdmin"
            class="dropdown-item"
            @click="handleAdmin"
          >
            <span class="icon">👑</span>
            <span>Admin Dashboard</span>
          </button>
          <div class="dropdown-divider"></div>
          <button class="dropdown-item theme-switcher">
            <span class="icon">🌙</span>
            <span>{{ currentTheme === 'dark' ? 'Light Mode' : 'Dark Mode' }}</span>
          </button>
          <div class="dropdown-divider"></div>
          <button
            class="dropdown-item logout"
            @click="handleLogout"
          >
            <span class="icon">🚪</span>
            <span>Logout</span>
          </button>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from "vue";
import NavItem from "./NavItem.vue";
import { notificationAPI } from "../services/api";
import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();

defineProps<{
  currentRoute: string
}>();

const emit = defineEmits<{
  navigate: [path: string]
  "open-search": []
  "open-notifications": []
  "open-create": []
  "open-settings": []
  "open-saved": []
  logout: []
}>();

const showMoreMenu = ref(false);
const currentTheme = ref("dark");
const unreadNotificationCount = ref(0);
let pollInterval: ReturnType<typeof setInterval> | null = null;

// Check if user is admin
const isAdmin = computed(() => {
  const token = authStore.token;
  if (!token) return false;
  
  try {
    const base64Url = token.split(".")[1];
    const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
    const jsonPayload = decodeURIComponent(window.atob(base64).split("").map(function(c) {
        return "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2);
    }).join(""));
    const decoded = JSON.parse(jsonPayload);
    return decoded.role === "admin";
  } catch {
    return false;
  }
});

const fetchUnreadCount = async () => {
  try {
    const data = await notificationAPI.getNotifications(1); // Only fetch 1 to get count
    unreadNotificationCount.value = data.unread_count;
  } catch {
    // Silently fail - don't spam console if backend is unavailable
    // Just keep the badge at 0
    unreadNotificationCount.value = 0;
  }
};

const handleSettings = () => {
  showMoreMenu.value = false;
  emit("open-settings");
};

const handleSaved = () => {
  showMoreMenu.value = false;
  emit("open-saved");
};

const handleAdmin = () => {
  showMoreMenu.value = false;
  emit("navigate", "admin");
};

const handleLogout = () => {
  showMoreMenu.value = false;
  emit("logout");
};

onMounted(() => {
  fetchUnreadCount();
  // Poll for unread count every 10 seconds
  pollInterval = setInterval(fetchUnreadCount, 10000);
});

onUnmounted(() => {
  if (pollInterval) {
    clearInterval(pollInterval);
  }
});
</script>

<style scoped lang="scss">
@import '@../../../styles/fonts.css';
.sidebar {
  width: 244px;
  height: 100vh;
  border-right: 1px solid #262626;
  background-color: #000;
  position: fixed;
  left: 0;
  top: 0;
  overflow-y: auto;
  padding: 16px 12px;
  z-index: 50;
}

.sidebar-content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 32px;
  text-decoration: none;
  color: #fff;
  cursor: pointer;
  transition: opacity 0.2s;

  &:hover {
    opacity: 0.7;
  }

  svg {
    width: 24px;
    height: 24px;
  }

  .logo-image {
    width: 32px;
    height: 32px;
    object-fit: contain;
  }

  .logo-text {
    font-family: 'Instagram', cursive;
    font-size: 20px;
    font-weight: 300;
  }
}

.nav-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  margin-bottom: auto;
}

.more-menu {
  position: relative;

  .more-btn {
    width: 100%;
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
    transition: background-color 0.2s;

    &:hover {
      background-color: #262626;
    }

    .icon {
      font-size: 20px;
    }

    .label {
      font-weight: 500;
    }
  }

  .more-dropdown {
    position: absolute;
    bottom: calc(100% + 8px);
    left: 0;
    background-color: #262626;
    border-radius: 8px;
    min-width: 200px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.8);
    z-index: 100;
    overflow: hidden;

    .dropdown-item {
      width: 100%;
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 16px;
      background: none;
      border: none;
      color: #fff;
      font-size: 14px;
      cursor: pointer;
      text-align: left;
      transition: background-color 0.2s;

      &:hover {
        background-color: #404040;
      }

      .icon {
        font-size: 16px;
      }

      &.logout {
        color: #ff4458;
      }
    }

    .dropdown-divider {
      height: 1px;
      background-color: #404040;
      margin: 4px 0;
    }
  }
}

@media (max-width: 1024px) {
  .sidebar {
    width: 72px;
    padding: 12px 0;

    .logo-text {
      display: none;
    }

    .logo {
      margin-bottom: 24px;
      justify-content: center;
    }

    .nav-items {
      gap: 12px;
    }

    .more-menu .more-btn .label {
      display: none;
    }
  }
}

@media (max-width: 768px) {
  .sidebar {
    width: 60px;
  }
}
</style>
