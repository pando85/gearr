import React, { useEffect, createContext, useContext, useState, useCallback } from 'react';
import { CheckCircle, Error as ErrorIcon, Warning, Info, Close } from '@mui/icons-material';
import '../styles/Toast.css';

type ToastType = 'success' | 'error' | 'warning' | 'info';

interface Toast {
  id: string;
  type: ToastType;
  title?: string;
  message: string;
  duration?: number;
}

interface ToastContextValue {
  showToast: (toast: Omit<Toast, 'id'>) => void;
  removeToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined);

export const useToast = (): ToastContextValue => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
};

const ToastIcon: React.FC<{ type: ToastType }> = ({ type }) => {
  const icons: Record<ToastType, React.ReactNode> = {
    success: <CheckCircle className="toast-icon" />,
    error: <ErrorIcon className="toast-icon" />,
    warning: <Warning className="toast-icon" />,
    info: <Info className="toast-icon" />,
  };
  return <>{icons[type]}</>;
};

const ToastItem: React.FC<{ toast: Toast; onClose: () => void }> = ({ toast, onClose }) => {
  useEffect(() => {
    if (toast.duration !== 0) {
      const timer = setTimeout(onClose, toast.duration || 5000);
      return () => clearTimeout(timer);
    }
  }, [toast.duration, onClose]);

  return (
    <div className={`toast toast-${toast.type}`}>
      <ToastIcon type={toast.type} />
      <div className="toast-content">
        {toast.title && <div className="toast-title">{toast.title}</div>}
        <div className="toast-message">{toast.message}</div>
      </div>
      <button className="toast-close" onClick={onClose} aria-label="Close">
        <Close fontSize="small" />
      </button>
    </div>
  );
};

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback((toast: Omit<Toast, 'id'>) => {
    const id = Math.random().toString(36).substring(2, 9);
    setToasts((prev) => [...prev, { ...toast, id }]);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ showToast, removeToast }}>
      {children}
      <div className="toast-container">
        {toasts.map((toast) => (
          <ToastItem
            key={toast.id}
            toast={toast}
            onClose={() => removeToast(toast.id)}
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
};