import { ref, onMounted, onUnmounted } from 'vue';

declare global {
  interface Window {
    turnstile?: {
      render: (element: string | HTMLElement, options: {
        sitekey: string;
        callback?: (token: string) => void;
        'error-callback'?: () => void;
        'expired-callback'?: () => void;
        theme?: 'light' | 'dark' | 'auto';
        size?: 'normal' | 'compact';
      }) => string;
      reset: (widgetId: string) => void;
      remove: (widgetId: string) => void;
    };
  }
}

export function useTurnstile() {
  const turnstileToken = ref<string>('');
  const turnstileWidgetId = ref<string>('');
  const turnstileError = ref<string>('');
  const turnstileReady = ref(false);

  const TURNSTILE_SITE_KEY = import.meta.env.VITE_TURNSTILE_SITE_KEY || '1x00000000000000000000AA'; // Test key

  const initTurnstile = (elementId: string) => {
    // Wait for Turnstile to be loaded
    const checkTurnstile = setInterval(() => {
      if (window.turnstile) {
        clearInterval(checkTurnstile);
        turnstileReady.value = true;

        try {
          const widgetId = window.turnstile.render(`#${elementId}`, {
            sitekey: TURNSTILE_SITE_KEY,
            callback: (token: string) => {
              turnstileToken.value = token;
              turnstileError.value = '';
            },
            'error-callback': () => {
              turnstileError.value = 'Verification failed. Please try again.';
              turnstileToken.value = '';
            },
            'expired-callback': () => {
              turnstileError.value = 'Verification expired. Please try again.';
              turnstileToken.value = '';
            },
            theme: 'dark',
            size: 'normal',
          });
          turnstileWidgetId.value = widgetId;
        } catch (error) {
          console.error('Failed to render Turnstile:', error);
          turnstileError.value = 'Failed to load verification. Please refresh the page.';
        }
      }
    }, 100);

    // Timeout after 10 seconds
    setTimeout(() => {
      clearInterval(checkTurnstile);
      if (!turnstileReady.value) {
        turnstileError.value = 'Verification service is not available. Please try again later.';
      }
    }, 10000);
  };

  const resetTurnstile = () => {
    if (window.turnstile && turnstileWidgetId.value) {
      window.turnstile.reset(turnstileWidgetId.value);
      turnstileToken.value = '';
      turnstileError.value = '';
    }
  };

  const removeTurnstile = () => {
    if (window.turnstile && turnstileWidgetId.value) {
      window.turnstile.remove(turnstileWidgetId.value);
      turnstileWidgetId.value = '';
      turnstileToken.value = '';
      turnstileError.value = '';
    }
  };

  return {
    turnstileToken,
    turnstileError,
    turnstileReady,
    initTurnstile,
    resetTurnstile,
    removeTurnstile,
  };
}
