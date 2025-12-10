<template>
  <div class="messages-page">
    <!-- Video Call Modal -->
    <div
      v-if="showVideoCall"
      class="video-call-overlay"
      @click="endVideoCall"
    >
      <div
        class="video-call-modal"
        @click.stop
      >
        <div class="video-call-header">
          <h3>Video Call with {{ activeConversationName }}</h3>
          <button
            class="close-btn"
            @click="endVideoCall"
          >
            ✕
          </button>
        </div>
        <div class="video-container">
          <video
            ref="localVideoRef"
            autoplay
            muted
            class="local-video"
          ></video>
          <video
            ref="remoteVideoRef"
            autoplay
            class="remote-video"
          ></video>
        </div>
        <div class="video-controls">
          <button
            :class="{ muted: isMuted }"
            @click="toggleMute"
          >
            {{ isMuted ? '🔇' : '🔊' }}
          </button>
          <button
            :class="{ 'video-off': !isVideoEnabled }"
            @click="toggleVideo"
          >
            {{ isVideoEnabled ? '📹' : '📷' }}
          </button>
          <button
            class="end-call-btn"
            @click="endVideoCall"
          >
            📞 End Call
          </button>
        </div>
      </div>
    </div>

    <!-- New Conversation Modal -->
    <div
      v-if="showNewConversation"
      class="modal-overlay"
      @click="showNewConversation = false"
    >
      <div
        class="modal"
        @click.stop
      >
        <div class="modal-header">
          <h3>New Message</h3>
          <button
            class="close-btn"
            @click="showNewConversation = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <!-- Selected participants chips -->
          <div
            v-if="selectedParticipants.length > 0"
            class="selected-participants"
          >
            <div
              v-for="participant in selectedParticipants"
              :key="participant.user_id"
              class="participant-chip"
            >
              <img
                :src="getMediaUrl(participant.profile_picture_url)"
                :alt="participant.username"
                class="chip-avatar"
              />
              <span class="chip-username">{{ participant.username }}</span>
              <button
                class="chip-remove"
                @click="removeSelectedParticipant(participant.user_id)"
              >
                ✕
              </button>
            </div>
          </div>

          <!-- Group name input (show if multiple participants selected) -->
          <input
            v-if="selectedParticipants.length > 1"
            v-model="newGroupName"
            type="text"
            placeholder="Group name (optional)"
            class="search-input group-name-input"
          />

          <input 
            v-model="searchQuery" 
            type="text" 
            placeholder="Search users to add..." 
            class="search-input"
            @input="searchUsers"
          />
          <div
            v-if="searchResults.length > 0"
            class="search-results"
          >
            <div 
              v-for="user in searchResults" 
              :key="user.user_id" 
              class="search-result-item"
              :class="{ selected: isParticipantSelected(user.user_id) }"
              @click="toggleParticipantSelection(user)"
            >
              <img
                :src="getMediaUrl(user.profile_picture_url)"
                :alt="user.username"
                class="avatar"
              />
              <div class="user-info">
                <div class="username">
                  {{ user.username }}
                </div>
                <div class="name">
                  {{ user.name }}
                </div>
              </div>
              <span
                v-if="isParticipantSelected(user.user_id)"
                class="selected-indicator"
              >
                ✓
              </span>
            </div>
          </div>

          <!-- Create button -->
          <button
            v-if="selectedParticipants.length > 0"
            class="create-conversation-btn"
            @click="createConversationWithSelectedUsers"
          >
            Create {{ selectedParticipants.length > 1 ? 'Group' : 'Chat' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Edit Participants Modal -->
    <div
      v-if="showEditParticipants"
      class="modal-overlay"
      @click="showEditParticipants = false"
    >
      <div
        class="modal"
        @click.stop
      >
        <div class="modal-header">
          <h3>Edit Participants</h3>
          <button
            class="close-btn"
            @click="showEditParticipants = false"
          >
            ✕
          </button>
        </div>
        <div class="modal-body">
          <div class="participants-section">
            <h4>Current Participants ({{ currentParticipants.length }})</h4>
            <div class="participants-list">
              <div 
                v-for="participant in currentParticipants" 
                :key="participant.id || participant.user_id"
                class="participant-item"
              >
                <img
                  :src="getParticipantAvatar(participant)"
                  :alt="participant.username"
                  class="avatar"
                />
                <div class="user-info">
                  <div class="username">
                    {{ participant.username }}
                  </div>
                  <div class="name">
                    {{ participant.name }}
                  </div>
                </div>
                <button 
                  v-if="(participant.id || participant.user_id) !== currentUserId && canEditParticipants"
                  class="remove-btn"
                  title="Remove participant"
                  @click="removeParticipant(participant)"
                >
                  ✕
                </button>
              </div>
            </div>
          </div>
          
          <div
            v-if="canEditParticipants"
            class="add-participants-section"
          >
            <h4>Add People</h4>
            <input 
              v-model="participantSearchQuery" 
              type="text" 
              placeholder="Search users to add..." 
              class="search-input"
              @input="searchUsersForParticipants"
            />
            <div
              v-if="participantSearchResults.length > 0"
              class="search-results"
            >
              <div 
                v-for="user in participantSearchResults" 
                :key="user.user_id" 
                class="search-result-item"
                @click="addParticipant(user)"
              >
                <img
                  :src="getMediaUrl(user.profile_picture_url)"
                  :alt="user.username"
                  class="avatar"
                />
                <div class="user-info">
                  <div class="username">
                    {{ user.username }}
                  </div>
                  <div class="name">
                    {{ user.name }}
                  </div>
                </div>
                <button class="add-btn">
                  +
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="messages-container">
      <!-- Conversations List -->
      <div class="conversations-list">
        <div class="conversations-header">
          <h1>Messages</h1>
          <button
            class="new-message-btn"
            @click="showNewConversation = true"
          >
            ✉
          </button>
        </div>

        <div class="conversations-search">
          <input 
            v-model="conversationSearch" 
            type="text" 
            placeholder="Search Direct" 
            class="search-input" 
          />
        </div>

        <div
          v-if="loading"
          class="loading"
        >
          Loading conversations...
        </div>
        <div
          v-else-if="filteredConversations.length === 0"
          class="empty-state"
        >
          <p>No conversations yet</p>
          <button
            class="start-conversation-btn"
            @click="showNewConversation = true"
          >
            Start a conversation
          </button>
        </div>
        <div
          v-else
          class="conversations"
        >
          <div 
            v-for="conversation in filteredConversations" 
            :key="conversation.id" 
            class="conversation-item" 
            :class="{ active: activeConversation?.id === conversation.id }"
            @click="selectConversation(conversation)"
          >
            <img 
              :src="getConversationAvatar(conversation)" 
              :alt="getConversationName(conversation)" 
              class="avatar" 
            />
            <div class="conversation-info">
              <div class="username">
                {{ getConversationName(conversation) }}
              </div>
              <div class="last-message">
                {{ conversation.last_message?.content || 'No messages yet' }}
              </div>
            </div>
            <div class="timestamp">
              {{ formatTimestamp(conversation.last_message?.sent_at || conversation.created_at) }}
            </div>
          </div>
        </div>
      </div>

      <!-- Chat Area -->
      <div
        v-if="!activeConversation"
        class="chat-area empty"
      >
        <div class="empty-chat-state">
          <h2>Your Messages</h2>
          <p>Send private messages to a friend</p>
          <button
            class="send-message-btn"
            @click="showNewConversation = true"
          >
            Send Message
          </button>
        </div>
      </div>
      <div
        v-else
        class="chat-area"
      >
        <div class="chat-header">
          <div class="chat-user">
            <img 
              :src="getConversationAvatar(activeConversation)" 
              :alt="getConversationName(activeConversation)" 
              class="avatar" 
            />
            <div>
              <div class="username">
                {{ getConversationName(activeConversation) }}
                <span v-if="isConversationUserVerified(activeConversation)" class="verified-badge" title="Verified">✓</span>
              </div>
              <div class="status">
                {{ getOnlineStatus() }}
              </div>
            </div>
          </div>
          <div class="chat-actions">
            <button
              title="Search in conversation"
              @click="toggleConversationSearch"
            >
              🔍
            </button>
            <button
              title="Edit Participants"
              @click="showEditParticipants = true"
            >
              ✏️
            </button>
            <button
              title="Audio Call"
              @click="startAudioCall"
            >
              📞
            </button>
            <button
              title="Video Call"
              @click="startVideoCall"
            >
              📹
            </button>
            <button
              title="Delete Chat"
              class="delete-chat-btn"
              @click="deleteConversation"
            >
              🗑️
            </button>
          </div>
        </div>

        <!-- Conversation Search Bar -->
        <div
          v-if="showConversationSearch"
          class="conversation-search-bar"
        >
          <input 
            v-model="conversationSearchQuery" 
            type="text" 
            placeholder="Search messages..." 
            class="search-input"
            @input="filterMessages"
          />
          <button
            class="close-search-btn"
            @click="toggleConversationSearch"
          >
            ✕
          </button>
          <div
            v-if="conversationSearchQuery"
            class="search-results-count"
          >
            {{ filteredMessagesCount }} result(s) found
          </div>
        </div>

        <div
          ref="messagesContainer"
          class="messages"
        >
          <div
            v-if="messagesLoading"
            class="loading"
          >
            Loading messages...
          </div>
          <div 
            v-for="message in displayMessages" 
            :key="message.id" 
            class="message" 
            :class="[Number(message.sender_id) === currentUserId ? 'sent' : 'received', { highlighted: isMessageHighlighted(message) }]"
            @contextmenu.prevent="openMessageMenu(message, $event)"
          >
            <!-- Show avatar for received messages -->
            <img 
              v-if="Number(message.sender_id) !== currentUserId" 
              :src="getSenderAvatar(message)" 
              :alt="message.sender_username" 
              class="message-avatar"
            />
            <div class="message-wrapper">
              <!-- Show username for received messages -->
              <div
                v-if="Number(message.sender_id) !== currentUserId"
                class="sender-name"
              >
                {{ message.sender_username || 'User' }}
                <span v-if="isSenderVerified(message)" class="verified-badge" title="Verified">✓</span>
              </div>
              <div class="message-content">
                <!-- Display media if present -->
                <div
                  v-if="message.media_url"
                  class="message-media"
                >
                  <img
                    v-if="isImage(message.media_url)"
                    :src="getMediaUrl(message.media_url)"
                    alt="Image"
                    class="media-image"
                  />
                  <video
                    v-else-if="isVideo(message.media_url)"
                    :src="getMediaUrl(message.media_url)"
                    controls
                    class="media-video"
                  ></video>
                </div>
                <div
                  v-if="message.content"
                  class="message-text"
                >
                  {{ message.content }}
                </div>
                <div class="message-footer">
                  <span class="message-time">{{ formatMessageTime(message.sent_at) }}</span>
                  <span
                    v-if="Number(message.sender_id) === currentUserId"
                    class="message-status"
                    :class="getMessageStatusClass(message)"
                    :title="getMessageStatusTitle(message)"
                  >
                    {{ getMessageStatusIcon(message) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Message Context Menu -->
        <div 
          v-if="showMessageMenu && selectedMessage" 
          class="context-menu"
          :style="{ top: menuPosition.y + 'px', left: menuPosition.x + 'px' }"
          @click.stop
        >
          <button 
            v-if="Number(selectedMessage.sender_id) === currentUserId" 
            class="context-menu-item danger"
            @click="deleteMessage(selectedMessage.id)"
          >
            🗑️ Delete Message
          </button>
          <button
            class="context-menu-item"
            @click="copyMessage"
          >
            📋 Copy
          </button>
          <button
            class="context-menu-item"
            @click="closeMessageMenu"
          >
            ✕ Cancel
          </button>
        </div>

        <div class="message-input-area">
          <input 
            ref="mediaFileInput" 
            type="file" 
            accept="image/*,video/*,.gif" 
            style="display: none" 
            @change="handleMediaUpload"
          />
          <button
            class="media-btn"
            title="Add Image/GIF/Video"
            @click="openMediaUpload"
          >
            📷
          </button>
          <button
            class="emoji-btn"
            title="Add emoji"
            @click="insertEmoji"
          >
            😊
          </button>
          <div
            v-if="selectedMedia"
            class="media-preview"
          >
            <img
              v-if="isImage(selectedMedia.name)"
              :src="selectedMedia.preview"
              alt="Preview"
              class="preview-image"
            />
            <video
              v-else-if="isVideo(selectedMedia.name)"
              :src="selectedMedia.preview"
              class="preview-video"
            ></video>
            <button
              class="clear-media-btn"
              @click="clearMediaSelection"
            >
              ✕
            </button>
          </div>
          <input 
            ref="messageInputRef" 
            v-model="messageText" 
            type="text" 
            placeholder="Message..."
            class="message-input"
            @keyup.enter="sendMessage"
          />
          <button 
            v-if="messageText.trim() || selectedMedia" 
            class="send-btn" 
            :disabled="sending"
            @click="sendMessage"
          >
            {{ sending ? '...' : 'Send' }}
          </button>
          <button
            v-else
            class="send-btn"
            @click="sendHeart"
          >
            ❤
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from "vue";
import { useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { messageAPI, userAPI } from "@/services/api";

interface Participant {
  id?: number
  user_id?: number  // Alternative field name from some APIs
  username?: string
  name?: string
  profile_url?: string
  profile_picture_url?: string  // From proto GetUserDataResponse
  profilePictureUrl?: string    // Possible camelCase variant
  is_verified?: boolean
}

interface Message {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  sent_at: string
  sender_username: string
  media_url?: string
  status?: "sent" | "delivered" | "seen"
  seen_by?: string[]
}

interface Conversation {
  id: string
  participants: Participant[]
  last_message?: Message
  created_at: string
  is_group: boolean
  group_name?: string
  group_image_url?: string
}

const route = useRoute();
const authStore = useAuthStore();

const conversations = ref<Conversation[]>([]);
const activeConversation = ref<Conversation | null>(null);
const messages = ref<Message[]>([]);
const messageText = ref("");
const conversationSearch = ref("");
const loading = ref(true);
const messagesLoading = ref(false);
const sending = ref(false);
const showNewConversation = ref(false);
const searchQuery = ref("");
const searchResults = ref<any[]>([]);
const selectedParticipants = ref<any[]>([]);
const newGroupName = ref("");
const messagesContainer = ref<HTMLElement | null>(null);
const messageInputRef = ref<HTMLInputElement | null>(null);

// Message context menu
const showMessageMenu = ref(false);
const selectedMessage = ref<Message | null>(null);
const menuPosition = ref({ x: 0, y: 0 });

// Video call states
const showVideoCall = ref(false);
const localVideoRef = ref<HTMLVideoElement | null>(null);
const remoteVideoRef = ref<HTMLVideoElement | null>(null);
const isMuted = ref(false);
const isVideoEnabled = ref(true);
const localStream = ref<MediaStream | null>(null);

// Edit Participants states
const showEditParticipants = ref(false);
const participantSearchQuery = ref("");
const participantSearchResults = ref<any[]>([]);
const currentParticipants = ref<Participant[]>([]);

// Media upload states
const mediaFileInput = ref<HTMLInputElement | null>(null);
const selectedMedia = ref<{ file: File; preview: string; name: string } | null>(null);

// Conversation search states
const showConversationSearch = ref(false);
const conversationSearchQuery = ref("");
const filteredMessagesIndices = ref<number[]>([]);

// WebSocket connection
let ws: WebSocket | null = null;

const currentUserId = computed(() => {
  const user = authStore.user as any;
  return user?.user_id || user?.id || 0;
});

const activeConversationName = computed(() => {
  if (!activeConversation.value) return "";
  return getConversationName(activeConversation.value);
});

const filteredConversations = computed(() => {
  if (!conversationSearch.value.trim()) return conversations.value;
  const search = conversationSearch.value.toLowerCase();
  return conversations.value.filter(conv => 
    getConversationName(conv).toLowerCase().includes(search)
  );
});

const canEditParticipants = computed(() => {
  if (!activeConversation.value) return false;
  // Allow editing for group chats or 1-on-1 that can become groups
  return activeConversation.value.participants.length >= 1;
});

const displayMessages = computed(() => {
  if (!conversationSearchQuery.value.trim()) {
    return messages.value;
  }
  return messages.value.filter((msg, index) => 
    filteredMessagesIndices.value.includes(index)
  );
});

const filteredMessagesCount = computed(() => {
  return filteredMessagesIndices.value.length;
});

onMounted(async () => {
  await loadConversations();
  connectWebSocket();
  
  // Check if we should open a specific user's conversation from route query
  const username = route.query.user as string;
  if (username) {
    // Search for user and create/open conversation
    const user = await searchUserByUsername(username);
    if (user) {
      await createConversationWithUser(user.user_id);
    }
  }
});

onUnmounted(() => {
  disconnectWebSocket();
  if (localStream.value) {
    localStream.value.getTracks().forEach(track => track.stop());
  }
});

watch(activeConversation, async (newConv) => {
  if (newConv) {
    await loadMessages(newConv.id);
    currentParticipants.value = newConv.participants || [];
    conversationSearchQuery.value = "";
    showConversationSearch.value = false;
  }
});

const loadConversations = async () => {
  loading.value = true;
  try {
    const data = await messageAPI.getConversations();
    conversations.value = Array.isArray(data) ? data : [];
    
    console.log("=== CONVERSATIONS DEBUG ===");
    console.log("Raw API response:", data);
    console.log("Conversations count:", conversations.value.length);
    console.log("Current user ID:", currentUserId.value);
    if (conversations.value.length > 0) {
      console.log("First conversation:", JSON.stringify(conversations.value[0], null, 2));
      if (conversations.value[0]?.participants) {
        console.log("First conversation participants:", conversations.value[0].participants);
        conversations.value[0].participants.forEach((p, i) => {
          console.log(`Participant ${i}:`, {
            id: p?.id,
            user_id: p?.user_id,
            username: p?.username,
            profile_picture_url: p?.profile_picture_url,
            profilePictureUrl: p?.profilePictureUrl,
            profile_url: p?.profile_url
          });
        });
      }
    }
    console.log("=== END DEBUG ===");
    
    // Auto-select first conversation if available
    if (conversations.value.length > 0 && !activeConversation.value) {
      activeConversation.value = conversations.value[0];
    }
  } catch (error) {
    console.error("Failed to load conversations:", error);
  } finally {
    loading.value = false;
  }
};

const loadMessages = async (conversationId: string) => {
  messagesLoading.value = true;
  try {
    const data = await messageAPI.getMessages(conversationId);
    const loadedMessages = Array.isArray(data) ? data : [];
    
    // FIX 2: Sort messages chronologically (Oldest -> Newest) to ensure Top-to-Bottom flow
    messages.value = loadedMessages.sort((a, b) => {
      return new Date(a.sent_at).getTime() - new Date(b.sent_at).getTime();
    });

    await nextTick();
    scrollToBottom();
  } catch (error) {
    console.error("Failed to load messages:", error);
  } finally {
    messagesLoading.value = false;
  }
};

const selectConversation = async (conversation: Conversation) => {
  activeConversation.value = conversation;
};

// Participant management functions
const searchUsersForParticipants = async () => {
  if (!participantSearchQuery.value.trim()) {
    participantSearchResults.value = [];
    return;
  }
  
  try {
    const data = await userAPI.searchUsers(participantSearchQuery.value);
    const users = Array.isArray(data) ? data : [];
    // Filter out users already in the conversation
    const currentParticipantIds = currentParticipants.value.map(p => p.id || p.user_id);
    participantSearchResults.value = users.filter(u => !currentParticipantIds.includes(u.user_id));
  } catch (error) {
    console.error("Failed to search users:", error);
  }
};

const addParticipant = async (user: any) => {
  if (!activeConversation.value) return;
  
  try {
    // Add participant to conversation via API
    await messageAPI.addParticipant(activeConversation.value.id, user.user_id);
    
    // Update local state
    currentParticipants.value.push({
      id: user.user_id,
      user_id: user.user_id,
      username: user.username,
      name: user.name,
      profile_picture_url: user.profile_picture_url
    });
    
    // Update conversation
    const convIndex = conversations.value.findIndex(c => c.id === activeConversation.value?.id);
    if (convIndex !== -1) {
      conversations.value[convIndex].participants = [...currentParticipants.value];
      conversations.value[convIndex].is_group = currentParticipants.value.length > 2;
    }
    
    participantSearchQuery.value = "";
    participantSearchResults.value = [];
  } catch (error) {
    console.error("Failed to add participant:", error);
    alert("Failed to add participant. This feature may require backend support.");
  }
};

const removeParticipant = async (participant: Participant) => {
  if (!activeConversation.value) return;
  
  const participantId = participant.id || participant.user_id;
  if (!participantId) return;
  
  if (!confirm(`Remove ${participant.username} from this conversation?`)) return;
  
  try {
    // Remove participant from conversation via API
    await messageAPI.removeParticipant(activeConversation.value.id, participantId);
    
    // Update local state
    currentParticipants.value = currentParticipants.value.filter(p => 
      (p.id || p.user_id) !== participantId
    );
    
    // Update conversation
    const convIndex = conversations.value.findIndex(c => c.id === activeConversation.value?.id);
    if (convIndex !== -1) {
      conversations.value[convIndex].participants = [...currentParticipants.value];
      conversations.value[convIndex].is_group = currentParticipants.value.length > 2;
    }
  } catch (error) {
    console.error("Failed to remove participant:", error);
    alert("Failed to remove participant. This feature may require backend support.");
  }
};

const getParticipantAvatar = (participant: Participant): string => {
  const avatar = participant.profile_picture_url || participant.profilePictureUrl || participant.profile_url || "";
  return getMediaUrl(avatar);
};

// Media upload functions
const openMediaUpload = () => {
  mediaFileInput.value?.click();
};

const handleMediaUpload = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;
  
  // Validate file type
  const validTypes = ["image/jpeg", "image/png", "image/gif", "image/webp", "video/mp4", "video/webm"];
  if (!validTypes.includes(file.type)) {
    alert("Please select a valid image or video file");
    return;
  }
  
  // Validate file size (max 50MB)
  if (file.size > 50 * 1024 * 1024) {
    alert("File size must be less than 50MB");
    return;
  }
  
  // Create preview
  const reader = new FileReader();
  reader.onload = (e) => {
    selectedMedia.value = {
      file: file,
      preview: e.target?.result as string,
      name: file.name
    };
  };
  reader.readAsDataURL(file);
};

const clearMediaSelection = () => {
  selectedMedia.value = null;
  if (mediaFileInput.value) {
    mediaFileInput.value.value = "";
  }
};

const isConversationUserVerified = (conversation: Conversation): boolean => {
  if (conversation.is_group) return false;
  
  if (!conversation.participants || conversation.participants.length === 0) return false;
  
  const otherUser = conversation.participants.find(p => {
    const participantId = p?.id || p?.user_id;
    return participantId && participantId !== currentUserId.value;
  });
  
  return otherUser?.is_verified || otherUser?.isVerified || false;
};

const isSenderVerified = (message: Message): boolean => {
  if (!activeConversation.value) return false;
  
  if (!activeConversation.value.participants || activeConversation.value.participants.length === 0) return false;
  
  const sender = activeConversation.value.participants.find(
    p => {
      if (!p) return false;
      const participantId = p.id || p.user_id;
      if (!participantId) return false;
      return participantId.toString() === message.sender_id;
    }
  );
  
  return sender?.is_verified || sender?.isVerified || false;
};

const isImage = (filename: string): boolean => {
  return /\.(jpg|jpeg|png|gif|webp)$/i.test(filename);
};

const isVideo = (filename: string): boolean => {
  return /\.(mp4|webm|mov)$/i.test(filename);
};

// Conversation search functions
const toggleConversationSearch = () => {
  showConversationSearch.value = !showConversationSearch.value;
  if (!showConversationSearch.value) {
    conversationSearchQuery.value = "";
    filteredMessagesIndices.value = [];
  }
};

const filterMessages = () => {
  if (!conversationSearchQuery.value.trim()) {
    filteredMessagesIndices.value = [];
    return;
  }
  
  const query = conversationSearchQuery.value.toLowerCase();
  filteredMessagesIndices.value = messages.value
    .map((msg, index) => ({ msg, index }))
    .filter(({ msg }) => msg.content.toLowerCase().includes(query))
    .map(({ index }) => index);
};

const isMessageHighlighted = (message: Message): boolean => {
  if (!conversationSearchQuery.value.trim()) return false;
  return message.content.toLowerCase().includes(conversationSearchQuery.value.toLowerCase());
};

// Message status functions
const getMessageStatus = (message: Message): string => {
  // Check if message has explicit status
  if (message.status === "seen") return "seen";
  if (message.status === "delivered") return "delivered";
  if (message.status === "sent") return "sent";
  
  // Default to sent status
  return "sent";
};

const getMessageStatusIcon = (message: Message): string => {
  const status = getMessageStatus(message);
  if (status === "seen") return "✓✓";      // Double check for read
  if (status === "delivered") return "✓✓";  // Double check for delivered
  return "✓";                               // Single check for sent
};

const getMessageStatusClass = (message: Message): string => {
  const status = getMessageStatus(message);
  if (status === "seen") return "status-seen";
  if (status === "delivered") return "status-delivered";
  return "status-sent";
};

const getMessageStatusTitle = (message: Message): string => {
  const status = getMessageStatus(message);
  if (status === "seen") return "Seen";
  if (status === "delivered") return "Delivered";
  return "Sent";
};

const sendMessage = async () => {
  if ((!messageText.value.trim() && !selectedMedia.value) || !activeConversation.value || sending.value) return;
  
  sending.value = true;
  const content = messageText.value;
  const media = selectedMedia.value;
  messageText.value = ""; // Clear immediately for better UX
  
  try {
    let newMessage;
    
    if (media) {
      // Upload media and send message with media
      console.log("Uploading media:", media.file.name);
      const formData = new FormData();
      formData.append("file", media.file);
      if (content.trim()) {
        formData.append("content", content);
      }
      formData.append("conversation_id", activeConversation.value.id);
      
      // This would require a media upload endpoint
      newMessage = await messageAPI.sendMessageWithMedia(activeConversation.value.id, formData);
      clearMediaSelection();
    } else {
      // Send text-only message
      console.log("Sending message:", content, "to conversation:", activeConversation.value.id);
      newMessage = await messageAPI.sendMessage(activeConversation.value.id, content);
    }
    
    console.log("Message sent successfully:", newMessage);
    
    // Add message immediately for instant feedback, but check for duplicates
    const exists = messages.value.some(m => m.id === newMessage.id);
    if (!exists) {
      messages.value.push(newMessage);
      
      // Sort messages by time to maintain chronological order
      messages.value.sort((a, b) => 
        new Date(a.sent_at).getTime() - new Date(b.sent_at).getTime()
      );
      
      await nextTick();
      scrollToBottom();
    }
    
    // Update last message in conversation
    const convIndex = conversations.value.findIndex(c => c.id === activeConversation.value?.id);
    if (convIndex !== -1) {
      conversations.value[convIndex].last_message = newMessage;
      // Move conversation to top
      const conv = conversations.value.splice(convIndex, 1)[0];
      conversations.value.unshift(conv);
    }
    
    await nextTick();
    scrollToBottom();
  } catch (error) {
    console.error("Failed to send message:", error);
    messageText.value = content; // Restore message on error
    alert("Failed to send message. Media upload may require backend support.");
  } finally {
    sending.value = false;
  }
};

const searchUsers = async () => {
  if (!searchQuery.value.trim()) {
    searchResults.value = [];
    return;
  }
  
  try {
    const data = await userAPI.searchUsers(searchQuery.value);
    console.log("Search results:", data);
    searchResults.value = Array.isArray(data) ? data : [];
    console.log("Parsed search results:", searchResults.value);
  } catch (error) {
    console.error("Failed to search users:", error);
  }
};

const searchUserByUsername = async (username: string) => {
  try {
    const data = await userAPI.searchUsers(username);
    const users = Array.isArray(data) ? data : [];
    return users.find(u => u.username === username);
  } catch (error) {
    console.error("Failed to search user:", error);
    return null;
  }
};

const toggleParticipantSelection = (user: any) => {
  const index = selectedParticipants.value.findIndex(p => p.user_id === user.user_id);
  if (index > -1) {
    selectedParticipants.value.splice(index, 1);
  } else {
    selectedParticipants.value.push(user);
  }
};

const isParticipantSelected = (userId: number): boolean => {
  return selectedParticipants.value.some(p => p.user_id === userId);
};

const removeSelectedParticipant = (userId: number) => {
  selectedParticipants.value = selectedParticipants.value.filter(p => p.user_id !== userId);
};

const createConversationWithSelectedUsers = async () => {
  if (selectedParticipants.value.length === 0) return;
  
  try {
    const participantIds = selectedParticipants.value.map(p => p.user_id);
    const isGroup = participantIds.length > 1;
    
    const payload: any = { participant_ids: participantIds };
    if (isGroup && newGroupName.value.trim()) {
      payload.group_name = newGroupName.value.trim();
    }
    
    const newConv = await messageAPI.createConversation(payload);
    console.log("Created conversation:", newConv);
    
    // Refresh conversations list
    await loadConversations();
    
    // Find and select the newly created conversation
    const createdConv = conversations.value.find(c => c.id === newConv.id);
    if (createdConv) {
      activeConversation.value = createdConv;
    } else {
      activeConversation.value = newConv;
    }
    
    // Reset modal state
    showNewConversation.value = false;
    selectedParticipants.value = [];
    newGroupName.value = "";
    searchQuery.value = "";
    searchResults.value = [];
  } catch (error) {
    console.error("Failed to create conversation:", error);
    alert("Failed to create conversation");
  }
};

const createConversationWithUser = async (userId: number) => {
  try {
    console.log("Creating conversation with userId:", userId, "type:", typeof userId);
    
    // Check if conversation already exists
    const existingConv = conversations.value.find(conv => 
      !conv.is_group && conv.participants.some(p => p.id === userId)
    );
    
    if (existingConv) {
      activeConversation.value = existingConv;
      showNewConversation.value = false;
      return;
    }
    
    // Create new conversation
    const newConv = await messageAPI.createConversation({ participant_ids: [userId] });
    console.log("Created conversation:", newConv);
    
    // Refresh conversations list to get the properly formatted conversation with participant data
    await loadConversations();
    
    // Find and select the newly created conversation
    const createdConv = conversations.value.find(c => c.id === newConv.id);
    if (createdConv) {
      activeConversation.value = createdConv;
    } else {
      activeConversation.value = newConv;
    }
    
    showNewConversation.value = false;
    searchQuery.value = "";
    searchResults.value = [];
  } catch (error) {
    console.error("Failed to create conversation:", error);
  }
};

const getConversationName = (conversation: Conversation): string => {
  if (conversation.is_group) {
    return conversation.group_name || "Group Chat";
  }
  
  // For 1-on-1, show the other person's name
  if (!conversation.participants || conversation.participants.length === 0) {
    console.warn("Conversation has no participants:", conversation);
    return "Loading...";
  }
  
  // Log for debugging
  console.log("Getting conversation name:", {
    conversationId: conversation.id,
    participants: conversation.participants,
    currentUserId: currentUserId.value
  });
  
  const otherUser = conversation.participants.find(p => {
    const participantId = p?.id || p?.user_id;
    return participantId && participantId !== currentUserId.value;
  });
  
  if (!otherUser) {
    console.warn("Could not find other user in conversation:", {
      conversation,
      currentUserId: currentUserId.value,
      participantIds: conversation.participants.map(p => p?.id || p?.user_id)
    });
    
    // Fallback: show the first participant if they're not the current user
    const firstParticipant = conversation.participants[0];
    const firstId = firstParticipant?.id || firstParticipant?.user_id;
    if (firstParticipant && firstId !== currentUserId.value) {
      return firstParticipant.username || firstParticipant.name || "User";
    }
    
    // If first is current user, try second
    if (conversation.participants[1]) {
      return conversation.participants[1].username || conversation.participants[1].name || "User";
    }
    
    return "Loading...";
  }
  
  return otherUser.username || otherUser.name || "User";
};

const getConversationAvatar = (conversation: Conversation): string => {
  if (conversation.is_group && conversation.group_image_url) {
    return getMediaUrl(conversation.group_image_url);
  }
  
  // For 1-on-1, show the other person's avatar
  if (!conversation.participants || conversation.participants.length === 0) {
    return "/placeholder.svg";
  }
  
  const otherUser = conversation.participants.find(p => {
    const participantId = p?.id || p?.user_id;
    return participantId && participantId !== currentUserId.value;
  });
  
  if (!otherUser) {
    // Fallback to first participant
    const firstParticipant = conversation.participants[0];
    const firstId = firstParticipant?.id || firstParticipant?.user_id;
    if (firstParticipant && firstId !== currentUserId.value) {
      const avatar = firstParticipant.profile_picture_url || firstParticipant.profilePictureUrl || firstParticipant.profile_url || "";
      return getMediaUrl(avatar);
    }
    if (conversation.participants[1]) {
      const avatar = conversation.participants[1].profile_picture_url || conversation.participants[1].profilePictureUrl || conversation.participants[1].profile_url || "";
      return getMediaUrl(avatar);
    }
    return "/placeholder.svg";
  }
  
  const avatar = otherUser.profile_picture_url || otherUser.profilePictureUrl || otherUser.profile_url || "";
  return getMediaUrl(avatar);
};

const getOnlineStatus = (): string => {
  // TODO: Implement real online status via WebSocket
  return "Active now";
};

const getMediaUrl = (url: string): string => {
  if (!url) return "/placeholder.svg";
  if (url.startsWith("http")) return url;
  
  // If it's a relative path from MinIO, construct the full URL
  // MinIO paths are stored as "user-X/..." in the database
  if (url.includes("/")) {
    return `http://localhost:9000/media/${url}`;
  }
  
  // Fallback to api-gateway proxy
  return `http://localhost:8000${url}`;
};

const getSenderAvatar = (message: Message): string => {
  if (!activeConversation.value) {
    console.log("No active conversation for avatar");
    return "/placeholder.svg";
  }
  
  if (!activeConversation.value.participants || activeConversation.value.participants.length === 0) {
    console.log("No participants in conversation");
    return "/placeholder.svg";
  }
  
  // Find the sender in the conversation participants
  const sender = activeConversation.value.participants.find(
    p => {
      if (!p) return false;
      const participantId = p.id || p.user_id;
      if (!participantId) return false;
      return participantId.toString() === message.sender_id;
    }
  );
  
  if (!sender) {
    console.log("Sender not found in participants:", {
      senderId: message.sender_id,
      participants: activeConversation.value.participants.map(p => ({
        id: p?.id,
        user_id: p?.user_id,
        username: p?.username
      }))
    });
    return "/placeholder.svg";
  }
  
  const avatarUrl = sender.profile_picture_url || sender.profilePictureUrl || sender.profile_url || "";
  return getMediaUrl(avatarUrl);
};

const formatTimestamp = (timestamp: string): string => {
  if (!timestamp) return "";
  const date = new Date(timestamp);
  const now = new Date();
  const diffInMs = now.getTime() - date.getTime();
  const diffInMins = Math.floor(diffInMs / 60000);
  const diffInHours = Math.floor(diffInMins / 60);
  const diffInDays = Math.floor(diffInHours / 24);

  if (diffInMins < 60) return `${diffInMins}m`;
  if (diffInHours < 24) return `${diffInHours}h`;
  if (diffInDays < 7) return `${diffInDays}d`;
  return date.toLocaleDateString();
};

const formatMessageTime = (timestamp: string): string => {
  const date = new Date(timestamp);
  return date.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true });
};

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
    // Use requestAnimationFrame for smoother scroll
    requestAnimationFrame(() => {
      if (messagesContainer.value) {
        messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
      }
    });
  }
};

