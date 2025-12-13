import axios, { AxiosInstance, AxiosError } from "axios";

// API Base URL - points to your API Gateway
const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8000";

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
  timeout: 30000, // 30 seconds
});

// Request interceptor - Add JWT token to requests
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("jwt_token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor - Handle errors globally
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    // Only redirect to login if:
    // 1. It's a 401 error
    // 2. User has a token (meaning they were logged in)
    // 3. It's NOT a login/register attempt
    const isAuthEndpoint = error.config?.url?.includes('/auth/login') || 
                          error.config?.url?.includes('/auth/register') ||
                          error.config?.url?.includes('/auth/verify');
    
    if (error.response?.status === 401 && !isAuthEndpoint && localStorage.getItem("jwt_token")) {
      // Unauthorized - clear token and redirect to login
      localStorage.removeItem("jwt_token");
      localStorage.removeItem("user");
      window.location.href = "/login";
    }
    return Promise.reject(error);
  }
);

// Types
export interface RegisterRequest {
  name: string
  username: string
  email: string
  password: string
  confirm_password: string
  gender: string
  date_of_birth: string
  enable_2fa?: boolean
}

export interface LoginRequest {
  email_or_username: string // username, email, or phone
  password: string
}

export interface LoginResponse {
  message?: string
  token?: string
  access_token?: string
  refresh_token?: string
  requires_2fa?: boolean
  is_2fa_required?: boolean
  user_id?: number
  username?: string
  email?: string
  name?: string
  profile_picture_url?: string
  needs_profile_completion?: boolean
}

export interface Verify2FARequest {
  email: string
  otp_code: string
}

export interface RequestOTPRequest {
  email: string
}

export interface ResetPasswordRequest {
  email: string
  otp_code: string
  new_password: string
}

export interface ForgotPasswordRequest {
  email: string
}

export interface GoogleAuthRequest {
  auth_code: string
}

export interface Post {
  id: string
  author_id: number
  caption: string
  author_username: string
  author_profile_url: string
  author_is_verified: boolean
  media_urls: string[]
  created_at: string
  is_reel: boolean
  
  // New Fields
  location?: string
  like_count: number
  comment_count: number
  share_count?: number
  is_liked?: boolean
  is_saved?: boolean
  thumbnail_url?: string
}

export interface UserProfile {
  id: number
  username: string
  name: string
  email: string
  profile_picture_url: string
  bio: string
  is_verified: boolean
  is_private: boolean
  is_banned: boolean
  followers_count: number
  following_count: number
  posts_count: number
}

export interface UploadURLResponse {
  upload_url: string;     // The URL to POST the file to
  final_media_url: string; // The URL to save in our database
}

export interface Story {
  id: string
  media_url: string
  media_type: string
  caption: string
  author_username: string
  author_profile_url: string
  created_at: string
  expires_at?: string
  is_liked?: boolean
  filter_name?: string
  stickers_json?: string
}

export interface StoryGroup {
  user_id: number
  username: string
  user_profile_url: string
  stories: Story[]
  all_seen: boolean
}

// API Functions

// Auth APIs
export const authAPI = {
  // Register new user
  register: async (data: RegisterRequest) => {
    const response = await apiClient.post("/auth/register", data);
    return response.data;
  },

  // Login
  login: async (data: LoginRequest) => {
    const response = await apiClient.post<LoginResponse>("/auth/login", data);
    return response.data;
  },

  // Verify 2FA OTP
  verify2FA: async (data: Verify2FARequest) => {
    const response = await apiClient.post("/auth/login/verify-2fa", data);
    return response.data;
  },

  // Request OTP (for registration)
  requestOTP: async (data: RequestOTPRequest) => {
    const response = await apiClient.post("/auth/send-otp", data);
    return response.data;
  },

  // Verify registration OTP
  verifyRegistrationOTP: async (data: { email: string; otp_code: string }) => {
    const response = await apiClient.post("/auth/verify-otp", data);
    return response.data;
  },

  // Forgot password - request OTP
  forgotPassword: async (data: ForgotPasswordRequest) => {
    const response = await apiClient.post("/auth/password-reset/request", data);
    return response.data;
  },

  // Reset password with OTP
  resetPassword: async (data: ResetPasswordRequest) => {
    const response = await apiClient.post("/auth/password-reset/submit", data);
    return response.data;
  },

  // Google OAuth
  googleAuth: async (data: GoogleAuthRequest) => {
    const response = await apiClient.post<LoginResponse>("/auth/google/callback", data);
    return response.data;
  },

  // Logout
  logout: () => {
    localStorage.removeItem("jwt_token");
    localStorage.removeItem("user");
    window.location.href = "/login";
  },
};

