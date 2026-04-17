import { createSignal, createResource, Show, For } from 'solid-js';
import { getOrders, getOrderAnalytics, getOrderDetails, updateOrderStatus, deleteOrder } from '../../services/adminService';
import type { OrderStats, OrderAnalytics, OrderItem } from '../../types';
import DataTable from '../../components/admin/DataTable';
import Pagination from '../../components/admin/Pagination';
import Modal, { useModal } from '../../components/admin/Modal';
import StatsCard from '../../components/admin/StatsCard';

export default function AdminOrders() {
  const [page, setPage] = createSignal(1);
  const [limit] = createSignal(20);
  const [statusFilter, setStatusFilter] = createSignal('');
  const [paymentStatusFilter, setPaymentStatusFilter] = createSignal('');
  const [searchQuery, setSearchQuery] = createSignal('');
  const [selectedOrder, setSelectedOrder] = createSignal<OrderStats | null>(null);
  const [orderItems, setOrderItems] = createSignal<OrderItem[]>([]);
  const [statusForm, setStatusForm] = createSignal({ status: '', payment_status: '', notes: '' });
  const [error, setError] = createSignal('');
  const [success, setSuccess] = createSignal('');

  const detailsModal = useModal();
  const statusModal = useModal();
  const deleteModal = useModal();

  const [orders, { refetch: refetchOrders }] = createResource(
    () => ({ page: page(), limit: limit(), status: statusFilter(), payment_status: paymentStatusFilter(), search: searchQuery() }),
    async (params) => await getOrders(params)
  );

  const [analytics] = createResource<{ analytics: OrderAnalytics }>(getOrderAnalytics);

  const handleViewDetails = async (order: OrderStats) => {
    setSelectedOrder(order);
    try {
      const response = await getOrderDetails(order.id);
      setOrderItems(response.items || []);
      detailsModal.open();
    } catch (err: any) {
      setError(err.message || 'Failed to load order details');
    }
  };

  const handleUpdateStatus = (order: OrderStats) => {
    setSelectedOrder(order);
    setStatusForm({ 
      status: order.status, 
      payment_status: order.payment_status,
      notes: order.notes || ''
    });
    statusModal.open();
  };

  const handleDelete = (order: OrderStats) => {
    setSelectedOrder(order);
    deleteModal.open();
  };

  const handleSearch = (e: Event) => {
    const input = e.target as HTMLInputElement;
    setSearchQuery(input.value);
    setPage(1);
  };

  const handleStatusFilter = (e: Event) => {
    const select = e.target as HTMLSelectElement;
    setStatusFilter(select.value);
    setPage(1);
  };

  const handlePaymentStatusFilter = (e: Event) => {
    const select = e.target as HTMLSelectElement;
    setPaymentStatusFilter(select.value);
    setPage(1);
  };

  const submitStatusUpdate = async (e: Event) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      await updateOrderStatus(selectedOrder()!.id, statusForm());
      setSuccess('Order status updated successfully');
      setTimeout(() => {
        statusModal.close();
        refetchOrders();
        setSuccess('');
      }, 1500);
    } catch (err: any) {
      setError(err.message || 'Failed to update order');
    }
  };

  const confirmDelete = async () => {
    setError('');
    setSuccess('');

    try {
      await deleteOrder(selectedOrder()!.id);
      setSuccess('Order deleted successfully');
      setTimeout(() => {
        deleteModal.close();
        refetchOrders();
        setSuccess('');
      }, 1500);
    } catch (err: any) {
      setError(err.message || 'Failed to delete order');
    }
  };

  const getStatusBadge = (status: string) => {
    const badges: Record<string, string> = {
      pending: 'badge-warning',
      processing: 'badge-info',
      shipped: 'badge-primary',
      delivered: 'badge-success',
      cancelled: 'badge-error',
    };
    return badges[status] || 'badge-ghost';
  };

  const getPaymentStatusBadge = (status: string) => {
    const badges: Record<string, string> = {
      pending: 'badge-warning',
      paid: 'badge-success',
      failed: 'badge-error',
      refunded: 'badge-info',
    };
    return badges[status] || 'badge-ghost';
  };

  const columns = [
    { key: 'order_number', label: 'Order #' },
    { key: 'user_name', label: 'Customer' },
    { key: 'user_email', label: 'Email' },
    {
      key: 'status',
      label: 'Status',
      render: (order: OrderStats) => (
        <span class={`badge ${getStatusBadge(order.status)}`}>
          {order.status}
        </span>
      ),
    },
    {
      key: 'payment_status',
      label: 'Payment',
      render: (order: OrderStats) => (
        <span class={`badge ${getPaymentStatusBadge(order.payment_status)}`}>
          {order.payment_status}
        </span>
      ),
    },
    {
      key: 'total_amount',
      label: 'Total',
      render: (order: OrderStats) => `$${order.total_amount.toFixed(2)}`,
    },
    {
      key: 'item_count',
      label: 'Items',
      render: (order: OrderStats) => order.item_count,
    },
    {
      key: 'created_at',
      label: 'Date',
      render: (order: OrderStats) => new Date(order.created_at).toLocaleDateString(),
    },
    {
      key: 'actions',
      label: 'Actions',
      render: (order: OrderStats) => (
        <div class="flex gap-2">
          <button class="btn btn-sm btn-info" onClick={() => handleViewDetails(order)}>
            View
          </button>
          <button class="btn btn-sm btn-primary" onClick={() => handleUpdateStatus(order)}>
            Update
          </button>
          <button class="btn btn-sm btn-error" onClick={() => handleDelete(order)}>
            Delete
          </button>
        </div>
      ),
    },
  ];

  return (
    <div class="p-6">
      <h1 class="text-3xl font-bold mb-6">Order Management</h1>

      {/* Analytics Cards */}
      <Show when={!analytics.loading && analytics()}>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          <StatsCard
            title="Total Orders"
            value={analytics()?.analytics.total_orders || 0}
            icon="📦"
            color="primary"
          />
          <StatsCard
            title="Total Revenue"
            value={`$${(analytics()?.analytics.total_revenue || 0).toFixed(2)}`}
            icon="💰"
            color="success"
          />
          <StatsCard
            title="Pending Orders"
            value={analytics()?.analytics.pending_orders || 0}
            icon="⏳"
            color="warning"
          />
          <StatsCard
            title="Delivered Orders"
            value={analytics()?.analytics.delivered_orders || 0}
            icon="✅"
            color="info"
          />
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <StatsCard
            title="Orders Today"
            value={analytics()?.analytics.orders_today || 0}
            icon="📅"
            color="accent"
          />
          <StatsCard
            title="Orders This Week"
            value={analytics()?.analytics.orders_this_week || 0}
            icon="📊"
            color="secondary"
          />
          <StatsCard
            title="Orders This Month"
            value={analytics()?.analytics.orders_this_month || 0}
            icon="📈"
            color="primary"
          />
        </div>
      </Show>

      {/* Filters */}
      <div class="card bg-base-100 shadow-xl mb-6">
        <div class="card-body">
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <div class="form-control flex-1">
              <input
                type="text"
                placeholder="Search by order #, customer name or email..."
                class="input input-bordered w-full"
                onInput={handleSearch}
                value={searchQuery()}
              />
            </div>
            <div class="form-control">
              <select class="select select-bordered" onChange={handleStatusFilter} value={statusFilter()}>
                <option value="">All Statuses</option>
                <option value="pending">Pending</option>
                <option value="processing">Processing</option>
                <option value="shipped">Shipped</option>
                <option value="delivered">Delivered</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </div>
            <div class="form-control">
              <select class="select select-bordered" onChange={handlePaymentStatusFilter} value={paymentStatusFilter()}>
                <option value="">All Payment Statuses</option>
                <option value="pending">Pending</option>
                <option value="paid">Paid</option>
                <option value="failed">Failed</option>
                <option value="refunded">Refunded</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {/* Orders Table */}
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <DataTable
            data={orders()?.orders || []}
            columns={columns}
            loading={orders.loading}
            emptyMessage="No orders found"
          />

          <Show when={orders() && orders()!.total > limit()}>
            <Pagination
              currentPage={page()}
              totalPages={Math.ceil(orders()!.total / limit())}
              totalItems={orders()!.total}
              itemsPerPage={limit()}
              onPageChange={setPage}
            />
          </Show>
        </div>
      </div>

      {/* Order Details Modal */}
      <Modal title="Order Details" isOpen={detailsModal.isOpen()} onClose={detailsModal.close} size="lg">
        <Show when={selectedOrder()}>
          <div class="space-y-4">
            {/* Order Info */}
            <div class="grid grid-cols-2 gap-4">
              <div>
                <p class="text-sm text-base-content/60">Order Number</p>
                <p class="font-semibold">{selectedOrder()?.order_number}</p>
              </div>
              <div>
                <p class="text-sm text-base-content/60">Customer</p>
                <p class="font-semibold">{selectedOrder()?.user_name}</p>
              </div>
              <div>
                <p class="text-sm text-base-content/60">Email</p>
                <p class="font-semibold">{selectedOrder()?.user_email}</p>
              </div>
              <div>
                <p class="text-sm text-base-content/60">Status</p>
                <span class={`badge ${getStatusBadge(selectedOrder()!.status)}`}>
                  {selectedOrder()?.status}
                </span>
              </div>
              <div>
                <p class="text-sm text-base-content/60">Payment Status</p>
                <span class={`badge ${getPaymentStatusBadge(selectedOrder()!.payment_status)}`}>
                  {selectedOrder()?.payment_status}
                </span>
              </div>
              <div>
                <p class="text-sm text-base-content/60">Payment Method</p>
                <p class="font-semibold">{selectedOrder()?.payment_method || 'N/A'}</p>
              </div>
              <div class="col-span-2">
                <p class="text-sm text-base-content/60">Shipping Address</p>
                <p class="font-semibold">{selectedOrder()?.shipping_address || 'N/A'}</p>
              </div>
              <Show when={selectedOrder()?.notes}>
                <div class="col-span-2">
                  <p class="text-sm text-base-content/60">Notes</p>
                  <p class="font-semibold">{selectedOrder()?.notes}</p>
                </div>
              </Show>
            </div>

            <div class="divider"></div>

            {/* Order Items */}
            <div>
              <h3 class="font-bold text-lg mb-3">Order Items</h3>
              <div class="overflow-x-auto">
                <table class="table table-zebra">
                  <thead>
                    <tr>
                      <th>Product</th>
                      <th>Quantity</th>
                      <th>Price</th>
                      <th>Subtotal</th>
                    </tr>
                  </thead>
                  <tbody>
                    <For each={orderItems()}>
                      {(item) => (
                        <tr>
                          <td>{item.product_name}</td>
                          <td>{item.quantity}</td>
                          <td>${item.price.toFixed(2)}</td>
                          <td>${item.subtotal.toFixed(2)}</td>
                        </tr>
                      )}
                    </For>
                  </tbody>
                </table>
              </div>
            </div>

            <div class="divider"></div>

            {/* Total */}
            <div class="flex justify-end">
              <div class="text-right">
                <p class="text-sm text-base-content/60">Total Amount</p>
                <p class="text-2xl font-bold text-primary">${selectedOrder()?.total_amount.toFixed(2)}</p>
              </div>
            </div>
          </div>
        </Show>
      </Modal>

      {/* Update Status Modal */}
      <Modal title="Update Order Status" isOpen={statusModal.isOpen()} onClose={statusModal.close}>
        <form onSubmit={submitStatusUpdate}>
          <Show when={error()}>
            <div class="alert alert-error shadow-lg mb-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{error()}</span>
            </div>
          </Show>
          <Show when={success()}>
            <div class="alert alert-success shadow-lg mb-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{success()}</span>
            </div>
          </Show>

          <div class="space-y-4">
            <div class="form-control w-full">
              <label class="label">
                <span class="label-text font-semibold">Order Status</span>
              </label>
              <select
                class="select select-bordered w-full"
                value={statusForm().status}
                onChange={(e) => setStatusForm({ ...statusForm(), status: e.currentTarget.value })}
              >
                <option value="pending">Pending</option>
                <option value="processing">Processing</option>
                <option value="shipped">Shipped</option>
                <option value="delivered">Delivered</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </div>

            <div class="form-control w-full">
              <label class="label">
                <span class="label-text font-semibold">Payment Status</span>
              </label>
              <select
                class="select select-bordered w-full"
                value={statusForm().payment_status}
                onChange={(e) => setStatusForm({ ...statusForm(), payment_status: e.currentTarget.value })}
              >
                <option value="pending">Pending</option>
                <option value="paid">Paid</option>
                <option value="failed">Failed</option>
                <option value="refunded">Refunded</option>
              </select>
            </div>

            <div class="form-control w-full">
              <label class="label">
                <span class="label-text font-semibold">Admin Notes</span>
              </label>
              <textarea
                placeholder="Add notes about this order..."
                class="textarea textarea-bordered h-24 resize-none"
                value={statusForm().notes}
                onInput={(e) => setStatusForm({ ...statusForm(), notes: e.currentTarget.value })}
              />
            </div>
          </div>

          <div class="divider"></div>
          <div class="flex justify-end gap-3">
            <button type="button" class="btn btn-ghost" onClick={statusModal.close}>
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              Update Order
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete Modal */}
      <Modal title="Delete Order" isOpen={deleteModal.isOpen()} onClose={deleteModal.close} size="sm">
        <Show when={error()}>
          <div class="alert alert-error shadow-lg mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{error()}</span>
          </div>
        </Show>
        <Show when={success()}>
          <div class="alert alert-success shadow-lg mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{success()}</span>
          </div>
        </Show>

        <div class="py-4">
          <div class="flex items-center gap-3 mb-4">
            <div class="text-error">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <div>
              <h4 class="font-bold text-lg">Confirm Deletion</h4>
              <p class="text-sm opacity-70">This action cannot be undone</p>
            </div>
          </div>

          <p class="mb-4">
            Are you sure you want to delete order <strong class="text-error">{selectedOrder()?.order_number}</strong>?
          </p>
        </div>

        <div class="divider"></div>
        <div class="flex justify-end gap-3">
          <button class="btn btn-ghost" onClick={deleteModal.close}>
            Cancel
          </button>
          <button class="btn btn-error" onClick={confirmDelete}>
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
            Delete Order
          </button>
        </div>
      </Modal>
    </div>
  );
}