// Emoji and heart functions
const insertEmoji = () => {
  const emojis = ["😊", "😂", "❤️", "👍", "🎉", "🔥", "😍", "🤔", "😎", "👏"];
  const randomEmoji = emojis[Math.floor(Math.random() * emojis.length)];
  messageText.value += randomEmoji;
  messageInputRef.value?.focus();
};

const sendHeart = async () => {
  if (!activeConversation.value || sending.value) return;
  messageText.value = "❤️";
  await sendMessage();
};

// Message context menu functions
const openMessageMenu = (message: Message, event: MouseEvent) => {
  selectedMessage.value = message;
  menuPosition.value = { x: event.clientX, y: event.clientY };
  showMessageMenu.value = true;
  
  // Close menu when clicking outside
  document.addEventListener("click", closeMessageMenu, { once: true });
};

const closeMessageMenu = () => {
  showMessageMenu.value = false;
  selectedMessage.value = null;
};

const deleteMessage = async (messageId: string) => {
  if (!confirm("Delete this message?")) return;
  
  try {
    await messageAPI.unsendMessage(messageId);
    
    // Remove message from list
    messages.value = messages.value.filter(m => m.id !== messageId);
    
    closeMessageMenu();
  } catch (error) {
    console.error("Failed to delete message:", error);
    alert("Failed to delete message");
  }
};