// User APIs
export const userAPI = {
  // Get current user profile
  getProfile: async (username: string) => {
    const response = await apiClient.get(`/users/${username}`);
    return response.data;
  },

  // Get user posts
  getUserPosts: async (username: string, page: number = 1, limit: number = 12) => {
    const response = await apiClient.get(`/users/${username}/posts?page=${page}&limit=${limit}`);
    return response.data;
  },

  // Get user reels
  getUserReels: async (username: string, page: number = 1, limit: number = 12) => {
    const response = await apiClient.get(`/users/${username}/reels?page=${page}&limit=${limit}`);
    return response.data;
  },

  getUserTagged: async (username: string) => {
    const response = await apiClient.get(`/users/${username}/tagged`);
    return response.data;
  },

  // Follow user
  followUser: async (userId: number) => {
    const response = await apiClient.post(`/users/${userId}/follow`);
    return response.data;
  },

  // Unfollow user
  unfollowUser: async (userId: number) => {
    const response = await apiClient.delete(`/users/${userId}/follow`);
    return response.data;
  },

  // Block user
  blockUser: async (userId: number) => {
    const response = await apiClient.post(`/users/${userId}/block`);
    return response.data;
  },

  // Unblock user
  unblockUser: async (userId: number) => {
    const response = await apiClient.delete(`/users/${userId}/block`);
    return response.data;
  },

  // Update profile
  updateProfile: async (data: { name: string; bio: string; gender: string }) => {
    const response = await apiClient.put("/profile/edit", data);
    return response.data;
  },

  // Set privacy
  setPrivacy: async (isPrivate: boolean) => {
    const response = await apiClient.put("/settings/privacy", { is_private: isPrivate });
    return response.data;
  },

  // Submit verification request
  submitVerification: async (data: {
    id_card_number: string
    face_picture_url: string
    reason: string
  }) => {
    const response = await apiClient.post("/profile/verify", data);
    return response.data;
  },

  // Search users
  searchUsers: async (query: string) => {
    const response = await apiClient.get(`/search/users?q=${encodeURIComponent(query)}`);
    return response.data;
  },

  // Get followers list
  getFollowers: async (userId: number) => {
    const response = await apiClient.get(`/users/${userId}/followers`);
    return response.data;
  },

  // Get following list
  getFollowing: async (userId: number) => {
    const response = await apiClient.get(`/users/${userId}/following`);
    return response.data;
  },

  // Get top users by follower count
  getTopUsers: async (limit: number = 5) => {
    const response = await apiClient.get(`/users/top?limit=${limit}`);
    return response.data;
  },

  reportUser: async (userId: number, reason: string) => {
    const response = await apiClient.post("/reports/user", { 
      reported_user_id: userId, 
      reason 
    });
    return response.data;
  }
};

// Media APIs
export const mediaAPI = {
  uploadMedia: async (file: File) => {
    // 1. Get the pre-signed URL from our gateway
    const urlResponse = await apiClient.get<UploadURLResponse>("/media/upload-url", {
      params: {
        filename: file.name,
        type: file.type,
      }
    });

    const { upload_url, final_media_url } = urlResponse.data;

    // 2. Upload the file DIRECTLY to MinIO using the pre-signed PUT URL
    // Note: Use PUT instead of POST for direct file upload
    await axios.put(upload_url, file, {
      headers: {
        "Content-Type": file.type,
      },
    });

    // 3. Return the final URL for our database
    return { media_url: final_media_url };
  },

  // Get pre-signed URL for viewing/downloading media
  getSecureMediaURL: async (objectName: string): Promise<string> => {
    const response = await apiClient.get<{media_url: string}>("/media/secure-url", {
      params: {
        object_name: objectName,
      }
    });
    return response.data.media_url;
  },
};

