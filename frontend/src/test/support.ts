import { vi } from 'vitest';

/** A Vitest mock standing in for the global `fetch`. */
export type FetchMock = ReturnType<typeof vi.fn<typeof fetch>>;

/**
 * Builds a stand-in for a `Response`.
 *
 * The service layer only ever reads `ok`, `status` and `json()`, so faking that
 * subset keeps the tests free of a fetch implementation (and of real I/O).
 *
 * @param body - Value `json()` resolves with.
 * @param init - Override the derived `ok` flag and status code (they must agree:
 *   a 2xx status is `ok`, anything else is not).
 * @returns A `Response`-shaped object accepted by the services under test.
 */
export const jsonResponse = (
  body: unknown,
  init: { ok?: boolean; status?: number } = {},
): Response => {
  const status = init.status ?? 200;
  return {
    ok: init.ok ?? status < 400,
    status,
    json: async () => body,
  } as unknown as Response;
};

/**
 * Builds a failed response whose body cannot be parsed as JSON.
 *
 * `authService.getProfile` inspects the status code, so the status has to be
 * real here rather than inferred.
 */
export const unparseableResponse = (status = 500): Response =>
  ({
    ok: status < 400,
    status,
    json: async () => {
      throw new SyntaxError('Unexpected token < in JSON at position 0');
    },
  }) as unknown as Response;

/**
 * Installs a `fetch` mock on `globalThis` for the current test.
 *
 * `unstubGlobals` in `vite.config.ts` removes it again after each test.
 */
export const installFetchMock = (): FetchMock => {
  const mock = vi.fn<typeof fetch>();
  vi.stubGlobal('fetch', mock);
  return mock;
};

/**
 * Returns the URL and init object of a recorded `fetch` call.
 *
 * @param mock - Mock returned by {@link installFetchMock}.
 * @param call - Zero-based index of the call to inspect.
 */
export const fetchCall = (mock: FetchMock, call = 0): { url: string; init: RequestInit } => {
  const [url, init] = mock.mock.calls[call] ?? [];
  return { url: String(url ?? ''), init: (init as RequestInit) ?? {} };
};

/** Reads a JSON request body back off a recorded `fetch` call. */
export const jsonBody = (init: RequestInit): unknown =>
  JSON.parse(String(init.body));