const copyMessage = () => {
  if (selectedMessage.value) {
    navigator.clipboard.writeText(selectedMessage.value.content);
    closeMessageMenu();
  }
};

const deleteConversation = async () => {
  if (!activeConversation.value) return;
  
  if (!confirm("Delete this conversation? All messages will be removed.")) return;
  
  try {
    await messageAPI.deleteConversation(activeConversation.value.id);
    
    // Clear active conversation first
    activeConversation.value = null;
    messages.value = [];
    
    // Reload conversations from backend to ensure it's removed
    await loadConversations();
    
    // Auto-select first conversation if available
    if (conversations.value.length > 0) {
      activeConversation.value = conversations.value[0];
      await loadMessages(conversations.value[0].id);
    }
  } catch (error) {
    console.error("Failed to delete conversation:", error);
    alert("Failed to delete conversation");
  }
};

// WebSocket connection
const connectWebSocket = () => {
  const token = localStorage.getItem("jwt_token");
  if (!token) return;
  
  const wsUrl = `ws://localhost:9004/ws?token=${token}`;
  ws = new WebSocket(wsUrl);
  
  ws.onopen = () => {
    console.log("WebSocket connected");
  };
  
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      let newMessage = null;
      
      if (data.type === "message" && data.message) {
        newMessage = data.message;
      } else if (data.id && data.content && data.sender_id) {
        newMessage = data;
      }
      
      if (newMessage) {
        // Handle active conversation updates
        if (activeConversation.value?.id === newMessage.conversation_id) {
          const exists = messages.value.some(m => m.id === newMessage.id);
          if (!exists) {
            messages.value.push(newMessage);
            
            // ★ KEY FIX: Re-sort strictly by time (Oldest -> Newest)
            // This ensures that even if a message arrives slightly late due to lag,
            // it slots into the correct position in the chat history.
            messages.value.sort((a, b) => 
              new Date(a.sent_at).getTime() - new Date(b.sent_at).getTime()
            );

            nextTick(() => {
              scrollToBottom();
            });
          }
        }
        
        // Update conversation list preview
        const convIndex = conversations.value.findIndex(c => c.id === newMessage.conversation_id);
        if (convIndex !== -1) {
          conversations.value[convIndex].last_message = newMessage;
          const conv = conversations.value.splice(convIndex, 1)[0];
          conversations.value.unshift(conv);
        } else {
          loadConversations();
        }
      }
    } catch (error) {
      console.error("Failed to parse WebSocket message:", error);
    }
  };
  
  ws.onerror = (error) => {
    console.error("WebSocket error:", error);
  };
  
  ws.onclose = () => {
    console.log("WebSocket disconnected");
    setTimeout(() => {
      if (!ws || ws.readyState === WebSocket.CLOSED) {
        connectWebSocket();
      }
    }, 3000);
  };
};

