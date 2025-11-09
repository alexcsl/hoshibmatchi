<template>
  <form @submit.prevent="handleRegister">
    <input type="text" v-model="form.name" placeholder="Full Name" required>
    <div v-if="errors.name" class="error">{{ errors.name }}</div>

    <input type="text" v-model="form.username" placeholder="Username" required>
    <div v-if="errors.username" class="error">{{ errors.username }}</div>

    <input type="email" v-model="form.email" placeholder="Email" required>
    <div v-if="errors.email" class="error">{{ errors.email }}</div>

    <input type="password" v-model="form.password" placeholder="Password" required>
    <div v-if="errors.password" class="error">{{ errors.password }}</div>

    <input type="date" v-model="form.dateOfBirth" required>
    <div v-if="errors.dateOfBirth" class="error">{{ errors.dateOfBirth }}</div>

    <select v-model="form.gender" required>
      <option value="">Select Gender</option>
      <option value="male">Male</option>
      <option value="female">Female</option>
      <option value="other">Other</option>
    </select>
    <div v-if="errors.gender" class="error">{{ errors.gender }}</div>

    <button type="submit">Sign Up</button>
  </form>
</template>

<script setup lang="ts">
import { reactive } from 'vue'

const form = reactive({
  name: '',
  username: '',
  email: '',
  password: '',
  dateOfBirth: '',
  gender: ''
})

const errors = reactive({
  name: '',
  username: '',
  email: '',
  password: '',
  dateOfBirth: '',
  gender: ''
})

const handleRegister = () => {
  // Clear previous errors
  Object.keys(errors).forEach(key => {
    errors[key as keyof typeof errors] = ''
  })
  
  // Basic validation
  if (!form.name.trim()) errors.name = 'Name is required'
  if (!form.username.trim()) errors.username = 'Username is required'
  if (!form.email.trim()) errors.email = 'Email is required'
  if (!form.password.trim()) errors.password = 'Password is required'
  if (!form.dateOfBirth) errors.dateOfBirth = 'Date of birth is required'
  if (!form.gender) errors.gender = 'Gender is required'
  
  // If no errors, proceed with registration
  const hasErrors = Object.values(errors).some(error => error !== '')
  if (!hasErrors) {
    console.log('Registration data:', form)
    // TODO: Implement actual registration logic
  }
}
</script>

<style scoped>
.error {
  color: red;
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

form {
  max-width: 400px;
  margin: 0 auto;
  padding: 2rem;
}

input, select {
  width: 100%;
  padding: 0.5rem;
  margin-bottom: 1rem;
  border: 1px solid #ccc;
  border-radius: 4px;
}

button {
  width: 100%;
  padding: 0.75rem;
  background-color: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

button:hover {
  background-color: #0056b3;
}
</style>