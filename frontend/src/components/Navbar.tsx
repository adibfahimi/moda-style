import { A, useNavigate } from "@solidjs/router";
import { Show, createSignal } from "solid-js";
import { useAuth } from "../context/AuthContext";
import { useCart } from "../context/CartContext";

export default function Navbar() {
  const auth = useAuth();
  const cart = useCart();
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = createSignal("");

  const handleLogout = () => {
    auth.logout();
    navigate("/login");
  };

  const handleSearchSubmit = (e: Event) => {
    e.preventDefault();
    const query = searchQuery().trim();
    if (!query) {
      navigate("/products");
      return;
    }
    navigate(`/products?search=${encodeURIComponent(query)}`);
  };

  return (
    <header class="sticky top-0 z-50 border-b border-base-300 bg-base-100/95 backdrop-blur supports-backdrop-filter:bg-base-100/80">
      <div class="hidden border-b border-base-300 bg-base-200 md:block">
        <div class="container mx-auto flex items-center justify-between px-4 py-2 text-sm">
          <p class="font-medium">Free shipping on orders over $75</p>
          <div class="flex items-center gap-5 text-base-content/80">
            <A href="/orders" class="hover:text-base-content">
              Order Tracking
            </A>
            <A href="/profile" class="hover:text-base-content">
              My Account
            </A>
            <A href="/login" class="hover:text-base-content">
              Support
            </A>
          </div>
        </div>
      </div>

      <div class="container mx-auto px-4">
        <div class="navbar gap-2 py-3">
          <div class="navbar-start w-auto lg:w-1/3">
            <div class="dropdown lg:hidden">
              <div tabIndex={0} role="button" class="btn btn-ghost btn-circle">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M4 6h16M4 12h16M4 18h16"
                  />
                </svg>
              </div>
              <ul
                tabIndex={0}
                class="menu menu-md dropdown-content rounded-box z-20 mt-3 w-72 gap-1 border border-base-300 bg-base-100 p-3 shadow-xl"
              >
                <li>
                  <A href="/">Home</A>
                </li>
                <li>
                  <A href="/products">New Arrivals</A>
                </li>
                <li>
                  <A href="/products?category=Women">Women</A>
                </li>
                <li>
                  <A href="/products?category=Men">Men</A>
                </li>
                <li>
                  <A href="/products?category=Accessories">Accessories</A>
                </li>
                <li>
                  <A href="/products?category=Footwear">Footwear</A>
                </li>
                <li>
                  <A href="/products">Sale</A>
                </li>
                <li class="menu-title mt-2">
                  <span>Account</span>
                </li>
                <Show
                  when={auth.isAuthenticated}
                  fallback={
                    <>
                      <li>
                        <A href="/login">Login</A>
                      </li>
                      <li>
                        <A href="/register">Register</A>
                      </li>
                    </>
                  }
                >
                  <li>
                    <A href="/profile">Profile</A>
                  </li>
                  <li>
                    <A href="/orders">Orders</A>
                  </li>
                  <Show when={auth.user?.role === "admin"}>
                    <li>
                      <A href="/admin">Admin Dashboard</A>
                    </li>
                  </Show>
                  <li>
                    <button type="button" onclick={handleLogout}>
                      Logout
                    </button>
                  </li>
                </Show>
              </ul>
            </div>

            <A href="/" class="ml-1 text-2xl font-black tracking-tight">
              Moda Style
            </A>
          </div>

          <div class="navbar-center hidden w-1/3 lg:flex">
            <ul class="menu menu-horizontal gap-1 px-1 text-sm font-semibold">
              <li>
                <A href="/">Home</A>
              </li>
              <li>
                <A href="/products">Shop</A>
              </li>
              <li>
                <A href="/products">Collections</A>
              </li>
            </ul>
          </div>

          <div class="navbar-end ml-auto w-auto gap-2 lg:w-1/3">
            <form class="hidden md:flex md:w-72" onSubmit={handleSearchSubmit}>
              <label class="input input-bordered h-10 w-full items-center gap-2">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-4 w-4 opacity-70"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M21 21l-4.35-4.35M11 18a7 7 0 100-14 7 7 0 000 14z"
                  />
                </svg>
                <input
                  type="text"
                  class="grow"
                  placeholder="Search products"
                  value={searchQuery()}
                  onInput={(e) => setSearchQuery(e.currentTarget.value)}
                />
              </label>
            </form>

            <A
              href="/products"
              class="btn btn-ghost btn-circle hidden md:inline-flex"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"
                />
              </svg>
            </A>

            <A href="/cart" class="btn btn-ghost btn-circle">
              <div class="indicator">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z"
                  />
                </svg>
                <Show when={cart.count > 0}>
                  <span class="badge badge-sm badge-primary indicator-item">
                    {cart.count}
                  </span>
                </Show>
              </div>
            </A>

            <Show
              when={auth.isAuthenticated}
              fallback={
                <div class="hidden gap-2 md:flex">
                  <A href="/login" class="btn btn-ghost">
                    Login
                  </A>
                  <A href="/register" class="btn btn-primary">
                    Register
                  </A>
                </div>
              }
            >
              <div class="dropdown dropdown-end">
                <div
                  tabIndex={0}
                  role="button"
                  class="btn btn-ghost btn-circle avatar"
                >
                  <div class="flex w-10 items-center justify-center rounded-full bg-primary text-primary-content">
                    <span class="text-lg font-bold">
                      {auth.user?.name.charAt(0).toUpperCase()}
                    </span>
                  </div>
                </div>
                <ul
                  tabIndex={0}
                  class="menu menu-sm dropdown-content rounded-box z-20 mt-3 w-56 gap-1 border border-base-300 bg-base-100 p-2 shadow-xl"
                >
                  <li class="menu-title">
                    <span>{auth.user?.name}</span>
                  </li>
                  <li>
                    <A href="/profile">Profile</A>
                  </li>
                  <li>
                    <A href="/orders">Orders</A>
                  </li>
                  <Show when={auth.user?.role === "admin"}>
                    <li>
                      <A href="/admin">Admin Dashboard</A>
                    </li>
                  </Show>
                  <li>
                    <button type="button" onclick={handleLogout}>
                      Logout
                    </button>
                  </li>
                </ul>
              </div>
            </Show>
          </div>
        </div>
      </div>

      <div class="hidden border-t border-base-300 lg:block">
        <div class="container mx-auto flex items-center justify-between px-4">
          <ul class="menu menu-horizontal gap-1 py-1 text-sm">
            <li>
              <A href="/products?category=Women">Women</A>
            </li>
            <li>
              <A href="/products?category=Men">Men</A>
            </li>
            <li>
              <A href="/products?category=Accessories">Accessories</A>
            </li>
            <li>
              <A href="/products?category=Footwear">Footwear</A>
            </li>
            <li>
              <A href="/products">New Arrivals</A>
            </li>
            <li>
              <A href="/products" class="font-semibold text-primary">
                Sale
              </A>
            </li>
          </ul>
          <A href="/products" class="btn btn-ghost btn-sm">
            Track Trends
          </A>
        </div>
      </div>
    </header>
  );
}
