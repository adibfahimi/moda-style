import { API_CONFIG, getAuthHeaders } from '../config/api';
import type { LoginRequest, RegisterRequest, AuthResponse, User, UpdateProfileRequest } from '../types';

/**
 * Authentication service — thin wrapper around the auth-service HTTP API.
 *
 * Successful `register` and `login` calls persist the returned JWT under the
 * `token` key of `localStorage`, which is exactly what {@link getAuthHeaders}
 * reads back for authenticated requests.
 */
export const authService = {
  /**
   * Creates an account and stores the returned JWT.
   *
   * @throws Error with the backend message, or `Registration failed`.
   */
  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Registration failed');
    }

    const result = await response.json();
    if (result.token) {
      localStorage.setItem('token', result.token);
    }
    return result;
  },

  /**
   * Signs in and stores the returned JWT.
   *
   * @throws Error with the backend message, or `Login failed`.
   */
  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Login failed');
    }

    const result = await response.json();
    if (result.token) {
      localStorage.setItem('token', result.token);
    }
    return result;
  },

  /**
   * Returns the signed-in user.
   *
   * The thrown error carries the HTTP `status`, so callers can distinguish an
   * expired session (401) from a server fault; a non-JSON body falls back to a
   * generic message.
   */
  async getProfile(): Promise<User> {
    const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/profile`, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      let message = 'Failed to fetch profile';
      try {
        const error = await response.json();
        message = error.error || message;
      } catch {
        // Keep fallback message when response body is not JSON.
      }

      const err = new Error(message) as Error & { status?: number };
      err.status = response.status;
      throw err;
    }

    const data = await response.json();
    return data.user;
  },

  /** Patches the current profile and returns the updated user. */
  async updateProfile(data: UpdateProfileRequest): Promise<User> {
    const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/profile`, {
      method: 'PATCH',
      headers: getAuthHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update profile');
    }

    const result = await response.json();
    return result.user;
  },

  /** Requests a password-reset e-mail for the given address. */
  async resetPassword(email: string): Promise<void> {
    const response = await fetch(`${API_CONFIG.AUTH_SERVICE}/api/v1/auth/reset-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to send reset email');
    }
  },

  /** Drops the stored JWT, signing the local session out. */
  logout() {
    localStorage.removeItem('token');
  },

  /** @returns `true` when a JWT is present in `localStorage`. */
  isAuthenticated(): boolean {
    return !!localStorage.getItem('token');
  },
};
