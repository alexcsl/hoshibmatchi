<template>
  <div class="admin-page">
    <div class="admin-container">
      <h1 class="page-title">Admin Dashboard</h1>
      
      <!-- Tab Navigation -->
      <div class="tabs">
        <button 
          v-for="tab in tabs" 
          :key="tab.id"
          class="tab-btn"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
          <span v-if="getTabBadge(tab.id)" class="badge">{{ getTabBadge(tab.id) }}</span>
        </button>
      </div>

      <!-- Tab Content -->
      <div class="tab-content">
        <!-- Users Management -->
        <div v-if="activeTab === 'users'" class="section">
          <div class="section-header">
            <h2>User Management</h2>
            <div class="search-box">
              <input 
                v-model="userSearchQuery" 
                type="text" 
                placeholder="Search users..." 
              />
            </div>
          </div>
          
          <div v-if="loadingUsers" class="loading">Loading users...</div>
          
          <div v-else class="users-list">
            <div 
              v-for="user in filteredUsers" 
              :key="user.user_id" 
              class="user-item"
            >
              <img 
                :src="user.profile_picture_url || '/default-avatar.svg'" 
                :alt="user.username"
                class="user-avatar"
              />
              <div class="user-info">
                <div class="user-name">
                  {{ user.name }}
                  <span v-if="user.is_verified" class="verified-badge">✓</span>
                </div>
                <div class="user-username">@{{ user.username }}</div>
                <div class="user-email">{{ user.email }}</div>
              </div>
              <div class="user-actions">
                <span 
                  v-if="user.is_banned" 
                  class="status-badge banned"
                >
                  Banned
                </span>
                <button 
                  v-if="!user.is_banned"
                  class="btn-ban"
                  @click="banUser(user.user_id, user.username)"
                  :disabled="actionLoading"
                >
                  Ban User
                </button>
                <button 
                  v-else
                  class="btn-unban"
                  @click="unbanUser(user.user_id, user.username)"
                  :disabled="actionLoading"
                >
                  Unban User
                </button>
              </div>
            </div>
            
            <div v-if="filteredUsers.length === 0" class="empty-state">
              No users found
            </div>
          </div>
        </div>

        <!-- Post Reports -->
        <div v-if="activeTab === 'post-reports'" class="section">
          <div class="section-header">
            <h2>Post Reports</h2>
            <div class="filter-buttons">
              <button 
                :class="{ active: postReportFilter === 'all' }"
                @click="postReportFilter = 'all'"
              >
                All
              </button>
              <button 
                :class="{ active: postReportFilter === 'pending' }"
                @click="postReportFilter = 'pending'"
              >
                Pending
              </button>
              <button 
                :class="{ active: postReportFilter === 'resolved' }"
                @click="postReportFilter = 'resolved'"
              >
                Resolved
              </button>
            </div>
          </div>
          
          <div v-if="loadingPostReports" class="loading">Loading reports...</div>
          
          <div v-else class="reports-list">
            <div 
              v-for="report in filteredPostReports" 
              :key="report.id" 
              class="report-item"
            >
              <div class="report-header">
                <span class="report-id">#{{ report.id }}</span>
                <span 
                  class="report-status"
                  :class="{ resolved: report.is_resolved }"
                >
                  {{ report.is_resolved ? 'Resolved' : 'Pending' }}
                </span>
              </div>
              <div class="report-details">
                <p><strong>Reporter:</strong> @{{ report.reporter_username }}</p>
                <p><strong>Post ID:</strong> {{ report.reported_post_id }}</p>
                <p><strong>Reason:</strong> {{ report.reason }}</p>
                <p><strong>Date:</strong> {{ formatDate(report.created_at) }}</p>
              </div>
              <div v-if="!report.is_resolved" class="report-actions">
                <button 
                  class="btn-accept"
                  @click="resolvePostReport(report.id, 'ACCEPT')"
                  :disabled="actionLoading"
                >
                  Accept & Delete Post
                </button>
                <button 
                  class="btn-reject"
                  @click="resolvePostReport(report.id, 'REJECT')"
                  :disabled="actionLoading"
                >
                  Reject
                </button>
              </div>
            </div>
            
            <div v-if="filteredPostReports.length === 0" class="empty-state">
              No reports found
            </div>
          </div>
        </div>

        <!-- User Reports -->
        <div v-if="activeTab === 'user-reports'" class="section">
          <div class="section-header">
            <h2>User Reports</h2>
            <div class="filter-buttons">
              <button 
                :class="{ active: userReportFilter === 'all' }"
                @click="userReportFilter = 'all'"
              >
                All
              </button>
              <button 
                :class="{ active: userReportFilter === 'pending' }"
                @click="userReportFilter = 'pending'"
              >
                Pending
              </button>
              <button 
                :class="{ active: userReportFilter === 'resolved' }"
                @click="userReportFilter = 'resolved'"
              >
                Resolved
              </button>
            </div>
          </div>
          
          <div v-if="loadingUserReports" class="loading">Loading reports...</div>
          
          <div v-else class="reports-list">
            <div 
              v-for="report in filteredUserReports" 
              :key="report.id" 
              class="report-item"
            >
              <div class="report-header">
                <span class="report-id">#{{ report.id }}</span>
                <span 
                  class="report-status"
                  :class="{ resolved: report.is_resolved }"
                >
                  {{ report.is_resolved ? 'Resolved' : 'Pending' }}
                </span>
              </div>
              <div class="report-details">
                <p><strong>Reporter:</strong> @{{ report.reporter_username }}</p>
                <p><strong>Reported User:</strong> @{{ report.reported_username }} (ID: {{ report.reported_user_id }})</p>
                <p><strong>Reason:</strong> {{ report.reason }}</p>
                <p><strong>Date:</strong> {{ formatDate(report.created_at) }}</p>
              </div>
              <div v-if="!report.is_resolved" class="report-actions">
                <button 
                  class="btn-accept"
                  @click="resolveUserReport(report.id, 'ACCEPT')"
                  :disabled="actionLoading"
                >
                  Accept & Ban User
                </button>
                <button 
                  class="btn-reject"
                  @click="resolveUserReport(report.id, 'REJECT')"
                  :disabled="actionLoading"
                >
                  Reject
                </button>
              </div>
            </div>
            
            <div v-if="filteredUserReports.length === 0" class="empty-state">
              No reports found
            </div>
          </div>
        </div>

        <!-- Verification Requests -->
        <div v-if="activeTab === 'verifications'" class="section">
          <div class="section-header">
            <h2>Verification Requests</h2>
            <div class="filter-buttons">
              <button 
                :class="{ active: verificationFilter === 'all' }"
                @click="verificationFilter = 'all'"
              >
                All
              </button>
              <button 
                :class="{ active: verificationFilter === 'pending' }"
                @click="verificationFilter = 'pending'"
              >
                Pending
              </button>
              <button 
                :class="{ active: verificationFilter === 'approved' }"
                @click="verificationFilter = 'approved'"
              >
                Approved
              </button>
              <button 
                :class="{ active: verificationFilter === 'rejected' }"
                @click="verificationFilter = 'rejected'"
              >
                Rejected
              </button>
            </div>
          </div>
          
          <div v-if="loadingVerifications" class="loading">Loading requests...</div>
          
          <div v-else class="verifications-list">
            <div 
              v-for="request in filteredVerifications" 
              :key="request.id" 
              class="verification-item"
            >
              <div class="verification-header">
                <span class="verification-id">#{{ request.id }}</span>
                <span 
                  class="verification-status"
                  :class="request.status.toLowerCase()"
                >
                  {{ request.status }}
                </span>
              </div>
              <div class="verification-details">
                <p><strong>User:</strong> @{{ request.username }} (ID: {{ request.user_id }})</p>
                <p><strong>ID Card Number:</strong> {{ request.id_card_number }}</p>
                <p><strong>Reason:</strong> {{ request.reason }}</p>
                <p><strong>Date:</strong> {{ formatDate(request.created_at) }}</p>
                <div class="verification-image">
                  <strong>Face Picture:</strong>
                  <img 
                    :src="request.face_picture_url" 
                    alt="Verification photo"
                    @click="showImageModal(request.face_picture_url)"
                  />
                </div>
              </div>
              <div v-if="request.status === 'pending'" class="verification-actions">
                <button 
                  class="btn-approve"
                  @click="resolveVerification(request.id, 'APPROVE')"
                  :disabled="actionLoading"
                >
                  Approve
                </button>
                <button 
                  class="btn-reject"
                  @click="showRejectModal(request.id)"
                  :disabled="actionLoading"
                >
                  Reject
                </button>
              </div>
            </div>
            
            <div v-if="filteredVerifications.length === 0" class="empty-state">
              No verification requests found
            </div>
          </div>
        </div>

        <!-- Newsletter -->
        <div v-if="activeTab === 'newsletter'" class="section">
          <div class="section-header">
            <h2>Send Newsletter</h2>
            <p class="description">Send newsletters to all subscribed users</p>
          </div>
          
          <div class="newsletter-form">
            <div class="form-group">
              <label>Subject</label>
              <input 
                v-model="newsletterSubject" 
                type="text" 
                placeholder="Enter newsletter subject"
                class="form-input"
              />
            </div>
            
            <div class="form-group">
              <label>Content</label>
              <textarea 
                v-model="newsletterContent" 
                placeholder="Enter newsletter content"
                rows="10"
                class="form-textarea"
              ></textarea>
            </div>
            
            <button 
              class="btn-send-newsletter"
              @click="sendNewsletter"
              :disabled="!newsletterSubject || !newsletterContent || actionLoading"
            >
              {{ actionLoading ? 'Sending...' : 'Send Newsletter' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Image Modal -->
    <div v-if="showImageModalFlag" class="modal-overlay" @click="closeImageModal">
      <div class="modal-content image-modal" @click.stop>
        <button class="modal-close" @click="closeImageModal">×</button>
        <img :src="modalImageUrl" alt="Verification photo" />
      </div>
    </div>

    <!-- Reject Reason Modal -->
    <div v-if="showRejectModalFlag" class="modal-overlay" @click="closeRejectModal">
      <div class="modal-content reject-modal" @click.stop>
        <button class="modal-close" @click="closeRejectModal">×</button>
        <h3>Reject Verification Request</h3>
        <p>Please provide a reason for rejection:</p>
        <textarea 
          v-model="rejectReason" 
          placeholder="Enter rejection reason"
          rows="4"
          class="form-textarea"
        ></textarea>
        <div class="modal-actions">
          <button class="btn-cancel" @click="closeRejectModal">Cancel</button>
          <button 
            class="btn-confirm-reject"
            @click="confirmRejectVerification"
            :disabled="!rejectReason || actionLoading"
          >
            Confirm Reject
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { adminAPI, type PostReport, type UserReport, type VerificationRequest, type UserListItem } from '@/services/api'
import { useRouter } from 'vue-router'

const router = useRouter()
const authStore = useAuthStore()

// Check if user is admin (you'll need to add role to auth store)
const isAdmin = computed(() => {
  // For now, we'll check the token. You might want to add role to the user object
  const token = authStore.token
  if (!token) return false
  
  try {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function(c) {
        return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2)
    }).join(''))
    const decoded = JSON.parse(jsonPayload)
    return decoded.role === 'admin'
  } catch {
    return false
  }
})

