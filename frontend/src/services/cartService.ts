import { API_CONFIG, getAuthHeaders } from '../config/api';
import type { CartResponse, AddToCartRequest, UpdateCartItemRequest, WishlistResponse } from '../types';

export const cartService = {
  async getCart(): Promise<CartResponse> {
    const response = await fetch(`${API_CONFIG.CART_SERVICE}/api/v1/cart`, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch cart');
    }

    return response.json();
  },

  async addToCart(data: AddToCartRequest): Promise<void> {
    const response = await fetch(`${API_CONFIG.CART_SERVICE}/api/v1/cart`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to add to cart');
    }
  },

  async updateCartItem(id: number, data: UpdateCartItemRequest): Promise<void> {
    const response = await fetch(`${API_CONFIG.CART_SERVICE}/api/v1/cart/${id}`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update cart item');
    }
  },

  async removeFromCart(id: number): Promise<void> {
    const response = await fetch(`${API_CONFIG.CART_SERVICE}/api/v1/cart/${id}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to remove from cart');
    }
  },

  async clearCart(): Promise<void> {
    const response = await fetch(`${API_CONFIG.CART_SERVICE}/api/v1/cart`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to clear cart');
    }
  },

  async getWishlist(): Promise<WishlistResponse> {
    const response = await fetch(`${API_CONFIG.CART_SERVICE}/api/v1/wishlist`, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch wishlist');
    }

    return response.json();
  },

  async toggleWishlistItem(productId: number): Promise<{ wishlisted: boolean }> {
    const response = await fetch(`${API_CONFIG.CART_SERVICE}/api/v1/wishlist/${productId}`, {
      method: 'POST',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to toggle wishlist');
    }

    return response.json();
  },
};
