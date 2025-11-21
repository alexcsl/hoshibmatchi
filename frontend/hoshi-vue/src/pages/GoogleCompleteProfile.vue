<template>
  <div class="complete-profile-container">
    <div class="complete-profile-card">
      <div class="logo">
        <img src="../../public/instagram-logo.png" alt="Hoshi Logo" />
      </div>
      
      <h2>Complete Your Profile</h2>
      <p class="subtitle">Welcome! Let's finish setting up your account</p>

      <div class="google-info">
        <img :src="googleData.picture" alt="Profile" class="google-avatar" />
        <div class="google-details">
          <p class="google-name">{{ googleData.name }}</p>
          <p class="google-email">{{ googleData.email }}</p>
        </div>
      </div>

      <form @submit.prevent="handleSubmit" class="form">
        <div class="form-group">
          <label for="username">Username</label>
          <input
            id="username"
            v-model="formData.username"
            type="text"
            placeholder="Choose a unique username"
            required
            :class="{ 'error': errors.username }"
            @input="validateUsername"
          />
          <span v-if="errors.username" class="error-message">{{ errors.username }}</span>
        </div>

        <div class="form-group">
          <label for="dob">Date of Birth</label>
          <input
            id="dob"
            v-model="formData.dateOfBirth"
            type="date"
            required
            :max="maxDate"
            :class="{ 'error': errors.dateOfBirth }"
          />
          <span v-if="errors.dateOfBirth" class="error-message">{{ errors.dateOfBirth }}</span>
        </div>

        <div class="form-group">
          <label>Gender</label>
          <div class="gender-options">
            <label class="gender-option">
              <input
                v-model="formData.gender"
                type="radio"
                value="male"
                required
              />
              <span>Male</span>
            </label>
            <label class="gender-option">
              <input
                v-model="formData.gender"
                type="radio"
                value="female"
                required
              />
              <span>Female</span>
            </label>
            <label class="gender-option">
              <input
                v-model="formData.gender"
                type="radio"
                value="other"
                required
              />
              <span>Other</span>
            </label>
          </div>
        </div>

        <button
          type="submit"
          class="submit-btn"
          :disabled="loading || !isFormValid"
        >
          {{ loading ? 'Completing...' : 'Complete Profile' }}
        </button>

        <p v-if="errorMessage" class="error-banner">{{ errorMessage }}</p>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import apiClient, { handleApiError } from '../services/api'

const router = useRouter()
const route = useRoute()

const googleData = ref({
  name: '',
  email: '',
  picture: ''
})

const formData = ref({
  username: '',
  dateOfBirth: '',
  gender: ''
})

const errors = ref({
  username: '',
  dateOfBirth: ''
})

const loading = ref(false)
const errorMessage = ref('')

// Calculate max date (must be at least 13 years old)
const maxDate = computed(() => {
  const date = new Date()
  date.setFullYear(date.getFullYear() - 13)
  return date.toISOString().split('T')[0]
})

const isFormValid = computed(() => {
  return formData.value.username.length >= 3 &&
         formData.value.dateOfBirth &&
         formData.value.gender &&
         !errors.value.username
})

onMounted(() => {
  // Get Google data from query params
  googleData.value.name = route.query.name as string || ''
  googleData.value.email = route.query.email as string || ''
  googleData.value.picture = route.query.picture as string || ''

  // Pre-fill username from email (user can change it)
  if (googleData.value.email) {
    formData.value.username = googleData.value.email.split('@')[0]
    validateUsername()
  }

  // If no Google data, redirect back to login
  if (!googleData.value.email) {
    router.push('/login')
  }
})