// Redirect if not admin
onMounted(() => {
  if (!isAdmin.value) {
    router.push('/feed')
  } else {
    loadData()
  }
})

// Tab Management
const activeTab = ref('users')
const tabs = [
  { id: 'users', label: 'Users' },
  { id: 'post-reports', label: 'Post Reports' },
  { id: 'user-reports', label: 'User Reports' },
  { id: 'verifications', label: 'Verifications' },
  { id: 'newsletter', label: 'Newsletter' }
]

// State
const loadingUsers = ref(false)
const loadingPostReports = ref(false)
const loadingUserReports = ref(false)
const loadingVerifications = ref(false)
const actionLoading = ref(false)

const users = ref<UserListItem[]>([])
const postReports = ref<PostReport[]>([])
const userReports = ref<UserReport[]>([])
const verifications = ref<VerificationRequest[]>([])

// Filters
const userSearchQuery = ref('')
const postReportFilter = ref<'all' | 'pending' | 'resolved'>('all')
const userReportFilter = ref<'all' | 'pending' | 'resolved'>('all')
const verificationFilter = ref<'all' | 'pending' | 'approved' | 'rejected'>('all')

// Newsletter
const newsletterSubject = ref('')
const newsletterContent = ref('')

// Modals
const showImageModalFlag = ref(false)
const modalImageUrl = ref('')
const showRejectModalFlag = ref(false)
const rejectReason = ref('')
const rejectVerificationId = ref(0)

