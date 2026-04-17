import { createSignal, createResource, Show } from 'solid-js';
import { getUsers, getUserAnalytics, updateUser, banUser, unbanUser } from '../../services/adminService';
import type { UserStats, UserAnalytics } from '../../types';
import DataTable from '../../components/admin/DataTable';
import Pagination from '../../components/admin/Pagination';
import Modal, { useModal } from '../../components/admin/Modal';
import StatsCard from '../../components/admin/StatsCard';

export default function AdminUsers() {
  const [page, setPage] = createSignal(1);
  const [limit] = createSignal(20);
  const [roleFilter, setRoleFilter] = createSignal('');
  const [searchQuery, setSearchQuery] = createSignal('');
  const [selectedUser, setSelectedUser] = createSignal<UserStats | null>(null);
  const [editForm, setEditForm] = createSignal({ name: '', email: '', role: '' });
  const [error, setError] = createSignal('');
  const [success, setSuccess] = createSignal('');

  const editModal = useModal();
  const banModal = useModal();

  const [users, { refetch: refetchUsers }] = createResource(
    () => ({ page: page(), limit: limit(), role: roleFilter(), search: searchQuery() }),
    async (params) => await getUsers(params)
  );

  const [analytics] = createResource<{ analytics: UserAnalytics }>(getUserAnalytics);

  const handleEdit = (user: UserStats) => {
    setSelectedUser(user);
    setEditForm({ name: user.name, email: user.email, role: user.role });
    editModal.open();
  };

  const handleBanToggle = (user: UserStats) => {
    setSelectedUser(user);
    banModal.open();
  };

  const handleSearch = (e: Event) => {
    const input = e.target as HTMLInputElement;
    setSearchQuery(input.value);
    setPage(1);
  };

  const handleRoleFilter = (e: Event) => {
    const select = e.target as HTMLSelectElement;
    setRoleFilter(select.value);
    setPage(1);
  };

  const submitEdit = async (e: Event) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      await updateUser(selectedUser()!.id, editForm());
      setSuccess('User updated successfully');
      setTimeout(() => {
        editModal.close();
        refetchUsers();
        setSuccess('');
      }, 1500);
    } catch (err: any) {
      setError(err.message || 'Failed to update user');
    }
  };

  const confirmBanToggle = async () => {
    setError('');
    setSuccess('');

    try {
      const user = selectedUser()!;
      if (user.banned) {
        await unbanUser(user.id);
        setSuccess('User unbanned successfully');
      } else {
        await banUser(user.id);
        setSuccess('User banned successfully');
      }
      setTimeout(() => {
        banModal.close();
        refetchUsers();
        setSuccess('');
      }, 1500);
    } catch (err: any) {
      setError(err.message || 'Failed to update user ban status');
    }
  };

  const columns = [
    { key: 'id', label: 'ID' },
    { key: 'name', label: 'Name' },
    { key: 'email', label: 'Email' },
    {
      key: 'role',
      label: 'Role',
      render: (user: UserStats) => (
        <span class={`badge ${user.role === 'admin' ? 'badge-primary' : 'badge-ghost'}`}>
          {user.role}
        </span>
      ),
    },
    {
      key: 'banned',
      label: 'Status',
      render: (user: UserStats) => (
        <span class={`badge ${user.banned ? 'badge-error' : 'badge-success'}`}>
          {user.banned ? 'Banned' : 'Active'}
        </span>
      ),
    },
    { key: 'review_count', label: 'Reviews' },
    { key: 'wishlist_count', label: 'Wishlist' },
    {
      key: 'created_at',
      label: 'Joined',
      render: (user: UserStats) => new Date(user.created_at).toLocaleDateString(),
    },
    {
      key: 'actions',
      label: 'Actions',
      render: (user: UserStats) => (
        <div class="flex gap-2">
          <button class="btn btn-sm btn-primary" onClick={() => handleEdit(user)}>
            Edit
          </button>
          <button 
            class={`btn btn-sm ${user.banned ? 'btn-success' : 'btn-warning'}`}
            onClick={() => handleBanToggle(user)}
          >
            {user.banned ? 'Unban' : 'Ban'}
          </button>
        </div>
      ),
    },
  ];

  return (
    <div class="p-6">
      <h1 class="text-3xl font-bold mb-6">User Management</h1>

      {/* Analytics Cards */}
      <Show when={!analytics.loading && analytics()}>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
          <StatsCard
            title="Total Users"
            value={analytics()?.analytics.total_users || 0}
            icon="👥"
            color="primary"
          />
          <StatsCard
            title="Admin Users"
            value={analytics()?.analytics.admin_users || 0}
            icon="👑"
            color="secondary"
          />
          <StatsCard
            title="Regular Users"
            value={analytics()?.analytics.regular_users || 0}
            icon="👤"
            color="accent"
          />
          <StatsCard
            title="New Today"
            value={analytics()?.analytics.new_users_today || 0}
            icon="🆕"
            color="success"
          />
          <StatsCard
            title="New This Week"
            value={analytics()?.analytics.new_users_this_week || 0}
            icon="📈"
            color="info"
          />
          <StatsCard
            title="Active Reviewers"
            value={analytics()?.analytics.active_reviewers || 0}
            icon="⭐"
            color="warning"
          />
        </div>
      </Show>

      {/* Filters */}
      <div class="card bg-base-100 shadow-xl mb-6">
        <div class="card-body">
          <div class="flex flex-col sm:flex-row gap-4">
            <div class="form-control flex-1">
              <input
                type="text"
                placeholder="Search by name or email..."
                class="input input-bordered w-full"
                onInput={handleSearch}
                value={searchQuery()}
              />
            </div>
            <div class="form-control w-full sm:w-48">
              <select class="select select-bordered" onChange={handleRoleFilter} value={roleFilter()}>
                <option value="">All Roles</option>
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {/* Users Table */}
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <DataTable
            data={users()?.users || []}
            columns={columns}
            loading={users.loading}
            emptyMessage="No users found"
          />

          <Show when={users() && users()!.total > limit()}>
            <Pagination
              currentPage={page()}
              totalPages={Math.ceil(users()!.total / limit())}
              totalItems={users()!.total}
              itemsPerPage={limit()}
              onPageChange={setPage}
            />
          </Show>
        </div>
      </div>

      {/* Edit Modal */}
      <Modal title="Edit User" isOpen={editModal.isOpen()} onClose={editModal.close}>
        <form onSubmit={submitEdit} class="space-y-4">
          <Show when={error()}>
            <div class="alert alert-error">{error()}</div>
          </Show>
          <Show when={success()}>
            <div class="alert alert-success">{success()}</div>
          </Show>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Name</span>
            </label>
            <input
              type="text"
              class="input input-bordered"
              value={editForm().name}
              onInput={(e) => setEditForm({ ...editForm(), name: e.currentTarget.value })}
              required
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Email</span>
            </label>
            <input
              type="email"
              class="input input-bordered"
              value={editForm().email}
              onInput={(e) => setEditForm({ ...editForm(), email: e.currentTarget.value })}
              required
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Role</span>
            </label>
            <select
              class="select select-bordered"
              value={editForm().role}
              onChange={(e) => setEditForm({ ...editForm(), role: e.currentTarget.value })}
            >
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
          </div>

          <div class="flex justify-end gap-2 mt-6">
            <button type="button" class="btn btn-ghost" onClick={editModal.close}>
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              Save Changes
            </button>
          </div>
        </form>
      </Modal>

      {/* Ban/Unban Confirmation Modal */}
      <Modal 
        title={selectedUser()?.banned ? "Unban User" : "Ban User"} 
        isOpen={banModal.isOpen()} 
        onClose={banModal.close} 
        size="sm"
      >
        <Show when={error()}>
          <div class="alert alert-error mb-4">{error()}</div>
        </Show>
        <Show when={success()}>
          <div class="alert alert-success mb-4">{success()}</div>
        </Show>

        <p class="mb-6">
          {selectedUser()?.banned ? (
            <>
              Are you sure you want to unban user <strong>{selectedUser()?.name}</strong>? 
              This will restore their access to the platform.
            </>
          ) : (
            <>
              Are you sure you want to permanently ban user <strong>{selectedUser()?.name}</strong>? 
              They will no longer be able to access the platform.
            </>
          )}
        </p>

        <div class="flex justify-end gap-2">
          <button class="btn btn-ghost" onClick={banModal.close}>
            Cancel
          </button>
          <button 
            class={`btn ${selectedUser()?.banned ? 'btn-success' : 'btn-warning'}`}
            onClick={confirmBanToggle}
          >
            {selectedUser()?.banned ? 'Unban User' : 'Ban User'}
          </button>
        </div>
      </Modal>
    </div>
  );
}
