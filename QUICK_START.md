# 🚀 Quick Start Guide - Frontend Integration

## What Just Happened?

✅ **Mini Message Component** - Now fully draggable!  
✅ **Integration Plan** - 500+ line comprehensive roadmap created  
✅ **All Backend APIs** - Mapped and documented (60+ endpoints)

---

## 🎯 Start Here (5 Minutes)

### 1. Install Pinia
```bash
cd frontend/hoshi-vue
npm install pinia
```

### 2. Create Environment File
**File:** `frontend/hoshi-vue/.env`
```env
VITE_API_URL=http://localhost:8000
VITE_WS_URL=ws://localhost:9004
```

### 3. Update main.ts
**File:** `frontend/hoshi-vue/src/main.ts`
```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/global.scss'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.mount('#app')
```

---

## 📋 Next: Implement Phase 1 (Day 1)

### Create Auth Store
**File:** `frontend/hoshi-vue/src/stores/auth.ts`

```typescript
import { defineStore } from 'pinia'
import { authAPI } from '@/services/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as any,
    token: localStorage.getItem('jwt_token'),
    isAuthenticated: !!localStorage.getItem('jwt_token')
  }),

  actions: {
    async login(credentials: { email_or_username: string; password: string }) {
      try {
        const response = await authAPI.login(credentials)
        
        if (response.is_2fa_required) {
          return { requires2FA: true, email: credentials.email_or_username }
        }

        this.setAuth(response.access_token!, response)
        return { success: true }
      } catch (error) {
        throw error
      }
    },

    setAuth(token: string, user: any) {
      this.token = token
      this.user = user
      this.isAuthenticated = true
      localStorage.setItem('jwt_token', token)
      localStorage.setItem('user', JSON.stringify(user))
    },

    logout() {
      this.token = null
      this.user = null
      this.isAuthenticated = false
      localStorage.clear()
    }
  }
})
```

### Update Login Page
**File:** `frontend/hoshi-vue/src/pages/Login.vue`

Add at the top of `<script setup>`:
```typescript
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const result = await authStore.login({
      email_or_username: email.value,
      password: password.value
    })

    if (result.requires2FA) {
      // Redirect to 2FA page
      router.push({ 
        name: 'LoginOTP', 
        query: { email: result.email } 
      })
    } else {
      // Redirect to feed
      router.push('/feed')
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Login failed'
  } finally {
    loading.value = false
  }
}
```

---

## 📚 Three Important Documents

1. **FRONTEND_BACKEND_INTEGRATION_PLAN.md** ← THE MAIN GUIDE
   - All 8 phases explained
   - Every API endpoint mapped
   - Complete testing strategy

2. **INTEGRATION_SUMMARY.md**
   - Quick overview of what was done
   - Next steps
   - Troubleshooting

3. **This file (QUICK_START.md)**
   - Get started in 5 minutes
   - Essential code snippets

---

## 🎨 Test the Draggable Mini Message

1. Start the dev server:
```bash
npm run dev
```

2. Login and navigate to any page

3. Try dragging the mini message component:
   - Should move anywhere on the page
   - Position saves when you release
   - Click (without drag) opens Messages page
   - Position persists across page refreshes

---

## ✅ Validation Checklist (Phase 1)

After implementing Phase 1, verify:

- [ ] Pinia installed and configured
- [ ] Auth store created
- [ ] Login uses auth store
- [ ] Token saves to localStorage
- [ ] Protected routes work
- [ ] Can logout
- [ ] Mini message is draggable
- [ ] Profile page shows user data (coming next!)

---

## 🔗 Useful Commands

```bash
# Install dependencies
npm install

# Run dev server
npm run dev

# Build for production
npm run build

# Type check
npm run type-check

# Check for errors
npm run lint
```

---

## 📡 Backend Services

Make sure these are running:
```bash
docker-compose up -d
```

Check they're healthy:
- API Gateway: http://localhost:8000/health
- Frontend: http://localhost:5173

---

## 🆘 If Something Goes Wrong

### Frontend won't start?
```bash
npm install
npm run dev
```

### Backend not responding?
```bash
docker-compose restart
```

### CORS errors?
- Backend already has CORS enabled
- Check your .env VITE_API_URL

### Mini message not draggable?
- Clear localStorage: `localStorage.clear()`
- Hard refresh: Ctrl + Shift + R

---

## 🎯 Your Mission

**Today:** Set up Pinia and auth store  
**This Week:** Complete Phase 1 (Auth & Profile)  
**Next 8 Weeks:** Complete all 8 phases

Follow the integration plan phase by phase. Don't skip ahead!

---

## 🎉 You're Ready!

Everything is set up. The plan is clear. The backend is working.

**Now go build something amazing! 🚀**

---

*For detailed implementation, see: FRONTEND_BACKEND_INTEGRATION_PLAN.md*
