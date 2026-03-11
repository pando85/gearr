import React, { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { WbSunny, DarkMode, SettingsBrightness, Menu, Close, GitHub, Work, People } from '@mui/icons-material';
import { useTheme } from '../contexts/ThemeContext';
import { themeSetting } from '../contexts/ThemeContext';
import './styles/Navbar.css';

const Navbar: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const { userPreference, setTheme } = useTheme();
  const location = useLocation();

  const toggleMobileMenu = () => {
    setMobileMenuOpen(!mobileMenuOpen);
  };

  const closeMobileMenu = () => {
    setMobileMenuOpen(false);
  };

  const isActive = (path: string) => {
    return location.pathname === path;
  };

  const themeButtons: { setting: themeSetting; icon: React.ReactNode; label: string }[] = [
    { setting: 'light', icon: <WbSunny />, label: 'Light theme' },
    { setting: 'dark', icon: <DarkMode />, label: 'Dark theme' },
    { setting: 'auto', icon: <SettingsBrightness />, label: 'System theme' },
  ];

  return (
    <>
      <nav className="navbar">
        <Link to="/jobs" className="navbar-brand" onClick={closeMobileMenu}>
          <img src="/logo.svg" alt="Gearr" />
          <span className="navbar-brand-text">Gearr</span>
        </Link>

        <ul className="navbar-nav">
          <li>
            <Link to="/jobs" className={`navbar-link ${isActive('/jobs') ? 'active' : ''}`}>
              <Work className="navbar-link-icon" />
              <span>Jobs</span>
            </Link>
          </li>
          <li>
            <Link to="/workers" className={`navbar-link ${isActive('/workers') ? 'active' : ''}`}>
              <People className="navbar-link-icon" />
              <span>Workers</span>
            </Link>
          </li>
          <li>
            <a 
              href="https://github.com/pando85/gearr" 
              target="_blank" 
              rel="noopener noreferrer"
              className="navbar-link"
            >
              <GitHub className="navbar-link-icon" />
              <span className="hide-mobile">GitHub</span>
            </a>
          </li>
        </ul>

        <div className="navbar-actions">
          <div className="navbar-theme-toggle">
            {themeButtons.map(({ setting, icon, label }) => (
              <button
                key={setting}
                className={`navbar-theme-btn ${userPreference === setting ? 'active' : ''}`}
                onClick={() => setTheme(setting)}
                title={label}
                aria-label={label}
              >
                {icon}
              </button>
            ))}
          </div>

          <button 
            className="navbar-mobile-toggle" 
            onClick={toggleMobileMenu}
            aria-label="Toggle menu"
          >
            {mobileMenuOpen ? <Close /> : <Menu />}
          </button>
        </div>
      </nav>

      <div className={`navbar-mobile-menu ${mobileMenuOpen ? 'open' : ''}`}>
        <ul className="navbar-mobile-nav">
          <li>
            <Link 
              to="/jobs" 
              className={`navbar-mobile-link ${isActive('/jobs') ? 'active' : ''}`}
              onClick={closeMobileMenu}
            >
              <Work />
              <span>Jobs</span>
            </Link>
          </li>
          <li>
            <Link 
              to="/workers" 
              className={`navbar-mobile-link ${isActive('/workers') ? 'active' : ''}`}
              onClick={closeMobileMenu}
            >
              <People />
              <span>Workers</span>
            </Link>
          </li>
          <li>
            <a 
              href="https://github.com/pando85/gearr"
              target="_blank"
              rel="noopener noreferrer"
              className="navbar-mobile-link"
              onClick={closeMobileMenu}
            >
              <GitHub />
              <span>GitHub</span>
            </a>
          </li>
        </ul>
      </div>
    </>
  );
};

export default Navbar;