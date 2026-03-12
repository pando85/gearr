import React, { useEffect } from 'react';
import { useTheme } from './contexts/ThemeContext';

export const themeLocalStorageKey = 'user-prefers-color-scheme';

export const Theme: React.FC = () => {
  const { theme } = useTheme();

  useEffect(() => {
    document.body.classList.remove('bootstrap', 'bootstrap-dark');
    document.body.classList.add(theme === 'dark' ? 'bootstrap-dark' : 'bootstrap');
  }, [theme]);

  return null;
};