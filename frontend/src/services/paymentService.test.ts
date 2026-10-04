import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  fetchCall,
  installFetchMock,
  jsonBody,
  jsonResponse,
} from '../test/support';

const validCard = {
  number: '4242424242424242',
  expMonth: 12,
  expYear: new Date().getFullYear() + 1,
  cvc: '123',
};

/** The card number Stripe documents as an always-declined test card. */
const declinedCard = { ...validCard, number: '4000000000000002' };

/**
 * Loads the module with the payment gateway either configured or not.
 *
 * `VITE_PAYMENT_SERVICE_URL` is read once at import time, so each case stubs the
 * variable and imports a fresh copy of the module.
 *
 * @param url - Gateway base URL; omit or pass an empty string for mock mode.
 */
const loadPaymentService = async (url = '') => {
  vi.resetModules();
  vi.stubEnv('VITE_PAYMENT_SERVICE_URL', url);
  return import('./paymentService');
};

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllEnvs();
});

describe('paymentService in mock mode', () => {
  it('reports mock mode when no gateway is configured', async () => {
    const { paymentService } = await loadPaymentService();

    expect(paymentService.isMockMode()).toBe(true);
    expect(paymentService.formatCardNumber(validCard.number)).toBe('•••• •••• •••• 4242');
  });

  it('creates a pending intent for the amount in cents', async () => {
    vi.useFakeTimers();
    const { paymentService } = await loadPaymentService();

    const pending = paymentService.createPaymentIntent(42.5);
    await vi.advanceTimersByTimeAsync(500);
    const intent = await pending;

    expect(intent.amount).toBe(4250);
    expect(intent.currency).toBe('usd');
    expect(intent.status).toBe('requires_payment_method');
    expect(intent.id).toMatch(/^pi_fake_/);
    expect(intent.client_secret).toMatch(/^pi_fake_secret_/);
  });

  it('rounds the amount up to the nearest cent', async () => {
    vi.useFakeTimers();
    const { paymentService } = await loadPaymentService();

    const pending = paymentService.createPaymentIntent(19.999);
    await vi.advanceTimersByTimeAsync(500);
    const intent = await pending;

    expect(intent.amount).toBe(2000);
  });

  it('accepts a well-formed card and derives the intent id from the secret', async () => {
    vi.useFakeTimers();
    const { paymentService } = await loadPaymentService();
    const secret = 'pi_fake_99_secret_abc';

    const confirming = paymentService.processPayment(secret, validCard);
    await vi.advanceTimersByTimeAsync(1500);
    const result = await confirming;

    expect(result).toEqual({ success: true, paymentIntentId: 'pi_fake_99' });
  });

  it('declines the always-failing test card', async () => {
    vi.useFakeTimers();
    const { paymentService } = await loadPaymentService();

    const confirming = paymentService.processPayment('pi_fake_7_secret_xyz', declinedCard);
    await vi.advanceTimersByTimeAsync(1500);
    const result = await confirming;

    expect(result.success).toBe(false);
    expect(result.error).toMatch(/declined/i);
  });

  it('rejects malformed card details', async () => {
    vi.useFakeTimers();
    const { paymentService } = await loadPaymentService();
    const malformed = [
      { ...validCard, number: '4242' },
      { ...validCard, expMonth: 13 },
      { ...validCard, expYear: new Date().getFullYear() - 1 },
      { ...validCard, cvc: '1' },
      { ...validCard, cvc: '12345' },
    ];

    for (const card of malformed) {
      const confirming = paymentService.processPayment('pi_fake_8_secret_xyz', card);
      await vi.advanceTimersByTimeAsync(1500);

      await expect(confirming).resolves.toMatchObject({ success: false });
    }
  });
});

describe('paymentService.validateCardNumber', () => {
  it('accepts numbers that satisfy the Luhn checksum', async () => {
    const { paymentService } = await loadPaymentService();

    expect(paymentService.validateCardNumber('4242 4242 4242 4242')).toBe(true);
    expect(paymentService.validateCardNumber('5555555555554444')).toBe(true);
  });

  it('rejects a mistyped number', async () => {
    const { paymentService } = await loadPaymentService();

    expect(paymentService.validateCardNumber('4242424242424241')).toBe(false);
  });

  it('rejects numbers that are too short or too long', async () => {
    const { paymentService } = await loadPaymentService();

    expect(paymentService.validateCardNumber('4242')).toBe(false);
    expect(paymentService.validateCardNumber('4'.repeat(20))).toBe(false);
  });
});

