import {
  createSignal,
  createResource,
  For,
  Show,
  createEffect,
} from "solid-js";
import { A, useSearchParams } from "@solidjs/router";
import Navbar from "../components/Navbar";
import { productService } from "../services/productService";
import type { Category } from "../types";

type ProductRatingSummary = {
  average: number;
  count: number;
};

export default function Products() {
  const [searchParams] = useSearchParams();
  const [selectedCategory, setSelectedCategory] = createSignal<string>("");
  const [searchQuery, setSearchQuery] = createSignal("");
  const [minPrice, setMinPrice] = createSignal<number>();
  const [maxPrice, setMaxPrice] = createSignal<number>();

  const [categories] = createResource(async () => {
    return await productService.listCategories();
  });

  const [products] = createResource(
    () => ({
      category: selectedCategory(),
      search: searchQuery(),
      minPrice: minPrice(),
      maxPrice: maxPrice(),
    }),
    async (params) => {
      return await productService.listProducts(params);
    },
  );

  const [productRatings] = createResource(
    () =>
      products()
        ?.map((product) => product.id)
        .join(",") || "",
    async () => {
      const currentProducts = products() || [];
      const ratingEntries = await Promise.all(
        currentProducts.map(async (product) => {
          try {
            const reviews = await productService.getProductReviews(product.id);
            const count = reviews.length;
            const average =
              count > 0
                ? reviews.reduce((sum, review) => sum + review.rating, 0) /
                  count
                : 0;

            return [product.id, { average, count }] as const;
          } catch {
            return [product.id, { average: 0, count: 0 }] as const;
          }
        }),
      );

      return Object.fromEntries(ratingEntries) as Record<
        number,
        ProductRatingSummary
      >;
    },
  );

  createEffect(() => {
    const category = searchParams.category?.trim() || "";
    const search = searchParams.search?.trim() || "";
    const min = searchParams.minPrice?.trim() || "";
    const max = searchParams.maxPrice?.trim() || "";

    setSearchQuery(search);
    setMinPrice(min === "" ? undefined : Number(min));
    setMaxPrice(max === "" ? undefined : Number(max));

    if (!category) {
      setSelectedCategory("");
      return;
    }

    if (/^\d+$/.test(category)) {
      setSelectedCategory(category);
      return;
    }

    const matchedCategory = categories()?.find((item) => {
      const raw = category.toLowerCase();
      return item.name.toLowerCase() === raw || item.slug.toLowerCase() === raw;
    });

    setSelectedCategory(matchedCategory ? String(matchedCategory.id) : "");
  });

  const getCategoryLabel = (category: Category) => {
    const parentName = categories()?.find(
      (c) => c.id === category.parent_id,
    )?.name;
    return parentName ? `${parentName} > ${category.name}` : category.name;
  };

  return (
    <>
      <Navbar />
      <div class="container mx-auto px-4 py-8">
        <div class="flex flex-col md:flex-row justify-between items-start md:items-center mb-6">
          <h1 class="text-4xl font-extrabold tracking-tight">Our Products</h1>
        </div>

        {/* Enhanced Filters Section */}
        <div class="bg-base-200/60 rounded-box p-4 md:p-6 mb-10 border border-base-300">
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-lg font-bold flex items-center gap-2">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 text-base-content/70"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
                />
              </svg>
              Filter Products
            </h2>
            <button
              class="btn btn-sm btn-ghost text-base-content/60 hover:text-error"
              onClick={() => {
                setSelectedCategory("");
                setSearchQuery("");
                setMinPrice(undefined);
                setMaxPrice(undefined);
              }}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-4 w-4 mr-1"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
              Clear
            </button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-12 gap-4">
            {/* Search - Takes up more space */}
            <div class="form-control md:col-span-5">
              <label class="label hidden md:block py-0 pb-1">
                <span class="label-text text-xs font-medium uppercase text-base-content/70">
                  Search
                </span>
              </label>
              <label class="input input-bordered flex items-center gap-2 focus-within:outline-primary">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                  class="w-4 h-4 opacity-60"
                >
                  <path
                    fill-rule="evenodd"
                    d="M9.965 11.026a5 5 0 1 1 1.06-1.06l2.755 2.754a.75.75 0 1 1-1.06 1.06l-2.755-2.754ZM10.5 7a3.5 3.5 0 1 1-7 0 3.5 3.5 0 0 1 7 0Z"
                    clip-rule="evenodd"
                  />
                </svg>
                <input
                  type="text"
                  class="grow"
                  placeholder="What are you looking for?"
                  value={searchQuery()}
                  onInput={(e) => setSearchQuery(e.currentTarget.value)}
                />
              </label>
            </div>

            {/* Category */}
            <div class="form-control md:col-span-3">
              <label class="label hidden md:block py-0 pb-1">
                <span class="label-text text-xs font-medium uppercase text-base-content/70">
                  Category
                </span>
              </label>
              <select
                class="select select-bordered w-full focus:outline-primary"
                value={selectedCategory()}
                onChange={(e) => {
                  setSelectedCategory(e.currentTarget.value);
                }}
              >
                <option value="">All Categories</option>
                <Show when={!categories.loading}>
                  <For each={categories()}>
                    {(category: Category) => (
                      <option value={String(category.id)}>
                        {getCategoryLabel(category)}
                      </option>
                    )}
                  </For>
                </Show>
              </select>
            </div>

            {/* Price Range via Join */}
            <div class="form-control md:col-span-4">
              <label class="label hidden md:block py-0 pb-1">
                <span class="label-text text-xs font-medium uppercase text-base-content/70">
                  Price Range
                </span>
              </label>
              <div class="join w-full">
                <input
                  type="number"
                  placeholder="Min $"
                  class="input input-bordered join-item w-1/2 focus:outline-primary"
                  value={minPrice() ?? ""}
                  onInput={(e) => {
                    const value = e.currentTarget.value.trim();
                    setMinPrice(value === "" ? undefined : Number(value));
                  }}
                />
                <div class="join-item flex items-center bg-base-100 border-y border-base-content/20 px-2 text-base-content/50">
                  -
                </div>
                <input
                  type="number"
                  placeholder="Max $"
                  class="input input-bordered join-item w-1/2 focus:outline-primary"
                  value={maxPrice() ?? ""}
                  onInput={(e) => {
                    const value = e.currentTarget.value.trim();
                    setMaxPrice(value === "" ? undefined : Number(value));
                  }}
                />
              </div>
            </div>
          </div>
        </div>

        {/* Products Grid */}
        <Show
          when={!products.loading}
          fallback={
            <div class="flex flex-col justify-center items-center py-32 space-y-4">
              <span class="loading loading-ring loading-lg text-primary"></span>
              <p class="text-base-content/60 font-medium">
                Loading products...
              </p>
            </div>
          }
        >
          <Show
            when={products() && products()!.length > 0}
            fallback={
              <div class="flex flex-col items-center justify-center py-32 text-center bg-base-100 rounded-box border border-base-200 border-dashed">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-16 w-16 text-base-300 mb-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
                  />
                </svg>
                <h3 class="text-xl font-bold text-base-content">
                  No products found
                </h3>
                <p class="text-base-content/60 mt-2 max-w-sm">
                  Try adjusting your filters or search query to find what you're
                  looking for.
                </p>
                <button
                  class="btn btn-primary mt-6"
                  onClick={() => {
                    setSelectedCategory("");
                    setSearchQuery("");
                    setMinPrice(undefined);
                    setMaxPrice(undefined);
                  }}
                >
                  Clear all filters
                </button>
              </div>
            }
          >
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
              <For each={products()}>
                {(product) => (
                  <A
                    href={`/products/${product.id}`}
                    class="card bg-base-100 border border-base-200 shadow-sm hover:shadow-xl hover:border-primary/30 transition-all duration-300 group"
                  >
                    <figure class="h-56 overflow-hidden bg-base-200">
                      <img
                        src={product.image_url || "/favicon.svg"}
                        alt={product.name}
                        class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                      />
                    </figure>
                    <div class="card-body p-5">
                      <div class="flex justify-between items-start gap-2">
                        <h2 class="card-title text-lg leading-tight">
                          {product.name}
                        </h2>
                        <Show
                          when={product.stock > 0}
                          fallback={
                            <div class="badge badge-error badge-sm whitespace-nowrap">
                              Out of Stock
                            </div>
                          }
                        >
                          <div class="badge badge-success badge-outline badge-sm whitespace-nowrap">
                            In Stock
                          </div>
                        </Show>
                      </div>
                      <p class="text-sm text-base-content/70 line-clamp-2 mt-2">
                        {product.description}
                      </p>
                      <div class="mt-3 flex items-center gap-2 text-sm">
                        <Show
                          when={!productRatings.loading && productRatings()}
                          fallback={
                            <div class="h-4 w-28 rounded bg-base-200 animate-pulse" />
                          }
                        >
                          {(() => {
                            const rating = productRatings()?.[product.id] || {
                              average: 0,
                              count: 0,
                            };

                            return (
                              <>
                                <div class="flex items-center">
                                  <For each={[1, 2, 3, 4, 5]}>
                                    {(star) => (
                                      <span
                                        class={
                                          star <= Math.round(rating.average)
                                            ? "text-warning"
                                            : "text-base-300"
                                        }
                                      >
                                        ★
                                      </span>
                                    )}
                                  </For>
                                </div>
                                <span class="font-medium text-base-content/80">
                                  {rating.count > 0
                                    ? `${rating.average.toFixed(1)} (${rating.count})`
                                    : "No reviews"}
                                </span>
                              </>
                            );
                          })()}
                        </Show>
                      </div>
                      <div class="card-actions justify-between items-end mt-4 pt-4 border-t border-base-200">
                        <div class="flex flex-col">
                          <span class="text-xs text-base-content/50 uppercase font-semibold">
                            Price
                          </span>
                          <span class="text-2xl font-bold text-base-content">
                            ${product.price.toFixed(2)}
                          </span>
                        </div>
                        <button class="btn btn-primary btn-sm px-6">
                          View
                        </button>
                      </div>
                    </div>
                  </A>
                )}
              </For>
            </div>
          </Show>
        </Show>
      </div>
    </>
  );
}
