<template>
  <div class="verify-otp-container">
    <div class="verify-otp-box">
      <div class="logo">Hoshibmatchi</div>
      
      <ErrorAlert v-if="error" :message="error" @close="error = ''" />
      
      <div v-if="success" class="success-message">
        <p>✓ Account verified successfully!</p>
        <p>Redirecting to login...</p>
      </div>
      
      <div v-else class="verify-content">
        <h2>Verify Your Email</h2>
        <p class="instruction">Enter the 6-digit code sent to<br><strong>{{ email }}</strong></p>
        
        <form @submit.prevent="handleSubmit">
          <OTPInput v-model="otp" :length="6" />
          
          <button type="submit" class="verify-btn" :disabled="authStore.loading || otp.length !== 6">
            {{ authStore.loading ? 'Verifying...' : 'Verify Account' }}
          </button>
        </form>
        
        <div class="resend-section">
          <button 
            v-if="canResend" 
            @click="handleResend" 
            :disabled="resending"
            class="resend-btn"
          >
            {{ resending ? 'Sending...' : 'Resend Code' }}
          </button>
          <p v-else class="resend-timer">
            Resend code in {{ resendTimer }}s
          </p>
        </div>
        
        <p class="back-link">
          <router-link to="/login">Back to Login</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import OTPInput from '../components/OTPInput.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const otp = ref('')
const resending = ref(false)
const error = ref('')
const success = ref(false)
const canResend = ref(false)
const resendTimer = ref(60)

onMounted(() => {
  // Get email from query params
  email.value = (route.query.email as string) || ''
  
  if (!email.value) {
    error.value = 'Email not provided. Please register again.'
    setTimeout(() => router.push('/signup'), 3000)
    return
  }
  
  // Start resend timer
  startResendTimer()
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

const handleSubmit = async () => {
  if (otp.value.length !== 6) {
    error.value = 'Please enter a valid 6-digit code'
    return
  }
  
  error.value = ''
  
  try {
    // Call verify-otp endpoint
    await authStore.verifyRegistrationOTP({
      email: email.value,
      otp_code: otp.value
    })
    
    success.value = true
    
    // Redirect to login after 2 seconds
    setTimeout(() => {
      router.push({
        path: '/login',
        query: { verified: 'true', email: email.value }
      })
    }, 2000)
  } catch (err: any) {
    error.value = err?.message || 'Verification failed. Please try again.'
  }
}

const handleResend = async () => {
  resending.value = true
  error.value = ''
  
  try {
    await authStore.requestOTP({ email: email.value })
    startResendTimer()
  } catch (err: any) {
    error.value = err?.message || 'Failed to resend code.'
  } finally {
    resending.value = false
  }
}
</script>

<style scoped lang="scss">
.verify-otp-container {
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
  padding: 20px;
}

.verify-otp-box {
  background-color: #fff;
  padding: 40px;
  border-radius: 8px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.logo {
  font-family: 'Branding', cursive;
  font-size: 48px;
  margin-bottom: 30px;
  color: #262626;
}

.verify-content {
  h2 {
    font-size: 24px;
    font-weight: 600;
    color: #262626;
    margin-bottom: 10px;
  }
  
  .instruction {
    color: #8e8e8e;
    font-size: 14px;
    margin-bottom: 30px;
    line-height: 1.5;
    
    strong {
      color: #262626;
      font-weight: 600;
    }
  }
}

form {
  margin-bottom: 20px;
}

.verify-btn {
  width: 100%;
  padding: 12px;
  background-color: #0095f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 20px;
  transition: background-color 0.2s;
  
  &:hover:not(:disabled) {
    background-color: #1877f2;
  }
  
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.resend-section {
  margin: 20px 0;
  
  .resend-btn {
    background: none;
    border: none;
    color: #0095f6;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    
    &:hover:not(:disabled) {
      color: #1877f2;
    }
    
    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }
  
  .resend-timer {
    color: #8e8e8e;
    font-size: 14px;
  }
}

.success-message {
  padding: 30px;
  color: #00a400;
  
  p {
    margin: 10px 0;
    
    &:first-child {
      font-size: 18px;
      font-weight: 600;
    }
    
    &:last-child {
      font-size: 14px;
      color: #8e8e8e;
    }
  }
}

.back-link {
  margin-top: 20px;
  font-size: 14px;
  color: #8e8e8e;
  
  a {
    color: #0095f6;
    text-decoration: none;
    font-weight: 600;
    
    &:hover {
      color: #1877f2;
    }
  }
}
</style>
