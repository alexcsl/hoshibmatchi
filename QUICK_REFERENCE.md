# Phase 1 Complete - Quick Reference

## ✅ What's Done

### Authentication System
Complete end-to-end authentication with:
- User registration with email verification
- Standard login
- Two-factor authentication (2FA)
- OTP verification and resend
- Session management
- JWT token handling

### State Management
Pinia stores created and integrated:
- **auth.ts** - Full authentication logic
- **feed.ts** - Ready for Phase 2
- **user.ts** - Ready for Phase 2

### API Integration
Comprehensive API service with all backend endpoints:
- 8 domain-specific API groups
- 40+ endpoint methods
- Full TypeScript interfaces
- Error handling
- JWT token interceptors

## 🎯 How to Use

### In Your Components

#### Check if User is Logged In
```vue
<script setup>
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
</script>

<template>
  <div v-if="authStore.isLoggedIn">
    Welcome {{ authStore.currentUser?.username }}!
  </div>
</template>
```

#### Perform Login
```vue
<script setup>
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const router = useRouter()

const handleLogin = async () => {
  try {
    const response = await authStore.login({
      email_or_username: 'user@example.com',
      password: 'Password123!'
    })
    
    if (response.requires_2fa) {
      // Store temp data and redirect to 2FA
      sessionStorage.setItem('temp_user_id', response.user_id)
      router.push('/login-otp')
    } else {
      // Login success, redirect to feed
      router.push('/feed')
    }
  } catch (error) {
    // Error is already in authStore.error
    console.error('Login failed:', authStore.error)
  }
}
</script>
```

#### Logout
```vue
<script setup>
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const router = useRouter()

const handleLogout = () => {
  authStore.logout()
  router.push('/login')
}
</script>
```

#### Show Loading States
```vue
<template>
  <button :disabled="authStore.loading">
    {{ authStore.loading ? 'Loading...' : 'Submit' }}
  </button>
</template>
```

#### Display Errors
```vue
<template>
  <div v-if="authStore.error" class="error">
    {{ authStore.error }}
    <button @click="authStore.clearError()">✕</button>
  </div>
</template>
```

## 📁 File Structure Reference

```
frontend/hoshi-vue/src/
├── stores/
│   ├── auth.ts          ← Auth state management
│   ├── feed.ts          ← Feed state (Phase 2)
│   └── user.ts          ← User profile state (Phase 2)
├── services/
│   └── api.ts           ← All API endpoints
├── pages/
│   ├── Login.vue        ← Updated for store
│   ├── SignUp.vue       ← Updated for store
│   ├── LoginOTP.vue     ← Updated for store
│   └── VerifyOTP.vue    ← Updated for store
└── main.ts              ← Pinia integrated
```

## 🔑 Auth Store API

### State
- `user: User | null` - Current user object
- `token: string | null` - JWT token
- `isAuthenticated: boolean` - Auth status
- `loading: boolean` - Request in progress
- `error: string | null` - Error message

### Getters
- `currentUser` - Get current user
- `isLoggedIn` - Check if authenticated
- `userId` - Get current user ID

### Actions
```typescript
// Register new user
await authStore.register({
  name: string,
  username: string,
  email: string,
  password: string,
  confirm_password: string,
  gender: string,
  date_of_birth: string
})

// Login
const response = await authStore.login({
  email_or_username: string,
  password: string
})
// Returns: { requires_2fa, user_id?, username? } or { success }

// Verify 2FA
await authStore.verify2FA({
  user_id: number,
  otp_code: string
})

// Verify registration OTP
await authStore.verifyRegistrationOTP({
  email: string,
  otp_code: string
})

// Request new OTP
await authStore.requestOTP({
  email: string
})

// Logout
authStore.logout()

// Clear errors
authStore.clearError()
```

## 🌐 API Endpoints Available

### Auth API
```typescript
import { authAPI } from '@/services/api'

// All auth endpoints ready:
authAPI.register(data)
authAPI.login(data)
authAPI.verify2FA(data)
authAPI.verifyRegistrationOTP(data)
authAPI.requestOTP(data)
authAPI.forgotPassword(data)
authAPI.resetPassword(data)
authAPI.googleAuth(data)
authAPI.logout()
```

