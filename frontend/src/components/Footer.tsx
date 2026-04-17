import { A } from "@solidjs/router";

export default function Footer() {
  return (
    <footer class="footer footer-horizontal bg-base-200 text-base-content p-10">
      <aside>
        <h3 class="text-xl font-bold">Moda Style</h3>
        <p class="max-w-xs text-base-content/70">
          Curated fashion essentials with secure checkout, fast delivery, and
          quality you can trust.
        </p>
      </aside>

      <nav>
        <h6 class="footer-title">Shop</h6>
        <A href="/products" class="link link-hover">
          All Products
        </A>
        <A href="/cart" class="link link-hover">
          Cart
        </A>
        <A href="/orders" class="link link-hover">
          Orders
        </A>
      </nav>

      <nav>
        <h6 class="footer-title">Account</h6>
        <A href="/login" class="link link-hover">
          Sign In
        </A>
        <A href="/register" class="link link-hover">
          Register
        </A>
        <A href="/profile" class="link link-hover">
          Profile
        </A>
      </nav>

      <nav>
        <h6 class="footer-title">Company</h6>
        <a class="link link-hover">About us</a>
        <a class="link link-hover">Contact</a>
        <a class="link link-hover">Privacy policy</a>
      </nav>
    </footer>
  );
}
