<template>
  <div class="settings-page">
    <div class="settings-container">
      <h1>Settings</h1>

      <!-- Tabs Navigation -->
      <div class="settings-tabs">
        <button
          v-for="tab in filteredTabs"
          :key="tab.id"
          class="tab-button"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- Tab Content -->
      <div class="tab-content">
        <!-- Edit Profile Tab -->
        <div
          v-if="activeTab === 'edit-profile'"
          class="tab-panel"
        >
          <h2>Edit Profile</h2>
          <div v-if="loadingProfile" class="loading">
            Loading profile...
          </div>
          <div v-else class="form-section">
            <div class="form-group">
              <label>Profile Picture</label>
              <div class="profile-pic-upload">
                <img
                  :src="editForm.profilePicture || '/default-avatar.svg'"
                  alt="Profile"
                  class="preview-img"
                />
                <button
                  class="upload-btn"
                  @click="triggerFileInput"
                >
                  Change Photo
                </button>
                <input
                  ref="fileInput"
                  type="file"
                  accept="image/*"
                  style="display: none"
                  @change="handleFileSelect"
                />
              </div>
            </div>

            <div class="form-group">
              <label>Name</label>
              <input
                v-model="editForm.name"
                type="text"
                placeholder="Your name"
                maxlength="100"
              />
              <span
                v-if="validationErrors.name"
                class="error"
              >{{ validationErrors.name }}</span>
            </div>

            <div class="form-group">
              <label>Username</label>
              <input
                v-model="editForm.username"
                type="text"
                placeholder="Username"
                maxlength="50"
                @input="validateUsername"
              />
              <span
                v-if="validationErrors.username"
                class="error"
              >{{ validationErrors.username }}</span>
            </div>

            <div class="form-group">
              <label>Gender</label>
              <select v-model="editForm.gender">
                <option value="">
                  Select gender...
                </option>
                <option value="Male">
                  Male
                </option>
                <option value="Female">
                  Female
                </option>
                <option value="Other">
                  Other
                </option>
              </select>
              <span
                v-if="validationErrors.gender"
                class="error"
              >{{ validationErrors.gender }}</span>
            </div>

            <div class="form-group">
              <label>Bio</label>
              <textarea
                v-model="editForm.bio"
                placeholder="Tell us about yourself..."
                maxlength="255"
                rows="4"
              ></textarea>
            </div>

            <button
              class="save-btn"
              :disabled="saving"
              @click="saveProfile"
            >
              {{ saving ? 'Saving...' : 'Save Changes' }}
            </button>
          </div>
        </div>

        <!-- Notifications Tab -->
        <div
          v-if="activeTab === 'notifications'"
          class="tab-panel"
        >
          <h2>Notification Settings</h2>
          <div class="settings-section">
            <div class="setting-item">
              <div class="setting-info">
                <div class="label">
                  Push Notifications
                </div>
                <div class="description">
                  Get push notifications for likes, comments, and follows
                </div>
              </div>
              <label class="toggle">
                <input
                  v-model="notificationSettings.pushNotifications"
                  type="checkbox"
                  @change="saveNotificationSettings"
                />
                <span class="slider"></span>
              </label>
            </div>

            <div class="setting-item">
              <div class="setting-info">
                <div class="label">
                  Email Notifications
                </div>
                <div class="description">
                  Receive email updates about your activity
                </div>
              </div>
              <label class="toggle">
                <input
                  v-model="notificationSettings.emailNotifications"
                  type="checkbox"
                  @change="saveNotificationSettings"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
        </div>

        <!-- Account Privacy Tab -->
        <div
          v-if="activeTab === 'privacy'"
          class="tab-panel"
        >
          <h2>Account Privacy</h2>
          <div class="settings-section">
            <div class="setting-item">
              <div class="setting-info">
                <div class="label">
                  Private Account
                </div>
                <div class="description">
                  When your account is private, only people you approve can see your posts and stories
                </div>
              </div>
              <label class="toggle">
                <input
                  v-model="privacySettings.isPrivate"
                  type="checkbox"
                  @change="savePrivacySettings"
                />
                <span class="slider"></span>
              </label>
            </div>
          </div>
        </div>

        <!-- Close Friends Tab -->
        <div
          v-if="activeTab === 'close-friends'"
          class="tab-panel"
        >
          <h2>Close Friends</h2>
          <p class="tab-description">
            Share stories with your close friends only
          </p>

          <button
            class="add-btn"
            @click="showCloseFriendsModal = true"
          >
            Add Close Friends
          </button>

          <div
            v-if="loadingCloseFriends"
            class="loading"
          >
            Loading...
          </div>
          <div
            v-else-if="closeFriends.length === 0"
            class="empty-state"
          >
            <p>No close friends added yet</p>
          </div>
          <div
            v-else
            class="friends-list"
          >
            <div
              v-for="friend in closeFriends"
              :key="friend.id"
              class="friend-item"
            >
              <img
                :src="friend.profile_picture_url || '/default-avatar.svg'"
                alt=""
                class="avatar"
              />
              <div class="friend-info">
                <div class="username">
                  {{ friend.username }}
                </div>
                <div class="name">
                  {{ friend.name }}
                </div>
              </div>
              <button
                class="remove-btn"
                @click="removeCloseFriend(friend.id)"
              >
                Remove
              </button>
            </div>
          </div>
        </div>

        <!-- Blocked Tab -->
        <div
          v-if="activeTab === 'blocked'"
          class="tab-panel"
        >
          <h2>Blocked Accounts</h2>
          <p class="tab-description">
            Manage accounts you've blocked
          </p>

          <div
            v-if="loadingBlocked"
            class="loading"
          >
            Loading...
          </div>
          <div
            v-else-if="blockedUsers.length === 0"
            class="empty-state"
          >
            <p>No blocked accounts</p>
          </div>
          <div
            v-else
            class="friends-list"
          >
            <div
              v-for="user in blockedUsers"
              :key="user.id"
              class="friend-item"
            >
              <img
                :src="user.profile_picture_url || '/default-avatar.svg'"
                alt=""
                class="avatar"
              />
              <div class="friend-info">
                <div class="username">
                  {{ user.username }}
                </div>
                <div class="name">
                  {{ user.name }}
                </div>
              </div>
              <button
                class="unblock-btn"
                @click="unblockUser(user.id)"
              >
                Unblock
              </button>
            </div>
          </div>
        </div>

        <!-- Hide Story Tab -->
        <div
          v-if="activeTab === 'hide-story'"
          class="tab-panel"
        >
          <h2>Hide Story From</h2>
          <p class="tab-description">
            Hide your story from specific people
          </p>

          <button
            class="add-btn"
            @click="showHideStoryModal = true"
          >
            Add People
          </button>

          <div
            v-if="loadingHiddenStory"
            class="loading"
          >
            Loading...
          </div>
          <div
            v-else-if="hiddenStoryUsers.length === 0"
            class="empty-state"
          >
            <p>Your story is visible to everyone</p>
          </div>
          <div
            v-else
            class="friends-list"
          >
            <div
              v-for="user in hiddenStoryUsers"
              :key="user.id"
              class="friend-item"
            >
              <img
                :src="user.profile_picture_url || '/default-avatar.svg'"
                alt=""
                class="avatar"
              />
              <div class="friend-info">
                <div class="username">
                  {{ user.username }}
                </div>
                <div class="name">
                  {{ user.name }}
                </div>
              </div>
              <button
                class="remove-btn"
                @click="removeHiddenStoryUser(user.id)"
              >
                Remove
              </button>
            </div>
          </div>
        </div>

        <!-- Request Verified Tab -->
        <div
          v-if="activeTab === 'verified'"
          class="tab-panel"
        >
          <h2>Request Verified Badge</h2>
          <p class="tab-description">
            Get the verified checkmark on your profile
          </p>

          <div
            v-if="verificationRequest"
            class="verification-status"
          >
            <div class="status-card">
              <h3>Request Status: {{ verificationRequest.status }}</h3>
              <p v-if="verificationRequest.status === 'pending'">
                Your verification request is being reviewed
              </p>
              <p v-else-if="verificationRequest.status === 'approved'">
                Congratulations! Your account has been verified
              </p>
              <p v-else-if="verificationRequest.status === 'rejected'">
                Your verification request was rejected. You can submit a new request.
              </p>
            </div>
          </div>

          <div
            v-else
            class="verification-form"
          >
            <div class="form-group">
              <label>ID Card Number</label>
              <input
                v-model="verificationForm.idCardNumber"
                type="text"
                placeholder="Enter your ID card number"
                maxlength="50"
              />
              <span
                v-if="validationErrors.idCardNumber"
                class="error"
              >{{ validationErrors.idCardNumber }}</span>
            </div>

            <div class="form-group">
              <label>Face Picture</label>
              <div class="file-upload">
                <input
                  ref="faceInput"
                  type="file"
                  accept="image/*"
                  @change="handleFaceUpload"
                />
                <div
                  v-if="verificationForm.facePicturePreview"
                  class="preview"
                >
                  <img
                    :src="verificationForm.facePicturePreview"
                    alt="Face"
                  />
                </div>
              </div>
            </div>

            <div class="form-group">
              <label>Reason for Verification</label>
              <textarea
                v-model="verificationForm.reason"
                placeholder="Why should your account be verified?"
                rows="4"
              ></textarea>
            </div>

            <button
              class="submit-btn"
              :disabled="submittingVerification"
              @click="submitVerificationRequest"
            >
              {{ submittingVerification ? 'Submitting...' : 'Submit Request' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Close Friends Modal -->
    <div
      v-if="showCloseFriendsModal"
      class="modal-overlay"
      @click="showCloseFriendsModal = false"
    >
      <div
        class="modal"
        @click.stop
      >
        <div class="modal-header">
          <h3>Add Close Friends</h3>
          <button
            class="close-btn"
            @click="showCloseFriendsModal = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <input
            v-model="closeFriendSearch"
            type="text"
            placeholder="Search followers..."
            class="search-input"
            @input="searchFollowers"
          />
          <div class="users-list">
            <div
              v-for="user in searchResults"
              :key="user.id"
              class="user-item"
            >
              <img
                :src="user.profile_picture_url || '/default-avatar.svg'"
                alt=""
                class="avatar"
              />
              <div class="user-info">
                <div class="username">
                  {{ user.username }}
                </div>
              </div>
              <button
                class="add-btn-small"
                @click="addCloseFriend(user.id)"
              >
                Add
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Hide Story Modal -->
    <div
      v-if="showHideStoryModal"
      class="modal-overlay"
      @click="showHideStoryModal = false"
    >
      <div
        class="modal"
        @click.stop
      >
        <div class="modal-header">
          <h3>Hide Story From</h3>
          <button
            class="close-btn"
            @click="showHideStoryModal = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <input
            v-model="hideStorySearch"
            type="text"
            placeholder="Search users..."
            class="search-input"
            @input="searchUsers"
          />
          <div class="users-list">
            <div
              v-for="user in hideStorySearchResults"
              :key="user.id"
              class="user-item"
            >
              <img
                :src="user.profile_picture_url || '/default-avatar.svg'"
                alt=""
                class="avatar"
              />
              <div class="user-info">
                <div class="username">
                  {{ user.username }}
                </div>
              </div>
              <button
                class="add-btn-small"
                @click="addHiddenStoryUser(user.id)"
              >
                Add
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch, computed } from 'vue';
import { userAPI, mediaAPI } from '../services/api';
import { useAuthStore } from '../stores/auth';

const authStore = useAuthStore();

const tabs = [
  { id: 'edit-profile', label: 'Edit Profile' },
  { id: 'notifications', label: 'Notifications' },
  { id: 'privacy', label: 'Account Privacy' },
  { id: 'close-friends', label: 'Close Friends' },
  { id: 'blocked', label: 'Blocked' },
  { id: 'hide-story', label: 'Hide Story' },
  { id: 'verified', label: 'Request Verified' }
];

// Filter tabs based on verification status
const filteredTabs = computed(() => {
  if (authStore.user?.is_verified) {
    return tabs.filter(tab => tab.id !== 'verified');
  }
  return tabs;
});

const activeTab = ref('edit-profile');
const saving = ref(false);
const loadingProfile = ref(true);
const fileInput = ref<HTMLInputElement | null>(null);
const faceInput = ref<HTMLInputElement | null>(null);

// Edit Profile Form
const editForm = reactive({
  name: '',
  username: '',
  gender: '',
  bio: '',
  profilePicture: ''
});

const validationErrors = reactive<Record<string, string>>({});

// Notification Settings
const notificationSettings = reactive({
  pushNotifications: true,
  emailNotifications: true
});

// Privacy Settings
const privacySettings = reactive({
  isPrivate: false
});

// Close Friends
const closeFriends = ref<any[]>([]);
const loadingCloseFriends = ref(false);
const showCloseFriendsModal = ref(false);
const closeFriendSearch = ref('');
const searchResults = ref<any[]>([]);

// Blocked Users
const blockedUsers = ref<any[]>([]);
const loadingBlocked = ref(false);

// Hide Story
const hiddenStoryUsers = ref<any[]>([]);
const loadingHiddenStory = ref(false);
const showHideStoryModal = ref(false);
const hideStorySearch = ref('');
const hideStorySearchResults = ref<any[]>([]);

// Verification
const verificationRequest = ref<any>(null);
const verificationForm = reactive({
  idCardNumber: '',
  facePicture: null as File | null,
  facePicturePreview: '',
  reason: ''
});
const submittingVerification = ref(false);

onMounted(async () => {
  await loadUserProfile();
  await loadNotificationSettings();
  await loadPrivacySettings();
});

watch(activeTab, (newTab) => {
  if (newTab === 'blocked' && blockedUsers.value.length === 0) {
    loadBlockedUsers();
  }
  if (newTab === 'close-friends' && closeFriends.value.length === 0) {
    loadCloseFriends();
  }
  if (newTab === 'hide-story' && hiddenStoryUsers.value.length === 0) {
    loadHiddenStoryUsers();
  }
});

const loadBlockedUsers = async () => {
  loadingBlocked.value = true;
  try {
    const response = await userAPI.getBlockedUsers();
    blockedUsers.value = response.blocked_users || [];
  } catch (error) {
    console.error('Failed to load blocked users:', error);
  } finally {
    loadingBlocked.value = false;
  }
};

const loadCloseFriends = async () => {
  try {
    const response = await userAPI.getCloseFriends();
    closeFriends.value = response.friends || [];
  } catch (error) {
    console.error('Failed to load close friends:', error);
  }
};

const loadHiddenStoryUsers = async () => {
  try {
    const response = await userAPI.getHiddenStoryUsers();
    hiddenStoryUsers.value = response.hidden_users || [];
  } catch (error) {
    console.error('Failed to load hidden story users:', error);
  }
};

const loadUserProfile = async () => {
  loadingProfile.value = true;
  try {
    const profile = await userAPI.getProfile(authStore.user?.username || '');
    editForm.name = profile.name;
    editForm.username = profile.username;
    editForm.gender = profile.gender || '';
    editForm.bio = profile.bio || '';
    editForm.profilePicture = profile.profile_picture_url || '';
  } catch (error) {
    console.error('Failed to load profile:', error);
  } finally {
    loadingProfile.value = false;
  }
};

const loadNotificationSettings = async () => {
  try {
    const settings = await userAPI.getNotificationSettings();
    notificationSettings.pushNotifications = settings.push_enabled ?? true;
    notificationSettings.emailNotifications = settings.email_enabled ?? true;
  } catch (error) {
    console.error('Failed to load notification settings:', error);
    // Set defaults
    notificationSettings.pushNotifications = true;
    notificationSettings.emailNotifications = true;
  }
};

const loadPrivacySettings = async () => {
  try {
    const profile = await userAPI.getProfile(authStore.user?.username || '');
    privacySettings.isPrivate = profile.is_private || false;
  } catch (error) {
    console.error('Failed to load privacy settings:', error);
  }
};

const validateUsername = () => {
  const usernameRegex = /^[a-zA-Z0-9_]+$/;
  if (!editForm.username) {
    validationErrors.username = 'Username is required';
  } else if (editForm.username.length < 3 || editForm.username.length > 30) {
    validationErrors.username = 'Username must be 3-30 characters';
  } else if (!usernameRegex.test(editForm.username)) {
    validationErrors.username = 'Username can only contain letters, numbers, and underscores';
  } else {
    delete validationErrors.username;
  }
};

const triggerFileInput = () => {
  fileInput.value?.click();
};

const handleFileSelect = async (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (file) {
    // Upload profile picture
    try {
      saving.value = true;
      const response = await mediaAPI.uploadMedia(file);
      editForm.profilePicture = response.media_url;
      
      // Update profile with ONLY the new picture URL
      await userAPI.updateProfile({
        profile_picture_url: response.media_url
      });
      
      // Update auth store and reload profile to propagate changes everywhere
      if (authStore.user && authStore.user.username) {
        authStore.user.profile_picture_url = response.media_url;
        
        // Reload full profile to ensure all components see the update
        const updatedProfile = await userAPI.getProfile(authStore.user.username);
        authStore.user = { ...authStore.user, ...updatedProfile };
      }
      
      alert('Profile picture updated successfully!');
    } catch (error: any) {
      console.error('Failed to upload profile picture:', error);
      console.error('Error details:', error.response?.data || error.message);
      alert(`Failed to upload profile picture: ${error.response?.data?.error || error.message}`);
    } finally {
      saving.value = false;
    }
  }
};

const saveProfile = async () => {
  // Clear previous errors
  validationErrors.username = '';
  validationErrors.name = '';
  
  // Only validate username if it was changed
  if (editForm.username && editForm.username !== authStore.user?.username) {
    validateUsername();
    if (validationErrors.username) {
      return;
    }
  }

  saving.value = true;
  try {
    // Build update data - only include fields that are filled or changed
    const updateData: any = {};
    
    if (editForm.name && editForm.name.trim()) {
      updateData.name = editForm.name.trim();
    }
    
    // Only include bio if it's different from current value (including empty to allow clearing)
    const currentBio = authStore.user?.bio || '';
    if (editForm.bio !== currentBio) {
      updateData.bio = editForm.bio || '';
    }
    
    if (editForm.gender) {
      updateData.gender = editForm.gender;
    }
    
    if (editForm.username && editForm.username !== authStore.user?.username) {
      updateData.username = editForm.username;
    }
    
    // Check if there are any changes to save
    if (Object.keys(updateData).length === 0) {
      alert('No changes to save');
      saving.value = false;
      return;
    }
    
    await userAPI.updateProfile(updateData);
    
    // Update auth store with ALL updated fields to propagate changes everywhere
    if (authStore.user) {
      if (updateData.name) authStore.user.name = updateData.name;
      if (updateData.username) authStore.user.username = updateData.username;
      if (updateData.bio !== undefined) authStore.user.bio = updateData.bio;
      if (updateData.gender) authStore.user.gender = updateData.gender;
      
      // Reload the full profile to ensure everything is in sync
      if (authStore.user.username) {
        const updatedProfile = await userAPI.getProfile(authStore.user.username);
        authStore.user = { ...authStore.user, ...updatedProfile };
      }
    }
    
    alert('Profile updated successfully!');
  } catch (error: any) {
    console.error('Failed to save profile:', error);
    alert(error.response?.data?.error || 'Failed to save profile');
  } finally {
    saving.value = false;
  }
};

const saveNotificationSettings = async () => {
  try {
    await userAPI.updateNotificationSettings(
      notificationSettings.pushNotifications,
      notificationSettings.emailNotifications
    );
    alert('Notification settings saved successfully!');
  } catch (error) {
    console.error('Failed to save notification settings:', error);
    alert('Failed to save notification settings');
  }
};

const savePrivacySettings = async () => {
  try {
    await userAPI.updatePrivacy(privacySettings.isPrivate);
    // Update auth store to reflect the new privacy setting
    if (authStore.user) {
      authStore.user.is_private = privacySettings.isPrivate;
    }
    alert('Privacy settings saved successfully!');
  } catch (error) {
    console.error('Failed to save privacy settings:', error);
    alert('Failed to save privacy settings');
  }
};

const searchFollowers = async () => {
  if (!closeFriendSearch.value.trim()) {
    searchResults.value = [];
    return;
  }

  try {
    const results = await userAPI.searchUsers(closeFriendSearch.value);
    // Filter out users already in close friends
    const closeFriendIds = closeFriends.value.map(f => f.user_id || f.id);
    searchResults.value = (results.users || []).filter(
      (user: any) => !closeFriendIds.includes(user.user_id || user.id)
    );
  } catch (error) {
    console.error('Failed to search users:', error);
  }
};

const addCloseFriend = async (userId: number) => {
  try {
    await userAPI.addCloseFriend(userId);
    // Reload close friends list
    const response = await userAPI.getCloseFriends();
    closeFriends.value = response.friends || [];
    // Close modal and clear search
    showCloseFriendsModal.value = false;
    closeFriendSearch.value = '';
    searchResults.value = [];
  } catch (error) {
    console.error('Failed to add close friend:', error);
    alert('Failed to add close friend');
  }
};

const removeCloseFriend = async (userId: number) => {
  try {
    await userAPI.removeCloseFriend(userId);
    closeFriends.value = closeFriends.value.filter(f => f.id !== userId);
  } catch (error) {
    console.error('Failed to remove close friend:', error);
    alert('Failed to remove close friend');
  }
};

const unblockUser = async (userId: number) => {
  try {
    await userAPI.unblockUser(userId);
    blockedUsers.value = blockedUsers.value.filter(u => u.id !== userId);
  } catch (error) {
    console.error('Failed to unblock user:', error);
  }
};

const searchUsers = async () => {
  if (!hideStorySearch.value.trim()) {
    hideStorySearchResults.value = [];
    return;
  }

  try {
    const results = await userAPI.searchUsers(hideStorySearch.value);
    // Filter out users already in hidden list
    const hiddenUserIds = hiddenStoryUsers.value.map(u => u.user_id || u.id);
    hideStorySearchResults.value = (results.users || []).filter(
      (user: any) => !hiddenUserIds.includes(user.user_id || user.id)
    );
  } catch (error) {
    console.error('Failed to search users:', error);
  }
};

const addHiddenStoryUser = async (userId: number) => {
  try {
    await userAPI.addHiddenStoryUser(userId);
    // Reload hidden users list
    const response = await userAPI.getHiddenStoryUsers();
    hiddenStoryUsers.value = response.hidden_users || [];
    // Close modal and clear search
    showHideStoryModal.value = false;
    hideStorySearch.value = '';
    hideStorySearchResults.value = [];
  } catch (error) {
    console.error('Failed to hide story from user:', error);
    alert('Failed to hide story from user');
  }
};

const removeHiddenStoryUser = async (userId: number) => {
  try {
    await userAPI.removeHiddenStoryUser(userId);
    hiddenStoryUsers.value = hiddenStoryUsers.value.filter(u => u.id !== userId);
  } catch (error) {
    console.error('Failed to remove hidden story user:', error);
    alert('Failed to remove hidden story user');
  }
};

const handleFaceUpload = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (file) {
    verificationForm.facePicture = file;
    verificationForm.facePicturePreview = URL.createObjectURL(file);
  }
};

