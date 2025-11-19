<template>
  <div class="signup-container">
    <div class="signup-wrapper">
      <!-- Left side: Carousel/Image -->
      <div class="carousel-section">
        <img src="/instagram-signup-hero.png" alt="Instagram" />
      </div>

      <!-- Right side: Form -->
      <div class="form-section">
        <div class="form-content">
          <!-- Logo -->
          <h1 class="logo">Instagram</h1>

          <!-- Main heading -->
          <p class="subtitle">Sign up to see photos and videos from your friends.</p>

          <!-- Google OAuth Button -->
          <button @click="handleGoogleSignup" class="google-btn">
            <img src="/google-icon.svg" alt="Google" class="google-icon" />
            Sign up with Google
          </button>

          <!-- Divider -->
          <div class="divider">
            <span>OR</span>
          </div>

          <!-- Error Alert -->
          <ErrorAlert v-if="error" :message="error" @close="error = ''" />

          <!-- Form -->
          <form @submit.prevent="handleSubmit" class="form">
            <FormInput
              v-model="form.name"
              type="text"
              placeholder="Full name"
              :errorMessage="errors.name"
            />

            <FormInput
              v-model="form.username"
              type="text"
              placeholder="Username"
              :errorMessage="errors.username"
            />

            <FormInput
              v-model="form.email"
              type="email"
              placeholder="Email address"
              :errorMessage="errors.email"
            />

            <FormInput
              v-model="form.password"
              type="password"
              placeholder="Password"
              :errorMessage="errors.password"
            />

            <PasswordStrengthValidator :password="form.password" />

            <FormInput
              v-model="form.confirmPassword"
              type="password"
              placeholder="Confirm password"
              :errorMessage="errors.confirmPassword"
            />

            <!-- Gender Select -->
            <div class="form-group">
              <select v-model="form.gender" class="select" :class="{ error: errors.gender }">
                <option value="">Select gender</option>
                <option value="male">Male</option>
                <option value="female">Female</option>
              </select>
              <p v-if="errors.gender" class="error-message">{{ errors.gender }}</p>
            </div>

            <!-- Date of Birth -->
            <div class="form-group">
              <input
                v-model="form.dob"
                type="date"
                class="input"
                :class="{ error: errors.dob }"
              />
              <p v-if="errors.dob" class="error-message">{{ errors.dob }}</p>
            </div>

            <!-- Profile Picture -->
            <div class="form-group">
              <label class="file-input-label">
                <input
                  type="file"
                  accept="image/*"
                  @change="handleProfilePictureChange"
                  class="file-input"
                />
                <span class="file-input-text">{{ profilePictureLabel }}</span>
              </label>
              <p v-if="errors.profilePicture" class="error-message">{{ errors.profilePicture }}</p>
            </div>

            <!-- Checkboxes -->
            <div class="checkbox-group">
              <label class="checkbox">
                <input v-model="form.newsletter" type="checkbox" />
                <span>Subscribe to our newsletter</span>
              </label>
              <label class="checkbox">
                <input v-model="form.twoFA" type="checkbox" />
                <span>Enable Two-Factor Authentication</span>
              </label>
            </div>

            <!-- Submit Button -->
            <button type="submit" class="submit-btn" :disabled="authStore.loading">
              {{ authStore.loading ? 'Creating account...' : 'Sign up' }}
            </button>
          </form>

          <!-- Login Link -->
          <p class="login-link">
            Have an account? <router-link to="/login">Log in</router-link>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import FormInput from '../components/FormInput.vue'
import PasswordStrengthValidator from '../components/PasswordStrengthValidator.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const error = ref('')
const profilePictureLabel = ref('Choose profile picture')

const form = reactive({
  name: '',
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  gender: '',
  dob: '',
  profilePicture: null as File | null,
  newsletter: false,
  twoFA: false
})

const errors = reactive({
  name: '',
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  gender: '',
  dob: '',
  profilePicture: ''
})



const handleProfilePictureChange = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.[0]) {
    form.profilePicture = input.files[0]
    profilePictureLabel.value = input.files[0].name
    errors.profilePicture = ''
  }
}