// Computed
const filteredUsers = computed(() => {
  if (!userSearchQuery.value) return users.value
  const query = userSearchQuery.value.toLowerCase()
  return users.value.filter(user => 
    user.username.toLowerCase().includes(query) ||
    user.name.toLowerCase().includes(query) ||
    user.email.toLowerCase().includes(query)
  )
})

const filteredPostReports = computed(() => {
  if (postReportFilter.value === 'all') return postReports.value
  if (postReportFilter.value === 'pending') return postReports.value.filter(r => !r.is_resolved)
  return postReports.value.filter(r => r.is_resolved)
})

const filteredUserReports = computed(() => {
  if (userReportFilter.value === 'all') return userReports.value
  if (userReportFilter.value === 'pending') return userReports.value.filter(r => !r.is_resolved)
  return userReports.value.filter(r => r.is_resolved)
})

const filteredVerifications = computed(() => {
  if (verificationFilter.value === 'all') return verifications.value
  return verifications.value.filter(v => v.status.toLowerCase() === verificationFilter.value)
})

const getTabBadge = (tabId: string): number => {
  if (tabId === 'post-reports') {
    return postReports.value.filter(r => !r.is_resolved).length
  }
  if (tabId === 'user-reports') {
    return userReports.value.filter(r => !r.is_resolved).length
  }
  if (tabId === 'verifications') {
    return verifications.value.filter(v => v.status === 'pending').length
  }
  return 0
}

