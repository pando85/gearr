import React from 'react';
import { LightMode, DarkMode, Settings } from '@mui/icons-material';
import { useTheme } from '../contexts/ThemeContext';

const ThemeToggle: React.FC = () => {
  const { userPreference, setTheme } = useTheme();

  return (
    <div className="theme-toggle">
      <button
        className={`theme-btn ${userPreference === 'light' ? 'active' : ''}`}
        onClick={() => setTheme('light')}
        title="Light theme"
      >
        <LightMode />
      </button>
      <button
        className={`theme-btn ${userPreference === 'dark' ? 'active' : ''}`}
        onClick={() => setTheme('dark')}
        title="Dark theme"
      >
        <DarkMode />
      </button>
      <button
        className={`theme-btn ${userPreference === 'auto' ? 'active' : ''}`}
        onClick={() => setTheme('auto')}
        title="System theme"
      >
        <Settings />
      </button>
    </div>
  );
};

export default ThemeToggle;