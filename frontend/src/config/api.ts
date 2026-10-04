/**
 * API configuration and the shared request helpers used by every service module.
 *
 * Service URLs are resolved once, when this module is first evaluated: an
 * explicit `VITE_<SERVICE>_SERVICE_URL` wins, otherwise requests target
 * `http://localhost:<port>` while `import.meta.env.DEV` is true, and the
 * current origin (the Nginx gateway in production) everywhere else.
 */

/**
 * Origin used in production when no explicit service URL is configured.
 *
 * @returns `window.location.origin`, or an empty prefix (same-origin) when
 *   there is no `window`, as in the Vitest environment.
 */
const getDefaultProductionOrigin = () => {
  if (typeof window !== 'undefined') {
    return window.location.origin;
  }

  return '';
};

/**
 * Resolves the base URL of a backend service.
 *
 * @param envUrl - Value of the matching `VITE_*_SERVICE_URL` variable.
 * @param devPort - Port the service is exposed on by `docker compose` locally.
 */
const resolveServiceUrl = (envUrl: string | undefined, devPort: number) => {
  if (envUrl) {
    return envUrl;
  }

  if (import.meta.env.DEV) {
    return `http://localhost:${devPort}`;
  }

  // In production, route API calls through the same origin (Nginx gateway).
  return getDefaultProductionOrigin();
};

/** Base URLs of every backend service, keyed by service name. */
export const API_CONFIG = {
  AUTH_SERVICE: resolveServiceUrl(import.meta.env.VITE_AUTH_SERVICE_URL, 8001),
  PRODUCT_SERVICE: resolveServiceUrl(import.meta.env.VITE_PRODUCT_SERVICE_URL, 8002),
  CART_SERVICE: resolveServiceUrl(import.meta.env.VITE_CART_SERVICE_URL, 8003),
  ADMIN_SERVICE: resolveServiceUrl(import.meta.env.VITE_ADMIN_SERVICE_URL, 8004),
  ORDER_SERVICE: resolveServiceUrl(import.meta.env.VITE_ORDER_SERVICE_URL, 8005),
};

/**
 * Builds the headers for an authenticated JSON request.
 *
 * The bearer token is read from `localStorage` on every call, so a login or
 * logout is picked up by the next request without any extra bookkeeping.
 *
 * @returns `Content-Type: application/json` plus an `Authorization` header when
 *   a token is present.
 */
export const getAuthHeaders = () => {
  const token = localStorage.getItem('token');
  return {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
  };
};
