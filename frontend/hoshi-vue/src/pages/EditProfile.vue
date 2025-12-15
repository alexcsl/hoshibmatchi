<template>
  <div class="edit-profile-page">
    <div class="edit-profile-container">
      <div class="edit-header">
        <h1>Edit profile</h1>
        <button
          class="close-btn"
          @click="goBack"
        >
          ✕
        </button>
      </div>

      <div class="profile-picture-section">
        <img
          :src="profilePicture || '/default-avatar.svg'"
          alt="Profile"
          class="profile-pic"
        />
        <button
          class="change-photo-btn"
          @click="triggerFileUpload"
        >
          Change photo
        </button>
        <input 
          ref="fileInput" 
          type="file" 
          accept="image/*" 
          style="display: none"
          @change="handleProfilePictureChange"
        />
      </div>

      <form
        class="edit-form"
        @submit.prevent="handleSubmit"
      >
        <div class="form-group">
          <label for="name">Name</label>
          <input
            id="name"
            v-model="formData.name"
            type="text"
            class="form-input"
            placeholder="Name"
          />
          <p
            v-if="errors.name"
            class="error-text"
          >
            {{ errors.name }}
          </p>
          <p class="field-hint">
            Help people discover your account by using the name you're known by.
          </p>
        </div>

        <div class="form-group">
          <label for="username">Username</label>
          <input
            id="username"
            v-model="formData.username"
            type="text"
            class="form-input"
            placeholder="Username"
          />
          <p
            v-if="errors.username"
            class="error-text"
          >
            {{ errors.username }}
          </p>
        </div>

        <div class="form-group">
          <label for="bio">Bio</label>
          <textarea
            id="bio"
            v-model="formData.bio"
            class="form-input bio-input"
            placeholder="Bio"
            maxlength="150"
          ></textarea>
          <div class="char-count">
            {{ formData.bio.length }} / 150
          </div>
        </div>

        <div class="form-group">
          <label for="gender">Gender</label>
          <select
            id="gender"
            v-model="formData.gender"
            class="form-input select-input"
          >
            <option value="male">
              Male
            </option>
            <option value="female">
              Female
            </option>
          </select>
        </div>

        <div class="form-actions">
          <button
            type="button"
            class="cancel-btn"
            @click="goBack"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="save-btn"
            :disabled="isSubmitting"
          >
            {{ isSubmitting ? 'Saving...' : 'Save' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import apiClient, { mediaAPI } from "@/services/api";

const router = useRouter();
const authStore = useAuthStore();
const fileInput = ref<HTMLInputElement | null>(null);

const formData = reactive({
  name: "",
  username: "",
  bio: "",
  gender: "Prefer not to say"
});

const profilePicture = ref("");
const isSubmitting = ref(false);
const errors = reactive({
  name: "",
  username: ""
});

// Load initial data
const fetchUserData = async () => {
  try {
    const username = authStore.user?.username;
    if (!username) return;
    
    const response = await apiClient.get(`/users/${username}`);
    const user = response.data.user || response.data;
    
    formData.name = user.name || "";
    formData.username = user.username || "";
    formData.bio = user.bio || "";
    formData.gender = user.gender || "Prefer not to say";
    profilePicture.value = user.profile_picture_url || "";
  } catch (err) {
    console.error("Failed to fetch user data", err);
  }
};

const triggerFileUpload = () => {
  fileInput.value?.click();
};

const handleProfilePictureChange = async (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files && target.files[0]) {
    const file = target.files[0];
    try {
        // Upload Logic
        const uploadRes = await mediaAPI.uploadMedia(file);
        profilePicture.value = uploadRes.media_url;
        // Auto-save the picture update or wait for form submit?
        // Usually better to wait for form submit, but for picture specifically 
        // it's often instant. Let's keep it for the form submit.
    } catch {
        alert("Failed to upload image");
    }
  }
};

const validate = () => {
  let isValid = true;
  errors.name = "";
  errors.username = "";

  if (!formData.name.trim()) {
    errors.name = "Name is required";
    isValid = false;
  }

  if (!formData.username.trim()) {
    errors.username = "Username is required";
    isValid = false;
  }

  return isValid;
};

const handleSubmit = async () => {
  if (!validate()) return;
  isSubmitting.value = true;
  
  try {
    // Construct update payload
    // Note: API might need specific structure. Based on gateway:
    // PUT /profile/edit -> handleUpdateProfile_Gin
    
    await apiClient.put("/profile/edit", {
        name: formData.name,
        username: formData.username,
        bio: formData.bio,
        gender: formData.gender,
        profile_picture_url: profilePicture.value // If your backend supports this in the same call
    });
    
    // Update local store
    if (authStore.user) {
        authStore.user.username = formData.username;
        authStore.user.name = formData.name;
        // Trigger a refresh of auth token/data if possible
    }
    
    router.push("/profile");
  } catch (err: any) {
    console.error(err);
    alert(err.response?.data?.error || "Failed to update profile");
  } finally {
    isSubmitting.value = false;
  }
};

const goBack = () => {
  router.back();
};

onMounted(() => {
  fetchUserData();
});
</script>

<style scoped lang="scss">
.edit-profile-page {
  background-color: #000;
  min-height: 100vh;
  color: #fff;
  padding: 20px;
  display: flex;
  justify-content: center;
}

.edit-profile-container {
  width: 100%;
  max-width: 600px;
  background-color: #000;
  border: 1px solid #262626;
  border-radius: 12px;
  padding: 24px;
}

.edit-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
  h1 { font-size: 24px; font-weight: 600; }
  .close-btn { background: none; border: none; color: #fff; font-size: 24px; cursor: pointer; }
}

.profile-picture-section {
  background-color: #262626;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
  
  .profile-pic {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    object-fit: cover;
  }
  
  .change-photo-btn {
    background-color: #0095f6;
    color: #fff;
    border: none;
    padding: 8px 16px;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    &:hover { background-color: #1877f2; }
  }
}

.form-group {
  margin-bottom: 24px;
  label { display: block; margin-bottom: 8px; font-weight: 600; font-size: 14px; }
  .form-input {
    width: 100%;
    background: #121212;
    border: 1px solid #363636;
    border-radius: 4px;
    padding: 10px;
    color: #fff;
    font-size: 16px;
    &:focus { border-color: #fff; outline: none; }
  }
  .bio-input { resize: none; height: 80px; }
  .field-hint { font-size: 12px; color: #a8a8a8; margin-top: 8px; line-height: 1.4; }
  .error-text { color: #ed4956; font-size: 12px; margin-top: 4px; }
  .char-count { text-align: right; font-size: 12px; color: #a8a8a8; margin-top: 4px; }
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-top: 40px;
  button {
    flex: 1; padding: 12px; border-radius: 8px; font-weight: 600; cursor: pointer; border: none;
    &.cancel-btn { background: transparent; border: 1px solid #363636; color: #fff; }
    &.save-btn { background: #0095f6; color: #fff; &:disabled { opacity: 0.7; } }
  }
}

/* Responsive Design */
@media (max-width: 768px) {
  .edit-profile-container {
    padding: 20px 16px;
  }
  
  .profile-picture-section {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }
  
  .form-actions {
    flex-direction: column;
    
    button {
      width: 100%;
    }
  }
}

@media (max-width: 640px) {
  .edit-header h1 {
    font-size: 20px;
  }
  
  .edit-profile-container {
    padding: 16px 12px;
  }
  
  .profile-pic {
    width: 80px;
    height: 80px;
  }
}
</style>