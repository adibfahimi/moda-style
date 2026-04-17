import { Show, For, createSignal, onMount } from "solid-js";
import { A } from "@solidjs/router";
import { useAuth } from "../context/AuthContext";
import Navbar from "../components/Navbar";
import { orderService } from "../services/orderService";
import type { Order } from "../types";

export default function Orders() {
  const auth = useAuth();
  const [orders, setOrders] = createSignal<Order[]>([]);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal("");
  const [cancellingOrderId, setCancellingOrderId] = createSignal<number | null>(
    null,
  );

  onMount(async () => {
    if (!auth.isAuthenticated) return;

    try {
      const data = await orderService.getMyOrders();
      setOrders(data);
    } catch (err: any) {
      setError(err.message || "Failed to load orders");
    } finally {
      setLoading(false);
    }
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case "delivered":
        return "badge-success";
      case "shipped":
        return "badge-info";
      case "processing":
        return "badge-warning";
      case "cancelled":
        return "badge-error";
      default:
        return "badge-ghost";
    }
  };

  const getPaymentStatusColor = (status: string) => {
    switch (status) {
      case "paid":
        return "badge-success";
      case "failed":
        return "badge-error";
      case "refunded":
        return "badge-warning";
      default:
        return "badge-ghost";
    }
  };

  const handleCancelOrder = async (orderId: number) => {
    setError("");
    setCancellingOrderId(orderId);

    try {
      await orderService.cancelOrder(orderId);
      setOrders((currentOrders) =>
        currentOrders.map((order) =>
          order.id === orderId
            ? { ...order, status: "cancelled", payment_status: "failed" }
            : order,
        ),
      );
    } catch (err: any) {
      setError(err.message || "Failed to cancel order");
    } finally {
      setCancellingOrderId(null);
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
              <p>Please login to view your orders</p>
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
        <div class="container mx-auto px-4 py-8 max-w-6xl">
          <h1 class="text-3xl font-bold mb-8">My Orders</h1>

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

          <Show
            when={loading()}
            fallback={
              <Show
                when={orders().length > 0}
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
                        d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                      />
                    </svg>
                    <h2 class="text-3xl font-bold mt-6 mb-2">No orders yet</h2>
                    <p class="text-base-content/60 mb-6">
                      You haven't placed any orders yet
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
                <div class="space-y-4">
                  <For each={orders()}>
                    {(order) => (
                      <div class="card bg-base-100 border border-base-300">
                        <div class="card-body">
                          <div class="flex flex-wrap gap-4 justify-between items-start mb-4">
                            <div>
                              <h2 class="card-title mb-1">
                                Order #{order.order_number}
                              </h2>
                              <p class="text-sm opacity-70">
                                Placed on{" "}
                                {new Date(order.created_at).toLocaleDateString(
                                  "en-US",
                                  {
                                    year: "numeric",
                                    month: "long",
                                    day: "numeric",
                                  },
                                )}
                              </p>
                            </div>
                            <div class="flex gap-2">
                              <span
                                class={`badge ${getStatusColor(order.status)}`}
                              >
                                {order.status.charAt(0).toUpperCase() +
                                  order.status.slice(1)}
                              </span>
                              <span
                                class={`badge ${getPaymentStatusColor(order.payment_status)}`}
                              >
                                {order.payment_status.charAt(0).toUpperCase() +
                                  order.payment_status.slice(1)}
                              </span>
                            </div>
                          </div>

                          <div class="divider my-2"></div>

                          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                            <div>
                              <p class="text-sm opacity-70 mb-1">
                                Shipping Address
                              </p>
                              <p class="text-sm">{order.shipping_address}</p>
                            </div>
                            <div>
                              <p class="text-sm opacity-70 mb-1">
                                Payment Method
                              </p>
                              <p class="text-sm capitalize">
                                {order.payment_method}
                              </p>
                            </div>
                          </div>

                          <Show when={order.notes}>
                            <div class="mb-4">
                              <p class="text-sm opacity-70 mb-1">Notes</p>
                              <p class="text-sm">{order.notes}</p>
                            </div>
                          </Show>

                          <Show when={order.items && order.items.length > 0}>
                            <div class="mb-4">
                              <p class="text-sm opacity-70 mb-2">Order Items</p>
                              <div class="space-y-2">
                                <For each={order.items}>
                                  {(item) => (
                                    <div class="flex justify-between items-center text-sm bg-base-200 p-3 rounded">
                                      <div>
                                        <span class="font-medium">
                                          {item.product_name}
                                        </span>
                                        <span class="opacity-70 ml-2">
                                          x{item.quantity}
                                        </span>
                                      </div>
                                      <span class="font-semibold">
                                        ${item.subtotal.toFixed(2)}
                                      </span>
                                    </div>
                                  )}
                                </For>
                              </div>
                            </div>
                          </Show>

                          <div class="flex justify-between items-center pt-4 border-t border-base-300">
                            <span class="text-lg font-bold">Total</span>
                            <span class="text-xl font-bold text-primary">
                              ${order.total_amount.toFixed(2)}
                            </span>
                          </div>

                          <Show
                            when={
                              order.status === "pending" &&
                              order.payment_status === "pending"
                            }
                          >
                            <div class="card-actions justify-end mt-4">
                              <button
                                class="btn btn-error btn-sm"
                                onClick={() => handleCancelOrder(order.id)}
                                disabled={cancellingOrderId() === order.id}
                              >
                                <Show
                                  when={cancellingOrderId() !== order.id}
                                  fallback={
                                    <span class="loading loading-spinner loading-xs"></span>
                                  }
                                >
                                  Cancel Order
                                </Show>
                              </button>
                              <A
                                href={`/orders/${order.id}`}
                                class="btn btn-primary btn-sm"
                              >
                                View Details
                              </A>
                            </div>
                          </Show>
                        </div>
                      </div>
                    )}
                  </For>
                </div>
              </Show>
            }
          >
            <div class="flex justify-center items-center py-20">
              <span class="loading loading-spinner loading-lg text-primary"></span>
            </div>
          </Show>
        </div>
      </div>
    </>
  );
}
