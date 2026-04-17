import { createSignal, createEffect, Show } from "solid-js";
import type { JSX } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { useAuth } from "../../context/AuthContext";
import AdminSidebar from "../../components/admin/AdminSidebar";

interface AdminLayoutProps {
  children: JSX.Element;
}

export default function AdminLayout(props: AdminLayoutProps) {
  const [sidebarOpen, setSidebarOpen] = createSignal(false);
  const auth = useAuth();
  const navigate = useNavigate();

  // Check if user is admin after loading completes
  createEffect(() => {
    // Only redirect after loading is complete
    if (
      !auth.isLoading &&
      (!auth.isAuthenticated || auth.user?.role !== "admin")
    ) {
      navigate("/login", { replace: true });
    }
  });

  // Show loading while checking authentication
  return (
    <Show
      when={!auth.isLoading}
      fallback={
        <div class="flex items-center justify-center h-screen">
          <span class="loading loading-spinner loading-lg"></span>
        </div>
      }
    >
      <div class="flex h-screen overflow-hidden bg-base-300">
        <AdminSidebar
          isOpen={sidebarOpen()}
          onClose={() => setSidebarOpen(false)}
        />

        <div class="flex-1 flex flex-col overflow-hidden">
          {/* Top Bar */}
          <header class="bg-base-100 shadow-md z-30">
            <div class="flex items-center justify-between px-6 py-4">
              <button
                class="btn btn-ghost btn-square lg:hidden"
                onClick={() => setSidebarOpen(true)}
              >
                <svg
                  class="w-6 h-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M4 6h16M4 12h16M4 18h16"
                  />
                </svg>
              </button>

              <h1 class="text-xl font-bold hidden lg:block">Admin Dashboard</h1>

              <div class="flex items-center gap-4">
                <div class="text-right hidden sm:block">
                  <p class="font-medium">{auth.user?.name}</p>
                  <p class="text-sm text-base-content/60">{auth.user?.email}</p>
                </div>
                <div class="avatar avatar-placeholder">
                  <div class="bg-primary text-primary-content rounded-full w-10">
                    <span class="text-xl">
                      {auth.user?.name?.[0]?.toUpperCase()}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </header>

          {/* Main Content */}
          <main class="flex-1 overflow-x-hidden overflow-y-auto">
            {props.children}
          </main>
        </div>
      </div>
    </Show>
  );
}
