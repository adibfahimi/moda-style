import { beforeEach, describe, expect, it } from 'vitest';
import { listCategories, productService } from './productService';
import { fetchCall, installFetchMock, jsonBody, jsonResponse, type FetchMock } from '../test/support';

let fetchMock: FetchMock;

beforeEach(() => {
  fetchMock = installFetchMock();
});

describe('productService.listProducts', () => {
  it('returns the catalogue without a query string when unfiltered', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ products: [{ id: 1, name: 'Silk Dress' }] }));

    const products = await productService.listProducts();

    expect(fetchCall(fetchMock).url).toBe('http://localhost:8002/api/v1/products');
    expect(products).toEqual([{ id: 1, name: 'Silk Dress' }]);
  });

  it('maps every filter onto the query string the backend expects', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ products: [] }));

    await productService.listProducts({
      category: 'Women',
      minPrice: 25,
      maxPrice: 200,
      search: 'dress',
    });

    const { url } = fetchCall(fetchMock);
    expect(url.startsWith('http://localhost:8002/api/v1/products?')).toBe(true);
    expect(Object.fromEntries(new URL(url).searchParams)).toEqual({
      category: 'Women',
      min_price: '25',
      max_price: '200',
      search: 'dress',
    });
  });

  it('skips filters that are zero or empty', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ products: [] }));

    await productService.listProducts({ category: '', minPrice: 0, search: '' });

    expect(fetchCall(fetchMock).url).toBe('http://localhost:8002/api/v1/products');
  });

  it('always yields an array, even when the payload omits products', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}));

    await expect(productService.listProducts()).resolves.toEqual([]);
  });

  it('throws the backend message on failure', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'Catalogue unavailable' }, { status: 500 }));

    await expect(productService.listProducts()).rejects.toThrow('Catalogue unavailable');
  });
});

describe('productService.getProduct', () => {
  it('unwraps the product from the envelope', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ product: { id: 7, name: 'Wool Coat' } }));

    const product = await productService.getProduct(7);

    expect(fetchCall(fetchMock).url).toBe('http://localhost:8002/api/v1/products/7');
    expect(product).toEqual({ id: 7, name: 'Wool Coat' });
  });

  it('falls back to a generic message when the body carries none', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 404 }));

    await expect(productService.getProduct(404)).rejects.toThrow('Failed to fetch product');
  });
});

describe('categories', () => {
  it('lists categories through the standalone helper', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ categories: [{ id: 1, name: 'Women' }] }));

    const categories = await listCategories();

    expect(fetchCall(fetchMock).url).toBe('http://localhost:8002/api/v1/categories');
    expect(categories).toEqual([{ id: 1, name: 'Women' }]);
  });

  it('exposes the same call on the service object', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}));

    await expect(productService.listCategories()).resolves.toEqual([]);
    expect(fetchCall(fetchMock).url).toBe('http://localhost:8002/api/v1/categories');
  });

  it('throws the backend message when the lookup fails', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'No categories' }, { status: 500 }));

    await expect(listCategories()).rejects.toThrow('No categories');
  });
});

describe('reviews', () => {
  it('lists the reviews of a product', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ reviews: [{ id: 1, rating: 5 }] }));

    const reviews = await productService.getProductReviews(7);

    expect(fetchCall(fetchMock).url).toBe('http://localhost:8002/api/v1/products/7/reviews');
    expect(reviews).toEqual([{ id: 1, rating: 5 }]);
  });

  it('yields an empty list when a product has no reviews yet', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ reviews: null }));

    await expect(productService.getProductReviews(7)).resolves.toEqual([]);
  });

  it('posts a new review with the auth header and unwraps it', async () => {
    localStorage.setItem('token', 'jwt-123');
    fetchMock.mockResolvedValue(jsonResponse({ review: { id: 9, rating: 4 } }));

    const review = await productService.createReview(7, { rating: 4, comment: 'Great fit' });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe('http://localhost:8002/api/v1/products/7/reviews');
    expect(init.method).toBe('POST');
    expect(init.headers).toMatchObject({ Authorization: 'Bearer jwt-123' });
    expect(jsonBody(init)).toEqual({ rating: 4, comment: 'Great fit' });
    expect(review).toEqual({ id: 9, rating: 4 });
  });

  it('throws when the review is rejected', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'You already reviewed this product' }, { status: 409 }),
    );

    await expect(productService.createReview(7, { rating: 5, comment: '' })).rejects.toThrow(
      'You already reviewed this product',
    );
  });

  it('falls back to a generic message when reviews cannot be loaded', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));

    await expect(productService.getProductReviews(7)).rejects.toThrow('Failed to fetch reviews');
  });
});
