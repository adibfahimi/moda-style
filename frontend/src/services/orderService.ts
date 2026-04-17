import { API_CONFIG, getAuthHeaders } from '../config/api';
import type { Order } from '../types';

export interface CreateOrderRequest {
  shipping_address: string;
  payment_method: string;
  notes?: string;
}

export interface CreateOrderResponse {
  order: Order;
  message: string;
}

export interface ProcessPaymentRequest {
  payment_intent_id: string;
}

export interface ProcessPaymentResponse {
  success: boolean;
  order: Order;
  message: string;
}

/**
 * Order Service
 * 
 * This service handles all order-related API calls with the backend order service.
 */
export const orderService = {
  /**
   * Create an order from the current cart
   */
  async createOrder(data: CreateOrderRequest): Promise<CreateOrderResponse> {
    const response = await fetch(`${API_CONFIG.ORDER_SERVICE}/api/v1/orders`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create order');
    }

    const result = await response.json();
    console.log('📦 [Order API] Order Created:', result.order);
    return result;
  },

  /**
   * Process payment for an order
   */
  async processPayment(
    orderId: number,
    data: ProcessPaymentRequest
  ): Promise<ProcessPaymentResponse> {
    const response = await fetch(`${API_CONFIG.ORDER_SERVICE}/api/v1/orders/${orderId}/pay`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to process payment');
    }

    const result = await response.json();
    console.log('💰 [Order API] Payment Processed:', result);
    return result;
  },

  /**
   * Get user's order history
   */
  async getMyOrders(): Promise<Order[]> {
    const response = await fetch(`${API_CONFIG.ORDER_SERVICE}/api/v1/orders/my-orders`, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch orders');
    }

    const orders = await response.json();
    console.log('📋 [Order API] Fetched Orders:', orders);
    return orders;
  },

  /**
   * Get order details by ID
   */
  async getOrderById(orderId: number): Promise<Order> {
    const response = await fetch(`${API_CONFIG.ORDER_SERVICE}/api/v1/orders/${orderId}`, {
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch order');
    }

    const order = await response.json();
    console.log('📄 [Order API] Fetched Order:', order);
    return order;
  },

  /**
   * Cancel an order
   */
  async cancelOrder(orderId: number): Promise<void> {
    const response = await fetch(`${API_CONFIG.ORDER_SERVICE}/api/v1/orders/${orderId}/cancel`, {
      method: 'POST',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to cancel order');
    }

    console.log('❌ [Order API] Order Cancelled:', orderId);
  },
};

/**
 * HOW TO INTEGRATE WITH REAL BACKEND:
 * 
 * 1. Add order service to docker-compose.yml (port 8005)
 * 
 * 2. Update frontend/src/config/api.ts:
 *    export const API_CONFIG = {
 *      AUTH_SERVICE: 'http://localhost:8001',
 *      PRODUCT_SERVICE: 'http://localhost:8002',
 *      CART_SERVICE: 'http://localhost:8003',
 *      ADMIN_SERVICE: 'http://localhost:8004',
 *      ORDER_SERVICE: 'http://localhost:8005', // Add this
 *    };
 * 
 * 3. Update this file to use real API calls:
 * 
 *    async createOrder(data: CreateOrderRequest): Promise<CreateOrderResponse> {
 *      const response = await fetch(`${API_CONFIG.ORDER_SERVICE}/api/v1/orders`, {
 *        method: 'POST',
 *        headers: getAuthHeaders(),
 *        body: JSON.stringify(data),
 *      });
 *      
 *      if (!response.ok) {
 *        const error = await response.json();
 *        throw new Error(error.error || 'Failed to create order');
 *      }
 *      
 *      return response.json();
 *    }
 * 
 * 4. Implement these backend endpoints:
 *    POST   /api/v1/orders              - Create order from cart
 *    POST   /api/v1/orders/:id/pay      - Process payment
 *    GET    /api/v1/orders/my-orders    - Get user orders
 *    GET    /api/v1/orders/:id          - Get order details
 *    POST   /api/v1/orders/:id/cancel   - Cancel order
 */