// Load Data
async function loadData() {
  await Promise.all([
    loadUsers(),
    loadPostReports(),
    loadUserReports(),
    loadVerifications()
  ])
}

async function loadUsers() {
  loadingUsers.value = true
  try {
    const response = await adminAPI.getAllUsers()
    users.value = response.users || []
  } catch (error) {
    console.error('Failed to load users:', error)
  } finally {
    loadingUsers.value = false
  }
}

async function loadPostReports() {
  loadingPostReports.value = true
  try {
    const response = await adminAPI.getPostReports()
    postReports.value = response.reports || []
  } catch (error) {
    console.error('Failed to load post reports:', error)
  } finally {
    loadingPostReports.value = false
  }
}

async function loadUserReports() {
  loadingUserReports.value = true
  try {
    const response = await adminAPI.getUserReports()
    userReports.value = response.reports || []
  } catch (error) {
    console.error('Failed to load user reports:', error)
  } finally {
    loadingUserReports.value = false
  }
}

async function loadVerifications() {
  loadingVerifications.value = true
  try {
    const response = await adminAPI.getVerifications()
    verifications.value = response.requests || []
  } catch (error) {
    console.error('Failed to load verifications:', error)
  } finally {
    loadingVerifications.value = false
  }
}

// Actions
async function banUser(userId: number, username: string) {
  if (!confirm(`Are you sure you want to ban @${username}?`)) return
  
  actionLoading.value = true
  try {
    await adminAPI.banUser(userId)
    alert(`User @${username} has been banned successfully`)
    await loadUsers()
  } catch (error) {
    console.error('Failed to ban user:', error)
    alert('Failed to ban user')
  } finally {
    actionLoading.value = false
  }
}

async function unbanUser(userId: number, username: string) {
  if (!confirm(`Are you sure you want to unban @${username}?`)) return
  
  actionLoading.value = true
  try {
    await adminAPI.unbanUser(userId)
    alert(`User @${username} has been unbanned successfully`)
    await loadUsers()
  } catch (error) {
    console.error('Failed to unban user:', error)
    alert('Failed to unban user')
  } finally {
    actionLoading.value = false
  }
}

async function resolvePostReport(reportId: number, action: 'ACCEPT' | 'REJECT') {
  const message = action === 'ACCEPT' 
    ? 'This will delete the reported post. Continue?' 
    : 'Are you sure you want to reject this report?'
  
  if (!confirm(message)) return
  
  actionLoading.value = true
  try {
    await adminAPI.resolvePostReport(reportId, action)
    alert(`Report has been ${action === 'ACCEPT' ? 'accepted and post deleted' : 'rejected'}`)
    await loadPostReports()
  } catch (error) {
    console.error('Failed to resolve report:', error)
    alert('Failed to resolve report')
  } finally {
    actionLoading.value = false
  }
}

async function resolveUserReport(reportId: number, action: 'ACCEPT' | 'REJECT') {
  const message = action === 'ACCEPT' 
    ? 'This will ban the reported user. Continue?' 
    : 'Are you sure you want to reject this report?'
  
  if (!confirm(message)) return
  
  actionLoading.value = true
  try {
    await adminAPI.resolveUserReport(reportId, action)
    alert(`Report has been ${action === 'ACCEPT' ? 'accepted and user banned' : 'rejected'}`)
    await loadUserReports()
    if (action === 'ACCEPT') {
      await loadUsers() // Refresh users list
    }
  } catch (error) {
    console.error('Failed to resolve report:', error)
    alert('Failed to resolve report')
  } finally {
    actionLoading.value = false
  }
}

