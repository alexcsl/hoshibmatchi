import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import RegisterView from '../views/RegisterView.vue' // Your existing view
import MainLayout from '../layouts/MainLayout.vue' // We will create this

const routes = [
  // Routes that use the main Instagram-like layout
  {
    path: '/',
    component: MainLayout,
    // These are "children" of the layout
    children: [
      { path: '', name: 'Home', component: HomeView },
      // TODO: Add routes for Explore, Reels, Profile, etc. [cite: 717-722]
      // { path: '/explore', name: 'Explore', component: () => import('../views/ExploreView.vue') },
      // { path: '/:username', name: 'Profile', component: () => import('../views/ProfileView.vue') },
    ]
  },
  
  // Routes that do *not* use the main layout (e.g., login/register)
  {
    path: '/register',
    name: 'Register',
    component: RegisterView,
    // TODO: Add meta guard for "guests only" [cite: 602]
  },
  // TODO: Add Login Page route [cite: 632]
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router