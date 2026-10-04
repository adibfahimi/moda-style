import { cleanup, fireEvent, render, screen } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';
import Pagination from './Pagination';

/** Renders the control and returns the recorded page changes. */
const renderPagination = (props: {
  currentPage: number;
  totalPages: number;
  totalItems?: number;
  itemsPerPage?: number;
}) => {
  const onPageChange = vi.fn();
  render(() => <Pagination {...props} onPageChange={onPageChange} />);
  return onPageChange;
};

/** Labels of every page button, in document order. */
const pageLabels = () =>
  Array.from(document.querySelectorAll('.join button')).map((button) => button.textContent);

describe('Pagination', () => {
  it('lists every page when there are few of them', () => {
    renderPagination({ currentPage: 1, totalPages: 5 });

    expect(pageLabels()).toEqual(['«', '1', '2', '3', '4', '5', '»']);
  });

  it('reports the page the user clicked', () => {
    const onPageChange = renderPagination({ currentPage: 1, totalPages: 5 });

    fireEvent.click(screen.getByText('3'));

    expect(onPageChange).toHaveBeenCalledWith(3);
  });

  it('steps forwards and backwards with the arrow buttons', () => {
    const onPageChange = renderPagination({ currentPage: 3, totalPages: 5 });

    fireEvent.click(screen.getByText('«'));
    fireEvent.click(screen.getByText('»'));

    expect(onPageChange).toHaveBeenNthCalledWith(1, 2);
    expect(onPageChange).toHaveBeenNthCalledWith(2, 4);
  });

  it('never leaves the page range through the arrow buttons', () => {
    const first = renderPagination({ currentPage: 1, totalPages: 5 });
    fireEvent.click(screen.getByText('«'));
    expect(first).not.toHaveBeenCalled();

    // Start a fresh tree so the same button text resolves to one element again.
    cleanup();
    const last = renderPagination({ currentPage: 5, totalPages: 5 });
    fireEvent.click(screen.getByText('»'));
    expect(last).not.toHaveBeenCalled();
  });

  it('summarises the visible range when the totals are known', () => {
    renderPagination({ currentPage: 2, totalPages: 5, totalItems: 42, itemsPerPage: 10 });

    expect(screen.getByText(/Showing/).textContent).toContain('Showing 11 to 20 of 42 results');
  });

  it('omits the summary when the totals are unknown', () => {
    renderPagination({ currentPage: 2, totalPages: 5 });

    expect(screen.queryByText(/Showing/)).toBeNull();
  });

  it('collapses the tail while near the start of a long list', () => {
    renderPagination({ currentPage: 2, totalPages: 20 });

    expect(pageLabels()).toEqual(['«', '1', '2', '3', '4', '5', '...', '20', '»']);
  });

  it('collapses both sides in the middle of a long list', () => {
    renderPagination({ currentPage: 10, totalPages: 20 });

    expect(pageLabels()).toEqual(['«', '1', '...', '9', '10', '11', '...', '20', '»']);
  });

  it('collapses the head while near the end of a long list', () => {
    renderPagination({ currentPage: 19, totalPages: 20 });

    expect(pageLabels()).toEqual(['«', '1', '...', '16', '17', '18', '19', '20', '»']);
  });

  it('marks the current page and ignores clicks on the gap markers', () => {
    const onPageChange = renderPagination({ currentPage: 10, totalPages: 20 });

    expect(screen.getByText('10').className).toContain('btn-active');

    fireEvent.click(screen.getAllByText('...')[0]);

    expect(onPageChange).not.toHaveBeenCalled();
  });
});