async function resolveVerification(verificationId: number, action: 'APPROVE' | 'REJECT', reason?: string) {
  actionLoading.value = true
  try {
    await adminAPI.resolveVerification(verificationId, action, reason)
    alert(`Verification request has been ${action.toLowerCase()}d. Email notification sent.`)
    await loadVerifications()
    if (action === 'APPROVE') {
      await loadUsers() // Refresh users list
    }
  } catch (error) {
    console.error('Failed to resolve verification:', error)
    alert('Failed to resolve verification request')
  } finally {
    actionLoading.value = false
  }
}

async function sendNewsletter() {
  if (!confirm('Send newsletter to all subscribed users?')) return
  
  actionLoading.value = true
  try {
    await adminAPI.sendNewsletter(newsletterSubject.value, newsletterContent.value)
    alert('Newsletter sent successfully!')
    newsletterSubject.value = ''
    newsletterContent.value = ''
  } catch (error) {
    console.error('Failed to send newsletter:', error)
    alert('Failed to send newsletter')
  } finally {
    actionLoading.value = false
  }
}

// Modal functions
function showImageModal(url: string) {
  modalImageUrl.value = url
  showImageModalFlag.value = true
}

function closeImageModal() {
  showImageModalFlag.value = false
  modalImageUrl.value = ''
}

function showRejectModal(verificationId: number) {
  rejectVerificationId.value = verificationId
  rejectReason.value = ''
  showRejectModalFlag.value = true
}

function closeRejectModal() {
  showRejectModalFlag.value = false
  rejectReason.value = ''
  rejectVerificationId.value = 0
}

function confirmRejectVerification() {
  resolveVerification(rejectVerificationId.value, 'REJECT', rejectReason.value)
  closeRejectModal()
}

// Utility
function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', { 
    year: 'numeric', 
    month: 'short', 
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped>
.admin-page {
  min-height: 100vh;
  background: #000;
  color: #fff;
  padding: 20px;
}

.admin-container {
  max-width: 1200px;
  margin: 0 auto;
}

.page-title {
  font-size: 32px;
  font-weight: 700;
  margin-bottom: 30px;
}

/* Tabs */
.tabs {
  display: flex;
  gap: 10px;
  margin-bottom: 30px;
  border-bottom: 1px solid #262626;
  overflow-x: auto;
}

.tab-btn {
  padding: 12px 20px;
  background: none;
  border: none;
  color: #a8a8a8;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
  white-space: nowrap;
  position: relative;
}

.tab-btn:hover {
  color: #fff;
}

.tab-btn.active {
  color: #fff;
  border-bottom-color: #fff;
}

.tab-btn .badge {
  background: #ed4956;
  color: #fff;
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 11px;
  margin-left: 6px;
}

/* Section */
.section {
  background: #000;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h2 {
  font-size: 24px;
  font-weight: 600;
}

.description {
  color: #a8a8a8;
  font-size: 14px;
  margin-top: 5px;
}

/* Search Box */
.search-box input {
  width: 300px;
  padding: 8px 16px;
  background: #262626;
  border: 1px solid #262626;
  border-radius: 8px;
  color: #fff;
  font-size: 14px;
}

.search-box input:focus {
  outline: none;
  border-color: #555;
}

/* Filter Buttons */
.filter-buttons {
  display: flex;
  gap: 10px;
}

.filter-buttons button {
  padding: 6px 16px;
  background: #262626;
  border: 1px solid #262626;
  border-radius: 8px;
  color: #a8a8a8;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-buttons button:hover {
  background: #363636;
}

.filter-buttons button.active {
  background: #0095f6;
  border-color: #0095f6;
  color: #fff;
}

/* Loading & Empty States */
.loading {
  text-align: center;
  padding: 40px;
  color: #a8a8a8;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #a8a8a8;
  font-size: 16px;
}

/* Users List */
.users-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.user-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #262626;
  border-radius: 12px;
  transition: background 0.2s;
}

.user-item:hover {
  background: #2a2a2a;
}

.user-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  object-fit: cover;
}

.user-info {
  flex: 1;
}

.user-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.verified-badge {
  color: #0095f6;
  font-size: 14px;
}

.user-username {
  color: #a8a8a8;
  font-size: 14px;
  margin-bottom: 2px;
}

.user-email {
  color: #737373;
  font-size: 13px;
}

