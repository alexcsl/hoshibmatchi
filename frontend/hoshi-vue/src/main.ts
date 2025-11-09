import { createApp } from 'vue'
import './style.css'
import App from './App.vue'

try {
  createApp(App).mount('#app')
} catch (error) {
  console.error('Failed to mount Vue app:', error)
}