### Other APIs (Phase 2+)
```typescript
import { 
  feedAPI,      // Home/Explore/Reels feeds
  postAPI,      // Posts CRUD
  storyAPI,     // Stories
  commentAPI,   // Comments
  collectionAPI,// Saved collections
  searchAPI,    // Search users/hashtags
  messageAPI,   // Direct messages
  reportAPI,    // Report content
  userAPI       // User profiles
} from '@/services/api'
```

## 🚨 Important Notes

### Path Aliases
Always use `@/` for imports:
```typescript
// ✅ Correct
import { useAuthStore } from '@/stores/auth'
import { authAPI } from '@/services/api'

// ❌ Avoid
import { useAuthStore } from '../stores/auth'
import { authAPI } from '../services/api'
```

### TypeScript Errors
If you see `Cannot find module '@/stores/auth'`:
- This is a cosmetic IDE error
- Code will run fine in browser
- Reload VS Code window to fix: `Ctrl+Shift+P` → "Developer: Reload Window"

### localStorage Keys
```typescript
// Used by auth system:
localStorage.getItem('jwt_token')      // JWT token
localStorage.getItem('user')           // User data JSON
localStorage.getItem('miniMessagePos') // MiniMessage position

// Temporary session storage (2FA flow):
sessionStorage.getItem('temp_user_id')
sessionStorage.getItem('temp_username')
```

## 🧪 Testing Commands

### Start Development Server
```powershell
cd frontend/hoshi-vue
npm run dev
```

### Check TypeScript Errors
```powershell
npm run type-check
```

### Build for Production
```powershell
npm run build
```

## 📝 Common Tasks

### Protect a Route (Router Guard)
```typescript
// In router/index.ts
{
  path: '/profile',
  component: Profile,
  meta: { requiresAuth: true }
}

// Add global guard:
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  
  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    next('/login')
  } else {
    next()
  }
})
```

### Add New API Endpoint
```typescript
// In services/api.ts
export const myAPI = {
  async myNewEndpoint(data: MyDataType) {
    const response = await apiClient.post('/api/my-endpoint', data)
    return response.data
  }
}
```

### Add Store Action
```typescript
// In stores/mystore.ts
export const useMyStore = defineStore('mystore', {
  state: () => ({ ... }),
  
  actions: {
    async myAction(params) {
      this.loading = true
      try {
        const result = await myAPI.myNewEndpoint(params)
        // Update state
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    }
  }
})
```

## 🐛 Troubleshooting

### Problem: Login succeeds but doesn't redirect
**Check**: 
- Router is imported correctly
- `/feed` route exists
- No console errors

### Problem: Token not persisting
**Check**:
- `saveAuthData()` is called after login
- localStorage is not blocked by browser
- Token is included in API requests (Network tab)

### Problem: 2FA flow not working
**Check**:
- Backend returns `requires_2fa: true`
- `user_id` is stored in sessionStorage
- `/login-otp` route exists
- OTP endpoint is correct

### Problem: Store not updating
**Check**:
- Pinia is installed and imported in main.ts
- `createPinia()` is added to app
- Store is imported correctly with `use` prefix

## 📚 Documentation Links

- **Phase 1 Summary**: `PHASE_1_IMPLEMENTATION_SUMMARY.md`
- **Testing Guide**: `TESTING_GUIDE.md`
- **Full Integration Plan**: `FRONTEND_BACKEND_INTEGRATION_PLAN.md`
- **Quick Start**: `QUICK_START.md`

## 🎉 Next Phase

Phase 2 is ready to start:
- Feed implementation
- Post creation and display
- Like/comment functionality
- User profiles
- Follow system

See `FRONTEND_BACKEND_INTEGRATION_PLAN.md` for Phase 2 details.

---

**Status**: Phase 1 ✅ Complete
**Backend Integration**: Ready for testing
**Next**: Phase 2 - Feed & Posts