const disconnectWebSocket = () => {
  if (ws) {
    ws.close();
    ws = null;
  }
};

// Video call functions
const startVideoCall = async () => {
  if (!activeConversation.value) return;
  
  try {
    // Get video call token
    const response = await messageAPI.getVideoToken(activeConversation.value.id);
    console.log("Video call response:", response);
    console.log("Room ID:", response.room_id, "Token:", response.token);
    
    // Get user media
    localStream.value = await navigator.mediaDevices.getUserMedia({ 
      video: true, 
      audio: true 
    });
    
    showVideoCall.value = true;
    
    await nextTick();
    
    if (localVideoRef.value && localStream.value) {
      localVideoRef.value.srcObject = localStream.value;
    }
    
    // Display room ID to user
    alert(`Video call started for conversation ${response.room_id}\\n\\nShare this room ID with the other person to join the same call.\\n\\nNote: This is a basic implementation showing only your local video. Full WebRTC peer-to-peer connection requires additional setup.`);
    
    // TODO: Implement WebRTC peer connection with the token
    // For now, just show the local video
  } catch (error) {
    console.error("Failed to start video call:", error);
    alert("Failed to start video call. Please check camera permissions.");
  }
};

const startAudioCall = async () => {
  if (!activeConversation.value) return;
  
  try {
    // Get video call token (same endpoint, but we won't enable video)
    const response = await messageAPI.getVideoToken(activeConversation.value.id);
    console.log("Audio call token:", response);
    
    // Get user media - AUDIO ONLY
    localStream.value = await navigator.mediaDevices.getUserMedia({ 
      video: false,  // No video for audio call
      audio: true 
    });
    
    showVideoCall.value = true;
    isVideoEnabled.value = false;  // Mark video as disabled
    
    // No need to attach video elements since there's no video
    // TODO: Implement WebRTC peer connection with the token
  } catch (error) {
    console.error("Failed to start audio call:", error);
    alert("Failed to start audio call. Please check microphone permissions.");
  }
};