// Feed APIs
export const feedAPI = {
  getHomeFeed: async (page: number = 1, limit: number = 20) => {
    const response = await apiClient.get(`/feed/home?page=${page}&limit=${limit}`);
    return response.data;
  },

  getExploreFeed: async (page: number = 1, limit: number = 20) => {
    const response = await apiClient.get(`/feed/explore?page=${page}&limit=${limit}`);
    return response.data;
  },

  getReelsFeed: async (page: number = 1, limit: number = 20) => {
    const response = await apiClient.get(`/feed/reels?page=${page}&limit=${limit}`);
    return response.data;
  }
};

// Post APIs
export const postAPI = {
  getPost: async (postId: number) => {
    const response = await apiClient.get(`/posts/${postId}`);
    return response.data;
  },

  createPost: async (data: {
    caption: string
    media_urls: string[]
    comments_disabled?: boolean
    is_reel?: boolean
    collaborator_ids?: number[]
    location?: string
    thumbnail_url?: string
  }) => {
    const response = await apiClient.post("/posts", data);
    return response.data;
  },

  likePost: async (postId: string) => {
    const response = await apiClient.post(`/posts/${postId}/like`);
    return response.data;
  },

  unlikePost: async (postId: string) => {
    const response = await apiClient.delete(`/posts/${postId}/like`);
    return response.data;
  },

  sharePost: async (postId: string, caption?: string) => {
    const response = await apiClient.post(`/posts/${postId}/share`, { caption });
    return response.data;
  },

  // Get post likers
  getPostLikers: async (postId: string) => {
    const response = await apiClient.get(`/posts/${postId}/likes`);
    return response.data;
  },

  summarizeCaption: async (postId: string) => {
    const response = await apiClient.post(`/posts/${postId}/summarize`);
    return response.data;
  },

  deletePost: async (postId: string) => {
    const response = await apiClient.delete(`/posts/${postId}`);
    return response.data;
  },

  reportPost: async (postId: number, reason: string) => {
    const response = await apiClient.post("/reports/post", { 
      post_id: postId, 
      reason 
    });
    return response.data;
  }
};

// Story APIs
export const storyAPI = {
  getStoryFeed: async () => {
    const response = await apiClient.get("/stories/feed");
    return response.data; // Returns StoryGroup[]
  },

  createStory: async (data: {
      media_url: string, 
      media_type: string, 
      caption?: string,
      filter_name?: string,
      stickers_json?: string
  }) => {
    const response = await apiClient.post("/stories", data);
    return response.data;
  },

  likeStory: async (storyId: string) => {
    const response = await apiClient.post(`/stories/${storyId}/like`);
    return response.data;
  },

  unlikeStory: async (storyId: string) => {
    const response = await apiClient.delete(`/stories/${storyId}/like`);
    return response.data;
  },

  viewStory: async (storyId: string) => {
    const response = await apiClient.post(`/stories/${storyId}/view`);
    return response.data;
  },

  getArchive: async () => {
    const response = await apiClient.get("/stories/archive");
    return response.data; // Returns Story[]
  }
};

// Comment APIs
export const commentAPI = {
  getCommentsByPost: async (postId: number, page: number = 1, limit: number = 20) => {
    const response = await apiClient.get(`/posts/${postId}/comments?page=${page}&limit=${limit}`);
    return response.data;
  },

  createComment: async (data: {
    post_id: number
    content: string
    parent_comment_id?: number
  }) => {
    const response = await apiClient.post("/comments", data);
    return response.data;
  },

  deleteComment: async (commentId: string) => {
    const response = await apiClient.delete(`/comments/${commentId}`);
    return response.data;
  },

  likeComment: async (commentId: string) => {
    const response = await apiClient.post(`/comments/${commentId}/like`);
    return response.data;
  },

  unlikeComment: async (commentId: string) => {
    const response = await apiClient.delete(`/comments/${commentId}/like`);
    return response.data;
  }
};

