import { defineStore } from "pinia";

interface UserProfile {
  user_id: number
  name: string
  username: string
  bio: string
  profile_picture_url: string
  is_verified: boolean
  follower_count: number
  following_count: number
  is_followed_by_self: boolean
  mutual_follower_count: number
  gender: string
  is_private: boolean
}

interface UserState {
  profiles: Record<string, UserProfile>
  loading: boolean
  error: string | null
}

export const useUserStore = defineStore("user", {
  state: (): UserState => ({
    profiles: {},
    loading: false,
    error: null
  }),

  getters: {
    getProfile: (state) => (username: string) => state.profiles[username]
  },

  actions: {
    async fetchUserProfile(username: string) {
      this.loading = true;
      this.error = null;

      try {
        // TODO: Call API
        // const response = await userAPI.getProfile(username)
        // this.profiles[username] = response
        // return response
        console.warn("fetchUserProfile not implemented for", username);
      } catch (error: any) {
        this.error = error.response?.data?.error || "Failed to load profile";
        throw error;
      } finally {
        this.loading = false;
      }
    },

    async followUser(userId: number, username: string) {
      try {
        // TODO: Call API
        // await userAPI.followUser(userId)
        
        // Update local state optimistically
        if (this.profiles[username]) {
          this.profiles[username].is_followed_by_self = true;
          this.profiles[username].follower_count++;
        }
      } catch (error) {
        // Revert on error
        if (this.profiles[username]) {
          this.profiles[username].is_followed_by_self = false;
          this.profiles[username].follower_count--;
        }
        throw error;
      }
    },

    async unfollowUser(userId: number, username: string) {
      try {
        // TODO: Call API
        // await userAPI.unfollowUser(userId)
        
        // Update local state optimistically
        if (this.profiles[username]) {
          this.profiles[username].is_followed_by_self = false;
          this.profiles[username].follower_count--;
        }
      } catch (error) {
        // Revert on error
        if (this.profiles[username]) {
          this.profiles[username].is_followed_by_self = true;
          this.profiles[username].follower_count++;
        }
        throw error;
      }
    },

    clearCache() {
      this.profiles = {};
    }
  }
});