const endVideoCall = () => {
  if (localStream.value) {
    localStream.value.getTracks().forEach(track => track.stop());
    localStream.value = null;
  }
  showVideoCall.value = false;
  isVideoEnabled.value = true;
  isMuted.value = false;
};

const toggleMute = () => {
  if (localStream.value) {
    const audioTrack = localStream.value.getAudioTracks()[0];
    if (audioTrack) {
      audioTrack.enabled = !audioTrack.enabled;
      isMuted.value = !audioTrack.enabled;
    }
  }
};

const toggleVideo = () => {
  if (localStream.value) {
    const videoTrack = localStream.value.getVideoTracks()[0];
    if (videoTrack) {
      videoTrack.enabled = !videoTrack.enabled;
      isVideoEnabled.value = videoTrack.enabled;
    }
  }
};
</script>

<style scoped lang="scss">
.messages-page {
  width: 100%;
  padding-left: calc(244px);
  height: 100vh;
  background-color: #000;
  display: flex;
}

// Video Call Modal
.video-call-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.95);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.video-call-modal {
  background-color: #1a1a1a;
  border-radius: 12px;
  width: 90%;
  max-width: 1200px;
  height: 80vh;
  display: flex;
  flex-direction: column;

  .video-call-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #262626;

    h3 {
      font-size: 18px;
      font-weight: 600;
    }

    .close-btn {
      background: none;
      border: none;
      color: #fff;
      font-size: 24px;
      cursor: pointer;
      padding: 0;
    }
  }

  .video-container {
    flex: 1;
    position: relative;
    background-color: #000;

    .remote-video {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .local-video {
      position: absolute;
      top: 20px;
      right: 20px;
      width: 200px;
      height: 150px;
      object-fit: cover;
      border-radius: 8px;
      border: 2px solid #fff;
    }
  }

  .video-controls {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 20px;
    padding: 20px;
    background-color: #1a1a1a;

    button {
      background-color: #262626;
      border: none;
      color: #fff;
      font-size: 24px;
      width: 50px;
      height: 50px;
      border-radius: 50%;
      cursor: pointer;
      transition: background-color 0.2s;

      &:hover {
        background-color: #333;
      }

      &.muted,
      &.video-off {
        background-color: #dc3545;
      }

      &.end-call-btn {
        background-color: #dc3545;
        width: auto;
        padding: 0 20px;
        border-radius: 25px;
        font-size: 16px;

        &:hover {
          background-color: #c82333;
        }
      }
    }
  }
}

