import React from 'react';
import ReactDOM from 'react-dom/client';
import { Provider } from 'react-redux';
import store from './store';
import App from './App';
import { Theme } from './components/Theme';
import { ToastProvider } from './components/Toast';
import './styles/global.css';
import './styles/layout.css';
import './styles/auth.css';
import './styles/jobs.css';
import './styles/workers.css';
import './styles/dashboard.css';
import './styles/toast.css';
import reportWebVitals from './reportWebVitals';

const root: ReactDOM.Root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement);
root.render(
  <React.StrictMode>
    <Provider store={store}>
      <Theme />
      <ToastProvider>
        <App />
      </ToastProvider>
    </Provider>
  </React.StrictMode>,
);

reportWebVitals();