const submitVerificationRequest = async () => {
  if (!verificationForm.idCardNumber) {
    validationErrors.idCardNumber = 'ID Card Number is required';
    return;
  }

  submittingVerification.value = true;
  try {
    // TODO: Upload face picture first, then submit request
    alert('Verification request submitted!');
    verificationRequest.value = { status: 'pending' };
  } catch (error) {
    console.error('Failed to submit verification:', error);
    alert('Failed to submit verification request');
  } finally {
    submittingVerification.value = false;
  }
};
</script>

<style scoped>
.settings-page {
  min-height: 100vh;
  background: #000;
  color: #fff;
  padding: 60px 20px 20px;
}

.settings-container {
  max-width: 900px;
  margin: 0 auto;
}

h1 {
  font-size: 28px;
  margin-bottom: 30px;
  font-weight: 600;
}

.settings-tabs {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  border-bottom: 1px solid #262626;
  margin-bottom: 30px;
  padding-bottom: 0;
}

.tab-button {
  padding: 12px 20px;
  background: none;
  border: none;
  color: #8e8e8e;
  font-size: 14px;
  cursor: pointer;
  white-space: nowrap;
  border-bottom: 2px solid transparent;
  transition: all 0.3s;
}

.tab-button:hover {
  color: #fff;
}