// Modal Styles
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
}

.modal {
  background-color: #1a1a1a;
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #262626;

    h3 {
      font-size: 18px;
      font-weight: 600;
    }

    .close-btn {
      background: none;
      border: none;
      color: #fff;
      font-size: 24px;
      cursor: pointer;
      padding: 0;
    }
  }

  .modal-body {
    padding: 20px;
    overflow-y: auto;

    .selected-participants {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-bottom: 16px;
      padding: 12px;
      background-color: #1a1a1a;
      border-radius: 8px;
      min-height: 60px;

      .participant-chip {
        display: flex;
        align-items: center;
        gap: 8px;
        background-color: #0a66c2;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 14px;

        .chip-avatar {
          width: 24px;
          height: 24px;
          border-radius: 50%;
          object-fit: cover;
        }

        .chip-username {
          color: #fff;
          font-weight: 500;
        }

        .chip-remove {
          background: none;
          border: none;
          color: #fff;
          cursor: pointer;
          font-size: 16px;
          padding: 0 0 0 4px;
          line-height: 1;

          &:hover {
            opacity: 0.8;
          }
        }
      }
    }

    .group-name-input {
      margin-bottom: 16px;
    }

    .search-input {
      width: 100%;
      background-color: #262626;
      border: none;
      border-radius: 20px;
      padding: 10px 16px;
      color: #fff;
      font-size: 14px;
      outline: none;
      margin-bottom: 16px;

      &::placeholder {
        color: #a8a8a8;
      }
    }

    .search-results {
      display: flex;
      flex-direction: column;
      gap: 8px;
      margin-bottom: 16px;

      .search-result-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        border-radius: 8px;
        cursor: pointer;
        transition: background-color 0.2s;
        position: relative;

        &:hover {
          background-color: #262626;
        }

        &.selected {
          background-color: rgba(10, 102, 194, 0.2);
          border: 1px solid #0a66c2;
        }

        .avatar {
          width: 50px;
          height: 50px;
          border-radius: 50%;
          object-fit: cover;
        }

        .user-info {
          flex: 1;

          .username {
            font-weight: 600;
            font-size: 14px;
          }

          .name {
            font-size: 12px;
            color: #a8a8a8;
          }
        }

        .selected-indicator {
          color: #0a66c2;
          font-size: 20px;
          font-weight: bold;
        }
      }
    }

    .create-conversation-btn {
      width: 100%;
      padding: 12px;
      background-color: #0a66c2;
      color: #fff;
      border: none;
      border-radius: 8px;
      font-size: 16px;
      font-weight: 600;
      cursor: pointer;
      transition: background-color 0.2s;

      &:hover {
        background-color: #0958a8;
      }

      &:disabled {
        background-color: #333;
        cursor: not-allowed;
        opacity: 0.5;
      }
    }
  }
}