.user-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-badge {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.status-badge.banned {
  background: #ed4956;
  color: #fff;
}

.btn-ban, .btn-unban {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-ban {
  background: #ed4956;
  color: #fff;
}

.btn-ban:hover {
  background: #d63447;
}

.btn-unban {
  background: #00ba7c;
  color: #fff;
}

.btn-unban:hover {
  background: #00a86b;
}

.btn-ban:disabled, .btn-unban:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Reports List */
.reports-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-item {
  padding: 20px;
  background: #262626;
  border-radius: 12px;
}

.report-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.report-id {
  font-size: 14px;
  color: #a8a8a8;
  font-weight: 600;
}

.report-status {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  background: #555;
  color: #fff;
}

.report-status.resolved {
  background: #00ba7c;
}

.report-details p {
  margin-bottom: 8px;
  font-size: 14px;
  line-height: 1.5;
}

.report-details strong {
  color: #a8a8a8;
}

.report-actions {
  display: flex;
  gap: 12px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #363636;
}

.btn-accept, .btn-reject {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-accept {
  background: #ed4956;
  color: #fff;
}

.btn-accept:hover {
  background: #d63447;
}

.btn-reject {
  background: #363636;
  color: #fff;
}

.btn-reject:hover {
  background: #404040;
}

.btn-accept:disabled, .btn-reject:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Verifications List */
.verifications-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.verification-item {
  padding: 20px;
  background: #262626;
  border-radius: 12px;
}

.verification-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.verification-id {
  font-size: 14px;
  color: #a8a8a8;
  font-weight: 600;
}

.verification-status {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  text-transform: capitalize;
}

.verification-status.pending {
  background: #f5a623;
  color: #000;
}

.verification-status.approved {
  background: #00ba7c;
  color: #fff;
}

.verification-status.rejected {
  background: #ed4956;
  color: #fff;
}

.verification-details p {
  margin-bottom: 8px;
  font-size: 14px;
  line-height: 1.5;
}

.verification-details strong {
  color: #a8a8a8;
}

.verification-image {
  margin-top: 12px;
}

.verification-image img {
  width: 200px;
  height: 200px;
  object-fit: cover;
  border-radius: 8px;
  margin-top: 8px;
  cursor: pointer;
  transition: transform 0.2s;
}

.verification-image img:hover {
  transform: scale(1.05);
}

.verification-actions {
  display: flex;
  gap: 12px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #363636;
}

.btn-approve {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
  background: #00ba7c;
  color: #fff;
}

.btn-approve:hover {
  background: #00a86b;
}

.btn-approve:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Newsletter Form */
.newsletter-form {
  max-width: 800px;
}

.form-group {
  margin-bottom: 24px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #a8a8a8;
}

.form-input, .form-textarea {
  width: 100%;
  padding: 12px;
  background: #262626;
  border: 1px solid #262626;
  border-radius: 8px;
  color: #fff;
  font-size: 14px;
  font-family: inherit;
}

.form-input:focus, .form-textarea:focus {
  outline: none;
  border-color: #555;
}

.form-textarea {
  resize: vertical;
  min-height: 120px;
}

.btn-send-newsletter {
  padding: 12px 24px;
  background: #0095f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-send-newsletter:hover {
  background: #0084e0;
}

.btn-send-newsletter:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Modals */
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

.modal-content {
  background: #262626;
  border-radius: 12px;
  padding: 24px;
  position: relative;
  max-width: 90vw;
  max-height: 90vh;
}

.modal-close {
  position: absolute;
  top: 12px;
  right: 12px;
  background: none;
  border: none;
  color: #fff;
  font-size: 32px;
  cursor: pointer;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background 0.2s;
}

.modal-close:hover {
  background: #363636;
}

.image-modal {
  padding: 0;
}

.image-modal img {
  max-width: 80vw;
  max-height: 80vh;
  border-radius: 12px;
}

.reject-modal {
  max-width: 500px;
  width: 100%;
}

.reject-modal h3 {
  font-size: 20px;
  margin-bottom: 12px;
}

.reject-modal p {
  color: #a8a8a8;
  margin-bottom: 16px;
}

.reject-modal .form-textarea {
  margin-bottom: 16px;
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.btn-cancel, .btn-confirm-reject {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-cancel {
  background: #363636;
  color: #fff;
}

.btn-cancel:hover {
  background: #404040;
}

.btn-confirm-reject {
  background: #ed4956;
  color: #fff;
}

.btn-confirm-reject:hover {
  background: #d63447;
}

.btn-confirm-reject:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