const validateUsername = async () => {
  const username = formData.value.username.trim()
  
  if (username.length < 3) {
    errors.value.username = 'Username must be at least 3 characters'
    return
  }
  
  if (!/^[a-zA-Z0-9._]+$/.test(username)) {
    errors.value.username = 'Username can only contain letters, numbers, dots, and underscores'
    return
  }

  // Check username availability (debounced check)
  try {
    const response = await apiClient.get(`/auth/check-username/${username}`)
    if (response.data.exists) {
      errors.value.username = 'Username is already taken'
    } else {
      errors.value.username = ''
    }
  } catch (err) {
    // If endpoint doesn't exist, skip validation
    errors.value.username = ''
  }
}

const handleSubmit = async () => {
  if (!isFormValid.value) return

  loading.value = true
  errorMessage.value = ''

  try {
    // Update user profile with the additional information
    await apiClient.put('/users/complete-profile', {
      username: formData.value.username,
      date_of_birth: formData.value.dateOfBirth,
      gender: formData.value.gender
    })

    // Update stored user data
    const storedUser = localStorage.getItem('user')
    if (storedUser) {
      const userData = JSON.parse(storedUser)
      userData.username = formData.value.username
      localStorage.setItem('user', JSON.stringify(userData))
    }

    // Redirect to feed
    router.push('/feed')
  } catch (err) {
    errorMessage.value = handleApiError(err)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.complete-profile-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #000;
  padding: 20px;
}

.complete-profile-card {
  background-color: #000;
  border: 1px solid #262626;
  border-radius: 8px;
  padding: 40px;
  width: 100%;
  max-width: 450px;
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.5);
}

.logo {
  text-align: center;
  margin-bottom: 30px;
}

.logo img {
  height: 60px;
}

h2 {
  color: #fff;
  font-size: 24px;
  font-weight: 600;
  text-align: center;
  margin: 0 0 8px 0;
}

.subtitle {
  color: #8e8e8e;
  font-size: 14px;
  text-align: center;
  margin: 0 0 30px 0;
}

.google-info {
  display: flex;
  align-items: center;
  gap: 15px;
  padding: 15px;
  background-color: #0a0a0a;
  border: 1px solid #262626;
  border-radius: 8px;
  margin-bottom: 30px;
}

.google-avatar {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  object-fit: cover;
}

.google-details {
  flex: 1;
}

.google-name {
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 4px 0;
}

.google-email {
  color: #8e8e8e;
  font-size: 13px;
  margin: 0;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
}

label {
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 8px;
}

input[type="text"],
input[type="date"] {
  background-color: #0a0a0a;
  border: 1px solid #262626;
  border-radius: 6px;
  color: #fff;
  padding: 12px 15px;
  font-size: 14px;
  transition: border-color 0.2s;
}

input[type="text"]:focus,
input[type="date"]:focus {
  outline: none;
  border-color: #0a66c2;
}

input.error {
  border-color: #ed4956;
}

.error-message {
  color: #ed4956;
  font-size: 12px;
  margin-top: 6px;
}

.gender-options {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.gender-option {
  flex: 1;
  min-width: 100px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 15px;
  background-color: #0a0a0a;
  border: 1px solid #262626;
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 0.2s;
}

.gender-option:hover {
  border-color: #404040;
}

.gender-option input[type="radio"] {
  margin: 0;
  cursor: pointer;
}

.gender-option span {
  color: #fff;
  font-size: 14px;
}

.submit-btn {
  background-color: #0a66c2;
  color: #fff;
  border: none;
  border-radius: 6px;
  padding: 14px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-top: 10px;
}

.submit-btn:hover:not(:disabled) {
  background-color: #004182;
}

.submit-btn:disabled {
  background-color: #004182;
  opacity: 0.5;
  cursor: not-allowed;
}

.error-banner {
  background-color: rgba(237, 73, 86, 0.1);
  border: 1px solid #ed4956;
  border-radius: 6px;
  color: #ed4956;
  padding: 12px;
  font-size: 13px;
  text-align: center;
  margin: 0;
}

@media (max-width: 480px) {
  .complete-profile-card {
    padding: 30px 20px;
  }

  .gender-options {
    flex-direction: column;
  }

  .gender-option {
    min-width: 100%;
  }
}
</style>
