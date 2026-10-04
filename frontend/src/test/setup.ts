import { afterEach } from 'vitest';
import { cleanup } from '@solidjs/testing-library';

/**
 * Solid keeps rendered trees mounted for the lifetime of the process, so every
 * test tears its tree down explicitly. The auth token also lives in
 * `localStorage`, which jsdom shares between tests in a file, so it is cleared
 * as well to keep tests order-independent.
 */
afterEach(() => {
  cleanup();
  localStorage.clear();
});
