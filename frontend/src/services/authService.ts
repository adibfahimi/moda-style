import { API_CONFIG, getAuthHeaders } from '../config/api';
import type { LoginRequest, RegisterRequest, AuthResponse, User, UpdateProfileRequest } from '../types';

export const authService = {
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

  logout() {
    localStorage.removeItem('token');
  },

  isAuthenticated(): boolean {
    return !!localStorage.getItem('token');
  },
};
