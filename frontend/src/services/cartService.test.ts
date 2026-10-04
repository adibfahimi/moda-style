import { beforeEach, describe, expect, it } from 'vitest';
import { cartService } from './cartService';
import {
  fetchCall,
  installFetchMock,
  jsonBody,
  jsonResponse,
  type FetchMock,
} from '../test/support';

const CART_URL = 'http://localhost:8003/api/v1/cart';

let fetchMock: FetchMock;

beforeEach(() => {
  fetchMock = installFetchMock();
  localStorage.setItem('token', 'jwt-123');
});

describe('cartService.getCart', () => {
  it('returns the cart snapshot with the bearer token attached', async () => {
    const cart = { items: [{ id: 1, quantity: 2 }], subtotal: 120, count: 2 };
    fetchMock.mockResolvedValue(jsonResponse(cart));

    await expect(cartService.getCart()).resolves.toEqual(cart);

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(CART_URL);
    expect(init.headers).toMatchObject({ Authorization: 'Bearer jwt-123' });
  });

  it('throws the backend message when the cart cannot be loaded', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'Cart service unavailable' }, { status: 503 }),
    );

    await expect(cartService.getCart()).rejects.toThrow('Cart service unavailable');
  });

  it('falls back to a generic message when the body carries none', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));

    await expect(cartService.getCart()).rejects.toThrow('Failed to fetch cart');
  });
});

describe('cart mutations', () => {
  it('adds an item through a POST of the size and quantity', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ message: 'Added' }));

    await cartService.addToCart({ product_id: 7, size_id: 3, quantity: 2 });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(CART_URL);
    expect(init.method).toBe('POST');
    expect(jsonBody(init)).toEqual({ product_id: 7, size_id: 3, quantity: 2 });
  });

  it('updates a quantity through a PUT of the item', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ message: 'Updated' }));

    await cartService.updateCartItem(11, { quantity: 5 });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(`${CART_URL}/11`);
    expect(init.method).toBe('PUT');
    expect(jsonBody(init)).toEqual({ quantity: 5 });
  });

  it('removes an item through a DELETE of the item', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ message: 'Removed' }));

    await cartService.removeFromCart(11);

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(`${CART_URL}/11`);
    expect(init.method).toBe('DELETE');
  });

  it('empties the whole cart through a DELETE of the collection', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ message: 'Cleared' }));

    await cartService.clearCart();

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(CART_URL);
    expect(init.method).toBe('DELETE');
  });

  it('reports mutation failures', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'Only 3 left in stock' }, { status: 409 }));

    await expect(cartService.addToCart({ product_id: 7, size_id: 3, quantity: 4 })).rejects.toThrow(
      'Only 3 left in stock',
    );
  });
});

describe('wishlist', () => {
  it('lists the wishlist', async () => {
    const wishlist = { items: [{ id: 1, product_id: 7 }], count: 1 };
    fetchMock.mockResolvedValue(jsonResponse(wishlist));

    await expect(cartService.getWishlist()).resolves.toEqual(wishlist);
    expect(fetchCall(fetchMock).url).toBe('http://localhost:8003/api/v1/wishlist');
  });

  it('toggles a product through a POST and reports the new state', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ wishlisted: true }));

    await expect(cartService.toggleWishlistItem(7)).resolves.toEqual({ wishlisted: true });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe('http://localhost:8003/api/v1/wishlist/7');
    expect(init.method).toBe('POST');
  });

  it('reports wishlist failures', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));

    await expect(cartService.getWishlist()).rejects.toThrow('Failed to fetch wishlist');
    await expect(cartService.toggleWishlistItem(7)).rejects.toThrow('Failed to toggle wishlist');
  });
});
