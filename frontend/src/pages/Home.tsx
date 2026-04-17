import { A } from "@solidjs/router";
import { createMemo, createResource, For, Show } from "solid-js";
import Navbar from "../components/Navbar";
import { productService } from "../services/productService";

export default function Home() {
  const [products] = createResource(async () => {
    return await productService.listProducts();
  });

  const trendingProducts = createMemo(() => (products() || []).slice(0, 2));
  const featuredProducts = createMemo(() => (products() || []).slice(0, 3));

  return (
    <>
      <Navbar />

      <div class="bg-base-100">
        <section class="hero min-h-[85vh] bg-base-200">
          <div class="hero-content w-full max-w-7xl flex-col gap-10 px-4 py-16 lg:flex-row lg:justify-between">
            <div class="max-w-2xl">
              <span class="badge badge-primary badge-outline mb-5">
                New Season Arrivals
              </span>
              <h1 class="text-4xl font-extrabold leading-tight sm:text-5xl lg:text-6xl">
                Curated Fashion For Everyday Confidence
              </h1>
              <p class="py-6 text-base-content/70 sm:text-lg">
                Discover premium essentials, statement looks, and timeless
                pieces designed to fit your lifestyle. Shop faster with a clean
                checkout and reliable delivery.
              </p>
              <div class="flex flex-wrap gap-3">
                <A href="/products" class="btn btn-primary btn-lg">
                  Shop Collection
                </A>
                <A href="/register" class="btn btn-outline btn-lg">
                  Create Account
                </A>
              </div>
              <div class="mt-8 flex flex-wrap gap-6 text-sm text-base-content/70">
                <div>
                  <div class="text-xl font-bold text-base-content">15k+</div>
                  Happy customers
                </div>
                <div>
                  <div class="text-xl font-bold text-base-content">1.2k+</div>
                  Curated products
                </div>
                <div>
                  <div class="text-xl font-bold text-base-content">4.9/5</div>
                  Average rating
                </div>
              </div>
            </div>

            <div class="w-full max-w-md">
              <div class="card bg-base-100 shadow-2xl">
                <div class="card-body gap-4">
                  <div class="flex items-center justify-between">
                    <h2 class="card-title">Trending Now</h2>
                    <span class="badge badge-secondary">Limited</span>
                  </div>

                  <div class="space-y-3">
                    <Show
                      when={!products.loading && trendingProducts().length > 0}
                      fallback={
                        <>
                          <div class="h-18 rounded-box bg-base-200 animate-pulse" />
                          <div class="h-18 rounded-box bg-base-200 animate-pulse" />
                        </>
                      }
                    >
                      <For each={trendingProducts()}>
                        {(item) => (
                          <A
                            href={`/products/${item.id}`}
                            class="flex items-center gap-3 rounded-box bg-base-200 p-3 transition hover:bg-base-300"
                          >
                            <div class="avatar">
                              <div class="w-12 rounded-lg bg-base-100">
                                <img
                                  src={item.image_url || "/favicon.svg"}
                                  alt={item.name}
                                  class="object-cover"
                                />
                              </div>
                            </div>
                            <div class="flex-1">
                              <p class="line-clamp-1 font-semibold">
                                {item.name}
                              </p>
                              <p class="text-sm text-base-content/60">
                                {item.stock > 0 ? "In stock" : "Out of stock"}
                              </p>
                            </div>
                            <p class="font-bold">${item.price.toFixed(0)}</p>
                          </A>
                        )}
                      </For>
                    </Show>
                  </div>

                  <A href="/products" class="btn btn-primary w-full">
                    View All Products
                  </A>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="container mx-auto px-4 py-16">
          <div class="mb-10 flex items-end justify-between gap-4">
            <div>
              <h2 class="text-3xl font-bold sm:text-4xl">Shop By Category</h2>
              <p class="mt-2 text-base-content/70">
                Explore collections tailored for every occasion.
              </p>
            </div>
            <A href="/products" class="btn btn-ghost">
              See all
            </A>
          </div>

          <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
            <div class="card bg-base-200 shadow-lg transition hover:-translate-y-1 hover:shadow-xl">
              <div class="card-body">
                <h3 class="card-title">Women</h3>
                <p class="text-base-content/70">Elegant fits and essentials</p>
                <div class="card-actions justify-end">
                  <A
                    href="/products?category=Women"
                    class="btn btn-sm btn-primary"
                  >
                    Explore
                  </A>
                </div>
              </div>
            </div>

            <div class="card bg-base-200 shadow-lg transition hover:-translate-y-1 hover:shadow-xl">
              <div class="card-body">
                <h3 class="card-title">Men</h3>
                <p class="text-base-content/70">Modern casual to formal wear</p>
                <div class="card-actions justify-end">
                  <A
                    href="/products?category=Men"
                    class="btn btn-sm btn-primary"
                  >
                    Explore
                  </A>
                </div>
              </div>
            </div>

            <div class="card bg-base-200 shadow-lg transition hover:-translate-y-1 hover:shadow-xl">
              <div class="card-body">
                <h3 class="card-title">Accessories</h3>
                <p class="text-base-content/70">Bags, belts, and details</p>
                <div class="card-actions justify-end">
                  <A
                    href="/products?category=Accessories"
                    class="btn btn-sm btn-primary"
                  >
                    Explore
                  </A>
                </div>
              </div>
            </div>

            <div class="card bg-base-200 shadow-lg transition hover:-translate-y-1 hover:shadow-xl">
              <div class="card-body">
                <h3 class="card-title">Footwear</h3>
                <p class="text-base-content/70">Comfort with standout style</p>
                <div class="card-actions justify-end">
                  <A
                    href="/products?category=Footwear"
                    class="btn btn-sm btn-primary"
                  >
                    Explore
                  </A>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="bg-base-200 py-16">
          <div class="container mx-auto px-4">
            <div class="mb-10 text-center">
              <h2 class="text-3xl font-bold sm:text-4xl">Featured Picks</h2>
              <p class="mt-3 text-base-content/70">
                Handpicked items customers are loving right now.
              </p>
            </div>

            <div class="grid grid-cols-1 gap-6 md:grid-cols-3">
              <Show
                when={!products.loading && featuredProducts().length > 0}
                fallback={
                  <>
                    <div class="h-96 rounded-box bg-base-300 animate-pulse" />
                    <div class="h-96 rounded-box bg-base-300 animate-pulse" />
                    <div class="h-96 rounded-box bg-base-300 animate-pulse" />
                  </>
                }
              >
                <For each={featuredProducts()}>
                  {(item) => (
                    <div class="card bg-base-100 shadow-xl">
                      <figure class="bg-base-300 px-10 pt-10">
                        <img
                          src={item.image_url || "/favicon.svg"}
                          alt={item.name}
                          class="h-48 w-full rounded-box object-cover"
                        />
                      </figure>
                      <div class="card-body">
                        <h3 class="card-title line-clamp-1">{item.name}</h3>
                        <p class="line-clamp-2 text-base-content/70">
                          {item.description ||
                            "Carefully selected for your style."}
                        </p>
                        <div class="card-actions items-center justify-between">
                          <span class="text-lg font-bold">
                            ${item.price.toFixed(2)}
                          </span>
                          <A
                            href={`/products/${item.id}`}
                            class="btn btn-sm btn-primary"
                          >
                            View
                          </A>
                        </div>
                      </div>
                    </div>
                  )}
                </For>
              </Show>
            </div>
          </div>
        </section>

        <section class="container mx-auto px-4 py-16">
          <h2 class="mb-10 text-center text-3xl font-bold sm:text-4xl">
            Why Customers Choose Us
          </h2>
          <div class="grid grid-cols-1 gap-6 md:grid-cols-3">
            <div class="card border border-base-300 bg-base-100">
              <div class="card-body text-center">
                <h3 class="card-title mx-auto">Premium Quality</h3>
                <p class="text-base-content/70">
                  Carefully selected materials and verified suppliers.
                </p>
              </div>
            </div>
            <div class="card border border-base-300 bg-base-100">
              <div class="card-body text-center">
                <h3 class="card-title mx-auto">Fast Delivery</h3>
                <p class="text-base-content/70">
                  Quick dispatch and reliable order tracking from checkout.
                </p>
              </div>
            </div>
            <div class="card border border-base-300 bg-base-100">
              <div class="card-body text-center">
                <h3 class="card-title mx-auto">Secure Checkout</h3>
                <p class="text-base-content/70">
                  Trusted payment flow and protected customer information.
                </p>
              </div>
            </div>
          </div>
        </section>

        <section class="bg-base-200 py-16">
          <div class="container mx-auto grid gap-8 px-4 lg:grid-cols-2 lg:items-center">
            <div>
              <h2 class="text-3xl font-bold sm:text-4xl">How It Works</h2>
              <p class="mt-3 text-base-content/70">
                We built a simple flow so you can discover, order, and receive
                your favorite items without friction.
              </p>
            </div>
            <ul class="steps steps-vertical">
              <li class="step step-primary">
                Browse curated products across categories
              </li>
              <li class="step step-primary">Add favorites to your cart</li>
              <li class="step step-primary">Checkout with secure payment</li>
              <li class="step">Track your order to delivery</li>
            </ul>
          </div>
        </section>

        <section class="container mx-auto px-4 py-16">
          <div class="mb-10 text-center">
            <h2 class="text-3xl font-bold sm:text-4xl">Customer Stories</h2>
            <p class="mt-3 text-base-content/70">
              Real feedback from shoppers who wear Moda Style every week.
            </p>
          </div>

          <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <div class="card bg-base-100 shadow-lg">
              <div class="card-body">
                <div class="rating rating-sm">
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                </div>
                <p>
                  Great quality and sizing. My order arrived quickly and exactly
                  as described.
                </p>
                <p class="font-semibold">Sarah N.</p>
              </div>
            </div>

            <div class="card bg-base-100 shadow-lg">
              <div class="card-body">
                <div class="rating rating-sm">
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                </div>
                <p>
                  The website is easy to use, and checkout took less than two
                  minutes. Will shop again.
                </p>
                <p class="font-semibold">David M.</p>
              </div>
            </div>

            <div class="card bg-base-100 shadow-lg">
              <div class="card-body">
                <div class="rating rating-sm">
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                  <input
                    type="radio"
                    class="mask mask-star-2 bg-orange-400"
                    checked
                  />
                </div>
                <p>
                  Love the curated style. It feels like shopping with a personal
                  stylist online.
                </p>
                <p class="font-semibold">Amina R.</p>
              </div>
            </div>
          </div>
        </section>

        <section class="bg-base-200 py-16">
          <div class="container mx-auto grid gap-8 px-4 lg:grid-cols-2">
            <div>
              <h2 class="text-3xl font-bold sm:text-4xl">Frequently Asked</h2>
              <p class="mt-3 text-base-content/70">
                Have questions? Here are the answers to what customers ask most.
              </p>
            </div>

            <div class="space-y-3">
              <div class="collapse collapse-plus bg-base-100">
                <input type="radio" name="faq-accordion" checked />
                <div class="collapse-title text-lg font-medium">
                  How long does delivery take?
                </div>
                <div class="collapse-content">
                  <p>Most orders arrive within 2 to 5 business days.</p>
                </div>
              </div>

              <div class="collapse collapse-plus bg-base-100">
                <input type="radio" name="faq-accordion" />
                <div class="collapse-title text-lg font-medium">
                  Can I return or exchange items?
                </div>
                <div class="collapse-content">
                  <p>
                    Yes. You can request returns or exchanges within 14 days of
                    delivery.
                  </p>
                </div>
              </div>

              <div class="collapse collapse-plus bg-base-100">
                <input type="radio" name="faq-accordion" />
                <div class="collapse-title text-lg font-medium">
                  Do I need an account to place an order?
                </div>
                <div class="collapse-content">
                  <p>
                    Guest browsing is available, but creating an account helps
                    you track orders and save favorites.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="container mx-auto px-4 py-16">
          <div class="hero rounded-box bg-primary text-primary-content">
            <div class="hero-content w-full flex-col gap-8 px-6 py-12 text-center lg:flex-row lg:justify-between lg:text-left">
              <div class="max-w-xl">
                <h2 class="text-3xl font-bold sm:text-4xl">
                  Get Style Updates Every Week
                </h2>
                <p class="mt-3 text-primary-content/90">
                  Be the first to know about new drops, seasonal edits, and
                  member-only deals.
                </p>
              </div>
              <div class="w-full max-w-md">
                <div class="join w-full">
                  <input
                    type="email"
                    placeholder="you@example.com"
                    class="input input-bordered join-item w-full text-base-content"
                  />
                  <button class="btn join-item btn-secondary">Subscribe</button>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="pb-20">
          <div class="container mx-auto px-4">
            <div class="rounded-box border border-base-300 bg-base-100 p-8 text-center sm:p-12">
              <h2 class="text-3xl font-bold sm:text-4xl">
                Ready To Refresh Your Wardrobe?
              </h2>
              <p class="mx-auto mt-4 max-w-2xl text-base-content/70">
                Explore our latest collection and find pieces that feel made for
                your everyday life.
              </p>
              <div class="mt-8 flex flex-wrap justify-center gap-3">
                <A href="/products" class="btn btn-primary btn-lg">
                  Start Shopping
                </A>
                <A href="/login" class="btn btn-outline btn-lg">
                  Sign In
                </A>
              </div>
            </div>
          </div>
        </section>
      </div>
    </>
  );
}