.messages-container {
  display: flex;
  width: 100%;
  height: 100%;
}

.loading,
.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: #a8a8a8;

  .start-conversation-btn {
    margin-top: 16px;
    background-color: #0a66c2;
    color: #fff;
    border: none;
    padding: 8px 24px;
    border-radius: 8px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 600;

    &:hover {
      background-color: #0958a8;
    }
  }
}

.conversations-list {
  width: 360px;
  border-right: 1px solid #262626;
  display: flex;
  flex-direction: column;

  .conversations-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #262626;

    h1 {
      font-size: 24px;
      font-weight: 700;
    }

    .new-message-btn {
      background: none;
      border: none;
      color: #fff;
      font-size: 20px;
      cursor: pointer;
      padding: 0;
    }
  }

  .conversations-search {
    padding: 8px 16px;

    .search-input {
      width: 100%;
      background-color: #262626;
      border: none;
      border-radius: 20px;
      padding: 10px 16px;
      color: #fff;
      font-size: 14px;
      outline: none;

      &::placeholder {
        color: #a8a8a8;
      }
    }
  }

  .conversations {
    flex: 1;
    overflow-y: auto;

    .conversation-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 16px;
      cursor: pointer;
      transition: background-color 0.2s;
      border-left: 3px solid transparent;
      min-height: 80px;

      &:hover {
        background-color: #1a1a1a;
      }

      &.active {
        background-color: #1a1a1a;
        border-left-color: #0a66c2;
      }

      .avatar {
        width: 56px;
        height: 56px;
        border-radius: 50%;
        object-fit: cover;
        flex-shrink: 0;
        background-color: #262626;
      }

      .conversation-info {
        flex: 1;
        min-width: 0;
        display: flex;
        flex-direction: column;
        gap: 4px;

        .username {
          font-weight: 600;
          font-size: 14px;
          color: #fff;
        }

        .last-message {
          font-size: 12px;
          color: #a8a8a8;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
          max-width: 100%;
        }
      }

      .timestamp {
        font-size: 12px;
        color: #a8a8a8;
        flex-shrink: 0;
        align-self: flex-start;
        padding-top: 2px;
      }
    }
  }
}

