import { afterEach, describe, expect, it, vi } from 'vitest';

/**
 * `API_CONFIG` resolves every service URL once, when the module is evaluated
 * for the first time, so each case stubs the environment and then re-imports
 * the module to observe the effect.
 */
const loadApiModule = async () => {
  vi.resetModules();
  return import('./api');
};

describe('API_CONFIG', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it('points at the local development ports by default', async () => {
    const { API_CONFIG } = await loadApiModule();

    expect(API_CONFIG.AUTH_SERVICE).toBe('http://localhost:8001');
    expect(API_CONFIG.PRODUCT_SERVICE).toBe('http://localhost:8002');
    expect(API_CONFIG.CART_SERVICE).toBe('http://localhost:8003');
    expect(API_CONFIG.ADMIN_SERVICE).toBe('http://localhost:8004');
    expect(API_CONFIG.ORDER_SERVICE).toBe('http://localhost:8005');
  });

  it('prefers a VITE_* override over the local port', async () => {
    vi.stubEnv('VITE_AUTH_SERVICE_URL', 'https://auth.moda-style.test');

    const { API_CONFIG } = await loadApiModule();

    expect(API_CONFIG.AUTH_SERVICE).toBe('https://auth.moda-style.test');
    // Services without an override keep their development fallback.
    expect(API_CONFIG.CART_SERVICE).toBe('http://localhost:8003');
  });

  it('routes through the current origin once the bundle is not a dev build', async () => {
    vi.stubEnv('DEV', false);

    const { API_CONFIG } = await loadApiModule();

    expect(API_CONFIG.AUTH_SERVICE).toBe(window.location.origin);
    expect(API_CONFIG.ORDER_SERVICE).toBe(window.location.origin);
  });
});

describe('getAuthHeaders', () => {
  it('sends JSON without an Authorization header when signed out', async () => {
    const { getAuthHeaders } = await loadApiModule();

    expect(getAuthHeaders()).toEqual({ 'Content-Type': 'application/json' });
  });

  it('attaches the stored token as a bearer credential', async () => {
    localStorage.setItem('token', 'jwt-123');

    const { getAuthHeaders } = await loadApiModule();

    expect(getAuthHeaders()).toEqual({
      'Content-Type': 'application/json',
      Authorization: 'Bearer jwt-123',
    });
  });
});
