import { createSignal, type JSX, Show } from 'solid-js';

interface ModalProps {
  title: string;
  isOpen: boolean;
  onClose: () => void;
  children: JSX.Element;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

export default function Modal(props: ModalProps) {
  const sizeClasses = {
    sm: 'max-w-sm',
    md: 'max-w-2xl',
    lg: 'max-w-4xl',
    xl: 'max-w-6xl',
  };

  return (
    <Show when={props.isOpen}>
      <div class="modal modal-open">
        <div class={`modal-box ${sizeClasses[props.size || 'md']}`}>
          <div class="flex items-center justify-between mb-4">
            <h3 class="font-bold text-lg">{props.title}</h3>
            <button
              class="btn btn-sm btn-circle btn-ghost"
              onClick={props.onClose}
            >
              ✕
            </button>
          </div>
          <div class="py-4">{props.children}</div>
        </div>
        <div class="modal-backdrop" onClick={props.onClose}></div>
      </div>
    </Show>
  );
}

export function useModal() {
  const [isOpen, setIsOpen] = createSignal(false);

  return {
    isOpen,
    open: () => setIsOpen(true),
    close: () => setIsOpen(false),
    toggle: () => setIsOpen(!isOpen()),
  };
}
