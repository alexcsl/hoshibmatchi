<template>
  <div class="callback-container">
    <div class="loading">
      <div class="spinner"></div>
      <p>{{ message }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { authAPI, saveAuthData, handleApiError } from "../services/api";

const router = useRouter();
const message = ref("Processing Google login...");

onMounted(async () => {
  try {
    // Get the auth_code from URL query params (OAuth 2.0 authorization code flow)
    const params = new URLSearchParams(window.location.search);
    const authCode = params.get("code");

    if (!authCode) {
      message.value = "No authorization code received";
      setTimeout(() => {
        router.push("/login");
      }, 2000);
      return;
    }

    // Send auth_code to backend
    const response = await authAPI.googleAuth({ auth_code: authCode });

    // Save token and user data (Google OAuth returns access_token)
    const token = response.access_token || response.token || "";
    saveAuthData(token, response);

    // Check if user needs to complete their profile
    if (response.needs_profile_completion) {
      // New Google user - redirect to profile completion
      message.value = "Account created! Please complete your profile...";
      setTimeout(() => {
        router.push({
          name: "google-complete-profile",
          query: {
            name: response.name || "",
            email: response.email || "",
            picture: response.profile_picture_url || ""
          }
        });
      }, 1000);
    } else {
      // Existing user - redirect to feed
      message.value = "Login successful! Redirecting...";
      setTimeout(() => {
        router.push("/feed");
      }, 1000);
    }
  } catch (err) {
    message.value = handleApiError(err);
    setTimeout(() => {
      router.push("/login");
    }, 3000);
  }
});
</script>

<style scoped>
.callback-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #000;
}

.loading {
  text-align: center;
}

.spinner {
  width: 50px;
  height: 50px;
  margin: 0 auto 20px;
  border: 4px solid #262626;
  border-top: 4px solid #0a66c2;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

p {
  color: #fff;
  font-size: 16px;
}
</style>