describe('paymentService against the gateway', () => {
  const GATEWAY = 'https://pay.moda-style.test';

  it('leaves mock mode once a gateway URL is configured', async () => {
    const { paymentService } = await loadPaymentService(GATEWAY);

    expect(paymentService.isMockMode()).toBe(false);
  });

  it('asks the gateway for a payment intent', async () => {
    const fetchMock = installFetchMock();
    fetchMock.mockResolvedValue(
      jsonResponse({ payment_intent: { id: 'pi_remote', amount: 2000, status: 'succeeded' } }),
    );
    const { paymentService } = await loadPaymentService(GATEWAY);

    const intent = await paymentService.createPaymentIntent(20);

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(`${GATEWAY}/api/v1/payments/intents`);
    expect(init.method).toBe('POST');
    expect(jsonBody(init)).toEqual({ amount: 2000, currency: 'usd' });
    expect(intent.id).toBe('pi_remote');
  });

  it('also understands a camelCase envelope and a bare payload', async () => {
    const fetchMock = installFetchMock();
    const { paymentService } = await loadPaymentService(GATEWAY);

    fetchMock.mockResolvedValue(jsonResponse({ paymentIntent: { id: 'pi_camel' } }));
    await expect(paymentService.createPaymentIntent(20)).resolves.toMatchObject({ id: 'pi_camel' });

    fetchMock.mockResolvedValue(jsonResponse({ id: 'pi_bare' }));
    await expect(paymentService.createPaymentIntent(20)).resolves.toMatchObject({ id: 'pi_bare' });
  });

  it('surfaces a failure to create an intent', async () => {
    const fetchMock = installFetchMock();
    const { paymentService } = await loadPaymentService(GATEWAY);

    fetchMock.mockResolvedValue(jsonResponse({ error: 'Amount too small' }, { status: 400 }));
    await expect(paymentService.createPaymentIntent(0.5)).rejects.toThrow('Amount too small');

    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));
    await expect(paymentService.createPaymentIntent(20)).rejects.toThrow(
      'Failed to create payment intent',
    );
  });

  it('confirms the payment through the gateway', async () => {
    const fetchMock = installFetchMock();
    fetchMock.mockResolvedValue(
      jsonResponse({ result: { success: true, paymentIntentId: 'pi_remote' } }),
    );
    const { paymentService } = await loadPaymentService(GATEWAY);

    const result = await paymentService.processPayment('pi_remote_secret_abc', validCard);

    const { url, init } = fetchCall(fetchMock);
    expect(url).toBe(`${GATEWAY}/api/v1/payments/confirm`);
    expect(init.method).toBe('POST');
    expect(jsonBody(init)).toEqual({ client_secret: 'pi_remote_secret_abc', card: validCard });
    expect(result).toEqual({ success: true, paymentIntentId: 'pi_remote' });
  });

  it('also understands a bare confirmation payload', async () => {
    const fetchMock = installFetchMock();
    fetchMock.mockResolvedValue(jsonResponse({ success: true, paymentIntentId: 'pi_bare' }));
    const { paymentService } = await loadPaymentService(GATEWAY);

    await expect(
      paymentService.processPayment('pi_bare_secret_abc', validCard),
    ).resolves.toMatchObject({ paymentIntentId: 'pi_bare' });
  });

  it('surfaces a failure to confirm a payment', async () => {
    const fetchMock = installFetchMock();
    const { paymentService } = await loadPaymentService(GATEWAY);

    fetchMock.mockResolvedValue(jsonResponse({ error: 'Card declined' }, { status: 402 }));
    await expect(
      paymentService.processPayment('pi_remote_secret_abc', validCard),
    ).rejects.toThrow('Card declined');

    fetchMock.mockResolvedValue(jsonResponse({}, { status: 500 }));
    await expect(
      paymentService.processPayment('pi_remote_secret_abc', validCard),
    ).rejects.toThrow('Failed to process payment');
  });
});
