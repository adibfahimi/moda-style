import { beforeEach, describe, expect, it } from 'vitest';
import { authService } from './authService';
import {
  fetchCall,
  installFetchMock,
  jsonBody,
  jsonResponse,
  unparseableResponse,
  type FetchMock,
} from '../test/support';

let fetchMock: FetchMock;

beforeEach(() => {
  fetchMock = installFetchMock();
});

describe('authService.register', () => {
  it('posts the payload and remembers the returned token', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ token: 'jwt-123', user: { id: 1 } }));

    const result = await authService.register({
      name: 'Ada',
      email: 'ada@moda.test',
      password: 'secret123',
    });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe('http://localhost:8001/api/v1/auth/register');
    expect(init.method).toBe('POST');
    expect(jsonBody(init)).toEqual({ name: 'Ada', email: 'ada@moda.test', password: 'secret123' });
    expect(localStorage.getItem('token')).toBe('jwt-123');
    expect(result).toEqual({ token: 'jwt-123', user: { id: 1 } });
  });

  it('surfaces the backend error message', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'Email already registered' }, { status: 409 }),
    );

    await expect(
      authService.register({ name: 'Ada', email: 'ada@moda.test', password: 'secret123' }),
    ).rejects.toThrow('Email already registered');
  });

  it('falls back to a generic message when the body carries none', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));

    await expect(
      authService.register({ name: 'Ada', email: 'ada@moda.test', password: 'secret123' }),
    ).rejects.toThrow('Registration failed');
  });

  it('does not store a token when the backend omits one', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ user: { id: 1 } }));

    await authService.register({ name: 'Ada', email: 'ada@moda.test', password: 'secret123' });

    expect(localStorage.getItem('token')).toBeNull();
  });
});

describe('authService.login', () => {
  it('posts credentials and stores the token', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ token: 'jwt-456', user: { id: 2, role: 'admin' } }));

    const result = await authService.login({ email: 'admin@moda.test', password: 'secret123' });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe('http://localhost:8001/api/v1/auth/login');
    expect(init.method).toBe('POST');
    expect(jsonBody(init)).toEqual({ email: 'admin@moda.test', password: 'secret123' });
    expect(localStorage.getItem('token')).toBe('jwt-456');
    expect(result.user).toEqual({ id: 2, role: 'admin' });
  });

  it('falls back to a generic message when rejected', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { status: 401 }));

    await expect(authService.login({ email: 'ada@moda.test', password: 'nope' })).rejects.toThrow(
      'Login failed',
    );
  });
});

describe('authService.getProfile', () => {
  it('unwraps the user from the response envelope', async () => {
    localStorage.setItem('token', 'jwt-123');
    fetchMock.mockResolvedValue(jsonResponse({ user: { id: 3, name: 'Ada' } }));

    const user = await authService.getProfile();

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe('http://localhost:8001/api/v1/auth/profile');
    expect(init.method).toBeUndefined();
    expect(init.headers).toMatchObject({ Authorization: 'Bearer jwt-123' });
    expect(user).toEqual({ id: 3, name: 'Ada' });
  });

  it('attaches the status code so callers can react to expired sessions', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse({ error: 'Invalid or expired token' }, { status: 401 }),
    );

    await expect(authService.getProfile()).rejects.toMatchObject({
      message: 'Invalid or expired token',
      status: 401,
    });
  });

  it('keeps a readable message when the error body is not JSON', async () => {
    fetchMock.mockResolvedValue(unparseableResponse(502));

    await expect(authService.getProfile()).rejects.toMatchObject({
      message: 'Failed to fetch profile',
      status: 502,
    });
  });
});

describe('authService.updateProfile', () => {
  it('patches the profile and unwraps the updated user', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ user: { id: 3, name: 'Ada Lovelace' } }));

    const user = await authService.updateProfile({ name: 'Ada Lovelace' });

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe('http://localhost:8001/api/v1/auth/profile');
    expect(init.method).toBe('PATCH');
    expect(jsonBody(init)).toEqual({ name: 'Ada Lovelace' });
    expect(user).toEqual({ id: 3, name: 'Ada Lovelace' });
  });

  it('throws when the update is refused', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'Email already in use' }, { status: 409 }));

    await expect(authService.updateProfile({ email: 'taken@moda.test' })).rejects.toThrow(
      'Email already in use',
    );
  });
});

describe('authService.resetPassword', () => {
  it('posts the email address', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ message: 'Email sent' }));

    await authService.resetPassword('ada@moda.test');

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe('http://localhost:8001/api/v1/auth/reset-password');
    expect(init.method).toBe('POST');
    expect(jsonBody(init)).toEqual({ email: 'ada@moda.test' });
  });

  it('throws when the request fails', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'Unknown email' }, { status: 404 }));

    await expect(authService.resetPassword('missing@moda.test')).rejects.toThrow('Unknown email');
  });
});

describe('session helpers', () => {
  it('reports the presence of a token', () => {
    expect(authService.isAuthenticated()).toBe(false);

    localStorage.setItem('token', 'jwt-123');
    expect(authService.isAuthenticated()).toBe(true);
  });

  it('drops the token on logout', () => {
    localStorage.setItem('token', 'jwt-123');

    authService.logout();

    expect(localStorage.getItem('token')).toBeNull();
    expect(authService.isAuthenticated()).toBe(false);
  });
});
