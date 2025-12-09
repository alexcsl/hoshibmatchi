import { AxiosError } from "axios";

/**
 * Formats error messages into user-friendly text
 * Converts technical error codes and messages into readable format
 */
export function formatErrorMessage(error: any): string {
  // Network errors
  if (!error.response) {
    if (error.message?.includes("Network Error")) {
      return "Unable to connect to the server. Please check your internet connection.";
    }
    if (error.code === "ECONNABORTED" || error.message?.includes("timeout")) {
      return "Request timed out. Please try again.";
    }
    return "Unable to reach the server. Please try again later.";
  }

  const status = error.response?.status;
  const serverMessage = error.response?.data?.error || error.response?.data?.message;

  // Handle specific HTTP status codes
  switch (status) {
    case 400:
      return serverMessage || "Invalid request. Please check your input.";
    case 401:
      // For login endpoints, show server message (like "Invalid credentials")
      // For other endpoints, show generic auth message
      return serverMessage || "You are not authorized. Please log in again.";
    case 403:
      return "You don't have permission to perform this action.";
    case 404:
      return serverMessage || "The requested resource was not found.";
    case 409:
      return serverMessage || "This action conflicts with existing data.";
    case 422:
      return serverMessage || "The data provided is invalid.";
    case 429:
      return "Too many requests. Please slow down and try again.";
    case 500:
      return "Server error. Please try again later.";
    case 502:
      return "Service temporarily unavailable. Please try again.";
    case 503:
      return "Service is under maintenance. Please try again later.";
    default:
      return serverMessage || "An unexpected error occurred. Please try again.";
  }
}

/**
 * Extracts validation errors from API response
 * Returns an object with field names as keys and error messages as values
 */
export function extractValidationErrors(error: AxiosError): Record<string, string> | null {
  if (error.response?.status === 422) {
    const data = error.response.data as any;
    if (data?.errors && typeof data.errors === "object") {
      return data.errors;
    }
  }
  return null;
}

/**
 * Shows a user-friendly error notification
 * Can be used with toast/notification libraries
 */
export function handleApiError(error: any, customMessage?: string): string {
  const message = customMessage || formatErrorMessage(error);
  console.error("API Error:", error);
  return message;
}
