<template>
  <img
    v-if="!loading && secureUrl"
    :src="secureUrl"
    :alt="alt"
    :class="className"
    @error="handleError"
  />
  <img
    v-else-if="loading"
    :src="loadingPlaceholder"
    :alt="alt"
    :class="className"
  />
  <img
    v-else
    :src="errorPlaceholder"
    :alt="alt"
    :class="className"
  />
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { getSecureURL } from '@/services/media'

interface Props {
  src: string | undefined
  alt?: string
  className?: string
  loadingPlaceholder?: string
  errorPlaceholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  alt: '',
  className: '',
  loadingPlaceholder: '/placeholder.svg?height=200&width=200&text=Loading',
  errorPlaceholder: '/placeholder.svg?height=200&width=200&text=Error'
})

const secureUrl = ref<string>('')
const loading = ref(true)

const loadSecureUrl = async () => {
  if (!props.src) {
    loading.value = false
    return
  }

  loading.value = true
  try {
    secureUrl.value = await getSecureURL(props.src, props.errorPlaceholder)
  } catch (error) {
    console.error('Failed to load secure URL:', error)
    secureUrl.value = props.errorPlaceholder
  } finally {
    loading.value = false
  }
}

const handleError = () => {
  secureUrl.value = props.errorPlaceholder
}

// Load on mount and when src changes
onMounted(() => {
  loadSecureUrl()
})

watch(() => props.src, () => {
  loadSecureUrl()
})
</script>
