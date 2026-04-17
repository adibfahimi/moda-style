import { For, Show } from 'solid-js';

interface Column<T> {
  key: keyof T | string;
  label: string;
  render?: (item: T) => any;
}

interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  loading?: boolean;
  emptyMessage?: string;
  onRowClick?: (item: T) => void;
}

export default function DataTable<T extends { id: number }>(props: DataTableProps<T>) {
  const getValue = (item: T, key: keyof T | string): any => {
    if (typeof key === 'string' && key.includes('.')) {
      const keys = key.split('.');
      let value: any = item;
      for (const k of keys) {
        value = value?.[k];
      }
      return value;
    }
    return item[key as keyof T];
  };

  return (
    <div class="overflow-x-auto">
      <table class="table table-zebra">
        <thead>
          <tr>
            <For each={props.columns}>
              {(column) => <th>{column.label}</th>}
            </For>
          </tr>
        </thead>
        <tbody>
          <Show
            when={!props.loading && props.data.length > 0}
            fallback={
              <tr>
                <td colSpan={props.columns.length} class="text-center py-8">
                  <Show
                    when={!props.loading}
                    fallback={
                      <span class="loading loading-spinner loading-lg"></span>
                    }
                  >
                    <p class="text-base-content/60">{props.emptyMessage || 'No data available'}</p>
                  </Show>
                </td>
              </tr>
            }
          >
            <For each={props.data}>
              {(item) => (
                <tr
                  class={props.onRowClick ? 'cursor-pointer hover:bg-base-200' : ''}
                  onClick={() => props.onRowClick?.(item)}
                >
                  <For each={props.columns}>
                    {(column) => (
                      <td>
                        {column.render
                          ? column.render(item)
                          : getValue(item, column.key)}
                      </td>
                    )}
                  </For>
                </tr>
              )}
            </For>
          </Show>
        </tbody>
      </table>
    </div>
  );
}
