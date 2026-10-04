import { fireEvent, render, screen } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';
import Modal, { useModal } from './Modal';

describe('Modal', () => {
  it('renders nothing while it is closed', () => {
    render(() => (
      <Modal title="Edit product" isOpen={false} onClose={() => {}}>
        <p>Body</p>
      </Modal>
    ));

    expect(screen.queryByText('Edit product')).toBeNull();
    expect(screen.queryByText('Body')).toBeNull();
  });

  it('renders the title and the children while it is open', () => {
    render(() => (
      <Modal title="Edit product" isOpen onClose={() => {}}>
        <p>Body</p>
      </Modal>
    ));

    expect(screen.getByText('Edit product')).toBeDefined();
    expect(screen.getByText('Body')).toBeDefined();
  });

  it('calls onClose from both the close button and the backdrop', () => {
    const onClose = vi.fn();
    render(() => (
      <Modal title="Edit product" isOpen onClose={onClose}>
        <p>Body</p>
      </Modal>
    ));

    fireEvent.click(screen.getByText('✕'));
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.click(document.querySelector('.modal-backdrop') as Element);
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it('defaults to the medium size and honours an explicit one', () => {
    render(() => (
      <Modal title="Edit product" isOpen onClose={() => {}}>
        <p>Body</p>
      </Modal>
    ));

    expect((document.querySelector('.modal-box') as Element).className).toContain('max-w-2xl');
  });

  it('applies the requested size class', () => {
    render(() => (
      <Modal title="Edit product" isOpen onClose={() => {}} size="xl">
        <p>Body</p>
      </Modal>
    ));

    expect((document.querySelector('.modal-box') as Element).className).toContain('max-w-6xl');
  });
});

describe('useModal', () => {
  it('opens, closes and toggles the state', () => {
    let modal!: ReturnType<typeof useModal>;
    render(() => {
      modal = useModal();
      return <span>{String(modal.isOpen())}</span>;
    });

    expect(screen.getByText('false')).toBeDefined();

    modal.open();
    expect(screen.getByText('true')).toBeDefined();

    modal.close();
    expect(screen.getByText('false')).toBeDefined();

    modal.toggle();
    expect(screen.getByText('true')).toBeDefined();

    modal.toggle();
    expect(screen.getByText('false')).toBeDefined();
  });
});
