import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from '../services/api'

// Layouts
import MainLayout from '../layouts/MainLayout.vue'

// Views
import HomeView from '../views/HomeView.vue'
import RegisterView from '../views/RegisterView.vue'

// Pages
import Login from '../pages/Login.vue'
import SignUp from '../pages/SignUp.vue'
import LoginOTP from '../pages/LoginOTP.vue'
import ForgotPassword from '../pages/ForgotPassword.vue'
import ResetPassword from '../pages/ResetPassword.vue'
import Feed from '../pages/Feed.vue'
import Explore from '../pages/Explore.vue'
import HashtagExplore from '../pages/HashtagExplore.vue'
import Reels from '../pages/Reels.vue'
import Messages from '../pages/Messages.vue'
import Profile from '../pages/Profile.vue'
import GoogleCallback from '../pages/GoogleCallback.vue'
import GoogleCompleteProfile from '../pages/GoogleCompleteProfile.vue'
import VerifyOTP from '../pages/VerifyOTP.vue'
import EditProfile from '../pages/EditProfile.vue'
import CreateStory from '../pages/CreateStory.vue'
import Admin from '../pages/Admin.vue'

const routes = [
  // Auth Routes (no sidebar layout)
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { guestsOnly: true }
  },
  {
    path: '/signup',
    name: 'SignUp',
    component: SignUp,
    meta: { guestsOnly: true }
  },
  {
    path: '/login-otp',
    name: 'LoginOTP',
    component: LoginOTP,
    meta: { guestsOnly: true }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: ForgotPassword,
    meta: { guestsOnly: true }
  },
  {
    path: '/reset-password',
    name: 'ResetPassword',
    component: ResetPassword,
    meta: { guestsOnly: true }
  },
  {
    path: '/verify-otp',
    name: 'VerifyOTP',
    component: VerifyOTP,
    meta: { guestsOnly: true }
  },
  {
    path: '/auth/google/callback',
    name: 'GoogleCallback',
    component: GoogleCallback,
    meta: { guestsOnly: true }
  },
  {
    path: '/auth/google/complete-profile',
    name: 'google-complete-profile',
    component: GoogleCompleteProfile,
    meta: { requiresAuth: true }
  },
  
  // Main App Routes (with sidebar layout)
  {
    path: '/',
    component: MainLayout,
    meta: { requiresAuth: true },
    children: [
      { 
        path: '', 
        redirect: '/feed'
      },
      { 
        path: 'feed', 
        name: 'Feed', 
        component: Feed 
      },
      { 
        path: 'explore', 
        name: 'Explore', 
        component: Explore 
      },
      { 
        path: 'explore/tags/:hashtag', 
        name: 'HashtagExplore', 
        component: HashtagExplore 
      },
      { 
        path: 'reels', 
        name: 'Reels', 
        component: Reels 
      },
      { 
        path: 'messages', 
        name: 'Messages', 
        component: Messages 
      },
      { 
        path: 'profile', 
        name: 'ProfileOwn', 
        component: Profile 
      },
      { 
        path: 'profile/:username?', 
        name: 'Profile', 
        component: Profile 
      },
      { 
        path: 'settings', 
        name: 'Settings', 
        component: () => import('../pages/Settings.vue')
      },
      { 
        path: 'archive', 
        name: 'Archive', 
        component: () => import('../pages/Archive.vue')
      },
      { 
        path: 'edit-profile', 
        name: 'EditProfile', 
        component: () => import('../pages/EditProfile.vue')
      },
      { 
      path: '/create-story', 
      name: 'create-story', 
      component: CreateStory 
      },
      { 
        path: 'admin', 
        name: 'Admin', 
        component: Admin,
        meta: { requiresAdmin: true }
      },

      // Future routes
      // { path: ':username', name: 'Profile', component: () => import('../views/ProfileView.vue') },
    ]
  },
  
  // Legacy routes (can be removed later)
  {
    path: '/register',
    name: 'Register',
    component: RegisterView,
    meta: { guestsOnly: true }
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guards
router.beforeEach((to, from, next) => {
  const authenticated = isAuthenticated()

  // Redirect authenticated users away from guest-only pages
  if (to.meta.guestsOnly && authenticated) {
    next('/feed')
    return
  }

  // Redirect unauthenticated users to login
  if (to.meta.requiresAuth && !authenticated) {
    next('/login')
    return
  }

  // Check admin access
  if (to.meta.requiresAdmin) {
    const token = localStorage.getItem('jwt_token')
    if (!token) {
      next('/login')
      return
    }
    
    try {
      const base64Url = token.split('.')[1]
      const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
      const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function(c) {
          return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2)
      }).join(''))
      const decoded = JSON.parse(jsonPayload)
      
      if (decoded.role !== 'admin') {
        next('/feed')
        return
      }
    } catch {
      next('/login')
      return
    }
  }

  next()
})

export default router