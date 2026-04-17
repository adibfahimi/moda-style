import { A, useLocation } from '@solidjs/router';
import { Show } from 'solid-js';

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function AdminSidebar(props: SidebarProps) {
  const location = useLocation();

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  const menuItems = [
    { path: '/admin', icon: '📊', label: 'Dashboard', exact: true },
    { path: '/admin/users', icon: '👥', label: 'Users' },
    { path: '/admin/products', icon: '🛍️', label: 'Products' },
    { path: '/admin/categories', icon: '📁', label: 'Categories' },
    { path: '/admin/orders', icon: '📦', label: 'Orders' },
    { path: '/admin/activity', icon: '📝', label: 'Activity Logs' },
  ];

  return (
    <>
      {/* Mobile Overlay */}
      <Show when={props.isOpen}>
        <div
          class="fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden"
          onClick={props.onClose}
        />
      </Show>

      {/* Sidebar */}
      <aside
        class={`fixed lg:sticky top-0 left-0 z-50 h-screen w-64 bg-base-200 transition-transform duration-300 ${
          props.isOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
        }`}
      >
        <div class="flex flex-col h-full">
          {/* Header */}
          <div class="flex items-center justify-between p-4 border-b border-base-300">
            <h2 class="text-xl font-bold">Admin Panel</h2>
            <button
              class="btn btn-ghost btn-sm lg:hidden"
              onClick={props.onClose}
            >
              ✕
            </button>
          </div>

          {/* Navigation */}
          <nav class="flex-1 overflow-y-auto p-4">
            <ul class="menu gap-2">
              {menuItems.map((item) => (
                <li>
                  <A
                    href={item.path}
                    class={
                      (item.exact
                        ? location.pathname === item.path
                        : isActive(item.path))
                        ? 'active'
                        : ''
                    }
                    onClick={props.onClose}
                  >
                    <span class="text-xl">{item.icon}</span>
                    <span>{item.label}</span>
                  </A>
                </li>
              ))}
            </ul>
          </nav>

          {/* Footer */}
          <div class="p-4 border-t border-base-300">
            <A href="/" class="btn btn-outline btn-sm w-full">
              ← Back to Store
            </A>
          </div>
        </div>
      </aside>
    </>
  );
}
