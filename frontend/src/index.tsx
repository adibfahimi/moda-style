/* @refresh reload */
import { Show } from "solid-js";
import { render } from "solid-js/web";
import "./index.css";
import { Router, Route, useLocation } from "@solidjs/router";
import { AuthProvider } from "./context/AuthContext";
import { CartProvider } from "./context/CartContext";
import Footer from "./components/Footer";
import Home from "./pages/Home";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Products from "./pages/Products";
import ProductDetail from "./pages/ProductDetail";
import Profile from "./pages/Profile";
import Cart from "./pages/Cart";
import Checkout from "./pages/Checkout";
import Orders from "./pages/Orders";
import OrderDetail from "./pages/OrderDetail";
import AdminLayout from "./components/admin/AdminLayout";
import AdminDashboard from "./pages/admin/Dashboard";
import AdminUsers from "./pages/admin/Users";
import AdminProducts from "./pages/admin/Products";
import AdminCategories from "./pages/admin/Categories";
import AdminOrders from "./pages/admin/Orders";
import AdminActivityLogs from "./pages/admin/ActivityLogs";

function Root(props: any) {
  const location = useLocation();

  const shouldHideFooter = () => {
    const path = location.pathname;
    const hiddenFooterRoutes = [
      "/admin",
      "/profile",
      "/login",
      "/register",
      "/checkout",
      "/orders",
    ];

    return hiddenFooterRoutes.some(
      (route) => path === route || path.startsWith(`${route}/`),
    );
  };

  return (
    <AuthProvider>
      <CartProvider>
        <div class="flex min-h-screen flex-col">
          <main class="flex-1">{props.children}</main>
          <Show when={!shouldHideFooter()}>
            <Footer />
          </Show>
        </div>
      </CartProvider>
    </AuthProvider>
  );
}

function AdminRoot(props: any) {
  return <AdminLayout>{props.children}</AdminLayout>;
}

render(
  () => (
    <Router root={Root}>
      <Route path="/" component={Home} />
      <Route path="/login" component={Login} />
      <Route path="/register" component={Register} />
      <Route path="/products" component={Products} />
      <Route path="/products/:id" component={ProductDetail} />
      <Route path="/profile" component={Profile} />
      <Route path="/cart" component={Cart} />
      <Route path="/checkout" component={Checkout} />
      <Route path="/orders" component={Orders} />
      <Route path="/orders/:id" component={OrderDetail} />

      {/* Admin Routes */}
      <Route path="/admin" component={AdminRoot}>
        <Route path="/" component={AdminDashboard} />
        <Route path="/users" component={AdminUsers} />
        <Route path="/products" component={AdminProducts} />
        <Route path="/categories" component={AdminCategories} />
        <Route path="/orders" component={AdminOrders} />
        <Route path="/activity" component={AdminActivityLogs} />
      </Route>
    </Router>
  ),
  document.getElementById("root")!,
);
