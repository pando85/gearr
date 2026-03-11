import React, { useEffect } from 'react';
import { useTheme } from './contexts/ThemeContext';

export const themeLocalStorageKey = 'user-prefers-color-scheme';

export const Theme: React.FC = () => {
  const { theme } = useTheme();

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  return null;
};

export { default as ThemeToggle } from './ThemeToggle';