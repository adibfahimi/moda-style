import { createResource, Show, For } from 'solid-js';
import { getDashboardStats, getRecentActivity } from '../../services/adminService';
import type { DashboardStats, RecentActivity } from '../../types';
import StatsCard from '../../components/admin/StatsCard';

export default function AdminDashboard() {
  const [stats] = createResource<{ stats: DashboardStats }>(getDashboardStats);
  const [activity] = createResource<{ activities: RecentActivity[]; count: number }>(
    () => getRecentActivity(10)
  );

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div class="p-6">
      <h1 class="text-3xl font-bold mb-6">Dashboard</h1>

      {/* Stats Grid */}
      <Show
        when={!stats.loading && stats()}
        fallback={
          <div class="flex justify-center items-center h-64">
            <span class="loading loading-spinner loading-lg"></span>
          </div>
        }
      >
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
          <StatsCard
            title="Total Users"
            value={stats()?.stats.total_users || 0}
            icon="👥"
            color="primary"
          />
          <StatsCard
            title="Total Products"
            value={stats()?.stats.total_products || 0}
            icon="🛍️"
            color="secondary"
          />
          <StatsCard
            title="Categories"
            value={stats()?.stats.total_categories || 0}
            icon="📁"
            color="accent"
          />
          <StatsCard
            title="Total Reviews"
            value={stats()?.stats.total_reviews || 0}
            icon="⭐"
            color="info"
          />
          <StatsCard
            title="Low Stock Products"
            value={stats()?.stats.low_stock_products || 0}
            icon="⚠️"
            color="warning"
          />
          <StatsCard
            title="Average Rating"
            value={(stats()?.stats.average_rating || 0).toFixed(1)}
            icon="📊"
            color="success"
          />
        </div>
      </Show>

      {/* Recent Activity */}
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <h2 class="card-title">Recent Activity</h2>
          <Show
            when={!activity.loading && activity()}
            fallback={
              <div class="flex justify-center py-8">
                <span class="loading loading-spinner loading-lg"></span>
              </div>
            }
          >
            <div class="space-y-4">
              <Show
                when={activity()?.activities.length! > 0}
                fallback={
                  <p class="text-base-content/60 text-center py-8">
                    No recent activity
                  </p>
                }
              >
                <For each={activity()?.activities}>
                  {(item) => (
                    <div class="flex items-start gap-4 p-4 rounded-lg bg-base-200">
                      <div class="text-2xl">
                        {item.type === 'user_registered' && '👤'}
                        {item.type === 'product_created' && '🛍️'}
                        {item.type === 'review_added' && '⭐'}
                      </div>
                      <div class="flex-1">
                        <p class="font-medium">{item.description}</p>
                        <p class="text-sm text-base-content/60">
                          {formatDate(item.created_at)}
                        </p>
                      </div>
                      <div class="badge badge-outline">{item.type}</div>
                    </div>
                  )}
                </For>
              </Show>
            </div>
          </Show>
        </div>
      </div>
    </div>
  );
}
