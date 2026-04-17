interface PaginationProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  totalItems?: number;
  itemsPerPage?: number;
}

export default function Pagination(props: PaginationProps) {
  const handlePageChange = (page: number) => {
    if (page >= 1 && page <= props.totalPages) {
      props.onPageChange(page);
    }
  };

  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    const maxVisible = 7;

    if (props.totalPages <= maxVisible) {
      for (let i = 1; i <= props.totalPages; i++) {
        pages.push(i);
      }
    } else {
      if (props.currentPage <= 4) {
        for (let i = 1; i <= 5; i++) pages.push(i);
        pages.push('...');
        pages.push(props.totalPages);
      } else if (props.currentPage >= props.totalPages - 3) {
        pages.push(1);
        pages.push('...');
        for (let i = props.totalPages - 4; i <= props.totalPages; i++) pages.push(i);
      } else {
        pages.push(1);
        pages.push('...');
        for (let i = props.currentPage - 1; i <= props.currentPage + 1; i++) pages.push(i);
        pages.push('...');
        pages.push(props.totalPages);
      }
    }

    return pages;
  };

  return (
    <div class="flex flex-col sm:flex-row items-center justify-between gap-4 mt-6">
      <div class="text-sm text-base-content/60">
        {props.totalItems && props.itemsPerPage && (
          <span>
            Showing {(props.currentPage - 1) * props.itemsPerPage + 1} to{' '}
            {Math.min(props.currentPage * props.itemsPerPage, props.totalItems)} of{' '}
            {props.totalItems} results
          </span>
        )}
      </div>

      <div class="join">
        <button
          class="join-item btn btn-sm"
          onClick={() => handlePageChange(props.currentPage - 1)}
          disabled={props.currentPage === 1}
        >
          «
        </button>

        {getPageNumbers().map((page) => (
          <button
            class={`join-item btn btn-sm ${
              page === props.currentPage ? 'btn-active' : ''
            } ${page === '...' ? 'btn-disabled' : ''}`}
            onClick={() => typeof page === 'number' && handlePageChange(page)}
            disabled={page === '...'}
          >
            {page}
          </button>
        ))}

        <button
          class="join-item btn btn-sm"
          onClick={() => handlePageChange(props.currentPage + 1)}
          disabled={props.currentPage === props.totalPages}
        >
          »
        </button>
      </div>
    </div>
  );
}
