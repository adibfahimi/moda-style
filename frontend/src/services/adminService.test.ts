import { beforeEach, describe, expect, it } from 'vitest';
import * as adminService from './adminService';
import {
  fetchCall,
  installFetchMock,
  jsonBody,
  jsonResponse,
  unparseableResponse,
  type FetchMock,
} from '../test/support';

const BASE = 'http://localhost:8004/api/v1/admin';

interface AdminCase {
  /** `describe`/`it` label describing the call. */
  name: string;
  /** Invokes the helper under test. */
  call: () => Promise<unknown>;
  method: string;
  url: string;
  /** Expected JSON request body; omit for requests without one. */
  body?: unknown;
  /** Message the helper rejects with on a non-2xx response. */
  error: string;
}

let fetchMock: FetchMock;

beforeEach(() => {
  fetchMock = installFetchMock();
  localStorage.setItem('token', 'jwt-123');
});

const cases: AdminCase[] = [
  {
    name: 'getDashboardStats',
    call: () => adminService.getDashboardStats(),
    method: 'GET',
    url: `${BASE}/dashboard/stats`,
    error: 'Failed to fetch dashboard stats',
  },
  {
    name: 'getRecentActivity defaults to twenty entries',
    call: () => adminService.getRecentActivity(),
    method: 'GET',
    url: `${BASE}/dashboard/activity?limit=20`,
    error: 'Failed to fetch recent activity',
  },
  {
    name: 'getRecentActivity honours an explicit limit',
    call: () => adminService.getRecentActivity(5),
    method: 'GET',
    url: `${BASE}/dashboard/activity?limit=5`,
    error: 'Failed to fetch recent activity',
  },
  {
    name: 'getActivityLogs serialises every filter',
    call: () =>
      adminService.getActivityLogs({
        page: 2,
        limit: 25,
        admin_id: 3,
        action: 'delete',
        resource: 'product',
      }),
    method: 'GET',
    url: `${BASE}/dashboard/logs?page=2&limit=25&admin_id=3&action=delete&resource=product`,
    error: 'Failed to fetch activity logs',
  },
  {
    name: 'getActivityLogs sends no query string without filters',
    call: () => adminService.getActivityLogs({}),
    method: 'GET',
    url: `${BASE}/dashboard/logs?`,
    error: 'Failed to fetch activity logs',
  },
  {
    name: 'getUsers serialises every filter',
    call: () =>
      adminService.getUsers({ page: 1, limit: 10, role: 'admin', search: 'ada' }),
    method: 'GET',
    url: `${BASE}/users?page=1&limit=10&role=admin&search=ada`,
    error: 'Failed to fetch users',
  },
  {
    name: 'getUserAnalytics',
    call: () => adminService.getUserAnalytics(),
    method: 'GET',
    url: `${BASE}/users/analytics`,
    error: 'Failed to fetch user analytics',
  },
  {
    name: 'getUserDetails',
    call: () => adminService.getUserDetails(9),
    method: 'GET',
    url: `${BASE}/users/9`,
    error: 'Failed to fetch user details',
  },
  {
    name: 'updateUser',
    call: () => adminService.updateUser(9, { name: 'Ada', role: 'admin' }),
    method: 'PUT',
    url: `${BASE}/users/9`,
    body: { name: 'Ada', role: 'admin' },
    error: 'Failed to update user',
  },
  {
    name: 'deleteUser',
    call: () => adminService.deleteUser(9),
    method: 'DELETE',
    url: `${BASE}/users/9`,
    error: 'Failed to delete user',
  },
  {
    name: 'banUser',
    call: () => adminService.banUser(9),
    method: 'POST',
    url: `${BASE}/users/9/ban`,
    error: 'Failed to ban user',
  },
  {
    name: 'unbanUser',
    call: () => adminService.unbanUser(9),
    method: 'POST',
    url: `${BASE}/users/9/unban`,
    error: 'Failed to unban user',
  },
];

cases.push(
  {
    name: 'getProductStats serialises pagination',
    call: () => adminService.getProductStats({ page: 1, limit: 20 }),
    method: 'GET',
    url: `${BASE}/products?page=1&limit=20`,
    error: 'Failed to fetch products',
  },
  {
    name: 'createProduct',
    call: () =>
      adminService.createProduct({ name: 'Silk shirt', description: 'Soft', price: 89, category_id: 2 }),
    method: 'POST',
    url: `${BASE}/products`,
    body: { name: 'Silk shirt', description: 'Soft', price: 89, category_id: 2 },
    error: 'Failed to create product',
  },
  {
    name: 'listProductImages',
    call: () => adminService.listProductImages(),
    method: 'GET',
    url: `${BASE}/products/images`,
    error: 'Failed to fetch uploaded product images',
  },
  {
    name: 'updateProduct',
    call: () => adminService.updateProduct(4, { price: 119 }),
    method: 'PUT',
    url: `${BASE}/products/4`,
    body: { price: 119 },
    error: 'Failed to update product',
  },
  {
    name: 'deleteProduct',
    call: () => adminService.deleteProduct(4),
    method: 'DELETE',
    url: `${BASE}/products/4`,
    error: 'Failed to delete product',
  },
  {
    name: 'addProductSize',
    call: () => adminService.addProductSize(4, { size: 'M', color: 'black', stock: 12 }),
    method: 'POST',
    url: `${BASE}/products/4/sizes`,
    body: { size: 'M', color: 'black', stock: 12 },
    error: 'Failed to add product size',
  },
  {
    name: 'getProductSizes',
    call: () => adminService.getProductSizes(4),
    method: 'GET',
    url: `${BASE}/products/4/sizes`,
    error: 'Failed to fetch product sizes',
  },
  {
    name: 'updateProductSize',
    call: () => adminService.updateProductSize(4, 7, { stock: 3 }),
    method: 'PUT',
    url: `${BASE}/products/4/sizes/7`,
    body: { stock: 3 },
    error: 'Failed to update product size',
  },
  {
    name: 'deleteProductSize',
    call: () => adminService.deleteProductSize(4, 7),
    method: 'DELETE',
    url: `${BASE}/products/4/sizes/7`,
    error: 'Failed to delete product size',
  },
);

