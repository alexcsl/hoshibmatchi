import apiClient from './api'

// Cache for pre-signed URLs
interface CachedURL {
  url: string
  expiry: number
}

const urlCache = new Map<string, CachedURL>()

/**
 * Get a secure pre-signed URL for media access
 * Implements caching to minimize API calls
 * 
 * @param objectPath - The object path in MinIO (e.g., "user-1/posts/abc.jpg")
 * @param expirySeconds - Optional expiry time in seconds (default: 3600)
 * @returns Pre-signed URL with expiration
 */
export const getSecureMediaURL = async (
  objectPath: string, 
  expirySeconds: number = 3600
): Promise<string> => {
  if (!objectPath || objectPath.trim() === '') {
    throw new Error('Object path is required')
  }

  // Extract object path if it's a full URL
  let cleanPath = objectPath
  if (objectPath.includes('localhost:9000/media/')) {
    cleanPath = objectPath.split('localhost:9000/media/')[1]
  } else if (objectPath.includes('/media/')) {
    cleanPath = objectPath.split('/media/')[1]
  }

  // Decode URL encoding
  cleanPath = decodeURIComponent(cleanPath)

  const now = Date.now()
  const cached = urlCache.get(cleanPath)

  // Return cached URL if still valid (with 5-minute safety buffer)
  if (cached && cached.expiry > now + 300000) {
    return cached.url
  }

  try {
    // Generate new pre-signed URL
    const response = await apiClient.get('/media/secure-url', {
      params: { 
        object_name: cleanPath,
        expiry_seconds: expirySeconds 
      }
    })

    const presignedUrl = response.data.media_url

    // Cache for (expiry - 10 minutes) to avoid using expired URLs
    urlCache.set(cleanPath, {
      url: presignedUrl,
      expiry: now + (expirySeconds - 600) * 1000
    })

    return presignedUrl
  } catch (error) {
    console.error('Failed to get secure media URL:', error)
    throw error
  }
}

/**
 * Batch load multiple pre-signed URLs at once
 * More efficient than loading one by one
 * 
 * @param objectPaths - Array of object paths
 * @returns Array of pre-signed URLs in the same order
 */
export const getSecureMediaURLs = async (objectPaths: string[]): Promise<string[]> => {
  const promises = objectPaths.map(path => getSecureMediaURL(path))
  return Promise.all(promises)
}

/**
 * Clear the URL cache (useful for testing or memory management)
 */
export const clearMediaURLCache = () => {
  urlCache.clear()
}

/**
 * Check if a URL needs to be converted to pre-signed URL
 * Returns false for already pre-signed URLs or external URLs
 */
export const needsSecureURL = (url: string): boolean => {
  if (!url) return false
  
  // Already a pre-signed URL (has X-Amz-Algorithm)
  if (url.includes('X-Amz-Algorithm')) return false
  
  // External URL (not our MinIO)
  if (url.startsWith('http') && !url.includes('localhost:9000')) return false
  
  // Placeholder or default images
  if (url.includes('placeholder.svg') || url.includes('default-avatar.svg')) return false
  
  // Needs secure URL
  return true
}

/**
 * Get secure URL for any media (post media, profile pictures, thumbnails, etc.)
 * Handles fallback for placeholders and external URLs
 * 
 * @param url - Any media URL (can be object path or full URL)
 * @param fallback - Fallback URL if loading fails
 * @returns Secure URL or fallback
 */
export const getSecureURL = async (
  url: string | undefined,
  fallback: string = '/placeholder.svg?height=400&width=400'
): Promise<string> => {
  if (!url || url.trim() === '') {
    return fallback
  }

  // Don't convert placeholders or external URLs
  if (!needsSecureURL(url)) {
    return url
  }

  try {
    return await getSecureMediaURL(url)
  } catch (error) {
    console.error('Failed to get secure URL:', error)
    return fallback
  }
}
