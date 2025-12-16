import { defineStore } from 'pinia'
import { ref, watch, computed } from 'vue'
import logger from '@/utils/logger'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type ActiveTheme = 'light' | 'dark'

export const useThemeStore = defineStore('theme', () => {
  // State
  const mode = ref<ThemeMode>('auto')
  const activeTheme = ref<ActiveTheme>('dark')

  // Computed
  const isDark = computed(() => activeTheme.value === 'dark')
  const isLight = computed(() => activeTheme.value === 'light')

  // Get system preference
  const getSystemTheme = (): ActiveTheme => {
    if (typeof window === 'undefined') return 'dark'
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }

  // Set theme class on document
  const applyTheme = (theme: ActiveTheme) => {
    if (typeof document === 'undefined') return
    
    document.documentElement.classList.remove('light-theme', 'dark-theme')
    document.documentElement.classList.add(`${theme}-theme`)
    document.documentElement.setAttribute('data-theme', theme)
    
    // Update meta theme-color for mobile browsers
    const metaThemeColor = document.querySelector('meta[name="theme-color"]')
    if (metaThemeColor) {
      metaThemeColor.setAttribute('content', theme === 'dark' ? '#000000' : '#ffffff')
    }
    
    logger.debug('Theme applied:', theme)
  }

  // Update active theme based on mode
  const updateActiveTheme = () => {
    let newTheme: ActiveTheme
    
    if (mode.value === 'auto') {
      newTheme = getSystemTheme()
    } else {
      newTheme = mode.value
    }
    
    if (newTheme !== activeTheme.value) {
      activeTheme.value = newTheme
      applyTheme(newTheme)
    }
  }

  // Set theme mode
  const setMode = (newMode: ThemeMode) => {
    logger.info('Setting theme mode:', newMode)
    mode.value = newMode
    localStorage.setItem('theme-mode', newMode)
    updateActiveTheme()
  }

  // Toggle between light and dark
  const toggle = () => {
    const newMode = activeTheme.value === 'dark' ? 'light' : 'dark'
    setMode(newMode)
  }

  // Initialize theme
  const initialize = () => {
    // Load saved preference
    const saved = localStorage.getItem('theme-mode') as ThemeMode | null
    if (saved && ['light', 'dark', 'auto'].includes(saved)) {
      mode.value = saved
    }

    // Apply initial theme
    updateActiveTheme()

    // Listen for system theme changes (only in auto mode)
    if (typeof window !== 'undefined') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      mediaQuery.addEventListener('change', () => {
        if (mode.value === 'auto') {
          updateActiveTheme()
        }
      })
    }

    logger.info('Theme system initialized. Mode:', mode.value, 'Active:', activeTheme.value)
  }

  // Watch mode changes
  watch(mode, () => {
    updateActiveTheme()
  })

  return {
    // State
    mode,
    activeTheme,
    
    // Computed
    isDark,
    isLight,
    
    // Actions
    setMode,
    toggle,
    initialize,
  }
})
