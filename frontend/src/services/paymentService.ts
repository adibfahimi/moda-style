export interface PaymentIntent {
  id: string;
  amount: number;
  currency: string;
  status: 'requires_payment_method' | 'requires_confirmation' | 'succeeded' | 'failed';
  client_secret: string;
}

export interface CardDetails {
  number: string;
  expMonth: number;
  expYear: number;
  cvc: string;
}

export interface PaymentResult {
  success: boolean;
  paymentIntentId: string;
  error?: string;
}

type RemotePaymentIntentResponse = {
  payment_intent?: PaymentIntent;
  paymentIntent?: PaymentIntent;
} & PaymentIntent;

type RemotePaymentResultResponse = {
  result?: PaymentResult;
} & PaymentResult;

const PAYMENT_SERVICE_URL = import.meta.env.VITE_PAYMENT_SERVICE_URL as
  | string
  | undefined;

class LocalPaymentGateway {
  async createPaymentIntent(amount: number, currency = 'usd'): Promise<PaymentIntent> {
    await this.delay(500);

    const paymentIntent: PaymentIntent = {
      id: `pi_fake_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      amount: Math.round(amount * 100),
      currency,
      status: 'requires_payment_method',
      client_secret: `pi_fake_secret_${Math.random().toString(36).substr(2, 16)}`,
    };

    return paymentIntent;
  }

  async confirmCardPayment(
    clientSecret: string,
    cardDetails: CardDetails
  ): Promise<PaymentResult> {
    await this.delay(1500);

    const isValidCard = this.validateCard(cardDetails);

    if (!isValidCard) {
      return {
        success: false,
        paymentIntentId: clientSecret.split('_secret_')[0],
        error: 'Your card was declined. Please try a different payment method.',
      };
    }

    return {
      success: true,
      paymentIntentId: clientSecret.split('_secret_')[0],
    };
  }

  private validateCard(cardDetails: CardDetails): boolean {
    const { number, expMonth, expYear, cvc } = cardDetails;

    if (number === '4000000000000002') {
      return false;
    }

    if (!number || number.length < 13 || number.length > 19) {
      return false;
    }

    if (expMonth < 1 || expMonth > 12) {
      return false;
    }

    const currentYear = new Date().getFullYear();
    if (expYear < currentYear) {
      return false;
    }

    if (!cvc || cvc.length < 3 || cvc.length > 4) {
      return false;
    }

    return true;
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

const localGateway = new LocalPaymentGateway();

const createRemotePaymentIntent = async (amount: number): Promise<PaymentIntent> => {
  const response = await fetch(`${PAYMENT_SERVICE_URL}/api/v1/payments/intents`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ amount: Math.round(amount * 100), currency: 'usd' }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to create payment intent');
  }

  const result: RemotePaymentIntentResponse = await response.json();
  return result.payment_intent || result.paymentIntent || (result as PaymentIntent);
};

const confirmRemotePayment = async (
  clientSecret: string,
  cardDetails: CardDetails,
): Promise<PaymentResult> => {
  const response = await fetch(`${PAYMENT_SERVICE_URL}/api/v1/payments/confirm`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ client_secret: clientSecret, card: cardDetails }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to process payment');
  }

  const result: RemotePaymentResultResponse = await response.json();
  return result.result || (result as PaymentResult);
};

export const paymentService = {
  isMockMode(): boolean {
    return !PAYMENT_SERVICE_URL;
  },

  async createPaymentIntent(amount: number): Promise<PaymentIntent> {
    if (!PAYMENT_SERVICE_URL) {
      return localGateway.createPaymentIntent(amount);
    }

    return createRemotePaymentIntent(amount);
  },

  async processPayment(
    clientSecret: string,
    cardDetails: CardDetails
  ): Promise<PaymentResult> {
    if (!PAYMENT_SERVICE_URL) {
      return localGateway.confirmCardPayment(clientSecret, cardDetails);
    }

    return confirmRemotePayment(clientSecret, cardDetails);
  },

  formatCardNumber(cardNumber: string): string {
    const last4 = cardNumber.slice(-4);
    return `•••• •••• •••• ${last4}`;
  },

  validateCardNumber(cardNumber: string): boolean {
    const digits = cardNumber.replace(/\D/g, '');
    
    if (digits.length < 13 || digits.length > 19) {
      return false;
    }

    let sum = 0;
    let isEven = false;

    for (let i = digits.length - 1; i >= 0; i--) {
      let digit = parseInt(digits[i]);

      if (isEven) {
        digit *= 2;
        if (digit > 9) {
          digit -= 9;
        }
      }

      sum += digit;
      isEven = !isEven;
    }

    return sum % 10 === 0;
  },
};