const handleGoogleSignup = () => {
  // Initialize Google OAuth flow
  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID
  
  if (!googleClientId) {
    error.value = 'Google OAuth is not configured'
    return
  }
  
  // Redirect to Google OAuth using authorization code flow
  const redirectUri = `${window.location.origin}/auth/google/callback`
  const scope = 'openid profile email'
  const googleAuthUrl = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${googleClientId}&redirect_uri=${redirectUri}&response_type=code&scope=${scope}&access_type=offline&prompt=consent`
  
  window.location.href = googleAuthUrl
}



const handleSubmit = async () => {
  // Clear previous errors
  Object.keys(errors).forEach(key => {
    errors[key as keyof typeof errors] = ''
  })
  error.value = ''

  // Frontend validations
  if (form.name.length <= 4 || /[0-9!@#$%^&*(),.?":{}|<>]/.test(form.name)) {
    errors.name = 'Name must be more than 4 characters with no symbols or numbers'
    return
  }

  if (form.username.length < 3 || form.username.length > 30) {
    errors.username = 'Username must be 3-30 characters'
    return
  }

  if (!/^[a-zA-Z0-9_]+$/.test(form.username)) {
    errors.username = 'Username can only contain letters, numbers, and underscores'
    return
  }

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(form.email)) {
    errors.email = 'Please enter a valid email address'
    return
  }

  if (form.password.length < 8 || !/[A-Z]/.test(form.password) || !/[a-z]/.test(form.password) || !/[0-9]/.test(form.password) || !/[!@#$%^&*(),.?":{}|<>]/.test(form.password)) {
    errors.password = 'Password must have at least 8 characters, 1 uppercase, 1 lowercase, 1 number, and 1 special character'
    return
  }

  if (form.password !== form.confirmPassword) {
    errors.confirmPassword = 'Passwords do not match'
    return
  }

  if (!form.gender || (form.gender !== 'male' && form.gender !== 'female')) {
    errors.gender = 'Please select a valid gender'
    return
  }

  if (!form.dob) {
    errors.dob = 'Please select your date of birth'
    return
  }

  // Age check (13+)
  const today = new Date()
  const birthDate = new Date(form.dob)
  let age = today.getFullYear() - birthDate.getFullYear()
  const monthDiff = today.getMonth() - birthDate.getMonth()
  if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birthDate.getDate())) {
    age--
  }
  
  if (age < 13) {
    errors.dob = 'You must be at least 13 years old to sign up'
    return
  }

  try {
    // Register user
    await authStore.register({
      name: form.name,
      username: form.username,
      email: form.email,
      password: form.password,
      confirm_password: form.confirmPassword,
      gender: form.gender,
      date_of_birth: form.dob,
      enable_2fa: form.twoFA
    })

    // Redirect to OTP verification page
    router.push({
      path: '/verify-otp',
      query: { email: form.email }
    })
  } catch (err: any) {
    error.value = err?.message || 'Registration failed. Please try again.'
  }
}
</script>

<style scoped lang="scss">
.signup-container {
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
  padding: 20px;
}

.signup-wrapper {
  display: flex;
  width: 100%;
  max-width: 900px;
  gap: 40px;

  @media (max-width: 768px) {
    flex-direction: column;
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
    border-radius: 8px;
  }

  @media (max-width: 768px) {
    display: none;
  }
}

.form-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.form-content {
  width: 100%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.logo {
  font-family: 'Branding', cursive;
  font-size: 48px;
  font-weight: 300;
  text-align: center;
  margin-bottom: 8px;
}

.subtitle {
  font-size: 15px;
  color: #a8a8a8;
  text-align: center;
  margin-bottom: 8px;
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

.divider {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #a8a8a8;
  font-size: 13px;

  &::before,
  &::after {
    content: '';
    flex: 1;
    height: 1px;
    background-color: #404040;
  }
}

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;

  label {
    font-size: 13px;
    color: #a8a8a8;
  }



  .select,
  .input {
    padding: 10px 16px;
    background-color: #262626;
    border: 1px solid #404040;
    border-radius: 5px;
    color: #fff;
    font-size: 14px;
    outline: none;
    transition: all 0.2s;

    &:focus {
      border-color: #818384;
      background-color: #1a1a1a;
    }

    &.error {
      border-color: #f52424;
      background-color: rgba(245, 36, 36, 0.1);
    }
  }

  .select {
    cursor: pointer;

    option {
      background-color: #262626;
      color: #fff;
    }
  }

  .error-message {
    font-size: 12px;
    color: #f52424;
  }
}

.file-input-label {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px 16px;
  background-color: #262626;
  border: 1px solid #404040;
  border-radius: 5px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: #818384;
    background-color: #1a1a1a;
  }

  .file-input {
    display: none;
  }

  .file-input-text {
    font-size: 14px;
    color: #a8a8a8;
  }
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 10px;

  .checkbox {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 13px;
    color: #a8a8a8;

    input {
      width: 18px;
      height: 18px;
      cursor: pointer;
      accent-color: #0a66c2;
    }
  }
}

.submit-btn {
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

.login-link {
  text-align: center;
  font-size: 13px;
  color: #a8a8a8;

  a {
    color: #0a66c2;
    text-decoration: none;
    font-weight: 600;

    &:hover {
      text-decoration: underline;
    }
  }
}
</style>
