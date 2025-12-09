<template>
  <div class="reset-password-container">
    <div class="reset-card">
      <h1 class="logo">
        Instagram
      </h1>

      <div class="content">
        <h2 class="title">
          Reset Password
        </h2>
        <p class="subtitle">
          Enter your new password below
        </p>

        <!-- Error Alert -->
        <ErrorAlert
          v-if="error"
          :message="error"
          @close="error = ''"
        />

        <!-- Form -->
        <form
          class="form"
          @submit.prevent="handleSubmit"
        >
          <FormInput
            v-model="form.newPassword"
            type="password"
            placeholder="New password"
            :error-message="errors.newPassword"
          />

          <PasswordStrengthValidator :password="form.newPassword" />

          <FormInput
            v-model="form.confirmPassword"
            type="password"
            placeholder="Confirm new password"
            :error-message="errors.confirmPassword"
          />

          <button
            type="submit"
            class="reset-btn"
            :disabled="loading"
          >
            {{ loading ? 'Resetting...' : 'Reset Password' }}
          </button>
        </form>

        <!-- Links -->
        <div class="links">
          <router-link
            to="/signup"
            class="link"
          >
            Don't have an account? Sign up
          </router-link>
          <router-link
            to="/login"
            class="link"
          >
            Back to login
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import FormInput from "../components/FormInput.vue";
import PasswordStrengthValidator from "../components/PasswordStrengthValidator.vue";
import ErrorAlert from "../components/ErrorAlert.vue";
import { authAPI, handleApiError } from "../services/api";

const router = useRouter();
const route = useRoute();
const loading = ref(false);
const error = ref("");
const email = ref("");
const otpCode = ref("");

const form = reactive({
  newPassword: "",
  confirmPassword: ""
});

const errors = reactive({
  newPassword: "",
  confirmPassword: ""
});

onMounted(() => {
  // Get email and OTP from query params
  email.value = (route.query.email as string) || "";
  otpCode.value = (route.query.otp as string) || "";

  if (!email.value || !otpCode.value) {
    error.value = "Invalid reset link. Please request a new password reset.";
    setTimeout(() => {
      router.push("/forgot-password");
    }, 3000);
  }
});

const handleSubmit = async () => {
  errors.newPassword = "";
  errors.confirmPassword = "";
  error.value = "";

  if (form.newPassword.length < 8 || !/[A-Z]/.test(form.newPassword) || !/[a-z]/.test(form.newPassword) || !/[0-9]/.test(form.newPassword) || !/[!@#$%^&*(),.?":{}|<>]/.test(form.newPassword)) {
    errors.newPassword = "Password must have at least 8 characters, 1 uppercase, 1 lowercase, 1 number, and 1 special character";
    return;
  }

  if (form.newPassword !== form.confirmPassword) {
    errors.confirmPassword = "Passwords do not match";
    return;
  }

  loading.value = true;

  try {
    await authAPI.resetPassword({
      email: email.value,
      otp_code: otpCode.value,
      new_password: form.newPassword
    });

    // Success - redirect to login with success message
    router.push({
      path: "/login",
      query: { reset: "success" }
    });
  } catch (err) {
    error.value = handleApiError(err);
  } finally {
    loading.value = false;
  }
};

</script>

<style scoped lang="scss">
.reset-password-container {
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
  padding: 20px;
}

.reset-card {
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

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
}

.reset-btn {
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

.links {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
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
