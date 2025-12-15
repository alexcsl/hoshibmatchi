import { useThemeStore } from '@/stores/theme'
import type { ThemeMode } from '@/stores/theme'

export const useTheme = () => {
  const themeStore = useThemeStore()

  return {
    mode: themeStore.mode,
    activeTheme: themeStore.activeTheme,
    isDark: themeStore.isDark,
    isLight: themeStore.isLight,
    setMode: (mode: ThemeMode) => themeStore.setMode(mode),
    toggle: () => themeStore.toggle(),
  }
}