// Collection APIs
export const collectionAPI = {
  create: async (name: string) => {
    const response = await apiClient.post("/collections", { name });
    return response.data;
  },

  getAll: async () => {
    const response = await apiClient.get("/collections");
    return response.data;
  },

  getPosts: async (collectionId: string, page: number = 1, limit: number = 12) => {
    const response = await apiClient.get(`/collections/${collectionId}?page=${page}&limit=${limit}`);
    return response.data;
  },

  getCollectionsForPost: async (postId: string) => {
    const response = await apiClient.get(`/posts/${postId}/collections`);
    return response.data; // Returns { collection_ids: string[] }
  },

  savePost: async (collectionId: string, postId: number) => {
    const response = await apiClient.post(`/collections/${collectionId}/posts`, { post_id: postId });
    return response.data;
  },

  unsavePost: async (collectionId: string, postId: string) => {
    const response = await apiClient.delete(`/collections/${collectionId}/posts/${postId}`);
    return response.data;
  },

  delete: async (collectionId: string) => {
    const response = await apiClient.delete(`/collections/${collectionId}`);
    return response.data;
  },

  rename: async (collectionId: string, newName: string) => {
    const response = await apiClient.put(`/collections/${collectionId}`, { new_name: newName });
    return response.data;
  }
};

// Search APIs
export const searchAPI = {
  users: async (query: string) => {
    const response = await apiClient.get(`/search/users?q=${encodeURIComponent(query)}`);
    return response.data;
  },

  hashtag: async (name: string) => {
    const response = await apiClient.get(`/search/hashtags/${name}`);
    return response.data;
  },

  trending: async () => {
    const response = await apiClient.get("/trending/hashtags");
    return response.data;
  }
};

// Message APIs
export const messageAPI = {
  createConversation: async (data: {
    participant_ids: number[]
    group_name?: string
    group_image_url?: string
  }) => {
    const response = await apiClient.post("/conversations", data);
    return response.data;
  },

  getConversations: async (page: number = 1, limit: number = 20) => {
    const response = await apiClient.get(`/conversations?page=${page}&limit=${limit}`);
    return response.data;
  },

  getMessages: async (conversationId: string, page: number = 1, limit: number = 50) => {
    const response = await apiClient.get(`/conversations/${conversationId}/messages?page=${page}&limit=${limit}`);
    return response.data;
  },

  searchMessages: async (conversationId: string, query: string) => {
    const response = await apiClient.get(`/conversations/${conversationId}/messages/search?q=${encodeURIComponent(query)}`);
    return response.data;
  },

  sendMessage: async (conversationId: string, content: string) => {
    const response = await apiClient.post(`/conversations/${conversationId}/messages`, { content });
    return response.data;
  },

  unsendMessage: async (messageId: string) => {
    const response = await apiClient.delete(`/messages/${messageId}`);
    return response.data;
  },

  deleteConversation: async (conversationId: string) => {
    const response = await apiClient.delete(`/conversations/${conversationId}`);
    return response.data;
  },

  getVideoToken: async (conversationId: string) => {
    const response = await apiClient.get(`/conversations/${conversationId}/video_token`);
    return response.data;
  },

  sendMessageWithMedia: async (conversationId: string, formData: FormData) => {
    const response = await apiClient.post(`/conversations/${conversationId}/messages/media`, formData, {
      headers: {
        "Content-Type": "multipart/form-data"
      }
    });
    return response.data;
  },

  addParticipant: async (conversationId: string, userId: number) => {
    const response = await apiClient.post(`/conversations/${conversationId}/participants`, { user_id: userId });
    return response.data;
  },

  removeParticipant: async (conversationId: string, userId: number) => {
    const response = await apiClient.delete(`/conversations/${conversationId}/participants/${userId}`);
    return response.data;
  }
};

// Report APIs
export const reportAPI = {
  reportPost: async (postId: number, reason: string) => {
    const response = await apiClient.post("/reports/post", { post_id: postId, reason });
    return response.data;
  },

  reportUser: async (userId: number, reason: string) => {
    const response = await apiClient.post("/reports/user", { reported_user_id: userId, reason });
    return response.data;
  }
};

// Admin APIs
export interface PostReport {
  id: number
  reporter_user_id: number
  reporter_username: string
  reported_post_id: number
  reason: string
  is_resolved: boolean
  created_at: string
}

export interface UserReport {
  id: number
  reporter_user_id: number
  reporter_username: string
  reported_user_id: number
  reported_username: string
  reason: string
  is_resolved: boolean
  created_at: string
}

