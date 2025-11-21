<template>
  <div class="login-container">
    <div class="login-wrapper">
      <!-- Left side: Carousel -->
      <div class="carousel-section">
        <img src="/instagram-phones.png" alt="Instagram phones" />
      </div>

      <!-- Right side: Form -->
      <div class="form-section">
        <div class="form-content">
          <!-- Logo -->
          <h1 class="logo">hoshiBmaTchi</h1>

          <!-- Success Message -->
          <div v-if="successMessage" class="success-alert">
            {{ successMessage }}
          </div>

          <!-- Error Alert -->
          <ErrorAlert v-if="error" :message="error" @close="error = ''" />
          
          <!-- Verification Link -->
          <div v-if="showVerificationLink" class="verification-prompt">
            <router-link 
              :to="{ path: '/verify-otp', query: { email: userEmailForVerification }}" 
              class="verify-link"
            >
              Click here to verify your account
            </router-link>
          </div>

          <!-- Form -->
          <form @submit.prevent="handleSubmit" class="form">
            <FormInput
              v-model="form.username"
              type="text"
              placeholder="Phone number, username, or email"
              :errorMessage="errors.username"
            />

            <FormInput
              v-model="form.password"
              type="password"
              placeholder="Password"
              :errorMessage="errors.password"
            />

            <button type="submit" class="login-btn" :disabled="authStore.loading">
              {{ authStore.loading ? 'Logging in...' : 'Log in' }}
            </button>
          </form>

          <!-- Divider -->
          <div class="divider">
            <span>OR</span>
          </div>

          <!-- Google OAuth Button -->
          <button @click="handleGoogleLogin" class="google-btn">
            <img src="/google-icon.svg" alt="Google" class="google-icon" />
            Log in with Google
          </button>

          <!-- Forgot Password Link -->
          <router-link to="/forgot-password" class="forgot-password">Forgot password?</router-link>

          <!-- Divider -->
          <div class="signup-divider"></div>

          <!-- Sign Up Link -->
          <p class="signup-text">
            Don't have an account? <router-link to="/signup">Sign up</router-link>
          </p>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <footer class="footer">
      <div class="footer-links">
        <a href="#">Meta</a>
        <a href="#">About</a>
        <a href="#">Blog</a>
        <a href="#">Jobs</a>
        <a href="#">Help</a>
        <a href="#">API</a>
        <a href="#">Privacy</a>
        <a href="#">Terms</a>
        <a href="#">Locations</a>
        <a href="#">Instagram Lite</a>
        <a href="#">Meta AI</a>
        <a href="#">Threads</a>
      </div>
      <p class="copyright">© 2025 Instagram from Meta</p>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import FormInput from '../components/FormInput.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const error = ref('')
const successMessage = ref('')
const showVerificationLink = ref(false)
const userEmailForVerification = ref('')

const form = reactive({
  username: '',
  password: ''
})

const errors = reactive({
  username: '',
  password: ''
})

onMounted(() => {
  // Check if user just registered
  if (route.query.registered === 'true') {
    successMessage.value = 'Registration successful! Please log in with your credentials.'
    if (route.query.email) {
      form.username = route.query.email as string
    }
  }
  
  // Check if user just verified email
  if (route.query.verified === 'true') {
    successMessage.value = 'Email verified successfully! Please log in with your credentials.'
    if (route.query.email) {
      form.username = route.query.email as string
    }
  }
  
  // Check if password was reset
  if (route.query.reset === 'success') {
    successMessage.value = 'Password reset successful! Please log in with your new password.'
  }
})

