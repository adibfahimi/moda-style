import { API_CONFIG, getAuthHeaders } from '../config/api';

const BASE_URL = `${API_CONFIG.ADMIN_SERVICE}/api/v1/admin`;

// Dashboard & Analytics
export const getDashboardStats = async () => {
  const response = await fetch(`${BASE_URL}/dashboard/stats`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch dashboard stats');
  return response.json();
};

export const getRecentActivity = async (limit = 20) => {
  const response = await fetch(`${BASE_URL}/dashboard/activity?limit=${limit}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch recent activity');
  return response.json();
};

export const getActivityLogs = async (params: {
  page?: number;
  limit?: number;
  admin_id?: number;
  action?: string;
  resource?: string;
}) => {
  const queryParams = new URLSearchParams();
  if (params.page) queryParams.append('page', params.page.toString());
  if (params.limit) queryParams.append('limit', params.limit.toString());
  if (params.admin_id) queryParams.append('admin_id', params.admin_id.toString());
  if (params.action) queryParams.append('action', params.action);
  if (params.resource) queryParams.append('resource', params.resource);

  const response = await fetch(`${BASE_URL}/dashboard/logs?${queryParams}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch activity logs');
  return response.json();
};

// User Management
export const getUsers = async (params: {
  page?: number;
  limit?: number;
  role?: string;
  search?: string;
}) => {
  const queryParams = new URLSearchParams();
  if (params.page) queryParams.append('page', params.page.toString());
  if (params.limit) queryParams.append('limit', params.limit.toString());
  if (params.role) queryParams.append('role', params.role);
  if (params.search) queryParams.append('search', params.search);

  const response = await fetch(`${BASE_URL}/users?${queryParams}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch users');
  return response.json();
};

export const getUserAnalytics = async () => {
  const response = await fetch(`${BASE_URL}/users/analytics`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch user analytics');
  return response.json();
};

export const getUserDetails = async (id: number) => {
  const response = await fetch(`${BASE_URL}/users/${id}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch user details');
  return response.json();
};

export const updateUser = async (id: number, data: {
  name?: string;
  email?: string;
  role?: string;
}) => {
  const response = await fetch(`${BASE_URL}/users/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to update user');
  return response.json();
};

export const deleteUser = async (id: number) => {
  const response = await fetch(`${BASE_URL}/users/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to delete user');
  return response.json();
};

export const banUser = async (id: number) => {
  const response = await fetch(`${BASE_URL}/users/${id}/ban`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to ban user');
  return response.json();
};

export const unbanUser = async (id: number) => {
  const response = await fetch(`${BASE_URL}/users/${id}/unban`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to unban user');
  return response.json();
};

// Product Management
export const getProductStats = async (params: {
  page?: number;
  limit?: number;
}) => {
  const queryParams = new URLSearchParams();
  if (params.page) queryParams.append('page', params.page.toString());
  if (params.limit) queryParams.append('limit', params.limit.toString());

  const response = await fetch(`${BASE_URL}/products?${queryParams}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch products');
  return response.json();
};

export const createProduct = async (data: {
  name: string;
  description: string;
  price: number;
  category_id: number;
  image_url?: string;
}) => {
  const response = await fetch(`${BASE_URL}/products`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to create product');
  return response.json();
};

export const uploadProductImage = async (file: File) => {
  const token = localStorage.getItem('token');
  const formData = new FormData();
  formData.append('image', file);

  const response = await fetch(`${BASE_URL}/products/upload-image`, {
    method: 'POST',
    headers: {
      ...(token && { Authorization: `Bearer ${token}` }),
    },
    body: formData,
  });

  if (!response.ok) {
    const errorPayload = await response.json().catch(() => null);
    throw new Error(errorPayload?.error || 'Failed to upload product image');
  }

  return response.json();
};

export const listProductImages = async () => {
  const response = await fetch(`${BASE_URL}/products/images`, {
    headers: getAuthHeaders(),
  });

  if (!response.ok) throw new Error('Failed to fetch uploaded product images');
  return response.json();
};

export const updateProduct = async (id: number, data: {
  name?: string;
  description?: string;
  price?: number;
  category_id?: number;
  image_url?: string;
}) => {
  const response = await fetch(`${BASE_URL}/products/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to update product');
  return response.json();
};

export const deleteProduct = async (id: number) => {
  const response = await fetch(`${BASE_URL}/products/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to delete product');
  return response.json();
};

// Product Size Management
export const addProductSize = async (productId: number, data: {
  size: string;
  color: string;
  stock: number;
}) => {
  const response = await fetch(`${BASE_URL}/products/${productId}/sizes`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to add product size');
  return response.json();
};

export const getProductSizes = async (productId: number) => {
  const response = await fetch(`${BASE_URL}/products/${productId}/sizes`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch product sizes');
  return response.json();
};

export const updateProductSize = async (productId: number, sizeId: number, data: {
  size?: string;
  color?: string;
  stock?: number;
}) => {
  const response = await fetch(`${BASE_URL}/products/${productId}/sizes/${sizeId}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to update product size');
  return response.json();
};

export const deleteProductSize = async (productId: number, sizeId: number) => {
  const response = await fetch(`${BASE_URL}/products/${productId}/sizes/${sizeId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to delete product size');
  return response.json();
};

// Category Management
export const getCategories = async () => {
  const response = await fetch(`${BASE_URL}/categories`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch categories');
  return response.json();
};

export const createCategory = async (data: {
  name: string;
  slug?: string;
  parent_id?: number;
}) => {
  const response = await fetch(`${BASE_URL}/categories`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to create category');
  return response.json();
};

export const updateCategory = async (id: number, data: {
  name?: string;
  slug?: string;
  parent_id?: number;
}) => {
  const response = await fetch(`${BASE_URL}/categories/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to update category');
  return response.json();
};

export const deleteCategory = async (id: number) => {
  const response = await fetch(`${BASE_URL}/categories/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to delete category');
  return response.json();
};

// Order Management
export const getOrders = async (params: {
  page?: number;
  limit?: number;
  status?: string;
  payment_status?: string;
  search?: string;
}) => {
  const queryParams = new URLSearchParams();
  if (params.page) queryParams.append('page', params.page.toString());
  if (params.limit) queryParams.append('limit', params.limit.toString());
  if (params.status) queryParams.append('status', params.status);
  if (params.payment_status) queryParams.append('payment_status', params.payment_status);
  if (params.search) queryParams.append('search', params.search);

  const response = await fetch(`${BASE_URL}/orders?${queryParams}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch orders');
  return response.json();
};

export const getOrderAnalytics = async () => {
  const response = await fetch(`${BASE_URL}/orders/analytics`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch order analytics');
  return response.json();
};

export const getOrderDetails = async (id: number) => {
  const response = await fetch(`${BASE_URL}/orders/${id}`, {
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to fetch order details');
  return response.json();
};

export const updateOrderStatus = async (id: number, data: {
  status?: string;
  payment_status?: string;
  notes?: string;
}) => {
  const response = await fetch(`${BASE_URL}/orders/${id}`, {
    method: 'PUT',
    headers: getAuthHeaders(),
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to update order');
  return response.json();
};

export const deleteOrder = async (id: number) => {
  const response = await fetch(`${BASE_URL}/orders/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });
  if (!response.ok) throw new Error('Failed to delete order');
  return response.json();
};

