<!-- frontend/hoshi-vue/src/views/RegisterView.vue -->

<template>
  <div class="register-container">
    <div class="register-box">
      <h1>hoshiBmatchi</h1>
      <p class="subtitle">Sign up to see photos and videos from your friends.</p>
      
      <form @submit.prevent="handleRegister">
        <input v-model="form.name" type="text" placeholder="Full Name" required />
        <input v-model="form.username" type="text" placeholder="Username" required />
        <input v-model="form.email" type="email" placeholder="Email" required />
        <input v-model="form.password" type="password" placeholder="Password" required />
        
        <div class="form-group">
          <label for="dob">Date of Birth:</label>
          <input v-model="form.date_of_birth" id="dob" type="date" required />
        </div>

        <div class="form-group">
          <label>Gender:</label>
          <select v-model="form.gender" required>
            <option disabled value="">Please select one</option>
            <option>Male</option>
            <option>Female</option>
            <option>Other</option>
          </select>
        </div>

        <button type="submit" :disabled="isLoading">
          {{ isLoading ? 'Signing Up...' : 'Sign Up' }}
        </button>
      </form>

      <div v-if="errorMessage" class="error-message">{{ errorMessage }}</div>
      <div v-if="successMessage" class="success-message">{{ successMessage }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import axios from 'axios';

// Reactive state for our form data
const form = ref({
  name: '',
  username: '',
  email: '',
  password: '',
  date_of_birth: '', // Will be in "YYYY-MM-DD" format from the input
  gender: ''
});

// State for loading and messages
const isLoading = ref(false);
const errorMessage = ref('');
const successMessage = ref('');

// Function to handle form submission
const handleRegister = async () => {
  isLoading.value = true;
  errorMessage.value = '';
  successMessage.value = '';

  try {
    // The URL for our API Gateway. We'll make 'api.hoshi.local' work in the next step.
    const apiUrl = 'http://api.hoshi.local/auth/register';
    
    // Send the form data to the API Gateway
    const response = await axios.post(apiUrl, form.value);

    // Handle success
    successMessage.value = `Successfully registered user ${response.data.username}! You can now log in.`;
    console.log('Registration successful:', response.data);
    
  } catch (error: any) {
    // Handle errors
    console.error('Registration failed:', error);
    if (error.response) {
      errorMessage.value = error.response.data.error || 'An unknown error occurred.';
    } else {
      errorMessage.value = 'Could not connect to the server. Please try again later.';
    }
  } finally {
    isLoading.value = false;
  }
};
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #fafafa;
}
.register-box {
  width: 350px;
  padding: 40px;
  border: 1px solid #dbdbdb;
  background-color: white;
  text-align: center;
}
h1 {
  font-family: 'Grand Hotel', cursive; /* A font similar to Instagram's */
  font-size: 3rem;
  margin-bottom: 10px;
}
.subtitle {
  color: #8e8e8e;
  margin-bottom: 20px;
}
input, select, button {
  width: 100%;
  padding: 10px;
  margin-bottom: 10px;
  border: 1px solid #dbdbdb;
  border-radius: 3px;
  box-sizing: border-box;
}
button {
  background-color: #0095f6;
  color: white;
  font-weight: bold;
  border: none;
  cursor: pointer;
}
button:disabled {
  background-color: #b2dffc;
  cursor: not-allowed;
}
.form-group {
  text-align: left;
  margin-bottom: 10px;
}
.form-group label {
  display: block;
  font-size: 0.9em;
  color: #8e8e8e;
  margin-bottom: 5px;
}
.error-message {
  color: red;
  margin-top: 15px;
}
.success-message {
  color: green;
  margin-top: 15px;
}
</style>