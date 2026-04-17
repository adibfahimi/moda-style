import { createSignal, createResource, Show } from 'solid-js';
import { getActivityLogs } from '../../services/adminService';
import type { ActivityLog } from '../../types';
import DataTable from '../../components/admin/DataTable';
import Pagination from '../../components/admin/Pagination';

export default function AdminActivityLogs() {
  const [page, setPage] = createSignal(1);
  const [limit] = createSignal(50);
  const [actionFilter, setActionFilter] = createSignal('');
  const [resourceFilter, setResourceFilter] = createSignal('');

  const [logs] = createResource(
    () => ({
      page: page(),
      limit: limit(),
      action: actionFilter(),
      resource: resourceFilter(),
    }),
    getActivityLogs
  );

  const columns = [
    { key: 'id', label: 'ID' },
    {
      key: 'admin_name',
      label: 'Admin',
      render: (log: ActivityLog) => (
        <div>
          <div class="font-medium">{log.admin_name}</div>
          <div class="text-xs text-base-content/60">ID: {log.admin_id}</div>
        </div>
      ),
    },
    {
      key: 'action',
      label: 'Action',
      render: (log: ActivityLog) => (
        <span
          class={`badge ${
            log.action === 'created'
              ? 'badge-success'
              : log.action === 'updated'
              ? 'badge-info'
              : 'badge-error'
          }`}
        >
          {log.action}
        </span>
      ),
    },
    {
      key: 'resource',
      label: 'Resource',
      render: (log: ActivityLog) => (
        <span class="badge badge-outline">
          {log.resource} #{log.resource_id}
        </span>
      ),
    },
    { key: 'description', label: 'Description' },
    { key: 'ip_address', label: 'IP Address' },
    {
      key: 'created_at',
      label: 'Timestamp',
      render: (log: ActivityLog) => new Date(log.created_at).toLocaleString(),
    },
  ];

  return (
    <div class="p-6">
      <h1 class="text-3xl font-bold mb-6">Activity Logs</h1>

      {/* Filters */}
      <div class="card bg-base-100 shadow-xl mb-6">
        <div class="card-body">
          <div class="flex flex-col sm:flex-row gap-4">
            <div class="form-control flex-1">
              <label class="label">
                <span class="label-text">Action</span>
              </label>
              <select
                class="select select-bordered"
                value={actionFilter()}
                onChange={(e) => {
                  setActionFilter(e.currentTarget.value);
                  setPage(1);
                }}
              >
                <option value="">All Actions</option>
                <option value="created">Created</option>
                <option value="updated">Updated</option>
                <option value="deleted">Deleted</option>
              </select>
            </div>
            <div class="form-control flex-1">
              <label class="label">
                <span class="label-text">Resource</span>
              </label>
              <select
                class="select select-bordered"
                value={resourceFilter()}
                onChange={(e) => {
                  setResourceFilter(e.currentTarget.value);
                  setPage(1);
                }}
              >
                <option value="">All Resources</option>
                <option value="product">Product</option>
                <option value="user">User</option>
                <option value="category">Category</option>
                <option value="size">Size</option>
              </select>
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">&nbsp;</span>
              </label>
              <button
                class="btn btn-outline"
                onClick={() => {
                  setActionFilter('');
                  setResourceFilter('');
                  setPage(1);
                }}
              >
                Clear Filters
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Logs Table */}
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <DataTable
            data={logs()?.activities || []}
            columns={columns}
            loading={logs.loading}
            emptyMessage="No activity logs found"
          />

          <Show when={logs() && logs()!.count > limit()}>
            <Pagination
              currentPage={page()}
              totalPages={Math.ceil(logs()!.count / limit())}
              totalItems={logs()!.count}
              itemsPerPage={limit()}
              onPageChange={setPage}
            />
          </Show>
        </div>
      </div>
    </div>
  );
}
