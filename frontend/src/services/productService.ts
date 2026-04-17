import { API_CONFIG, getAuthHeaders } from '../config/api';
import type { Product, Category, Review, CreateReviewRequest } from '../types';

// Export listCategories as a standalone function
export const listCategories = async (): Promise<Category[]> => {
  const response = await fetch(`${API_CONFIG.PRODUCT_SERVICE}/api/v1/categories`);

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to fetch categories');
  }

  const data = await response.json();
  return data.categories || [];
};

export const productService = {
  async listProducts(params?: {
    category?: string;
    minPrice?: number;
    maxPrice?: number;
    search?: string;
  }): Promise<Product[]> {
    const queryParams = new URLSearchParams();
    if (params?.category) queryParams.append('category', params.category);
    if (params?.minPrice) queryParams.append('min_price', params.minPrice.toString());
    if (params?.maxPrice) queryParams.append('max_price', params.maxPrice.toString());
    if (params?.search) queryParams.append('search', params.search);

    const url = `${API_CONFIG.PRODUCT_SERVICE}/api/v1/products${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
    const response = await fetch(url);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch products');
    }

    const data = await response.json();
    return data.products || [];
  },

  async getProduct(id: number): Promise<Product> {
    const response = await fetch(`${API_CONFIG.PRODUCT_SERVICE}/api/v1/products/${id}`);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch product');
    }

    const data = await response.json();
    return data.product;
  },

  async listCategories(): Promise<Category[]> {
    const response = await fetch(`${API_CONFIG.PRODUCT_SERVICE}/api/v1/categories`);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch categories');
    }

    const data = await response.json();
    return data.categories || [];
  },

  async getProductReviews(productId: number): Promise<Review[]> {
    const response = await fetch(
      `${API_CONFIG.PRODUCT_SERVICE}/api/v1/products/${productId}/reviews`
    );

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch reviews');
    }

    const data = await response.json();
    return data.reviews || [];
  },

  async createReview(productId: number, data: CreateReviewRequest): Promise<Review> {
    const response = await fetch(
      `${API_CONFIG.PRODUCT_SERVICE}/api/v1/products/${productId}/reviews`,
      {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify(data),
      }
    );

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create review');
    }

    const result = await response.json();
    return result.review;
  },
};
