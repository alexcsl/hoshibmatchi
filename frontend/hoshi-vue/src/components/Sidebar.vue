<template>
  <aside class="sidebar">
    <div class="sidebar-content">
      <!-- Logo -->
      <router-link to="/feed" class="logo">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" fill="none"/>
          <path d="M8 12a4 4 0 108 0 4 4 0 00-8 0z" stroke="currentColor" stroke-width="2" fill="none"/>
          <circle cx="17.5" cy="6.5" r="1.5" fill="currentColor"/>
        </svg>
        <span class="logo-text">Instagram</span>
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
        <button class="more-btn" @click="showMoreMenu = !showMoreMenu">
          <span class="icon">☰</span>
          <span class="label">More</span>
        </button>
        
        <div v-if="showMoreMenu" class="more-dropdown">
          <button class="dropdown-item" @click="handleSettings">
            <span class="icon">⚙</span>
            <span>Settings</span>
          </button>
          <button class="dropdown-item" @click="handleSaved">
            <span class="icon">🔖</span>
            <span>Saved</span>
          </button>
          <div class="dropdown-divider"></div>
          <button class="dropdown-item theme-switcher">
            <span class="icon">🌙</span>
            <span>{{ currentTheme === 'dark' ? 'Light Mode' : 'Dark Mode' }}</span>
          </button>
          <div class="dropdown-divider"></div>
          <button class="dropdown-item logout" @click="handleLogout">
            <span class="icon">🚪</span>
            <span>Logout</span>
          </button>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import NavItem from './NavItem.vue'

const props = defineProps<{
  currentRoute: string
}>()

const emit = defineEmits<{
  navigate: [path: string]
  'open-search': []
  'open-notifications': []
  'open-create': []
  'open-settings': []
  'open-saved': []
  logout: []
}>()

const showMoreMenu = ref(false)
const currentTheme = ref('dark')

const handleSettings = () => {
  showMoreMenu.value = false
  emit('open-settings')
}

const handleSaved = () => {
  showMoreMenu.value = false
  emit('open-saved')
}

const handleLogout = () => {
  showMoreMenu.value = false
  emit('logout')
}
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
