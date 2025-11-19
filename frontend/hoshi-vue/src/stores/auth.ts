import { defineStore } from 'pinia'
import { authAPI, saveAuthData, getStoredUser } from '@/services/api'

interface User {
  user_id?: number
  username?: string
  email?: string
  name?: string
  profile_picture_url?: string
}

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  loading: boolean
  error: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: getStoredUser(),
    token: localStorage.getItem('jwt_token'),
    isAuthenticated: !!localStorage.getItem('jwt_token'),
    loading: false,
    error: null
  }),

  getters: {
    currentUser: (state) => state.user,
    isLoggedIn: (state) => state.isAuthenticated,
    userId: (state) => state.user?.user_id
  },

  actions: {
    async register(data: {
      name: string
      username: string
      email: string
      password: string
      confirm_password: string
      gender: string
      date_of_birth: string
      enable_2fa?: boolean
    }) {
      this.loading = true
      this.error = null

      try {
        const response = await authAPI.register(data)
        return { success: true, message: response.message, email: data.email }
      } catch (error: any) {
        this.error = error.response?.data?.error || 'Registration failed'
        throw error
      } finally {
        this.loading = false
      }
    },

    async login(credentials: { email_or_username: string; password: string }) {
      this.loading = true
      this.error = null

      try {
        const response = await authAPI.login(credentials)

        // Check if 2FA is required
        if (response.requires_2fa || response.is_2fa_required) {
          return { 
            requires_2fa: true,
            user_id: response.user_id,
            username: response.username || credentials.email_or_username
          }
        }

        // Set authentication data
        const token = response.token || response.access_token || ''
        this.setAuth(token, response)
        return { requires_2fa: false, success: true }
      } catch (error: any) {
        this.error = error.response?.data?.error || 'Login failed'
        throw error
      } finally {
        this.loading = false
      }
    },

    async verify2FA(data: { email: string; otp_code: string }) {
      this.loading = true
      this.error = null

      try {
        const response = await authAPI.verify2FA(data)

        const token = response.token || response.access_token || ''
        this.setAuth(token, response)
        return { success: true }
      } catch (error: any) {
        this.error = error.response?.data?.error || '2FA verification failed'
        throw error
      } finally {
        this.loading = false
      }
    },

    async verifyRegistrationOTP(data: { email: string; otp_code: string }) {
      this.loading = true
      this.error = null

      try {
        const response = await authAPI.verifyRegistrationOTP(data)
        // Don't auto-login after registration verification
        // User should login with credentials
        return { success: true, message: response.message || 'Account verified successfully' }
      } catch (error: any) {
        this.error = error.response?.data?.error || 'OTP verification failed'
        throw error
      } finally {
        this.loading = false
      }
    },

    async requestOTP(data: { email: string }) {
      this.loading = true
      this.error = null

      try {
        const response = await authAPI.requestOTP(data)
        return { success: true, message: response.message || 'OTP sent successfully' }
      } catch (error: any) {
        this.error = error.response?.data?.error || 'Failed to send OTP'
        throw error
      } finally {
        this.loading = false
      }
    },

    setAuth(token: string, userData: any) {
      this.token = token
      this.user = {
        user_id: userData.user_id,
        username: userData.username,
        email: userData.email,
        name: userData.name,
        profile_picture_url: userData.profile_picture_url
      }
      this.isAuthenticated = true
      saveAuthData(token, userData)
    },

    logout() {
      this.token = null
      this.user = null
      this.isAuthenticated = false
      this.error = null
      authAPI.logout()
    },

    clearError() {
      this.error = null
    }
  }
})
