import { createSignal, createResource, Show, For } from "solid-js";
import {
  getCategories,
  createCategory,
  updateCategory,
  deleteCategory,
} from "../../services/adminService";
import type { CategoryWithCount } from "../../types";
import DataTable from "../../components/admin/DataTable";
import Modal, { useModal } from "../../components/admin/Modal";

export default function AdminCategories() {
  const [selectedCategory, setSelectedCategory] =
    createSignal<CategoryWithCount | null>(null);
  const [formData, setFormData] = createSignal({
    name: "",
    slug: "",
    parent_id: "" as number | "",
  });
  const [isEditMode, setIsEditMode] = createSignal(false);
  const [error, setError] = createSignal("");
  const [success, setSuccess] = createSignal("");

  const categoryModal = useModal();
  const deleteModal = useModal();

  const [categories, { refetch }] = createResource(getCategories);

  const getCategoryLabel = (category: CategoryWithCount) => {
    return category.parent_name
      ? `${category.parent_name} > ${category.name}`
      : category.name;
  };

  const getParentOptions = () => {
    const currentId = selectedCategory()?.id;
    return (categories()?.categories || []).filter(
      (cat: CategoryWithCount) => cat.id !== currentId,
    );
  };

  const handleCreate = () => {
    setIsEditMode(false);
    setSelectedCategory(null);
    setFormData({ name: "", slug: "", parent_id: "" });
    categoryModal.open();
  };

  const handleEdit = (category: CategoryWithCount) => {
    setIsEditMode(true);
    setSelectedCategory(category);
    setFormData({
      name: category.name,
      slug: category.slug,
      parent_id: category.parent_id || "",
    });
    categoryModal.open();
  };

  const handleDelete = (category: CategoryWithCount) => {
    setSelectedCategory(category);
    deleteModal.open();
  };

  const submitForm = async (e: Event) => {
    e.preventDefault();
    setError("");
    setSuccess("");

    try {
      if (isEditMode()) {
        await updateCategory(selectedCategory()!.id, {
          name: formData().name,
          slug: formData().slug,
          parent_id:
            formData().parent_id === ""
              ? undefined
              : Number(formData().parent_id),
        });
        setSuccess("Category updated successfully");
      } else {
        await createCategory({
          name: formData().name,
          slug: formData().slug,
          parent_id:
            formData().parent_id === ""
              ? undefined
              : Number(formData().parent_id),
        });
        setSuccess("Category created successfully");
      }
      setTimeout(() => {
        categoryModal.close();
        refetch();
        setSuccess("");
      }, 1500);
    } catch (err: any) {
      setError(err.message || "Operation failed");
    }
  };

  const confirmDelete = async () => {
    setError("");
    try {
      await deleteCategory(selectedCategory()!.id);
      setSuccess("Category deleted successfully");
      setTimeout(() => {
        deleteModal.close();
        refetch();
        setSuccess("");
      }, 1500);
    } catch (err: any) {
      setError(err.message || "Failed to delete category");
    }
  };

  const columns = [
    { key: "id", label: "ID" },
    { key: "name", label: "Category Name" },
    {
      key: "parent_name",
      label: "Parent",
      render: (cat: CategoryWithCount) => cat.parent_name || "-",
    },
    { key: "slug", label: "Slug" },
    {
      key: "product_count",
      label: "Products",
      render: (cat: CategoryWithCount) => (
        <span class="badge badge-neutral">{cat.product_count}</span>
      ),
    },
    {
      key: "created_at",
      label: "Created",
      render: (cat: CategoryWithCount) =>
        new Date(cat.created_at).toLocaleDateString(),
    },
    {
      key: "actions",
      label: "Actions",
      render: (category: CategoryWithCount) => (
        <div class="flex gap-2">
          <button
            class="btn btn-sm btn-primary"
            onClick={() => handleEdit(category)}
          >
            Edit
          </button>
          <button
            class="btn btn-sm btn-error"
            onClick={() => handleDelete(category)}
            disabled={category.product_count > 0}
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
        <h1 class="text-3xl font-bold">Category Management</h1>
        <button class="btn btn-primary" onClick={handleCreate}>
          + Add Category
        </button>
      </div>

      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <DataTable
            data={categories()?.categories || []}
            columns={columns}
            loading={categories.loading}
            emptyMessage="No categories found"
          />
        </div>
      </div>

      {/* Category Form Modal */}
      <Modal
        title={isEditMode() ? "Edit Category" : "Create Category"}
        isOpen={categoryModal.isOpen()}
        onClose={categoryModal.close}
        size="sm"
      >
        <form onSubmit={submitForm} class="space-y-4">
          <Show when={error()}>
            <div class="alert alert-error">{error()}</div>
          </Show>
          <Show when={success()}>
            <div class="alert alert-success">{success()}</div>
          </Show>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Category Name</span>
            </label>
            <input
              type="text"
              class="input input-bordered"
              value={formData().name}
              onInput={(e) =>
                setFormData({ ...formData(), name: e.currentTarget.value })
              }
              required
            />
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Parent Category (optional)</span>
            </label>
            <select
              class="select select-bordered"
              value={formData().parent_id}
              onChange={(e) =>
                setFormData({
                  ...formData(),
                  parent_id: e.currentTarget.value
                    ? Number(e.currentTarget.value)
                    : "",
                })
              }
            >
              <option value="">No parent (top level)</option>
              <For each={getParentOptions()}>
                {(category) => (
                  <option value={category.id}>
                    {getCategoryLabel(category)}
                  </option>
                )}
              </For>
            </select>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Slug (optional)</span>
            </label>
            <input
              type="text"
              class="input input-bordered"
              value={formData().slug}
              onInput={(e) =>
                setFormData({ ...formData(), slug: e.currentTarget.value })
              }
              placeholder="Auto-generated if empty"
            />
            <label class="label">
              <span class="label-text-alt">
                URL-friendly version of the name
              </span>
            </label>
          </div>

          <div class="flex justify-end gap-2 mt-6">
            <button
              type="button"
              class="btn btn-ghost"
              onClick={categoryModal.close}
            >
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              {isEditMode() ? "Update" : "Create"} Category
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete Modal */}
      <Modal
        title="Delete Category"
        isOpen={deleteModal.isOpen()}
        onClose={deleteModal.close}
        size="sm"
      >
        <Show when={error()}>
          <div class="alert alert-error mb-4">{error()}</div>
        </Show>
        <Show when={success()}>
          <div class="alert alert-success mb-4">{success()}</div>
        </Show>

        <p class="mb-6">
          Are you sure you want to delete category{" "}
          <strong>{selectedCategory()?.name}</strong>?
        </p>
        <Show when={selectedCategory()?.product_count! > 0}>
          <div class="alert alert-warning mb-4">
            This category has {selectedCategory()?.product_count} products and
            cannot be deleted.
          </div>
        </Show>

        <div class="flex justify-end gap-2">
          <button class="btn btn-ghost" onClick={deleteModal.close}>
            Cancel
          </button>
          <button
            class="btn btn-error"
            onClick={confirmDelete}
            disabled={selectedCategory()?.product_count! > 0}
          >
            Delete
          </button>
        </div>
      </Modal>
    </div>
  );
}
