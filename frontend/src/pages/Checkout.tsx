import { Show, createSignal, onMount } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { useCart } from "../context/CartContext";
import { useAuth } from "../context/AuthContext";
import Navbar from "../components/Navbar";
import { orderService } from "../services/orderService";
import { paymentService, type CardDetails } from "../services/paymentService";

export default function Checkout() {
  const cart = useCart();
  const auth = useAuth();
  const navigate = useNavigate();

  const [step, setStep] = createSignal<
    "shipping" | "payment" | "processing" | "success"
  >("shipping");
  const [processing, setProcessing] = createSignal(false);

  // Shipping form
  const [shippingAddress, setShippingAddress] = createSignal("");
  const [city, setCity] = createSignal("");
  const [state, setState] = createSignal("");
  const [zipCode, setZipCode] = createSignal("");
  const [country, setCountry] = createSignal("United States");
  const [notes, setNotes] = createSignal("");

  // Payment form
  const [cardNumber, setCardNumber] = createSignal("");
  const [cardName, setCardName] = createSignal("");
  const [expMonth, setExpMonth] = createSignal("");
  const [expYear, setExpYear] = createSignal("");
  const [cvc, setCvc] = createSignal("");

  // Order state
  const [orderNumber, setOrderNumber] = createSignal("");
  const [error, setError] = createSignal("");

  // Calculate totals
  const subtotal = () => cart.subtotal;
  const shipping = () => (subtotal() > 100 ? 0 : 9.99); // Free shipping over $100
  const tax = () => subtotal() * 0.08; // 8% tax
  const total = () => subtotal() + shipping() + tax();

  onMount(() => {
    // Redirect if cart is empty
    if (cart.items.length === 0) {
      navigate("/cart");
    }
  });

  const handleShippingSubmit = async (e: Event) => {
    e.preventDefault();
    setError("");

    if (!shippingAddress() || !city() || !state() || !zipCode()) {
      setError("Please fill in all required fields");
      return;
    }

    setStep("payment");
  };

  const handlePaymentSubmit = async (e: Event) => {
    e.preventDefault();
    setError("");
    setProcessing(true);
    let createdOrderId: number | null = null;

    try {
      // Validate card
      const cleanCardNumber = cardNumber().replace(/\s/g, "");
      if (!paymentService.validateCardNumber(cleanCardNumber)) {
        throw new Error("Invalid card number");
      }

      if (!expMonth() || !expYear() || !cvc()) {
        throw new Error("Please fill in all card details");
      }

      setStep("processing");

      // Step 1: Create order
      const fullAddress = `${shippingAddress()}, ${city()}, ${state()} ${zipCode()}, ${country()}`;
      const orderResponse = await orderService.createOrder({
        shipping_address: fullAddress,
        payment_method: "card",
        notes: notes(),
      });
      createdOrderId = orderResponse.order.id;

      setOrderNumber(orderResponse.order.order_number);

      // Step 2: Create payment intent
      const paymentIntent = await paymentService.createPaymentIntent(total());

      // Step 3: Process payment
      const cardDetails: CardDetails = {
        number: cleanCardNumber,
        expMonth: parseInt(expMonth()),
        expYear: parseInt(expYear()),
        cvc: cvc(),
      };

      const paymentResult = await paymentService.processPayment(
        paymentIntent.client_secret,
        cardDetails,
      );

      if (!paymentResult.success) {
        throw new Error(paymentResult.error || "Payment failed");
      }

      // Step 4: Confirm payment with backend
      await orderService.processPayment(orderResponse.order.id, {
        payment_intent_id: paymentResult.paymentIntentId,
      });

      // Step 5: Clear cart
      await cart.clearCart();

      // Success!
      setStep("success");
    } catch (err: any) {
      if (createdOrderId) {
        try {
          await orderService.cancelOrder(createdOrderId);
        } catch {
          // Ignore rollback failures and show original checkout error.
        }
      }
      console.error("Checkout error:", err);
      setError(err.message || "An error occurred during checkout");
      setStep("payment");
    } finally {
      setProcessing(false);
    }
  };

  const formatCardNumberInput = (value: string) => {
    const cleaned = value.replace(/\s/g, "");
    const chunks = cleaned.match(/.{1,4}/g) || [];
    return chunks.join(" ");
  };

  if (!auth.isAuthenticated) {
    return (
      <>
        <Navbar />
        <div class="min-h-screen bg-base-200 flex items-center justify-center">
          <div class="card bg-base-100 shadow-xl w-96">
            <div class="card-body items-center text-center">
              <h2 class="card-title">Login Required</h2>
              <p>Please login to checkout</p>
              <div class="card-actions">
                <button
                  onclick={() => navigate("/login")}
                  class="btn btn-primary"
                >
                  Login
                </button>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <Navbar />
      <div class="min-h-screen bg-base-100">
        <div class="container mx-auto px-4 py-8 max-w-6xl">
          {/* Header */}
          <div class="mb-8">
            <h1 class="text-3xl font-bold mb-2">Checkout</h1>
            <ul class="steps steps-horizontal w-full">
              <li class={`step ${step() !== "shipping" ? "step-primary" : ""}`}>
                Shipping
              </li>
              <li
                class={`step ${step() === "processing" || step() === "success" ? "step-primary" : ""}`}
              >
                Payment
              </li>
              <li class={`step ${step() === "success" ? "step-primary" : ""}`}>
                Complete
              </li>
            </ul>
          </div>

          {/* Error Alert */}
          <Show when={error()}>
            <div class="alert alert-error mb-6">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="stroke-current shrink-0 h-6 w-6"
                fill="none"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <span>{error()}</span>
            </div>
          </Show>

          <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
            {/* Main Content */}
            <div class="lg:col-span-2">
              {/* Shipping Form */}
              <Show when={step() === "shipping"}>
                <div class="card bg-base-100 border border-base-300">
                  <div class="card-body">
                    <h2 class="card-title mb-4">Shipping Information</h2>
                    <form onSubmit={handleShippingSubmit} class="space-y-4">
                      <div class="form-control">
                        <label class="label">
                          <span class="label-text">Street Address *</span>
                        </label>
                        <input
                          type="text"
                          placeholder="123 Main St"
                          class="input input-bordered w-full"
                          value={shippingAddress()}
                          onInput={(e) =>
                            setShippingAddress(e.currentTarget.value)
                          }
                          required
                        />
                      </div>

                      <div class="grid grid-cols-2 gap-4">
                        <div class="form-control">
                          <label class="label">
                            <span class="label-text">City *</span>
                          </label>
                          <input
                            type="text"
                            placeholder="New York"
                            class="input input-bordered w-full"
                            value={city()}
                            onInput={(e) => setCity(e.currentTarget.value)}
                            required
                          />
                        </div>

                        <div class="form-control">
                          <label class="label">
                            <span class="label-text">State *</span>
                          </label>
                          <input
                            type="text"
                            placeholder="NY"
                            class="input input-bordered w-full"
                            value={state()}
                            onInput={(e) => setState(e.currentTarget.value)}
                            required
                          />
                        </div>
                      </div>

                      <div class="grid grid-cols-2 gap-4">
                        <div class="form-control">
                          <label class="label">
                            <span class="label-text">ZIP Code *</span>
                          </label>
                          <input
                            type="text"
                            placeholder="10001"
                            class="input input-bordered w-full"
                            value={zipCode()}
                            onInput={(e) => setZipCode(e.currentTarget.value)}
                            required
                          />
                        </div>

                        <div class="form-control">
                          <label class="label">
                            <span class="label-text">Country *</span>
                          </label>
                          <select
                            class="select select-bordered w-full"
                            value={country()}
                            onChange={(e) => setCountry(e.currentTarget.value)}
                          >
                            <option>United States</option>
                            <option>Canada</option>
                            <option>United Kingdom</option>
                            <option>Australia</option>
                          </select>
                        </div>
                      </div>

                      <div class="form-control">
                        <label class="label">
                          <span class="label-text">Order Notes (Optional)</span>
                        </label>
                        <textarea
                          class="textarea textarea-bordered h-24"
                          placeholder="Any special instructions for your order..."
                          value={notes()}
                          onInput={(e) => setNotes(e.currentTarget.value)}
                        ></textarea>
                      </div>

                      <div class="card-actions justify-end mt-6">
                        <button
                          type="button"
                          onclick={() => navigate("/cart")}
                          class="btn btn-ghost"
                        >
                          Back to Cart
                        </button>
                        <button type="submit" class="btn btn-primary">
                          Continue to Payment
                        </button>
                      </div>
                    </form>
                  </div>
                </div>
              </Show>

              {/* Payment Form */}
              <Show when={step() === "payment"}>
                <div class="card bg-base-100 border border-base-300">
                  <div class="card-body">
                    <h2 class="card-title mb-4">Payment Information</h2>

                    <Show when={paymentService.isMockMode()}>
                      <div class="alert alert-info mb-4">
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          fill="none"
                          viewBox="0 0 24 24"
                          class="stroke-current shrink-0 w-6 h-6"
                        >
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                          ></path>
                        </svg>
                        <div class="text-sm">
                          <p class="font-semibold">
                            Test Mode - Use these cards:
                          </p>
                          <p>Success: 4242 4242 4242 4242</p>
                          <p>Decline: 4000 0000 0000 0002</p>
                        </div>
                      </div>
                    </Show>

                    <form onSubmit={handlePaymentSubmit} class="space-y-4">
                      <div class="form-control">
                        <label class="label">
                          <span class="label-text">Cardholder Name *</span>
                        </label>
                        <input
                          type="text"
                          placeholder="John Doe"
                          class="input input-bordered w-full"
                          value={cardName()}
                          onInput={(e) => setCardName(e.currentTarget.value)}
                          required
                        />
                      </div>

                      <div class="form-control">
                        <label class="label">
                          <span class="label-text">Card Number *</span>
                        </label>
                        <input
                          type="text"
                          placeholder="1234 5678 9012 3456"
                          class="input input-bordered w-full font-mono"
                          value={cardNumber()}
                          onInput={(e) => {
                            const formatted = formatCardNumberInput(
                              e.currentTarget.value,
                            );
                            if (formatted.replace(/\s/g, "").length <= 19) {
                              setCardNumber(formatted);
                            }
                          }}
                          maxLength={19}
                          required
                        />
                      </div>

                      <div class="grid grid-cols-3 gap-4">
                        <div class="form-control">
                          <label class="label">
                            <span class="label-text">Exp Month *</span>
                          </label>
                          <select
                            class="select select-bordered w-full"
                            value={expMonth()}
                            onChange={(e) => setExpMonth(e.currentTarget.value)}
                            required
                          >
                            <option value="">MM</option>
                            {Array.from({ length: 12 }, (_, i) => i + 1).map(
                              (month) => (
                                <option value={month}>
                                  {month.toString().padStart(2, "0")}
                                </option>
                              ),
                            )}
                          </select>
                        </div>

                        <div class="form-control">
                          <label class="label">
                            <span class="label-text">Exp Year *</span>
                          </label>
                          <select
                            class="select select-bordered w-full"
                            value={expYear()}
                            onChange={(e) => setExpYear(e.currentTarget.value)}
                            required
                          >
                            <option value="">YYYY</option>
                            {Array.from(
                              { length: 10 },
                              (_, i) => new Date().getFullYear() + i,
                            ).map((year) => (
                              <option value={year}>{year}</option>
                            ))}
                          </select>
                        </div>

                        <div class="form-control">
                          <label class="label">
                            <span class="label-text">CVC *</span>
                          </label>
                          <input
                            type="text"
                            placeholder="123"
                            class="input input-bordered w-full font-mono"
                            value={cvc()}
                            onInput={(e) => {
                              const value = e.currentTarget.value.replace(
                                /\D/g,
                                "",
                              );
                              if (value.length <= 4) {
                                setCvc(value);
                              }
                            }}
                            maxLength={4}
                            required
                          />
                        </div>
                      </div>

                      <div class="card-actions justify-end mt-6">
                        <button
                          type="button"
                          onclick={() => setStep("shipping")}
                          class="btn btn-ghost"
                        >
                          Back
                        </button>
                        <button
                          type="submit"
                          class="btn btn-primary"
                          disabled={processing()}
                        >
                          <Show
                            when={!processing()}
                            fallback={
                              <span class="loading loading-spinner"></span>
                            }
                          >
                            Pay ${total().toFixed(2)}
                          </Show>
                        </button>
                      </div>
                    </form>
                  </div>
                </div>
              </Show>

              {/* Processing State */}
              <Show when={step() === "processing"}>
                <div class="card bg-base-100 border border-base-300">
                  <div class="card-body items-center text-center py-12">
                    <span class="loading loading-spinner loading-lg text-primary"></span>
                    <h2 class="text-2xl font-bold mt-6">
                      Processing Payment...
                    </h2>
                    <p class="opacity-70">Please don't close this page</p>
                  </div>
                </div>
              </Show>

              {/* Success State */}
              <Show when={step() === "success"}>
                <div class="card bg-base-100 border border-success">
                  <div class="card-body items-center text-center py-12">
                    <div class="w-20 h-20 rounded-full bg-success/20 flex items-center justify-center mb-6">
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-12 w-12 text-success"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M5 13l4 4L19 7"
                        />
                      </svg>
                    </div>
                    <h2 class="text-3xl font-bold mb-2">Order Confirmed!</h2>
                    <p class="text-xl mb-2">Order #{orderNumber()}</p>
                    <p class="opacity-70 mb-8">
                      Thank you for your purchase. Your order has been confirmed
                      and will be shipped soon.
                    </p>
                    <div class="card-actions">
                      <button
                        onclick={() => navigate("/orders")}
                        class="btn btn-primary"
                      >
                        View Orders
                      </button>
                      <button
                        onclick={() => navigate("/products")}
                        class="btn btn-ghost"
                      >
                        Continue Shopping
                      </button>
                    </div>
                  </div>
                </div>
              </Show>
            </div>

            {/* Order Summary Sidebar */}
            <div class="lg:col-span-1">
              <div class="card bg-base-100 border border-base-300 sticky top-4">
                <div class="card-body">
                  <h2 class="card-title mb-4">Order Summary</h2>

                  <div class="space-y-2 mb-4">
                    <div class="flex justify-between text-sm">
                      <span class="opacity-80">
                        Subtotal ({cart.count} items)
                      </span>
                      <span>${subtotal().toFixed(2)}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                      <span class="opacity-80">Shipping</span>
                      <span>
                        <Show
                          when={shipping() === 0}
                          fallback={`$${shipping().toFixed(2)}`}
                        >
                          <span class="text-success">FREE</span>
                        </Show>
                      </span>
                    </div>
                    <div class="flex justify-between text-sm">
                      <span class="opacity-80">Tax</span>
                      <span>${tax().toFixed(2)}</span>
                    </div>
                  </div>

                  <div class="divider my-2"></div>

                  <div class="flex justify-between text-lg font-bold mb-4">
                    <span>Total</span>
                    <span class="text-primary">${total().toFixed(2)}</span>
                  </div>

                  <Show when={shipping() > 0}>
                    <div class="alert alert-info text-sm">
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
                        class="stroke-current shrink-0 w-5 h-5"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                        ></path>
                      </svg>
                      <span>
                        Add ${(100 - subtotal()).toFixed(2)} more for free
                        shipping!
                      </span>
                    </div>
                  </Show>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
