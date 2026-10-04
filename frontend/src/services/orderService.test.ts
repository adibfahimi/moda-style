import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderService } from './orderService';
import {
  fetchCall,
  installFetchMock,
  jsonBody,
  jsonResponse,
  type FetchMock,
} from '../test/support';

const ORDERS_URL = 'http://localhost:8005/api/v1/orders';

let fetchMock: FetchMock;

beforeEach(() => {
  fetchMock = installFetchMock();
  localStorage.setItem('token', 'jwt-123');
  // The service logs every payload it receives; keep the test output readable.
  vi.spyOn(console, 'log').mockImplementation(() => {});
});

describe('orderService.createOrder', () => {
  it('posts the checkout form and returns the created order', async () => {
    const response = { order: { id: 42, order_number: 'MS-42' }, message: 'Order created' };
    fetchMock.mockResolvedValue(jsonResponse(response));

    const result = await orderService.createOrder({
      shipping_address: '12 Rue de la Mode, Paris',
      payment_method: 'card',
      notes: 'Leave with the concierge',
    });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(ORDERS_URL);
    expect(init.method).toBe('POST');
    expect(init.headers).toMatchObject({ Authorization: 'Bearer jwt-123' });
    expect(jsonBody(init)).toEqual({
      shipping_address: '12 Rue de la Mode, Paris',
      payment_method: 'card',
      notes: 'Leave with the concierge',
    });
    expect(result).toEqual(response);
  });

  it('throws when the cart cannot be checked out', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'Cart is empty' }, { status: 400 }));

    await expect(
      orderService.createOrder({ shipping_address: 'x', payment_method: 'card' }),
    ).rejects.toThrow('Cart is empty');
  });

  it('falls back to a generic message when the body carries none', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));

    await expect(
      orderService.createOrder({ shipping_address: 'x', payment_method: 'card' }),
    ).rejects.toThrow('Failed to create order');
  });
});

describe('orderService.processPayment', () => {
  it('posts the payment intent to the order', async () => {
    const response = { success: true, order: { id: 42 }, message: 'Paid' };
    fetchMock.mockResolvedValue(jsonResponse(response));

    await expect(
      orderService.processPayment(42, { payment_intent_id: 'pi_123' }),
    ).resolves.toEqual(response);

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(`${ORDERS_URL}/42/pay`);
    expect(init.method).toBe('POST');
    expect(jsonBody(init)).toEqual({ payment_intent_id: 'pi_123' });
  });

  it('throws the backend message when the payment is refused', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'Card declined' }, { status: 402 }));

    await expect(
      orderService.processPayment(42, { payment_intent_id: 'pi_123' }),
    ).rejects.toThrow('Card declined');
  });
});

describe('orderService reads', () => {
  it('returns the order history as a bare array', async () => {
    fetchMock.mockResolvedValue(jsonResponse([{ id: 1 }, { id: 2 }]));

    await expect(orderService.getMyOrders()).resolves.toEqual([{ id: 1 }, { id: 2 }]);
    expect(fetchCall(fetchMock).url).toBe(`${ORDERS_URL}/my-orders`);
  });

  it('returns a single order by id', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ id: 42, status: 'shipped' }));

    await expect(orderService.getOrderById(42)).resolves.toEqual({ id: 42, status: 'shipped' });
    expect(fetchCall(fetchMock).url).toBe(`${ORDERS_URL}/42`);
  });

  it('surfaces read failures', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));

    await expect(orderService.getMyOrders()).rejects.toThrow('Failed to fetch orders');
    await expect(orderService.getOrderById(42)).rejects.toThrow('Failed to fetch order');
  });
});

describe('orderService.cancelOrder', () => {
  it('posts the cancellation', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ message: 'Cancelled' }));

    await orderService.cancelOrder(42);

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(`${ORDERS_URL}/42/cancel`);
    expect(init.method).toBe('POST');
  });

  it('throws when the order can no longer be cancelled', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'Shipped orders cannot be cancelled' }, { status: 409 }),
    );

    await expect(orderService.cancelOrder(42)).rejects.toThrow(
      'Shipped orders cannot be cancelled',
    );
  });
});
