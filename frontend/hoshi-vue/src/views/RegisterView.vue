<!-- frontend/hoshi-vue/src/views/RegisterView.vue -->

<template>
  <div class="register-container">
    <div class="register-box">
      <h1>hoshiBmatchi</h1>
      <p class="subtitle">
        Sign up to see photos and videos from your friends.
      </p>
      
      <form @submit.prevent="handleRegister">
        <input
          v-model="form.name"
          type="text"
          placeholder="Full Name"
          required
        />
        <input
          v-model="form.username"
          type="text"
          placeholder="Username"
          required
        />
        <input
          v-model="form.email"
          type="email"
          placeholder="Email"
          required
        />
        <input
          v-model="form.password"
          type="password"
          placeholder="Password"
          required
        />
        
        <div class="form-group">
          <label for="dob">Date of Birth:</label>
          <input
            id="dob"
            v-model="form.date_of_birth"
            type="date"
            required
          />
        </div>

        <div class="form-group">
          <label>Gender:</label>
          <select
            v-model="form.gender"
            required
          >
            <option
              disabled
              value=""
            >
              Please select one
            </option>
            <option>Male</option>
            <option>Female</option>
            <option>Other</option>
          </select>
        </div>

        <div class="form-group newsletter-checkbox">
          <label class="checkbox-label">
            <input
              v-model="form.subscribe_to_newsletter"
              type="checkbox"
            />
            <span>Subscribe to Newsletter</span>
          </label>
          <p class="checkbox-description">
            Get the latest updates, features, and news delivered to your inbox
          </p>
        </div>

        <!-- Cloudflare Turnstile -->
        <div class="turnstile-container">
          <div id="register-turnstile"></div>
          <div
            v-if="turnstileError"
            class="turnstile-error"
          >
            {{ turnstileError }}
          </div>
        </div>

        <button
          type="submit"
          :disabled="isLoading"
        >
          {{ isLoading ? 'Signing Up...' : 'Sign Up' }}
        </button>
      </form>

      <div
        v-if="errorMessage"
        class="error-message"
      >
        {{ errorMessage }}
      </div>
      <div
        v-if="successMessage"
        class="success-message"
      >
        {{ successMessage }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import axios from "axios";
import { useTurnstile } from "@/composables/useTurnstile";

// Turnstile setup
const { turnstileToken, turnstileError, initTurnstile, resetTurnstile, removeTurnstile } = useTurnstile();

// Reactive state for our form data
const form = ref({
  name: "",
  username: "",
  email: "",
  password: "",
  date_of_birth: "", // Will be in "YYYY-MM-DD" format from the input
  gender: "",
  subscribe_to_newsletter: false // Newsletter subscription
});

// State for loading and messages
const isLoading = ref(false);
const errorMessage = ref("");
const successMessage = ref("");

// Function to handle form submission
const handleRegister = async () => {
  // Validate Turnstile token
  if (!turnstileToken.value) {
    errorMessage.value = "Please complete the verification challenge.";
    return;
  }

  isLoading.value = true;
  errorMessage.value = "";
  successMessage.value = "";

  try {
    // The URL for our API Gateway. We'll make 'api.hoshi.local' work in the next step.
    const apiUrl = "http://api.hoshi.local/auth/register";
    
    // Send the form data to the API Gateway with Turnstile token
    const response = await axios.post(apiUrl, {
      ...form.value,
      turnstile_token: turnstileToken.value
    });

    // Handle success
    successMessage.value = `Successfully registered user ${response.data.username}! You can now log in.`;
    console.log("Registration successful:", response.data);
    
  } catch (error: any) {
    // Handle errors
    console.error("Registration failed:", error);
    if (error.response) {
      errorMessage.value = error.response.data.error || "An unknown error occurred.";
    } else {
      errorMessage.value = "Could not connect to the server. Please try again later.";
    }
    // Reset Turnstile on error
    resetTurnstile();
  } finally {
    isLoading.value = false;
  }
};

// Initialize Turnstile on mount
onMounted(() => {
  initTurnstile('register-turnstile');
});

// Cleanup Turnstile on unmount
onUnmounted(() => {
  removeTurnstile();
});
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: #000;
  padding: 20px;
}
.register-box {
  width: 100%;
  max-width: 400px;
  padding: 40px;
  border: 1px solid #262626;
  background-color: #000;
  text-align: center;
  border-radius: 3px;
}
h1 {
  font-family: 'Grand Hotel', cursive;
  font-size: 3rem;
  margin-bottom: 10px;
  color: #fff;
}
.subtitle {
  color: #8e8e8e;
  margin-bottom: 20px;
  font-size: 1.1em;
  font-weight: 600;
}
input, select {
  width: 100%;
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #262626;
  border-radius: 3px;
  box-sizing: border-box;
  background-color: #121212;
  color: #fff;
  font-size: 14px;
}
input::placeholder {
  color: #737373;
}
input:focus, select:focus {
  outline: none;
  border-color: #3897f0;
}
button {
  width: 100%;
  padding: 12px;
  margin-top: 10px;
  background-color: #0095f6;
  color: white;
  font-weight: bold;
  border: none;
  cursor: pointer;
  border-radius: 8px;
  font-size: 14px;
  transition: background-color 0.2s;
}
button:hover:not(:disabled) {
  background-color: #1877f2;
}
button:disabled {
  background-color: #0095f64d;
  cursor: not-allowed;
  opacity: 0.7;
}
.form-group {
  text-align: left;
  margin-bottom: 12px;
}
.form-group label {
  display: block;
  font-size: 0.9em;
  color: #a8a8a8;
  margin-bottom: 5px;
}
.form-group select {
  color: #a8a8a8;
}
.form-group select option {
  background-color: #121212;
  color: #fff;
}
.newsletter-checkbox {
  text-align: left;
  padding: 12px;
  background-color: #121212;
  border-radius: 5px;
  margin-bottom: 12px;
  border: 1px solid #262626;
}
.checkbox-label {
  display: flex;
  align-items: center;
  cursor: pointer;
  font-size: 0.9em;
  color: #e4e6eb;
}
.checkbox-label input[type="checkbox"] {
  width: auto;
  margin-right: 10px;
  margin-bottom: 0;
  cursor: pointer;
  accent-color: #0095f6;
}
.checkbox-label span {
  font-weight: 500;
}
.checkbox-description {
  font-size: 0.8em;
  color: #737373;
  margin: 6px 0 0 30px;
  line-height: 1.4;
}
.turnstile-container {
  margin-bottom: 15px;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 65px;
  justify-content: center;
}
.turnstile-error {
  color: #ed4956;
  font-size: 0.85em;
  margin-top: 8px;
  text-align: center;
}
.error-message {
  color: #ed4956;
  margin-top: 15px;
  font-size: 14px;
  background-color: #1a1a1a;
  padding: 10px;
  border-radius: 5px;
  border: 1px solid #ed4956;
}
.success-message {
  color: #00ba7c;
  margin-top: 15px;
  font-size: 14px;
  background-color: #1a1a1a;
  padding: 10px;
  border-radius: 5px;
  border: 1px solid #00ba7c;
}
</style>