cases.push(
  {
    name: 'getCategories',
    call: () => adminService.getCategories(),
    method: 'GET',
    url: `${BASE}/categories`,
    error: 'Failed to fetch categories',
  },
  {
    name: 'createCategory',
    call: () => adminService.createCategory({ name: 'Bags', parent_id: 2 }),
    method: 'POST',
    url: `${BASE}/categories`,
    body: { name: 'Bags', parent_id: 2 },
    error: 'Failed to create category',
  },
  {
    name: 'updateCategory',
    call: () => adminService.updateCategory(6, { slug: 'handbags' }),
    method: 'PUT',
    url: `${BASE}/categories/6`,
    body: { slug: 'handbags' },
    error: 'Failed to update category',
  },
  {
    name: 'deleteCategory',
    call: () => adminService.deleteCategory(6),
    method: 'DELETE',
    url: `${BASE}/categories/6`,
    error: 'Failed to delete category',
  },
  {
    name: 'getOrders serialises every filter',
    call: () =>
      adminService.getOrders({
        page: 3,
        limit: 10,
        status: 'pending',
        payment_status: 'paid',
        search: 'MS-1',
      }),
    method: 'GET',
    url: `${BASE}/orders?page=3&limit=10&status=pending&payment_status=paid&search=MS-1`,
    error: 'Failed to fetch orders',
  },
  {
    name: 'getOrderAnalytics',
    call: () => adminService.getOrderAnalytics(),
    method: 'GET',
    url: `${BASE}/orders/analytics`,
    error: 'Failed to fetch order analytics',
  },
  {
    name: 'getOrderDetails',
    call: () => adminService.getOrderDetails(42),
    method: 'GET',
    url: `${BASE}/orders/42`,
    error: 'Failed to fetch order details',
  },
  {
    name: 'updateOrderStatus',
    call: () => adminService.updateOrderStatus(42, { status: 'shipped' }),
    method: 'PUT',
    url: `${BASE}/orders/42`,
    body: { status: 'shipped' },
    error: 'Failed to update order',
  },
  {
    name: 'deleteOrder',
    call: () => adminService.deleteOrder(42),
    method: 'DELETE',
    url: `${BASE}/orders/42`,
    error: 'Failed to delete order',
  },
);

describe('adminService helpers', () => {
  cases.forEach(({ name, call, method, url, body, error }) => {
    it(`${name} calls ${method} ${url}`, async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({ ok: true }));

      await expect(call()).resolves.toEqual({ ok: true });

      const { url: calledUrl, init } = fetchCall(fetchMock);
      expect(calledUrl).toBe(url);
      expect(init.method ?? 'GET').toBe(method);
      expect(init.headers).toMatchObject({ Authorization: 'Bearer jwt-123' });
      if (body !== undefined) {
        expect(jsonBody(init)).toEqual(body);
      }
    });

    it(`${name} reports failures`, async () => {
      fetchMock.mockResolvedValue(jsonResponse({ error: 'nope' }, { status: 500 }));

      await expect(call()).rejects.toThrow(error);
    });
  });
});

describe('adminService.uploadProductImage', () => {
  it('posts the file as multipart data with the bearer token', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ url: '/uploads/look.png' }));
    const file = new File(['binary'], 'look.png', { type: 'image/png' });

    await expect(adminService.uploadProductImage(file)).resolves.toEqual({
      url: '/uploads/look.png',
    });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(`${BASE}/products/upload-image`);
    expect(init.method).toBe('POST');
    expect(init.headers).toMatchObject({ Authorization: 'Bearer jwt-123' });
    const formData = init.body as FormData;
    expect(formData.get('image')).toBe(file);
  });

  it('surfaces the backend message when the upload is refused', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'Only PNG and JPEG' }, { status: 415 }));

    await expect(
      adminService.uploadProductImage(new File(['x'], 'look.gif', { type: 'image/gif' })),
    ).rejects.toThrow('Only PNG and JPEG');
  });

  it('falls back to a generic message when the error body is not JSON', async () => {
    fetchMock.mockResolvedValue(unparseableResponse(413));

    await expect(
      adminService.uploadProductImage(new File(['x'], 'look.png', { type: 'image/png' })),
    ).rejects.toThrow('Failed to upload product image');
  });
});