.tab-button.active {
  color: #fff;
  border-bottom-color: #fff;
}

.tab-content {
  animation: fadeIn 0.3s;
}

.tab-panel h2 {
  font-size: 20px;
  margin-bottom: 20px;
}

.tab-description {
  color: #8e8e8e;
  margin-bottom: 20px;
}

/* Edit Profile Form */
.form-section {
  max-width: 600px;
}

.form-group {
  margin-bottom: 24px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #fff;
  font-weight: 500;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 12px;
  background: #121212;
  border: 1px solid #262626;
  border-radius: 8px;
  color: #fff;
  font-size: 14px;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  outline: none;
  border-color: #0095f6;
}

.form-group .error {
  display: block;
  color: #ed4956;
  font-size: 12px;
  margin-top: 4px;
}

.profile-pic-upload {
  display: flex;
  align-items: center;
  gap: 20px;
}

.preview-img {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
}

.upload-btn {
  padding: 8px 16px;
  background: #0095f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
}

.save-btn {
  padding: 12px 32px;
  background: #0095f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}

.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Settings Section */
.settings-section {
  background: #121212;
  border-radius: 12px;
  overflow: hidden;
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #262626;
}

.setting-item:last-child {
  border-bottom: none;
}

.setting-info {
  flex: 1;
}

.setting-info .label {
  font-size: 16px;
  font-weight: 500;
  margin-bottom: 4px;
}

.setting-info .description {
  font-size: 14px;
  color: #8e8e8e;
}

/* Toggle Switch */
.toggle {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #262626;
  transition: 0.4s;
  border-radius: 34px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.4s;
  border-radius: 50%;
}

input:checked + .slider {
  background-color: #0095f6;
}

input:checked + .slider:before {
  transform: translateX(22px);
}

/* Lists */
.friends-list {
  margin-top: 20px;
}

.friend-item,
.user-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #121212;
  border-radius: 8px;
  margin-bottom: 8px;
}

.avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
}

.friend-info,
.user-info {
  flex: 1;
}

.username {
  font-weight: 600;
  font-size: 14px;
}

.name {
  font-size: 14px;
  color: #8e8e8e;
}

.remove-btn,
.unblock-btn,
.add-btn-small {
  padding: 6px 16px;
  border-radius: 8px;
  border: none;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.remove-btn {
  background: #262626;
  color: #fff;
}

.unblock-btn {
  background: #0095f6;
  color: #fff;
}

.add-btn-small {
  background: #0095f6;
  color: #fff;
}

.add-btn {
  padding: 10px 20px;
  background: #0095f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 20px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #8e8e8e;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #8e8e8e;
}

/* Verification */
.verification-status {
  max-width: 600px;
}

.status-card {
  background: #121212;
  padding: 30px;
  border-radius: 12px;
  text-align: center;
}

.status-card h3 {
  font-size: 18px;
  margin-bottom: 10px;
}

.verification-form {
  max-width: 600px;
}

.file-upload input[type="file"] {
  margin-bottom: 10px;
}

.file-upload .preview {
  margin-top: 10px;
}

.file-upload .preview img {
  max-width: 200px;
  border-radius: 8px;
}

.submit-btn {
  padding: 12px 32px;
  background: #0095f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #262626;
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #363636;
}

.modal-header h3 {
  font-size: 18px;
  margin: 0;
}

.modal-header .close-btn {
  background: none;
  border: none;
  color: #fff;
  font-size: 24px;
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
}

.search-input {
  width: 100%;
  padding: 12px;
  background: #121212;
  border: 1px solid #363636;
  border-radius: 8px;
  color: #fff;
  margin-bottom: 16px;
}

.users-list {
  max-height: 400px;
  overflow-y: auto;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@media (max-width: 768px) {
  .settings-page {
    padding: 60px 16px 16px;
  }

  h1 {
    font-size: 24px;
  }

  .settings-tabs {
    gap: 5px;
  }

  .tab-button {
    padding: 10px 16px;
    font-size: 13px;
  }
}
</style>
