export interface User {
  id: number;
  name: string;
  email: string;
  role: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  name: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface Product {
  id: number;
  name: string;
  description: string;
  price: number;
  image_url: string;
  category_id: number;
  stock: number;
  sizes?: Size[];
  category?: Category;
  created_at: string;
  updated_at: string;
}

export interface Category {
  id: number;
  name: string;
  slug: string;
  parent_id?: number;
  parent?: Category;
  children?: Category[];
  created_at: string;
  updated_at: string;
}

export interface Review {
  id: number;
  product_id: number;
  user_id: number;
  rating: number;
  comment: string;
  user_name: string;
  created_at: string;
  updated_at: string;
}

export interface CreateReviewRequest {
  rating: number;
  comment: string;
}

export interface UpdateProfileRequest {
  name?: string;
  email?: string;
}

export interface ApiError {
  error: string;
}

export interface Size {
  id: number;
  product_id: number;
  size: string;
  color: string;
  stock: number;
}

export interface CartItem {
  id: number;
  product_id: number;
  name: string;
  image_url: string;
  price: number;
  size: string;
  color: string;
  size_id: number;
  quantity: number;
  stock: number;
}

export interface AddToCartRequest {
  product_id: number;
  size_id: number;
  quantity: number;
}

export interface UpdateCartItemRequest {
  quantity: number;
}

export interface CartResponse {
  items: CartItem[];
  subtotal: number;
  count: number;
}

export interface WishlistItem {
  id: number;
  product_id: number;
  name: string;
  image_url: string;
  price: number;
  in_stock: boolean;
}

export interface WishlistResponse {
  items: WishlistItem[];
  count: number;
}

// Admin Dashboard Types
export interface DashboardStats {
  total_users: number;
  total_products: number;
  total_categories: number;
  total_reviews: number;
  low_stock_products: number;
  average_rating: number;
}

export interface RecentActivity {
  id: number;
  type: string;
  description: string;
  user_id?: number;
  user_name?: string;
  created_at: string;
}

export interface ActivityLog {
  id: number;
  admin_id: number;
  admin_name: string;
  action: string;
  resource: string;
  resource_id: number;
  description: string;
  ip_address: string;
  created_at: string;
}

export interface UserStats {
  id: number;
  name: string;
  email: string;
  role: string;
  banned: boolean;
  review_count: number;
  wishlist_count: number;
  created_at: string;
}

export interface UserAnalytics {
  total_users: number;
  admin_users: number;
  regular_users: number;
  new_users_today: number;
  new_users_this_week: number;
  active_reviewers: number;
}

export interface ProductStats {
  id: number;
  name: string;
  description: string;
  category_id: number;
  category_name: string;
  price: number;
  stock: number;
  image_url: string;
  review_count: number;
  average_rating: number;
  wishlist_count: number;
}

export interface CategoryWithCount {
  id: number;
  name: string;
  slug: string;
  parent_id?: number;
  parent_name?: string;
  product_count: number;
  created_at: string;
}

// Order Management Types
export interface Order {
  id: number;
  user_id: number;
  order_number: string;
  status: 'pending' | 'processing' | 'shipped' | 'delivered' | 'cancelled';
  total_amount: number;
  shipping_address: string;
  payment_method: string;
  payment_status: 'pending' | 'paid' | 'failed' | 'refunded';
  notes: string;
  created_at: string;
  updated_at: string;
  items?: OrderItem[];
}

export interface OrderItem {
  id: number;
  order_id: number;
  product_id: number;
  product_name: string;
  quantity: number;
  price: number;
  subtotal: number;
  created_at: string;
}

export interface OrderStats {
  id: number;
  order_number: string;
  user_id: number;
  user_name: string;
  user_email: string;
  status: string;
  payment_status: string;
  payment_method: string;
  total_amount: number;
  item_count: number;
  shipping_address: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface OrderAnalytics {
  total_orders: number;
  pending_orders: number;
  processing_orders: number;
  shipped_orders: number;
  delivered_orders: number;
  cancelled_orders: number;
  total_revenue: number;
  pending_revenue: number;
  paid_revenue: number;
  orders_today: number;
  orders_this_week: number;
  orders_this_month: number;
}

