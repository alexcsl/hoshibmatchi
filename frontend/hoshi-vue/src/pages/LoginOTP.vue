<template>
  <div class="login-otp-container">
    <div class="otp-card">
      <h1 class="logo">Instagram</h1>

      <div class="content">
        <h2 class="title">Two-Factor Authentication</h2>
        <p class="subtitle">Enter the 6-digit code sent to your email</p>

        <!-- Error Alert -->
        <ErrorAlert v-if="error" :message="error" @close="error = ''" />

        <!-- OTP Input -->
        <form @submit.prevent="handleSubmit" class="otp-form">
          <OTPInput v-model="form.otp" :errorMessage="errors.otp" />

          <button type="submit" class="verify-btn" :disabled="authStore.loading">
            {{ authStore.loading ? 'Verifying...' : 'Verify' }}
          </button>
        </form>

        <!-- Resend Code -->
        <p class="resend-text">
          Didn't receive the code?
          <button @click="handleResendCode" class="resend-btn" :disabled="resendDisabled">
            {{ resendDisabled ? `Resend in ${resendCountdown}s` : 'Resend' }}
          </button>
        </p>

        <!-- Back to Login -->
        <router-link to="/login" class="back-link">Back to login</router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import OTPInput from '../components/OTPInput.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const error = ref('')
const resendDisabled = ref(false)
const resendCountdown = ref(0)
let countdownInterval: number

const form = reactive({
  otp: ''
})

const errors = reactive({
  otp: ''
})

const handleResendCode = async () => {
  // Get email from sessionStorage
  const email = sessionStorage.getItem('temp_email')
  if (!email) {
    error.value = 'Session expired. Please login again.'
    router.push('/login')
    return
  }

  resendDisabled.value = true
  resendCountdown.value = 60
  error.value = ''

  try {
    // Request new OTP code
    // Note: Backend should send new OTP to user's email
    console.log('Requesting new OTP for email:', email)
  } catch (err: any) {
    error.value = err?.message || 'Failed to resend code'
  }

  countdownInterval = setInterval(() => {
    resendCountdown.value--
    if (resendCountdown.value <= 0) {
      clearInterval(countdownInterval)
      resendDisabled.value = false
    }
  }, 1000)
}

const handleSubmit = async () => {
  errors.otp = ''
  error.value = ''

  if (form.otp.length !== 6) {
    errors.otp = 'Please enter a valid 6-digit code'
    return
  }

  // Get email from sessionStorage
  const email = sessionStorage.getItem('temp_email')
  if (!email) {
    error.value = 'Session expired. Please login again.'
    router.push('/login')
    return
  }

  try {
    await authStore.verify2FA({
      email: email,
      otp_code: form.otp
    })

    // Clear temporary session data
    sessionStorage.removeItem('temp_email')
    sessionStorage.removeItem('temp_username')

    // Redirect to feed
    router.push('/feed')
  } catch (err: any) {
    error.value = err?.message || 'Verification failed. Please try again.'
  }
}

onMounted(() => {
  // Check if email exists in sessionStorage
  const email = sessionStorage.getItem('temp_email')
  if (!email) {
    error.value = 'Please login first'
    router.push('/login')
    return
  }

  // Initialize 60-second rate limit
  resendCountdown.value = 60
  resendDisabled.value = true

  countdownInterval = setInterval(() => {
    resendCountdown.value--
    if (resendCountdown.value <= 0) {
      clearInterval(countdownInterval)
      resendDisabled.value = false
    }
  }, 1000)
})

onUnmounted(() => {
  if (countdownInterval) {
    clearInterval(countdownInterval)
  }
})

</script>

<style scoped lang="scss">
@import '../styles/fonts.css';
.login-otp-container {
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
  padding: 20px;
}

.otp-card {
  width: 100%;
  max-width: 350px;
  padding: 40px;
  border: 1px solid #262626;
  border-radius: 1px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.logo {
  font-family: 'Branding', cursive;
  font-size: 48px;
  font-weight: 300;
  text-align: center;
}

.content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.title {
  font-size: 18px;
  font-weight: 600;
  text-align: center;
}

.subtitle {
  font-size: 14px;
  color: #a8a8a8;
  text-align: center;
}

.otp-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 8px;
}

.verify-btn {
  width: 100%;
  padding: 10px;
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

.resend-text {
  font-size: 13px;
  color: #a8a8a8;
  text-align: center;

  .resend-btn {
    background: none;
    border: none;
    color: #0a66c2;
    cursor: pointer;
    font-weight: 600;
    text-decoration: none;
    padding: 0;

    &:hover:not(:disabled) {
      text-decoration: underline;
    }

    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  }
}

.back-link {
  text-align: center;
  font-size: 13px;
  color: #0a66c2;
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}
</style>
