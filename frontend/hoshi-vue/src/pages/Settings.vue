<template>
  <div class="settings-page">
    <div class="settings-container">
      <h1>Settings and privacy</h1>

      <div class="settings-sections">
        <!-- Account Settings -->
        <div class="settings-section">
          <h2>Account</h2>
          <div class="setting-item">
            <div class="setting-info">
              <div class="label">Email</div>
              <div class="value">user@example.com</div>
            </div>
            <button class="edit-btn">Edit</button>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <div class="label">Password</div>
              <div class="value">••••••••</div>
            </div>
            <button class="edit-btn">Change</button>
          </div>
        </div>

        <!-- Privacy Settings -->
        <div class="settings-section">
          <h2>Privacy and safety</h2>
          <div class="setting-item toggle">
            <div class="setting-info">
              <div class="label">Private account</div>
              <div class="value">Make your profile private</div>
            </div>
            <input type="checkbox" class="toggle-switch" />
          </div>
          <div class="setting-item toggle">
            <div class="setting-info">
              <div class="label">Allow comments</div>
              <div class="value">Anyone can comment on your posts</div>
            </div>
            <input type="checkbox" class="toggle-switch" checked />
          </div>
        </div>

        <!-- Notifications -->
        <div class="settings-section">
          <h2>Notifications</h2>
          <div class="setting-item toggle">
            <div class="setting-info">
              <div class="label">Push notifications</div>
              <div class="value">Get updates about your account</div>
            </div>
            <input type="checkbox" class="toggle-switch" checked />
          </div>
          <div class="setting-item toggle">
            <div class="setting-info">
              <div class="label">Email notifications</div>
              <div class="value">Receive emails about activity</div>
            </div>
            <input type="checkbox" class="toggle-switch" />
          </div>
        </div>

        <!-- Help -->
        <div class="settings-section">
          <h2>Help</h2>
          <div class="setting-item">
            <button class="text-btn">Help Center</button>
          </div>
          <div class="setting-item">
            <button class="text-btn">Report a problem</button>
          </div>
          <div class="setting-item">
            <button class="text-btn">Terms of Service</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped lang="scss">
.settings-page {
  width: 100%;
  padding: 20px;
  padding-left: calc(244px + 20px);
  background-color: #000;
  min-height: 100vh;
}

.settings-container {
  max-width: 600px;

  h1 {
    font-size: 32px;
    font-weight: 700;
    margin-bottom: 32px;
  }
}

.settings-sections {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.settings-section {
  border-top: 1px solid #262626;
  padding-top: 16px;

  h2 {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 12px;
    color: #fff;
  }

  .setting-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 0;
    border-bottom: 1px solid #262626;

    &:last-child {
      border-bottom: none;
    }

    .setting-info {
      flex: 1;

      .label {
        font-weight: 500;
        font-size: 14px;
        margin-bottom: 4px;
      }

      .value {
        font-size: 12px;
        color: #a8a8a8;
      }
    }

    .edit-btn {
      background: none;
      border: none;
      color: #0a66c2;
      cursor: pointer;
      font-weight: 600;
      font-size: 14px;
      padding: 0;
    }

    .text-btn {
      background: none;
      border: none;
      color: #fff;
      cursor: pointer;
      font-size: 14px;
      padding: 0;

      &:hover {
        text-decoration: underline;
      }
    }

    .toggle-switch {
      width: 40px;
      height: 24px;
      cursor: pointer;
    }
  }
}

@media (max-width: 1024px) {
  .settings-page {
    padding-left: calc(72px + 20px);
  }
}

@media (max-width: 768px) {
  .settings-page {
    padding-left: 20px;
  }
}
</style>