.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;

  &.empty {
    align-items: center;
    justify-content: center;

    .empty-chat-state {
      text-align: center;
      max-width: 400px;

      h2 {
        font-size: 24px;
        font-weight: 300;
        margin-bottom: 8px;
      }

      p {
        color: #a8a8a8;
        margin-bottom: 24px;
      }

      .send-message-btn {
        background-color: #0a66c2;
        color: #fff;
        border: none;
        padding: 8px 24px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 14px;
        font-weight: 600;

        &:hover {
          background-color: #0958a8;
        }
      }
    }
  }

  .chat-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 20px;
    border-bottom: 1px solid #262626;

    .chat-user {
      display: flex;
      align-items: center;
      gap: 12px;

      .avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        object-fit: cover;
      }

      .username {
        font-weight: 600;
        font-size: 14px;
        display: flex;
        align-items: center;
        gap: 4px;
        
        .verified-badge {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          background-color: #0a66c2;
          color: white;
          border-radius: 50%;
          width: 16px;
          height: 16px;
          font-size: 10px;
          font-weight: bold;
        }
      }

      .status {
        font-size: 12px;
        color: #a8a8a8;
      }
    }

    .chat-actions {
      display: flex;
      gap: 16px;

      button {
        background: none;
        border: none;
        color: #0a66c2;
        font-size: 18px;
        cursor: pointer;
        padding: 0;
        transition: color 0.2s;

        &:hover {
          color: #0958a8;
        }

        &.delete-chat-btn {
          color: #dc3545;

          &:hover {
            color: #c82333;
          }
        }
      }
    }
  }

  .messages {
    flex: 1;
    overflow-y: auto;
    padding: 16px 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    position: relative;

    .loading {
      text-align: center;
      color: #a8a8a8;
      padding: 20px;
    }

    .message {
      display: flex;
      align-items: flex-start;
      gap: 8px;
      cursor: context-menu;

      &:hover {
        .message-content {
          opacity: 0.9;
        }
      }

      .message-avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        object-fit: cover;
        flex-shrink: 0;
      }

      .message-wrapper {
        display: flex;
        flex-direction: column;
        max-width: 70%;
      }

      .sender-name {
        font-size: 12px;
        color: #a8a8a8;
        margin-bottom: 4px;
        padding-left: 12px;
        display: flex;
        align-items: center;
        gap: 4px;
        
        .verified-badge {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          background-color: #0a66c2;
          color: white;
          border-radius: 50%;
          width: 14px;
          height: 14px;
          font-size: 9px;
          font-weight: bold;
        }
      }

      .message-content {
        padding: 12px 16px;
        border-radius: 18px;
        font-size: 14px;
        word-wrap: break-word;
        position: relative;

        .message-time {
          font-size: 10px;
          color: rgba(255, 255, 255, 0.6);
          margin-top: 4px;
        }
      }

      &.sent {
        justify-content: flex-end;

        .message-wrapper {
          align-items: flex-end;
        }

        .message-content {
          background-color: #0a66c2;
          color: #fff;
        }
        
        .message-footer {
          .message-time {
            color: rgba(255, 255, 255, 0.8);
          }
          
          .message-status {
            &.status-sent {
              color: rgba(255, 255, 255, 0.8);
            }
            
            &.status-delivered {
              color: rgba(255, 255, 255, 0.9);
            }
            
            &.status-seen {
              color: #80d8ff;
            }
          }
        }
      }

      &.received {
        justify-content: flex-start;

        .message-wrapper {
          align-items: flex-start;
        }

        .message-content {
          background-color: #262626;
          color: #fff;
        }
      }
    }
  }

  .message-input-area {
    display: flex;
    gap: 12px;
    padding: 12px 20px;
    border-top: 1px solid #262626;
    align-items: center;

    .emoji-btn {
      background: none;
      border: none;
      color: #fff;
      font-size: 20px;
      cursor: pointer;
      padding: 0;
    }

    .message-input {
      flex: 1;
      background-color: #262626;
      border: none;
      border-radius: 20px;
      padding: 10px 16px;
      color: #fff;
      font-size: 14px;
      outline: none;

      &::placeholder {
        color: #a8a8a8;
      }
    }

    .send-btn {
      background: none;
      border: none;
      color: #0a66c2;
      font-size: 18px;
      cursor: pointer;
      padding: 0;
    }
  }
}

// Context Menu
.context-menu {
  position: fixed;
  background-color: #1a1a1a;
  border-radius: 8px;
  border: 1px solid #262626;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 1000;
  min-width: 180px;
  padding: 4px 0;

  .context-menu-item {
    display: block;
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    color: #fff;
    padding: 10px 16px;
    font-size: 14px;
    cursor: pointer;
    transition: background-color 0.2s;

    &:hover {
      background-color: #262626;
    }

    &.danger {
      color: #dc3545;

      &:hover {
        background-color: rgba(220, 53, 69, 0.1);
      }
    }
  }
}

@media (max-width: 1024px) {
  .messages-page {
    padding-left: calc(72px);
  }

  .conversations-list {
    width: 300px;
  }
}

@media (max-width: 768px) {
  .messages-page {
    padding-left: 0;
  }

  .messages-container {
    flex-direction: column;
  }

  .conversations-list {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #262626;
    height: auto;
    max-height: 200px;
  }

  .chat-area {
    flex: 1;
  }
}

// Conversation Search Bar
.conversation-search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background-color: #1a1a1a;
  border-bottom: 1px solid #262626;

  .search-input {
    flex: 1;
    background-color: #262626;
    border: none;
    border-radius: 20px;
    padding: 8px 16px;
    color: #fff;
    font-size: 14px;
    outline: none;

    &::placeholder {
      color: #a8a8a8;
    }
  }

  .close-search-btn {
    background: none;
    border: none;
    color: #fff;
    font-size: 18px;
    cursor: pointer;
    padding: 0;
  }

  .search-results-count {
    font-size: 12px;
    color: #a8a8a8;
    white-space: nowrap;
  }
}

// Participants Section in Modal
.participants-section,
.add-participants-section {
  margin-bottom: 20px;

  h4 {
    font-size: 14px;
    font-weight: 600;
    margin-bottom: 12px;
    color: #fff;
  }

  .participants-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 200px;
    overflow-y: auto;
  }

  .participant-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px;
    border-radius: 8px;
    transition: background-color 0.2s;

    &:hover {
      background-color: #262626;
    }

    .avatar {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      object-fit: cover;
    }

    .user-info {
      flex: 1;

      .username {
        font-weight: 600;
        font-size: 14px;
      }

      .name {
        font-size: 12px;
        color: #a8a8a8;
      }
    }

    .remove-btn,
    .add-btn {
      background-color: #dc3545;
      border: none;
      color: #fff;
      width: 28px;
      height: 28px;
      border-radius: 50%;
      cursor: pointer;
      font-size: 16px;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: background-color 0.2s;

      &:hover {
        background-color: #c82333;
      }
    }

    .add-btn {
      background-color: #28a745;

      &:hover {
        background-color: #218838;
      }
    }
  }
}

// Media Upload Styles
.media-btn {
  background: none;
  border: none;
  color: #fff;
  font-size: 20px;
  cursor: pointer;
  padding: 0;
  transition: transform 0.2s;

  &:hover {
    transform: scale(1.1);
  }
}

.media-preview {
  position: absolute;
  bottom: 60px;
  left: 50px;
  background-color: #262626;
  border-radius: 8px;
  padding: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);

  .preview-image,
  .preview-video {
    width: 80px;
    height: 80px;
    object-fit: cover;
    border-radius: 4px;
  }

  .clear-media-btn {
    background-color: #dc3545;
    border: none;
    color: #fff;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      background-color: #c82333;
    }
  }
}

// Message Media Display
.message-media {
  margin-bottom: 8px;

  .media-image {
    max-width: 300px;
    max-height: 400px;
    border-radius: 8px;
    cursor: pointer;
  }

  .media-video {
    max-width: 300px;
    max-height: 400px;
    border-radius: 8px;
  }
}

.message-text {
  margin-bottom: 4px;
}

.message-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  margin-top: 2px;
  font-size: 11px;

  .message-time {
    color: rgba(255, 255, 255, 0.6);
  }

  .message-status {
    font-size: 14px;
    font-weight: 600;
    line-height: 1;
    transition: color 0.3s ease;
    margin-left: 2px;
    
    &.status-sent {
      color: rgba(255, 255, 255, 0.6);
    }
    
    &.status-delivered {
      color: rgba(255, 255, 255, 0.8);
    }
    
    &.status-seen {
      color: #4fc3f7;
    }
  }
}

// Highlighted message for search
.message.highlighted {
  .message-content {
    background-color: #1a4d7a !important;
    animation: highlight-pulse 1s ease-in-out;
  }
}

@keyframes highlight-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}
</style>
