import { A, useParams } from "@solidjs/router";
import { For, Show, createResource } from "solid-js";
import Navbar from "../components/Navbar";
import { useAuth } from "../context/AuthContext";
import { orderService } from "../services/orderService";
import type { OrderItem } from "../types";

export default function OrderDetail() {
  const params = useParams();
  const auth = useAuth();

  const orderId = () => Number(params.id);

  const [order] = createResource(orderId, async (id) => {
    if (!Number.isFinite(id) || id <= 0) {
      throw new Error("Invalid order id");
    }

    const response = await orderService.getOrderById(id);
    return (response as any).order ? (response as any).order : response;
  });

  if (!auth.isAuthenticated) {
    return (
      <>
        <Navbar />
        <div class="min-h-screen bg-base-200 flex items-center justify-center">
          <div class="card bg-base-100 shadow-xl w-96">
            <div class="card-body items-center text-center">
              <h2 class="card-title">Login Required</h2>
              <p>Please login to view order details</p>
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
        <div class="container mx-auto max-w-5xl px-4 py-8">
          <div class="mb-6">
            <A href="/orders" class="btn btn-ghost btn-sm">
              Back to Orders
            </A>
          </div>

          <Show
            when={!order.loading}
            fallback={
              <div class="flex justify-center items-center py-20">
                <span class="loading loading-spinner loading-lg text-primary"></span>
              </div>
            }
          >
            <Show
              when={!order.error && order()}
              fallback={
                <div class="alert alert-error">
                  <span>Unable to load order details.</span>
                </div>
              }
            >
              <div class="card border border-base-300 bg-base-100">
                <div class="card-body">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h1 class="text-2xl font-bold">
                        Order #{order()!.order_number}
                      </h1>
                      <p class="text-sm opacity-70">
                        Placed on{" "}
                        {new Date(order()!.created_at).toLocaleDateString(
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
                      <span class="badge badge-outline">{order()!.status}</span>
                      <span class="badge badge-outline">
                        {order()!.payment_status}
                      </span>
                    </div>
                  </div>

                  <div class="divider"></div>

                  <div class="grid gap-4 md:grid-cols-2">
                    <div>
                      <p class="text-sm opacity-60">Shipping Address</p>
                      <p class="font-medium">{order()!.shipping_address}</p>
                    </div>
                    <div>
                      <p class="text-sm opacity-60">Payment Method</p>
                      <p class="font-medium capitalize">
                        {order()!.payment_method}
                      </p>
                    </div>
                  </div>

                  <Show when={order()!.notes}>
                    <div class="mt-4">
                      <p class="text-sm opacity-60">Notes</p>
                      <p>{order()!.notes}</p>
                    </div>
                  </Show>

                  <Show when={order()!.items && order()!.items!.length > 0}>
                    <div class="mt-6">
                      <h2 class="mb-3 text-lg font-semibold">Items</h2>
                      <div class="space-y-2">
                        <For each={order()!.items as OrderItem[]}>
                          {(item) => (
                            <div class="flex items-center justify-between rounded bg-base-200 p-3 text-sm">
                              <div>
                                <p class="font-medium">{item.product_name}</p>
                                <p class="opacity-70">Qty: {item.quantity}</p>
                              </div>
                              <p class="font-semibold">
                                ${item.subtotal.toFixed(2)}
                              </p>
                            </div>
                          )}
                        </For>
                      </div>
                    </div>
                  </Show>

                  <div class="mt-6 flex items-center justify-between border-t border-base-300 pt-4">
                    <span class="text-lg font-bold">Total</span>
                    <span class="text-xl font-bold text-primary">
                      ${order()!.total_amount.toFixed(2)}
                    </span>
                  </div>
                </div>
              </div>
            </Show>
          </Show>
        </div>
      </div>
    </>
  );
}
