import { ref, watch } from 'vue'
import { getSecureMediaURL } from '@/services/media'

/**
 * Composable for loading secure media URLs with caching
 * Handles loading state and error handling
 */
export function useSecureMedia() {
  const loadingMedia = ref(false)
  const secureUrl = ref<string>('')

  /**
   * Load a single secure URL
   */
  const loadSecureUrl = async (mediaPath: string | undefined): Promise<void> => {
    if (!mediaPath) {
      loadingMedia.value = false
      return
    }

    loadingMedia.value = true
    try {
      secureUrl.value = await getSecureMediaURL(mediaPath)
    } catch (error) {
      console.error('Failed to load secure media URL:', error)
      secureUrl.value = mediaPath // Fallback to original
    } finally {
      loadingMedia.value = false
    }
  }

  /**
   * Load multiple secure URLs
   */
  const loadSecureUrls = async (mediaPaths: string[]): Promise<string[]> => {
    if (!mediaPaths || mediaPaths.length === 0) {
      return []
    }

    loadingMedia.value = true
    try {
      const urls = await Promise.all(
        mediaPaths.map(path => getSecureMediaURL(path))
      )
      return urls
    } catch (error) {
      console.error('Failed to load secure media URLs:', error)
      return mediaPaths // Fallback to originals
    } finally {
      loadingMedia.value = false
    }
  }

  /**
   * Create a watcher for reactive media path loading
   */
  const watchAndLoad = (
    mediaPathGetter: () => string | undefined,
    options?: { immediate?: boolean }
  ) => {
    watch(
      mediaPathGetter,
      async (newPath) => {
        if (newPath) {
          await loadSecureUrl(newPath)
        }
      },
      { immediate: options?.immediate ?? true }
    )
  }

  return {
    loadingMedia,
    secureUrl,
    loadSecureUrl,
    loadSecureUrls,
    watchAndLoad
  }
}
