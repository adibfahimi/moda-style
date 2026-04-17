import { createSignal, createResource, Show, Index } from "solid-js";
import {
  getProductStats,
  createProduct,
  updateProduct,
  deleteProduct,
  uploadProductImage,
  listProductImages,
  createCategory,
  addProductSize,
  getProductSizes,
  updateProductSize,
  deleteProductSize,
} from "../../services/adminService";
import { listCategories } from "../../services/productService";
import type { ProductStats, Size } from "../../types";
import DataTable from "../../components/admin/DataTable";
import Pagination from "../../components/admin/Pagination";
import Modal, { useModal } from "../../components/admin/Modal";

export default function AdminProducts() {
  type ProductVariant = {
    id?: number;
    size: string;
    color: string;
    stock: number;
  };

  const [page, setPage] = createSignal(1);
  const [limit] = createSignal(20);
  const [selectedProduct, setSelectedProduct] =
    createSignal<ProductStats | null>(null);
  const [formData, setFormData] = createSignal({
    name: "",
    description: "",
    price: 0,
    category_id: 0,
    image_url: "",
  });
  const [priceInput, setPriceInput] = createSignal("0");
  const [isEditMode, setIsEditMode] = createSignal(false);
  const [isUploadingImage, setIsUploadingImage] = createSignal(false);
  const [isCreatingCategory, setIsCreatingCategory] = createSignal(false);
  const [newCategoryName, setNewCategoryName] = createSignal("");
  const [newCategoryParentId, setNewCategoryParentId] = createSignal<
    number | ""
  >("");
  const [imageLibraryRefreshKey, setImageLibraryRefreshKey] = createSignal(0);
  const [mediaTab, setMediaTab] = createSignal<"upload" | "library">("library");
  const [variants, setVariants] = createSignal<ProductVariant[]>([
    { size: "M", color: "Default", stock: 0 },
  ]);
  const [removedVariantIds, setRemovedVariantIds] = createSignal<number[]>([]);
  const [isSavingVariants, setIsSavingVariants] = createSignal(false);
  const [error, setError] = createSignal("");
  const [success, setSuccess] = createSignal("");

  const productModal = useModal();
  const deleteModal = useModal();

  const [products, { refetch }] = createResource(
    () => ({ page: page(), limit: limit() }),
    getProductStats,
  );

  const [categories, { refetch: refetchCategories }] =
    createResource(listCategories);

  const [uploadedImages] = createResource(
    () => imageLibraryRefreshKey(),
    listProductImages,
  );

  const slugifyCategoryName = (value: string) =>
    value
      .toLowerCase()
      .trim()
      .replace(/&/g, "and")
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "");

  const normalizePriceValue = (value: string) =>
    value.replace(/\s+/g, "").replace(",", ".");

  const getCategoryLabel = (category: any) => {
    const parentName = categories()?.find(
      (c: any) => c.id === category.parent_id,
    )?.name;
    return parentName ? `${parentName} > ${category.name}` : category.name;
  };

  const getImageNameFromUrl = (url: string) => {
    if (!url) return "No image selected";
    try {
      const cleanUrl = url.split("?")[0];
      return decodeURIComponent(
        cleanUrl.substring(cleanUrl.lastIndexOf("/") + 1),
      );
    } catch {
      return "Selected image";
    }
  };

  const handleCreate = () => {
    setIsEditMode(false);
    setNewCategoryName("");
    setNewCategoryParentId("");
    setPriceInput("0");
    setMediaTab("library");
    setImageLibraryRefreshKey((k) => k + 1);
    setFormData({
      name: "",
      description: "",
      price: 0,
      category_id: 0,
      image_url: "",
    });
    setVariants([{ size: "M", color: "Default", stock: 0 }]);
    setRemovedVariantIds([]);
    productModal.open();
  };

  const handleEdit = async (product: ProductStats) => {
    setIsEditMode(true);
    setNewCategoryName("");
    setNewCategoryParentId("");
    setPriceInput(String(product.price));
    setMediaTab("library");
    setImageLibraryRefreshKey((k) => k + 1);
    setSelectedProduct(product);
    setFormData({
      name: product.name,
      description: product.description || "",
      price: product.price,
      category_id: product.category_id,
      image_url: product.image_url || "",
    });

    setRemovedVariantIds([]);
    try {
      const response = await getProductSizes(product.id);
      const existingSizes: Size[] = response?.sizes || [];
      if (existingSizes.length > 0) {
        setVariants(
          existingSizes.map((size) => ({
            id: size.id,
            size: size.size,
            color: size.color,
            stock: size.stock,
          })),
        );
      } else {
        setVariants([{ size: "M", color: "Default", stock: 0 }]);
      }
    } catch {
      setVariants([{ size: "M", color: "Default", stock: 0 }]);
    }

    productModal.open();
  };

  const handleDelete = (product: ProductStats) => {
    setSelectedProduct(product);
    deleteModal.open();
  };

  const handleCreateCategory = async () => {
    const name = newCategoryName().trim();
    if (name.length < 2) {
      setError("Category name must be at least 2 characters");
      return;
    }

    setError("");
    setSuccess("");

    try {
      setIsCreatingCategory(true);
      const created = await createCategory({
        name,
        slug: slugifyCategoryName(name),
        parent_id:
          newCategoryParentId() === ""
            ? undefined
            : Number(newCategoryParentId()),
      });

      const createdCategory = created?.category || created?.data?.category;

      await refetchCategories();

      if (createdCategory?.id) {
        setFormData({ ...formData(), category_id: createdCategory.id });
      }

      setNewCategoryName("");
      setNewCategoryParentId("");
      setSuccess("Category created and selected");
    } catch (err: any) {
      setError(err.message || "Failed to create category");
    } finally {
      setIsCreatingCategory(false);
    }
  };

  const handleImageUpload = async (e: Event) => {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    setError("");
    setSuccess("");

    try {
      setIsUploadingImage(true);
      const response = await uploadProductImage(file);
      const imageUrl = response?.data?.image_url || response?.image_url || "";

      if (!imageUrl) {
        throw new Error("Upload succeeded but image URL is missing");
      }

      setFormData({ ...formData(), image_url: imageUrl });
      setMediaTab("library");
      setImageLibraryRefreshKey((k) => k + 1);
      setSuccess("Image uploaded successfully");
    } catch (err: any) {
      setError(err.message || "Failed to upload image");
    } finally {
      setIsUploadingImage(false);
      input.value = "";
    }
  };

  const addVariantRow = () => {
    setVariants([...variants(), { size: "", color: "", stock: 0 }]);
  };

  const updateVariantField = (
    index: number,
    field: keyof ProductVariant,
    value: string | number,
  ) => {
    setVariants((current) => {
      const updated = [...current];
      updated[index] = {
        ...updated[index],
        [field]: field === "stock" ? Number(value) || 0 : value,
      };
      return updated;
    });
  };

  const removeVariantRow = (index: number) => {
    const current = variants();
    const row = current[index];
    if (row?.id) {
      setRemovedVariantIds((ids) =>
        ids.includes(row.id!) ? ids : [...ids, row.id!],
      );
    }

    const next = current.filter((_, i) => i !== index);
    setVariants(
      next.length > 0 ? next : [{ size: "M", color: "Default", stock: 0 }],
    );
  };

  const isVariantValid = (variant: ProductVariant) =>
    variant.size.trim().length > 0 &&
    variant.color.trim().length > 0 &&
    Number.isFinite(variant.stock) &&
    variant.stock >= 0;

  const submitForm = async (e: Event) => {
    e.preventDefault();
    setError("");
    setSuccess("");

    if (!formData().image_url) {
      setError("Product image is required. Please upload an image.");
      return;
    }

    if (
      variants().length === 0 ||
      variants().some((variant) => !isVariantValid(variant))
    ) {
      setError("Please provide valid size, color, and stock for all variants.");
      return;
    }

    try {
      setIsSavingVariants(true);

      if (isEditMode()) {
        const productId = selectedProduct()!.id;
        await updateProduct(productId, formData());

        for (const variantId of removedVariantIds()) {
          await deleteProductSize(productId, variantId);
        }

        for (const variant of variants()) {
          const payload = {
            size: variant.size.trim(),
            color: variant.color.trim(),
            stock: variant.stock,
          };

          if (variant.id) {
            await updateProductSize(productId, variant.id, payload);
          } else {
            await addProductSize(productId, payload);
          }
        }

        setSuccess("Product updated successfully");
      } else {
        const created = await createProduct(formData());
        const createdProduct = created?.product || created?.data?.product;
        const productId = createdProduct?.id;

        if (!productId) {
          throw new Error(
            "Product was created but no product ID was returned.",
          );
        }

        for (const variant of variants()) {
          await addProductSize(productId, {
            size: variant.size.trim(),
            color: variant.color.trim(),
            stock: variant.stock,
          });
        }

        setSuccess("Product created successfully");
      }
      setTimeout(() => {
        productModal.close();
        refetch();
        setSuccess("");
      }, 1500);
    } catch (err: any) {
      setError(err.message || "Operation failed");
    } finally {
      setIsSavingVariants(false);
    }
  };

  const confirmDelete = async () => {
    setError("");
    try {
      await deleteProduct(selectedProduct()!.id);
      setSuccess("Product deleted successfully");
      setTimeout(() => {
        deleteModal.close();
        refetch();
        setSuccess("");
      }, 1500);
    } catch (err: any) {
      setError(err.message || "Failed to delete product");
    }
  };

  const columns = [
    { key: "id", label: "ID" },
    { key: "name", label: "Product Name" },
    { key: "category_name", label: "Category" },
    {
      key: "price",
      label: "Price",
      render: (p: ProductStats) => `$${p.price.toFixed(2)}`,
    },
    {
      key: "stock",
      label: "Stock",
      render: (p: ProductStats) => (
        <span
          class={`badge ${p.stock < 10 ? "badge-warning" : "badge-success"}`}
        >
          {p.stock}
        </span>
      ),
    },
    {
      key: "average_rating",
      label: "Rating",
      render: (p: ProductStats) =>
        `⭐ ${p.average_rating.toFixed(1)} (${p.review_count})`,
    },
    { key: "wishlist_count", label: "Wishlisted" },
    {
      key: "actions",
      label: "Actions",
      render: (product: ProductStats) => (
        <div class="flex gap-2">
          <button
            class="btn btn-sm btn-primary"
            onClick={() => handleEdit(product)}
          >
            Edit
          </button>
          <button
            class="btn btn-sm btn-error"
            onClick={() => handleDelete(product)}
          >
            Delete
          </button>
        </div>
      ),
    },
  ];

  return (
    <div class="p-6">
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold">Product Management</h1>
        <button class="btn btn-primary" onClick={handleCreate}>
          + Add Product
        </button>
      </div>

      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <DataTable
            data={products()?.products || []}
            columns={columns}
            loading={products.loading}
            emptyMessage="No products found"
          />

          <Show when={products() && products()!.total > limit()}>
            <Pagination
              currentPage={page()}
              totalPages={Math.ceil(products()!.total / limit())}
              totalItems={products()!.total}
              itemsPerPage={limit()}
              onPageChange={setPage}
            />
          </Show>
        </div>
      </div>

      {/* Product Form Modal */}
      <Modal
        title={isEditMode() ? "Edit Product" : "Create Product"}
        isOpen={productModal.isOpen()}
        onClose={productModal.close}
        size="lg"
      >
        <form onSubmit={submitForm} class="space-y-5">
          <Show when={error()}>
            <div class="alert alert-error">
              <span>{error()}</span>
            </div>
          </Show>

          <Show when={success()}>
            <div class="alert alert-success">
              <span>{success()}</span>
            </div>
          </Show>

          <div class="grid grid-cols-1 gap-5 xl:grid-cols-5">
            <div class="space-y-5 xl:col-span-3">
              <div class="card border border-base-300 bg-base-100">
                <div class="card-body gap-4">
                  <h4 class="card-title text-base">Product Details</h4>

                  <label class="form-control w-full">
                    <span class="mb-2 block text-sm font-semibold">
                      Product Name <span class="text-error">*</span>
                    </span>
                    <input
                      type="text"
                      placeholder="Enter product name"
                      class="input input-bordered w-full"
                      value={formData().name}
                      onInput={(e) =>
                        setFormData({
                          ...formData(),
                          name: e.currentTarget.value,
                        })
                      }
                      required
                    />
                  </label>

                  <label class="form-control w-full">
                    <span class="mb-2 block text-sm font-semibold">
                      Description
                    </span>
                    <textarea
                      placeholder="Enter product description"
                      class="textarea textarea-bordered h-28 resize-none"
                      value={formData().description}
                      onInput={(e) =>
                        setFormData({
                          ...formData(),
                          description: e.currentTarget.value,
                        })
                      }
                    />
                  </label>

                  <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <label class="form-control w-full">
                      <span class="mb-2 block text-sm font-semibold">
                        Price <span class="text-error">*</span>
                      </span>
                      <div class="join w-full">
                        <span class="join-item inline-flex items-center border border-base-300 bg-base-200 px-3 text-sm text-base-content/70">
                          $
                        </span>
                        <input
                          type="text"
                          inputMode="decimal"
                          pattern="^[0-9]+([.,][0-9]{1,2})?$"
                          placeholder="0.00"
                          class="input input-bordered join-item w-full"
                          value={priceInput()}
                          onInput={(e) => {
                            const rawValue = e.currentTarget.value;
                            setPriceInput(rawValue);
                            const normalizedValue =
                              normalizePriceValue(rawValue);
                            const parsedPrice = parseFloat(normalizedValue);
                            setFormData({
                              ...formData(),
                              price: Number.isFinite(parsedPrice)
                                ? parsedPrice
                                : 0,
                            });
                          }}
                          onBlur={(e) => {
                            const normalizedValue = normalizePriceValue(
                              e.currentTarget.value,
                            );
                            const parsedPrice = parseFloat(normalizedValue);
                            if (Number.isFinite(parsedPrice)) {
                              setPriceInput(parsedPrice.toFixed(2));
                            }
                          }}
                          required
                        />
                      </div>
                    </label>

                    <label class="form-control w-full">
                      <span class="mb-2 block text-sm font-semibold">
                        Category <span class="text-error">*</span>
                      </span>
                      <select
                        class="select select-bordered w-full"
                        value={formData().category_id}
                        onChange={(e) =>
                          setFormData({
                            ...formData(),
                            category_id: parseInt(e.currentTarget.value),
                          })
                        }
                        required
                      >
                        <option value={0} disabled>
                          Select a category
                        </option>
                        <Show when={categories()}>
                          {categories()!.map((cat: any) => (
                            <option value={cat.id}>
                              {getCategoryLabel(cat)}
                            </option>
                          ))}
                        </Show>
                      </select>

                      <div class="mt-3">
                        <span class="mb-2 block text-sm font-semibold">
                          Add New Category
                        </span>
                        <select
                          class="select select-bordered mb-2 w-full"
                          value={newCategoryParentId()}
                          onChange={(e) =>
                            setNewCategoryParentId(
                              e.currentTarget.value
                                ? Number(e.currentTarget.value)
                                : "",
                            )
                          }
                        >
                          <option value="">No parent (top level)</option>
                          <Show when={categories()}>
                            {categories()!.map((cat: any) => (
                              <option value={cat.id}>
                                {getCategoryLabel(cat)}
                              </option>
                            ))}
                          </Show>
                        </select>
                        <div class="join w-full">
                          <input
                            type="text"
                            placeholder="e.g. Outerwear"
                            class="input input-bordered join-item w-full"
                            value={newCategoryName()}
                            onInput={(e) =>
                              setNewCategoryName(e.currentTarget.value)
                            }
                            onKeyDown={(e) => {
                              if (e.key === "Enter") {
                                e.preventDefault();
                                handleCreateCategory();
                              }
                            }}
                          />
                          <button
                            type="button"
                            class="btn btn-outline join-item"
                            onClick={handleCreateCategory}
                            disabled={
                              isCreatingCategory() ||
                              newCategoryName().trim().length < 2
                            }
                          >
                            {isCreatingCategory() ? "Adding..." : "Add"}
                          </button>
                        </div>
                      </div>
                    </label>
                  </div>

                  <div class="rounded-box border border-base-300 bg-base-200 p-4">
                    <div class="mb-3 flex items-center justify-between">
                      <h5 class="text-sm font-semibold">Stock Variants</h5>
                      <button
                        type="button"
                        class="btn btn-xs btn-outline"
                        onClick={addVariantRow}
                      >
                        + Add Variant
                      </button>
                    </div>

                    <div class="space-y-3">
                      <Show
                        when={variants().length > 0}
                        fallback={
                          <div class="text-sm text-base-content/70">
                            No variants yet.
                          </div>
                        }
                      >
                        <Index each={variants()}>
                          {(variant, index) => (
                            <div class="grid grid-cols-1 gap-2 rounded-box border border-base-300 bg-base-100 p-3 md:grid-cols-12">
                              <div class="md:col-span-3">
                                <span class="mb-1 block text-xs font-semibold uppercase text-base-content/70">
                                  Size
                                </span>
                                <input
                                  type="text"
                                  class="input input-bordered input-sm w-full"
                                  value={variant().size}
                                  onInput={(e) =>
                                    updateVariantField(
                                      index(),
                                      "size",
                                      e.currentTarget.value,
                                    )
                                  }
                                  placeholder="M"
                                />
                              </div>
                              <div class="md:col-span-4">
                                <span class="mb-1 block text-xs font-semibold uppercase text-base-content/70">
                                  Color
                                </span>
                                <input
                                  type="text"
                                  class="input input-bordered input-sm w-full"
                                  value={variant().color}
                                  onInput={(e) =>
                                    updateVariantField(
                                      index(),
                                      "color",
                                      e.currentTarget.value,
                                    )
                                  }
                                  placeholder="Black"
                                />
                              </div>
                              <div class="md:col-span-3">
                                <span class="mb-1 block text-xs font-semibold uppercase text-base-content/70">
                                  Stock
                                </span>
                                <input
                                  type="number"
                                  min="0"
                                  class="input input-bordered input-sm w-full"
                                  value={variant().stock}
                                  onInput={(e) =>
                                    updateVariantField(
                                      index(),
                                      "stock",
                                      Number(e.currentTarget.value),
                                    )
                                  }
                                />
                              </div>
                              <div class="flex items-end md:col-span-2">
                                <button
                                  type="button"
                                  class="btn btn-error btn-sm w-full"
                                  onClick={() => removeVariantRow(index())}
                                  disabled={variants().length === 1}
                                >
                                  Remove
                                </button>
                              </div>
                            </div>
                          )}
                        </Index>
                      </Show>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="space-y-5 xl:col-span-2">
              <div class="card border border-base-300 bg-base-100">
                <div class="card-body gap-4">
                  <div class="flex items-start justify-between gap-3">
                    <h4 class="card-title text-base">Product Image</h4>
                    <span class="badge badge-outline max-w-45 truncate">
                      {getImageNameFromUrl(formData().image_url)}
                    </span>
                  </div>

                  <div class="tabs tabs-box w-full bg-base-200 p-1">
                    <button
                      type="button"
                      class={`tab flex-1 ${mediaTab() === "upload" ? "tab-active" : ""}`}
                      onClick={() => setMediaTab("upload")}
                    >
                      Upload Files
                    </button>
                    <button
                      type="button"
                      class={`tab flex-1 ${mediaTab() === "library" ? "tab-active" : ""}`}
                      onClick={() => setMediaTab("library")}
                    >
                      Media Library
                    </button>
                  </div>

                  <Show when={mediaTab() === "upload"}>
                    <div class="rounded-box border border-base-300 bg-base-100 p-4">
                      <label class="form-control w-full">
                        <span class="mb-2 block text-sm font-semibold">
                          Upload image
                        </span>
                        <input
                          type="file"
                          accept="image/png,image/jpeg,image/webp"
                          class="file-input file-input-bordered w-full"
                          onChange={handleImageUpload}
                          disabled={isUploadingImage()}
                        />
                        <span class="mt-2 text-xs text-base-content/70">
                          {isUploadingImage()
                            ? "Uploading image..."
                            : "Select JPG, PNG, or WEBP up to 5MB"}
                        </span>
                      </label>
                    </div>
                  </Show>

                  <Show when={mediaTab() === "library"}>
                    <div class="rounded-box border border-base-300 bg-base-100 p-3">
                      <div class="mb-3 flex items-center justify-between">
                        <p class="text-sm font-semibold">
                          Choose Existing Image
                        </p>
                        <button
                          type="button"
                          class="btn btn-xs btn-ghost"
                          onClick={() =>
                            setImageLibraryRefreshKey((k) => k + 1)
                          }
                        >
                          Refresh
                        </button>
                      </div>

                      <Show
                        when={!uploadedImages.loading}
                        fallback={
                          <div class="flex h-40 items-center justify-center rounded-box border border-dashed border-base-300 text-sm text-base-content/70">
                            Loading uploaded images...
                          </div>
                        }
                      >
                        <Show
                          when={(uploadedImages()?.images || []).length > 0}
                          fallback={
                            <div class="flex h-40 items-center justify-center rounded-box border border-dashed border-base-300 text-sm text-base-content/70">
                              No uploaded images yet
                            </div>
                          }
                        >
                          <div class="grid max-h-56 grid-cols-3 gap-2 overflow-y-auto pr-1">
                            {(uploadedImages()?.images || []).map(
                              (image: any) => (
                                <button
                                  type="button"
                                  class={`rounded-box border p-1 text-left transition ${
                                    formData().image_url === image.url
                                      ? "border-primary bg-primary/5 ring-2 ring-primary/20"
                                      : "border-base-300 hover:border-primary/50"
                                  }`}
                                  onClick={() =>
                                    setFormData({
                                      ...formData(),
                                      image_url: image.url,
                                    })
                                  }
                                  title={image.name}
                                >
                                  <img
                                    src={image.url}
                                    alt={image.name}
                                    class="h-20 w-full rounded object-cover"
                                  />
                                </button>
                              ),
                            )}
                          </div>
                        </Show>
                      </Show>
                    </div>
                  </Show>

                  <div class="rounded-box border border-base-300 bg-base-200 p-3">
                    <div class="mb-2 flex items-center justify-between gap-2">
                      <p class="text-sm font-semibold">Preview</p>
                      <Show when={formData().image_url}>
                        <button
                          type="button"
                          class="btn btn-xs btn-ghost"
                          onClick={() =>
                            setFormData({ ...formData(), image_url: "" })
                          }
                        >
                          Remove
                        </button>
                      </Show>
                    </div>
                    <Show
                      when={formData().image_url}
                      fallback={
                        <div class="flex h-40 items-center justify-center rounded-box border border-dashed border-base-300 bg-base-100 text-sm text-base-content/60">
                          No image selected
                        </div>
                      }
                    >
                      <img
                        src={formData().image_url}
                        alt="Product preview"
                        class="h-44 w-full rounded-box bg-base-100 object-contain"
                        onError={(e) => {
                          e.currentTarget.src =
                            "https://via.placeholder.com/400x240?text=Invalid+Image";
                        }}
                      />
                    </Show>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-action mt-0 border-t border-base-300 pt-4">
            <button
              type="button"
              class="btn btn-ghost"
              onClick={productModal.close}
            >
              Cancel
            </button>
            <button
              type="submit"
              class="btn btn-primary"
              disabled={isUploadingImage() || isSavingVariants()}
            >
              <Show
                when={isEditMode()}
                fallback={
                  <>
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      class="h-5 w-5"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fill-rule="evenodd"
                        d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                        clip-rule="evenodd"
                      />
                    </svg>
                    Create Product
                  </>
                }
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-5 w-5"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                </svg>
                Update Product
              </Show>
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete Modal */}
      <Modal
        title="Delete Product"
        isOpen={deleteModal.isOpen()}
        onClose={deleteModal.close}
        size="sm"
      >
        <Show when={error()}>
          <div class="alert alert-error shadow-lg mb-4">
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
            <span>{error()}</span>
          </div>
        </Show>
        <Show when={success()}>
          <div class="alert alert-success shadow-lg mb-4">
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
            <span>{success()}</span>
          </div>
        </Show>

        <div class="py-4">
          <div class="flex items-center gap-3 mb-4">
            <div class="text-error">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-12 w-12"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <div>
              <h4 class="font-bold text-lg">Confirm Deletion</h4>
              <p class="text-sm opacity-70">This action cannot be undone</p>
            </div>
          </div>

          <p class="mb-4">
            Are you sure you want to delete the product{" "}
            <strong class="text-error">{selectedProduct()?.name}</strong>?
          </p>
        </div>

        <div class="divider"></div>
        <div class="flex justify-end gap-3">
          <button class="btn btn-ghost" onClick={deleteModal.close}>
            Cancel
          </button>
          <button class="btn btn-error" onClick={confirmDelete}>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fill-rule="evenodd"
                d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
                clip-rule="evenodd"
              />
            </svg>
            Delete Product
          </button>
        </div>
      </Modal>
    </div>
  );
}
