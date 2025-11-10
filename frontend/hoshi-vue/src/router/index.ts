import { createRouter, createWebHistory } from 'vue-router'
// We'll create these components in a moment
import MainLayout from '../layouts/MainLayout.vue'
import HomeView from '../views/HomeView.vue'
import RegisterView from '../views/RegisterView.vue'

const routes = [
  // Routes that use the main Instagram-like layout
  {
    path: '/',
    component: MainLayout,
    children: [
      { path: '', name: 'Home', component: HomeView },
      // { path: 'explore', name: 'Explore', component: () => import('../views/ExploreView.vue') },
      // { path: 'reels', name: 'Reels', component: () => import('../views/ReelsView.vue') },
      // { path: 'messages', name: 'Messages', component: () => import('../views/MessagesView.vue') },
      // { path: ':username', name: 'Profile', component: () => import('../views/ProfileView.vue') },
    ]
  },
  
  // Routes that *don't* use the sidebar (e.g., Login, Register)
  {
    path: '/register',
    name: 'Register',
    component: RegisterView,
    // meta: { guestsOnly: true } // For auth guards later
  },
  // {
  //   path: '/login',
  //   name: 'Login',
  //   component: () => import('../views/LoginView.vue'),
  //   meta: { guestsOnly: true }
  // },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router