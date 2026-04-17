import { createSignal, createResource, For, Show } from "solid-js";
import { useParams, useNavigate } from "@solidjs/router";
import Navbar from "../components/Navbar";
import { productService } from "../services/productService";
import { useAuth } from "../context/AuthContext";
import { useCart } from "../context/CartContext";

export default function ProductDetail() {
  const params = useParams();
  const navigate = useNavigate();
  const auth = useAuth();
  const cart = useCart();

  const productId = () => Number(params.id);

  const [rating, setRating] = createSignal(5);
  const [comment, setComment] = createSignal("");
  const [reviewError, setReviewError] = createSignal("");
  const [reviewSuccess, setReviewSuccess] = createSignal(false);
  const [submittingReview, setSubmittingReview] = createSignal(false);

  const [selectedSize, setSelectedSize] = createSignal<string>("");
  const [selectedColor, setSelectedColor] = createSignal<string>("");
  const [quantity, setQuantity] = createSignal(1);
  const [addingToCart, setAddingToCart] = createSignal(false);
  const [feedbackMessage, setFeedbackMessage] = createSignal<string | null>(
    null,
  );
  const [feedbackTitle, setFeedbackTitle] = createSignal("Notice");

  const [product] = createResource(productId, async (id) => {
    return await productService.getProduct(id);
  });

  const [reviews, { refetch: refetchReviews }] = createResource(
    productId,
    async (id) => {
      return await productService.getProductReviews(id);
    },
  );

  const handleSubmitReview = async (e: Event) => {
    e.preventDefault();
    if (!auth.isAuthenticated) {
      navigate("/login");
      return;
    }

    setReviewError("");
    setReviewSuccess(false);
    setSubmittingReview(true);

    try {
      await productService.createReview(productId(), {
        rating: rating(),
        comment: comment(),
      });

      setReviewSuccess(true);
      setComment("");
      setRating(5);
      refetchReviews();
    } catch (err: any) {
      setReviewError(err.message || "Failed to submit review");
    } finally {
      setSubmittingReview(false);
    }
  };

  const averageRating = () => {
    const data = reviews();
    if (!data || data.length === 0) return 0;
    const sum = data.reduce((acc, review) => acc + review.rating, 0);
    return (sum / data.length).toFixed(1);
  };

  const availableSizes = () => {
    const p = product();
    if (!p || !p.sizes) return [];
    const sizes = [...new Set(p.sizes.map((s) => s.size))];
    return sizes;
  };

  const availableColors = () => {
    const p = product();
    if (!p || !p.sizes) return [];
    const size = selectedSize();
    if (!size) {
      return [...new Set(p.sizes.map((s) => s.color))];
    }
    return [
      ...new Set(p.sizes.filter((s) => s.size === size).map((s) => s.color)),
    ];
  };

  const selectedVariant = () => {
    const p = product();
    if (!p || !p.sizes || !selectedSize() || !selectedColor()) return null;
    return p.sizes.find(
      (s) => s.size === selectedSize() && s.color === selectedColor(),
    );
  };

  const handleAddToCart = async () => {
    if (!auth.isAuthenticated) {
      navigate("/login");
      return;
    }

    const variant = selectedVariant();
    if (!variant) {
      setFeedbackTitle("Selection Required");
      setFeedbackMessage("Please select both size and color");
      return;
    }

    setAddingToCart(true);
    try {
      await cart.addToCart({
        product_id: productId(),
        size_id: variant.id,
        quantity: quantity(),
      });

      setFeedbackTitle("Added To Cart");
      setFeedbackMessage("Item was added to your cart successfully.");
      setQuantity(1);
    } catch (err: any) {
      setFeedbackTitle("Could Not Add Item");
      setFeedbackMessage(err.message || "Failed to add to cart");
    } finally {
      setAddingToCart(false);
    }
  };

  return (
    <div class="min-h-screen bg-base-100">
      <Navbar />

      <main class="container mx-auto px-4 py-8 max-w-7xl">
        <Show
          when={!product.loading}
          fallback={
            <div class="flex justify-center items-center min-h-[60vh]">
              <span class="loading loading-spinner loading-lg text-primary"></span>
            </div>
          }
        >
          <Show when={product()} keyed>
            {(p) => (
              <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-16">
                {/* Product Image Section */}
                <div class="sticky top-4 h-fit">
                  <div class="card bg-base-200 shadow-2xl overflow-hidden">
                    <figure class="bg-linear-to-br from-base-300 to-base-100 p-8">
                      <img
                        src={
                          p.image_url ||
                          "https://via.placeholder.com/600x600?text=No+Image"
                        }
                        alt={p.name}
                        class="w-full h-auto max-h-150 object-contain rounded-lg"
                      />
                    </figure>
                  </div>
                </div>

                {/* Product Details Section */}
                <div class="flex flex-col space-y-6">
                  {/* Header */}
                  <div>
                    <h1 class="text-5xl font-bold mb-3 tracking-tight">
                      {p.name}
                    </h1>

                    <div class="flex items-center gap-3 mb-4">
                      <div class="rating rating-md">
                        <For each={[1, 2, 3, 4, 5]}>
                          {(star) => (
                            <input
                              type="radio"
                              name="display-rating"
                              class="mask mask-star-2 bg-warning"
                              checked={
                                star === Math.round(Number(averageRating()))
                              }
                              disabled
                            />
                          )}
                        </For>
                      </div>
                      <span class="text-sm font-semibold">
                        {averageRating()}
                      </span>
                      <span class="text-sm opacity-60">
                        ({reviews()?.length || 0} reviews)
                      </span>
                    </div>

                    <div class="stats shadow bg-base-200">
                      <div class="stat py-4">
                        <div class="stat-title text-xs">Price</div>
                        <div class="stat-value text-primary text-4xl">
                          ${p.price?.toFixed(2)}
                        </div>
                      </div>
                      <div class="stat py-4">
                        <div class="stat-title text-xs">Availability</div>
                        <Show
                          when={p.stock > 0}
                          fallback={
                            <div class="badge badge-error gap-2 mt-2">
                              <svg
                                xmlns="http://www.w3.org/2000/svg"
                                fill="none"
                                viewBox="0 0 24 24"
                                class="inline-block w-4 h-4 stroke-current"
                              >
                                <path
                                  stroke-linecap="round"
                                  stroke-linejoin="round"
                                  stroke-width="2"
                                  d="M6 18L18 6M6 6l12 12"
                                ></path>
                              </svg>
                              Out of Stock
                            </div>
                          }
                        >
                          <div class="badge badge-success gap-2 mt-2">
                            <svg
                              xmlns="http://www.w3.org/2000/svg"
                              fill="none"
                              viewBox="0 0 24 24"
                              class="inline-block w-4 h-4 stroke-current"
                            >
                              <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M5 13l4 4L19 7"
                              ></path>
                            </svg>
                            In Stock
                          </div>
                        </Show>
                      </div>
                    </div>
                  </div>

                  {/* Description */}
                  <div class="card bg-base-200 shadow-lg">
                    <div class="card-body">
                      <h3 class="card-title text-lg">Product Description</h3>
                      <p class="opacity-80 leading-relaxed">{p.description}</p>
                    </div>
                  </div>

                  {/* Size Selection */}
                  <Show when={p.sizes && p.sizes.length > 0}>
                    <div class="card bg-base-200 shadow-lg">
                      <div class="card-body">
                        <h3 class="card-title text-lg mb-3">Select Options</h3>

                        {/* Size Dropdown */}
                        <div class="form-control mb-4">
                          <label class="label">
                            <span class="label-text font-semibold">Size</span>
                          </label>
                          <select
                            class="select select-bordered select-lg w-full"
                            value={selectedSize()}
                            onChange={(e) => {
                              setSelectedSize(e.currentTarget.value);
                              setSelectedColor("");
                              setQuantity(1);
                            }}
                          >
                            <option value="" disabled>
                              Choose size
                            </option>
                            <For each={availableSizes()}>
                              {(size) => <option value={size}>{size}</option>}
                            </For>
                          </select>
                        </div>

                        {/* Color Dropdown */}
                        <div class="form-control">
                          <label class="label">
                            <span class="label-text font-semibold">Color</span>
                          </label>
                          <select
                            class="select select-bordered select-lg w-full"
                            value={selectedColor()}
                            onChange={(e) => {
                              setSelectedColor(e.currentTarget.value);
                              setQuantity(1);
                            }}
                            disabled={!selectedSize()}
                          >
                            <option value="" disabled>
                              Choose color
                            </option>
                            <For each={availableColors()}>
                              {(color) => (
                                <option value={color}>{color}</option>
                              )}
                            </For>
                          </select>
                        </div>

                        {/* Stock Info */}
                        <Show when={selectedVariant()}>
                          <div class="mt-4">
                            <Show
                              when={selectedVariant()!.stock > 0}
                              fallback={
                                <div class="alert alert-error">
                                  <span>Out of stock</span>
                                </div>
                              }
                            >
                              <div class="alert alert-success">
                                <span>
                                  {selectedVariant()!.stock} items available
                                </span>
                              </div>
                            </Show>
                          </div>
                        </Show>
                      </div>
                    </div>

                    {/* Quantity Selector */}
                    <Show when={selectedVariant()}>
                      <div class="card bg-base-200 shadow-lg">
                        <div class="card-body">
                          <h3 class="card-title text-lg mb-3">Quantity</h3>
                          <div class="flex items-center gap-4">
                            <div class="join join-horizontal">
                              <button
                                class="btn join-item btn-lg"
                                onclick={() =>
                                  setQuantity(Math.max(1, quantity() - 1))
                                }
                              >
                                <svg
                                  xmlns="http://www.w3.org/2000/svg"
                                  class="h-6 w-6"
                                  fill="none"
                                  viewBox="0 0 24 24"
                                  stroke="currentColor"
                                >
                                  <path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M20 12H4"
                                  />
                                </svg>
                              </button>
                              <div class="btn join-item no-animation btn-lg min-w-20 font-bold text-xl">
                                {quantity()}
                              </div>
                              <button
                                class="btn join-item btn-lg"
                                onclick={() => {
                                  const variant = selectedVariant();
                                  if (variant && quantity() < variant.stock) {
                                    setQuantity(quantity() + 1);
                                  }
                                }}
                                disabled={
                                  !selectedVariant() ||
                                  quantity() >= (selectedVariant()?.stock || 0)
                                }
                              >
                                <svg
                                  xmlns="http://www.w3.org/2000/svg"
                                  class="h-6 w-6"
                                  fill="none"
                                  viewBox="0 0 24 24"
                                  stroke="currentColor"
                                >
                                  <path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M12 4v16m8-8H4"
                                  />
                                </svg>
                              </button>
                            </div>
                            <Show when={selectedVariant()}>
                              <div class="text-sm opacity-60">
                                Max: {selectedVariant()?.stock}
                              </div>
                            </Show>
                          </div>
                        </div>
                      </div>
                    </Show>
                  </Show>

                  {/* Add to Cart Button */}
                  <div class="card bg-linear-to-br from-primary to-secondary text-primary-content shadow-2xl">
                    <div class="card-body items-center text-center p-6">
                      <button
                        class="btn btn-lg btn-block bg-base-100 text-base-content hover:bg-base-200 border-none"
                        disabled={
                          p.stock === 0 || !selectedVariant() || addingToCart()
                        }
                        onclick={handleAddToCart}
                      >
                        <Show
                          when={addingToCart()}
                          fallback={
                            <>
                              <svg
                                xmlns="http://www.w3.org/2000/svg"
                                class="h-6 w-6"
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
                              Add to Cart
                            </>
                          }
                        >
                          <span class="loading loading-spinner"></span>
                          Adding...
                        </Show>
                      </button>
                      <Show when={!selectedVariant() && p.stock > 0}>
                        <p class="text-xs opacity-70 mt-2">
                          Please select size and color
                        </p>
                      </Show>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </Show>

          <div class="divider my-20"></div>

          {/* Reviews Section */}
          <div class="max-w-5xl mx-auto">
            <div class="mb-10">
              <h2 class="text-3xl font-bold mb-2">Customer Reviews</h2>
              <div class="flex items-center gap-3">
                <div class="flex items-center gap-2">
                  <div class="rating rating-md">
                    <For each={[1, 2, 3, 4, 5]}>
                      {(star) => (
                        <input
                          type="radio"
                          name="avg-rating-display"
                          class="mask mask-star-2 bg-warning"
                          checked={star === Math.round(Number(averageRating()))}
                          disabled
                        />
                      )}
                    </For>
                  </div>
                  <span class="text-lg font-semibold">
                    {averageRating()} out of 5
                  </span>
                </div>
                <div class="text-sm opacity-60">
                  {reviews()?.length || 0} global ratings
                </div>
              </div>
            </div>

            {/* Review Form */}
            <Show when={auth.isAuthenticated}>
              <div class="border-2 border-base-300 rounded-lg p-6 mb-10 bg-base-100">
                <h3 class="text-xl font-bold mb-6">Write a customer review</h3>

                <Show when={reviewError()}>
                  <div class="alert alert-error mb-4">
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      class="stroke-current shrink-0 h-6 w-6"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                    <span>{reviewError()}</span>
                  </div>
                </Show>

                <Show when={reviewSuccess()}>
                  <div class="alert alert-success mb-4">
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      class="stroke-current shrink-0 h-6 w-6"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                    <span>Thank you for your review!</span>
                  </div>
                </Show>

                <form onSubmit={handleSubmitReview} class="space-y-5">
                  <div>
                    <label class="block font-bold text-sm mb-2">
                      Overall rating
                    </label>
                    <div class="flex items-center gap-3">
                      <div class="rating rating-lg">
                        <For each={[1, 2, 3, 4, 5]}>
                          {(star) => (
                            <input
                              type="radio"
                              name="user-input-rating"
                              class="mask mask-star-2 bg-warning cursor-pointer hover:scale-110 transition-transform"
                              value={star}
                              checked={rating() === star}
                              onInput={() => setRating(star)}
                            />
                          )}
                        </For>
                      </div>
                      <span class="text-sm font-semibold">
                        {rating() === 5
                          ? "Excellent"
                          : rating() === 4
                            ? "Good"
                            : rating() === 3
                              ? "Average"
                              : rating() === 2
                                ? "Poor"
                                : "Terrible"}
                      </span>
                    </div>
                  </div>

                  <div>
                    <label class="block font-bold text-sm mb-2">
                      Write your review
                    </label>
                    <textarea
                      class="textarea textarea-bordered w-full h-32 text-base resize-none focus:outline-none focus:border-primary"
                      placeholder="What did you like or dislike? What did you use this product for?"
                      value={comment()}
                      onInput={(e) => setComment(e.currentTarget.value)}
                      required
                    ></textarea>
                    <p class="text-xs opacity-60 mt-1">Minimum 10 characters</p>
                  </div>

                  <div class="pt-2">
                    <button
                      type="submit"
                      class="btn btn-warning text-base-100 font-bold px-8"
                      disabled={submittingReview() || comment().length < 10}
                    >
                      <Show when={submittingReview()} fallback="Submit">
                        <span class="loading loading-spinner"></span>
                        Submitting...
                      </Show>
                    </button>
                  </div>
                </form>
              </div>
            </Show>

            {/* Reviews List */}
            <div class="border-t border-base-300 pt-8">
              <h3 class="text-xl font-bold mb-6">Top reviews</h3>
              <Show
                when={!reviews.loading}
                fallback={
                  <div class="flex justify-center py-20">
                    <span class="loading loading-dots loading-lg text-primary"></span>
                  </div>
                }
              >
                <div class="space-y-6">
                  <For
                    each={reviews()}
                    fallback={
                      <div class="text-center py-12 border border-dashed border-base-300 rounded-lg">
                        <p class="opacity-60">
                          No reviews yet. Be the first to review this product!
                        </p>
                      </div>
                    }
                  >
                    {(review) => (
                      <div class="border-b border-base-300 pb-6 last:border-0">
                        <div class="flex items-center gap-3 mb-2">
                          <div class="avatar avatar-placeholder">
                            <div class="bg-neutral text-neutral-content rounded-full w-10">
                              <span class="text-sm">
                                {review.user_name.charAt(0).toUpperCase()}
                              </span>
                            </div>
                          </div>
                          <div>
                            <div class="font-semibold">{review.user_name}</div>
                          </div>
                        </div>
                        <div class="flex items-center gap-2 mb-2 ml-13">
                          <div class="rating rating-sm">
                            <For each={[1, 2, 3, 4, 5]}>
                              {(i) => (
                                <input
                                  type="radio"
                                  class="mask mask-star-2 bg-warning"
                                  checked={i <= review.rating}
                                  disabled
                                />
                              )}
                            </For>
                          </div>
                          <span class="text-xs opacity-60">
                            Reviewed on{" "}
                            {new Date(review.created_at).toLocaleDateString(
                              undefined,
                              {
                                year: "numeric",
                                month: "long",
                                day: "numeric",
                              },
                            )}
                          </span>
                        </div>
                        <p class="ml-13 leading-relaxed">{review.comment}</p>
                      </div>
                    )}
                  </For>
                </div>
              </Show>
            </div>
          </div>
        </Show>
      </main>

      <Show when={feedbackMessage()}>
        <dialog class="modal modal-open">
          <div class="modal-box">
            <h3 class="text-lg font-bold">{feedbackTitle()}</h3>
            <p class="py-3">{feedbackMessage()}</p>
            <div class="modal-action">
              <button
                class="btn btn-primary"
                onClick={() => setFeedbackMessage(null)}
              >
                OK
              </button>
            </div>
          </div>
        </dialog>
      </Show>
    </div>
  );
}