export interface VerificationRequest {
  id: number
  user_id: number
  username: string
  id_card_number: string
  face_picture_url: string
  reason: string
  status: string
  created_at: string
}

export interface UserListItem {
  user_id: number
  username: string
  name: string
  email: string
  profile_picture_url: string
  is_verified: boolean
  is_banned: boolean
  created_at: string
}

export const adminAPI = {
  // User Management
  getAllUsers: async () => {
    const response = await apiClient.get<UserListItem[]>("/admin/users");
    return { users: response.data };
  },

  banUser: async (userId: number) => {
    const response = await apiClient.post(`/admin/users/${userId}/ban`);
    return response.data;
  },

  unbanUser: async (userId: number) => {
    const response = await apiClient.post(`/admin/users/${userId}/unban`);
    return response.data;
  },

  // Reports Management - backend returns arrays directly
  getPostReports: async () => {
    const response = await apiClient.get<PostReport[]>("/admin/reports/posts?unresolved_only=false");
    return { reports: response.data };
  },

  getUserReports: async () => {
    const response = await apiClient.get<UserReport[]>("/admin/reports/users?unresolved_only=false");
    return { reports: response.data };
  },

  resolvePostReport: async (reportId: number, action: "ACCEPT" | "REJECT") => {
    const response = await apiClient.post(`/admin/reports/posts/${reportId}/resolve`, { action });
    return response.data;
  },

  resolveUserReport: async (reportId: number, action: "ACCEPT" | "REJECT") => {
    const response = await apiClient.post(`/admin/reports/users/${reportId}/resolve`, { action });
    return response.data;
  },

  // Verification Requests - backend returns array directly
  getVerifications: async () => {
    const response = await apiClient.get<VerificationRequest[]>("/admin/verifications?status=all");
    return { requests: response.data };
  },

  resolveVerification: async (verificationId: number, action: "APPROVE" | "REJECT", reason?: string) => {
    const response = await apiClient.post(`/admin/verifications/${verificationId}/resolve`, { action, reason });
    return response.data;
  },

  // Newsletter
  sendNewsletter: async (subject: string, content: string) => {
    const response = await apiClient.post("/admin/newsletters", { subject, body: content });
    return response.data;
  }
};

// Notification APIs
export interface NotificationItem {
  id: number
  user_id: number
  actor_id: number
  actor_username: string
  actor_profile_picture_url: string
  actor_is_verified: boolean
  type: string // e.g., "post.liked", "user.followed", "comment.created"
  entity_id: number
  is_read: boolean
  created_at: string
}

export const notificationAPI = {
  getNotifications: async (limit: number = 50) => {
    const response = await apiClient.get<{
      notifications: NotificationItem[]
      unread_count: number
    }>(`/notifications?limit=${limit}`);
    return response.data;
  },

  markAsRead: async (notificationId: number) => {
    const response = await apiClient.put(`/notifications/${notificationId}/read`);
    return response.data;
  },

  markAllAsRead: async () => {
    const response = await apiClient.put("/notifications/read-all");
    return response.data;
  }
};

// Helper function to handle API errors
export const handleApiError = (error: unknown): string => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<{ error?: string; message?: string }>;
    
    // Check if there's a response from server
    if (axiosError.response) {
      const errorMessage = 
        axiosError.response.data?.error || 
        axiosError.response.data?.message || 
        "An error occurred";
      return errorMessage;
    }
    
    // Network error
    if (axiosError.request) {
      return "Network error. Please check your connection.";
    }
  }
  
  return "An unexpected error occurred";
};

// Save token and user to localStorage
export const saveAuthData = (token: string, user?: LoginResponse) => {
  localStorage.setItem("jwt_token", token);
  if (user) {
    localStorage.setItem("user", JSON.stringify(user));
  }
};

// Get stored user data
export const getStoredUser = (): LoginResponse | null => {
  const userStr = localStorage.getItem("user");
  if (userStr) {
    try {
      return JSON.parse(userStr);
    } catch {
      return null;
    }
  }
  return null;
};

// Check if user is authenticated
export const isAuthenticated = (): boolean => {
  return !!localStorage.getItem("jwt_token");
};

export default apiClient;
