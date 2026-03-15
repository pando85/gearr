import { writable } from 'svelte/store';

export interface ToastData {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title?: string;
  message: string;
  duration?: number;
}

function createToastStore() {
  const { subscribe, update } = writable<ToastData[]>([]);

  const showToast = (toast: Omit<ToastData, 'id'>) => {
    const id = Math.random().toString(36).substring(2, 9);
    update((toasts) => [...toasts, { ...toast, id }]);
    return id;
  };

  const removeToast = (id: string) => {
    update((toasts) => toasts.filter((t) => t.id !== id));
  };

  return {
    subscribe,
    success: (message: string, title?: string) => showToast({ type: 'success', message, title }),
    error: (message: string, title?: string) => showToast({ type: 'error', message, title }),
    warning: (message: string, title?: string) => showToast({ type: 'warning', message, title }),
    info: (message: string, title?: string) => showToast({ type: 'info', message, title }),
    show: showToast,
    remove: removeToast,
  };
}

export const toastStore = createToastStore();
