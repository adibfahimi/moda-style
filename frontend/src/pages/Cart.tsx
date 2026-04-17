import { Show, For, createSignal } from "solid-js";
import { A, useNavigate } from "@solidjs/router";
import { useCart } from "../context/CartContext";
import { useAuth } from "../context/AuthContext";
import Navbar from "../components/Navbar";

export default function Cart() {
  const cart = useCart();
  const auth = useAuth();
  const navigate = useNavigate();
  const [updating, setUpdating] = createSignal<number | null>(null);
  const [infoMessage, setInfoMessage] = createSignal<string | null>(null);
  const [confirmAction, setConfirmAction] = createSignal<
    "remove" | "clear" | null
  >(null);
  const [pendingItemId, setPendingItemId] = createSignal<number | null>(null);
  const [confirming, setConfirming] = createSignal(false);

  const handleUpdateQuantity = async (id: number, newQuantity: number) => {
    if (newQuantity < 1) return;
    setUpdating(id);
    try {
      await cart.updateQuantity(id, newQuantity);
    } catch (error: any) {
      setInfoMessage(error.message || "Failed to update quantity");
    } finally {
      setUpdating(null);
    }
  };

  const removeCartItem = async (id: number) => {
    setUpdating(id);
    try {
      await cart.removeItem(id);
    } catch (error: any) {
      setInfoMessage(error.message || "Failed to remove item");
    } finally {
      setUpdating(null);
    }
  };

  const clearAllCartItems = async () => {
    try {
      await cart.clearCart();
    } catch (error: any) {
      setInfoMessage(error.message || "Failed to clear cart");
    }
  };

  const handleRemove = async (id: number) => {
    setPendingItemId(id);
    setConfirmAction("remove");
  };

  const handleClearCart = async () => {
    setPendingItemId(null);
    setConfirmAction("clear");
  };

  const handleConfirm = async () => {
    const action = confirmAction();
    if (!action) return;

    setConfirming(true);
    try {
      if (action === "remove" && pendingItemId() !== null) {
        await removeCartItem(pendingItemId()!);
      }

      if (action === "clear") {
        await clearAllCartItems();
      }
    } finally {
      setConfirming(false);
      setConfirmAction(null);
      setPendingItemId(null);
    }
  };

  if (!auth.isAuthenticated) {
    return (
      <>
        <Navbar />
        <div class="min-h-screen bg-base-200 flex items-center justify-center">
          <div class="card bg-base-100 shadow-xl w-96">
            <div class="card-body items-center text-center">
              <h2 class="card-title">Login Required</h2>
              <p>Please login to view your cart</p>
              <div class="card-actions">
                <A href="/login" class="btn btn-primary">
                  Login
                </A>
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
        <div class="container mx-auto px-4 py-8 max-w-7xl">
          <h1 class="text-3xl font-bold mb-8">Shopping Cart</h1>

          <Show
            when={!cart.isLoading}
            fallback={
              <div class="flex justify-center items-center py-20">
                <span class="loading loading-spinner loading-lg text-primary"></span>
              </div>
            }
          >
            <Show
              when={cart.items.length > 0}
              fallback={
                <div class="text-center py-20">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-32 w-32 mx-auto text-base-300"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="1.5"
                      d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z"
                    />
                  </svg>
                  <h2 class="text-3xl font-bold mt-6 mb-2">
                    Your cart is empty
                  </h2>
                  <p class="text-base-content/60 mb-6">
                    Looks like you haven't added anything to your cart yet
                  </p>
                  <A href="/products" class="btn btn-primary btn-lg">
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      class="h-5 w-5 mr-2"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z"
                      />
                    </svg>
                    Start Shopping
                  </A>
                </div>
              }
            >
              <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* Cart Items */}
                <div class="lg:col-span-2">
                  <div class="bg-base-100 rounded-lg border border-base-300 p-6">
                    <div class="flex justify-between items-center mb-6 pb-4 border-b border-base-300">
                      <h2 class="text-xl font-bold">
                        Cart ({cart.count} {cart.count === 1 ? "item" : "items"}
                        )
                      </h2>
                      <button
                        class="btn btn-ghost btn-sm text-error hover:bg-error/10"
                        onclick={handleClearCart}
                      >
                        Clear all
                      </button>
                    </div>

                    <div class="space-y-4">
                      <For each={cart.items}>
                        {(item) => (
                          <div class="flex gap-4 pb-4 border-b border-base-200 last:border-0">
                            <A
                              href={`/products/${item.product_id}`}
                              class="shrink-0"
                            >
                              <img
                                src={item.image_url || "/placeholder.png"}
                                alt={item.name}
                                class="w-28 h-28 object-cover rounded-lg border border-base-300"
                              />
                            </A>

                            <div class="flex-1 min-w-0">
                              <div class="flex justify-between gap-4">
                                <div class="flex-1">
                                  <A
                                    href={`/products/${item.product_id}`}
                                    class="font-semibold text-lg hover:text-primary line-clamp-2"
                                  >
                                    {item.name}
                                  </A>
                                  <div class="text-sm opacity-70 mt-1 space-y-1">
                                    <p>
                                      <span class="font-medium">Size:</span>{" "}
                                      {item.size} |
                                      <span class="font-medium ml-1">
                                        Color:
                                      </span>{" "}
                                      {item.color}
                                    </p>
                                    <Show
                                      when={item.stock > 0}
                                      fallback={
                                        <p class="text-error font-medium">
                                          Out of stock
                                        </p>
                                      }
                                    >
                                      <Show when={item.stock < 5}>
                                        <p class="text-warning font-medium">
                                          Only {item.stock} left in stock
                                        </p>
                                      </Show>
                                    </Show>
                                  </div>
                                </div>

                                <div class="text-right">
                                  <p class="text-xl font-bold text-primary">
                                    ${(item.price * item.quantity).toFixed(2)}
                                  </p>
                                  <Show when={item.quantity > 1}>
                                    <p class="text-sm opacity-60">
                                      ${item.price.toFixed(2)} each
                                    </p>
                                  </Show>
                                </div>
                              </div>

                              <div class="flex items-center gap-4 mt-4">
                                <div class="join border border-base-300">
                                  <button
                                    class="btn btn-sm join-item bg-base-100 hover:bg-base-200 border-0"
                                    onclick={() =>
                                      handleUpdateQuantity(
                                        item.id,
                                        item.quantity - 1,
                                      )
                                    }
                                    disabled={
                                      item.quantity <= 1 ||
                                      updating() === item.id
                                    }
                                  >
                                    <svg
                                      xmlns="http://www.w3.org/2000/svg"
                                      class="h-4 w-4"
                                      fill="none"
                                      viewBox="0 0 24 24"
                                      stroke="currentColor"
                                    >
                                      <path
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="2"
                                        d="M20 12H4"
                                      />
                                    </svg>
                                  </button>
                                  <div class="btn btn-sm join-item no-animation bg-base-100 border-0 font-semibold min-w-12.5">
                                    {item.quantity}
                                  </div>
                                  <button
                                    class="btn btn-sm join-item bg-base-100 hover:bg-base-200 border-0"
                                    onclick={() =>
                                      handleUpdateQuantity(
                                        item.id,
                                        item.quantity + 1,
                                      )
                                    }
                                    disabled={
                                      item.quantity >= item.stock ||
                                      updating() === item.id
                                    }
                                  >
                                    <svg
                                      xmlns="http://www.w3.org/2000/svg"
                                      class="h-4 w-4"
                                      fill="none"
                                      viewBox="0 0 24 24"
                                      stroke="currentColor"
                                    >
                                      <path
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="2"
                                        d="M12 4v16m8-8H4"
                                      />
                                    </svg>
                                  </button>
                                </div>

                                <button
                                  class="btn btn-ghost btn-sm text-error hover:bg-error/10 gap-2"
                                  onclick={() => handleRemove(item.id)}
                                  disabled={updating() === item.id}
                                >
                                  <svg
                                    xmlns="http://www.w3.org/2000/svg"
                                    class="h-4 w-4"
                                    fill="none"
                                    viewBox="0 0 24 24"
                                    stroke="currentColor"
                                  >
                                    <path
                                      stroke-linecap="round"
                                      stroke-linejoin="round"
                                      stroke-width="2"
                                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                                    />
                                  </svg>
                                  Remove
                                </button>

                                <Show when={updating() === item.id}>
                                  <span class="loading loading-spinner loading-sm"></span>
                                </Show>
                              </div>
                            </div>
                          </div>
                        )}
                      </For>
                    </div>
                  </div>
                </div>

                {/* Order Summary */}
                <div class="lg:col-span-1">
                  <div class="bg-base-100 rounded-lg border border-base-300 p-6 sticky top-4">
                    <h2 class="text-xl font-bold mb-6">Order Summary</h2>

                    <div class="space-y-3 mb-4">
                      <div class="flex justify-between text-base">
                        <span class="opacity-80">
                          Subtotal ({cart.count} items)
                        </span>
                        <span class="font-semibold">
                          ${cart.subtotal.toFixed(2)}
                        </span>
                      </div>
                      <div class="flex justify-between text-base">
                        <span class="opacity-80">Shipping</span>
                        <span class="text-sm opacity-70">
                          Calculated at checkout
                        </span>
                      </div>
                      <div class="flex justify-between text-base">
                        <span class="opacity-80">Tax</span>
                        <span class="text-sm opacity-70">
                          Calculated at checkout
                        </span>
                      </div>
                    </div>

                    <div class="divider my-4"></div>

                    <div class="flex justify-between text-xl font-bold mb-6">
                      <span>Total</span>
                      <span class="text-primary">
                        ${cart.subtotal.toFixed(2)}
                      </span>
                    </div>

                    <button
                      class="btn btn-primary btn-lg btn-block mb-3"
                      onclick={() => navigate("/checkout")}
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-5 w-5 mr-2"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                        />
                      </svg>
                      Proceed to Checkout
                    </button>

                    <A href="/products" class="btn btn-ghost btn-block">
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        class="h-5 w-5 mr-2"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M10 19l-7-7m0 0l7-7m-7 7h18"
                        />
                      </svg>
                      Continue Shopping
                    </A>

                    <div class="mt-6 p-4 bg-base-200 rounded-lg">
                      <div class="flex gap-3">
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          class="h-5 w-5 text-success shrink-0 mt-0.5"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                        >
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                          />
                        </svg>
                        <div class="text-sm">
                          <p class="font-semibold">Secure Checkout</p>
                          <p class="opacity-70 text-xs mt-1">
                            Your payment information is protected
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </Show>
          </Show>
        </div>
      </div>

      <Show when={confirmAction()}>
        <dialog class="modal modal-open">
          <div class="modal-box">
            <h3 class="text-lg font-bold">Please Confirm</h3>
            <p class="py-3">
              {confirmAction() === "remove"
                ? "Remove this item from your cart?"
                : "Clear all items from your cart?"}
            </p>
            <div class="modal-action">
              <button
                class="btn btn-ghost"
                onClick={() => {
                  setConfirmAction(null);
                  setPendingItemId(null);
                }}
                disabled={confirming()}
              >
                Cancel
              </button>
              <button
                class="btn btn-error"
                onClick={handleConfirm}
                disabled={confirming()}
              >
                <Show
                  when={!confirming()}
                  fallback={
                    <span class="loading loading-spinner loading-xs"></span>
                  }
                >
                  Confirm
                </Show>
              </button>
            </div>
          </div>
        </dialog>
      </Show>

      <Show when={infoMessage()}>
        <dialog class="modal modal-open">
          <div class="modal-box">
            <h3 class="text-lg font-bold">Something went wrong</h3>
            <p class="py-3">{infoMessage()}</p>
            <div class="modal-action">
              <button
                class="btn btn-primary"
                onClick={() => setInfoMessage(null)}
              >
                OK
              </button>
            </div>
          </div>
        </dialog>
      </Show>
    </>
  );
}
