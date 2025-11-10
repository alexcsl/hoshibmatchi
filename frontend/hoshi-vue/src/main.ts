import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router' // <-- Import the router

try {
  createApp(App)
    .use(router) // <-- Use the router
    .mount('#app')
} catch (error) {
  console.error('Failed to mount Vue app:', error)
}