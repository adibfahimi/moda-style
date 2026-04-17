const getDefaultProductionOrigin = () => {
  if (typeof window !== 'undefined') {
    return window.location.origin;
  }

  return '';
};

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

export const API_CONFIG = {
  AUTH_SERVICE: resolveServiceUrl(import.meta.env.VITE_AUTH_SERVICE_URL, 8001),
  PRODUCT_SERVICE: resolveServiceUrl(import.meta.env.VITE_PRODUCT_SERVICE_URL, 8002),
  CART_SERVICE: resolveServiceUrl(import.meta.env.VITE_CART_SERVICE_URL, 8003),
  ADMIN_SERVICE: resolveServiceUrl(import.meta.env.VITE_ADMIN_SERVICE_URL, 8004),
  ORDER_SERVICE: resolveServiceUrl(import.meta.env.VITE_ORDER_SERVICE_URL, 8005),
};

export const getAuthHeaders = () => {
  const token = localStorage.getItem('token');
  return {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
  };
};