const handleGoogleLogin = () => {
  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID
  
  if (!googleClientId) {
    error.value = 'Google OAuth is not configured'
    return
  }
  
  const redirectUri = `${window.location.origin}/auth/google/callback`
  const scope = 'openid profile email'
  const googleAuthUrl = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${googleClientId}&redirect_uri=${redirectUri}&response_type=code&scope=${scope}&access_type=offline&prompt=consent`
  
  window.location.href = googleAuthUrl
}

const handleSubmit = async () => {
  // Clear previous errors
  errors.username = ''
  errors.password = ''
  error.value = ''
  successMessage.value = ''

  // Frontend validations
  if (!form.username.trim()) {
    errors.username = 'Username, email, or phone number is required'
    return
  }

  if (!form.password.trim()) {
    errors.password = 'Password is required'
    return
  }

  try {
    const response = await authStore.login({
      email_or_username: form.username,
      password: form.password
    })
    
    // Check if 2FA is required
    if (response.requires_2fa) {
      // Store temporary session data for 2FA
      const userEmail: string = (response as any).email || (response as any).username || form.username
      const userName: string = (response as any).username || form.username
      
      sessionStorage.setItem('temp_email', userEmail)
      sessionStorage.setItem('temp_username', userName)
      
      // Redirect to OTP verification
      router.push('/login-otp')
    } else {
      // Login successful, redirect to feed
      router.push('/feed')
    }
  } catch (err: any) {
    // Check if error is due to inactive account
    const errorMessage = err?.message || String(err)
    
    if (errorMessage.toLowerCase().includes('deactivated') || 
        errorMessage.toLowerCase().includes('not verified') || 
        errorMessage.toLowerCase().includes('inactive') ||
        errorMessage.toLowerCase().includes('not active')) {
      error.value = 'Your account is not verified. Please check your email for the verification code.'
      showVerificationLink.value = true
      userEmailForVerification.value = form.username
    } else {
      error.value = errorMessage
      showVerificationLink.value = false
    }
  }
}
</script>

<style scoped lang="scss">
@import '../styles/fonts.css';
.login-container {
  width: 100%;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #000;
  padding: 20px;
}

.login-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 60px;
  margin:0 auto ;

  @media (max-width: 768px) {
    gap: 20px;
  }
}

.carousel-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;

  img {
    max-width: 100%;
    height: auto;
  }

  @media (max-width: 768px) {
    display: none;
  }
}

.form-section {
  flex: 1;
  max-width: 350px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 40px 0;
  border: 1px solid #262626;
  border-radius: 1px;
  padding: 40px;
}

.logo {
  font-family: 'Branding', cursive;
  font-size: 48px;
  font-weight: 300;
  text-align: center;
  margin-bottom: 24px;
}

.success-alert {
  padding: 12px 16px;
  background-color: rgba(74, 222, 128, 0.1);
  border: 1px solid #4ade80;
  border-radius: 5px;
  color: #4ade80;
  font-size: 13px;
  text-align: center;
}

.verification-prompt {
  padding: 12px 16px;
  background-color: rgba(255, 168, 0, 0.1);
  border: 1px solid #ffa800;
  border-radius: 5px;
  text-align: center;
  font-size: 13px;

  .verify-link {
    color: #0095f6;
    text-decoration: none;
    font-weight: 600;
    transition: color 0.2s;

    &:hover {
      color: #1877f2;
    }
  }
}

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.login-btn {
  width: 100%;
  padding: 10px;
  margin-top: 8px;
  background-color: #0a66c2;
  border: none;
  border-radius: 5px;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;

  &:hover:not(:disabled) {
    background-color: #0853a1;
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.divider {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #a8a8a8;
  font-size: 13px;
  margin: 8px 0;

  &::before,
  &::after {
    content: '';
    flex: 1;
    height: 1px;
    background-color: #404040;
  }
}

.google-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  padding: 10px;
  background-color: transparent;
  border: 1px solid #404040;
  border-radius: 5px;
  color: #0a66c2;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  transition: all 0.2s;

  &:hover {
    background-color: #0a66c2;
    color: #fff;
    border-color: #0a66c2;
  }

  .google-icon {
    width: 18px;
    height: 18px;
  }
}

.forgot-password {
  text-align: center;
  font-size: 12px;
  color: #0a66c2;
  text-decoration: none;
  margin-top: 8px;

  &:hover {
    text-decoration: underline;
  }
}

.signup-divider {
  height: 1px;
  background-color: #262626;
  margin: 16px 0;
}

.signup-text {
  text-align: center;
  font-size: 14px;
  color: #a8a8a8;

  a {
    color: #0a66c2;
    font-weight: 600;
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }
}

.footer {
  padding-top: 40px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  border-top: 1px solid #262626;
}

.footer-links {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 16px;

  a {
    font-size: 12px;
    color: #a8a8a8;
    text-decoration: none;

    &:hover {
      color: #fff;
    }
  }
}

.copyright {
  font-size: 12px;
  color: #666;
}
</style>
