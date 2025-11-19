<template>
  <div class="forgot-password-container">
    <div class="forgot-card">
      <h1 class="logo">Instagram</h1>

      <div class="content">
        <h2 class="title">Trouble logging in?</h2>
        <p class="subtitle">Enter your email and we'll send you a code to reset your password.</p>

        <!-- Error Alert -->
        <ErrorAlert v-if="error" :message="error" @close="error = ''" />

        <!-- Success Message -->
        <div v-if="otpSent" class="success-message">
          Code sent to {{ form.email }}! Check your email and enter the code below.
        </div>

        <!-- Form -->
        <form @submit.prevent="handleSubmit" class="form">
          <FormInput
            v-model="form.email"
            type="email"
            placeholder="Email address"
            :errorMessage="errors.email"
            :disabled="otpSent"
          />

          <!-- OTP Input (shown after email is sent) -->
          <div v-if="otpSent" class="otp-section">
            <label>6-Digit Code</label>
            <OTPInput v-model="form.otp" :errorMessage="errors.otp" />
          </div>

          <button type="submit" class="submit-btn" :disabled="loading">
            {{ loading ? 'Sending...' : otpSent ? 'Verify Code' : 'Send Reset Code' }}
          </button>
        </form>

        <!-- Resend Code (shown after OTP is sent) -->
        <p v-if="otpSent" class="resend-text">
          Didn't receive the code?
          <button @click="handleResend" class="resend-btn" :disabled="!canResend">
            {{ canResend ? 'Resend' : `Resend in ${resendTimer}s` }}
          </button>
        </p>

        <!-- Links -->
        <div class="links">
          <router-link to="/signup" class="link">Create new account</router-link>
          <router-link to="/login" class="link">Back to login</router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import FormInput from '../components/FormInput.vue'
import OTPInput from '../components/OTPInput.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import { authAPI, handleApiError } from '../services/api'

const router = useRouter()
const loading = ref(false)
const error = ref('')
const otpSent = ref(false)
const canResend = ref(true)
const resendTimer = ref(60)

const form = reactive({
  email: '',
  otp: ''
})

const errors = reactive({
  email: '',
  otp: ''
})

const startResendTimer = () => {
  canResend.value = false
  resendTimer.value = 60

  const interval = setInterval(() => {
    resendTimer.value--
    if (resendTimer.value === 0) {
      canResend.value = true
      clearInterval(interval)
    }
  }, 1000)
}

const handleResend = async () => {
  error.value = ''
  loading.value = true

  try {
    await authAPI.forgotPassword({ email: form.email })
    startResendTimer()
  } catch (err) {
    error.value = handleApiError(err)
  } finally {
    loading.value = false
  }
}

const handleSubmit = async () => {
  errors.email = ''
  errors.otp = ''
  error.value = ''

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(form.email)) {
    errors.email = 'Please enter a valid email address'
    return
  }

  if (otpSent.value) {
    // Verify OTP and proceed to reset password
    if (form.otp.length !== 6) {
      errors.otp = 'Please enter a valid 6-digit code'
      return
    }

    loading.value = true

    try {
      // Verify OTP by attempting to use it
      // We'll pass the email and OTP to the reset password page
      router.push({
        path: '/reset-password',
        query: {
          email: form.email,
          otp: form.otp
        }
      })
    } catch (err) {
      error.value = handleApiError(err)
    } finally {
      loading.value = false
    }
  } else {
    // Send OTP
    loading.value = true

    try {
      await authAPI.forgotPassword({ email: form.email })
      otpSent.value = true
      startResendTimer()
    } catch (err) {
      error.value = handleApiError(err)
    } finally {
      loading.value = false
    }
  }
}
</script>

<style scoped lang="scss">
.forgot-password-container {
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
  padding: 20px;
}

.forgot-card {
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
  line-height: 1.5;
}

.success-message {
  padding: 12px 16px;
  background-color: rgba(74, 222, 128, 0.1);
  border: 1px solid #4ade80;
  border-radius: 5px;
  color: #4ade80;
  font-size: 13px;
  text-align: center;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
}

.otp-section {
  display: flex;
  flex-direction: column;
  gap: 8px;

  label {
    font-size: 13px;
    color: #a8a8a8;
  }
}

.submit-btn {
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

.links {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
  padding-top: 16px;
  border-top: 1px solid #262626;
}

.link {
  text-align: center;
  font-size: 13px;
  color: #0a66c2;
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}
</style>
