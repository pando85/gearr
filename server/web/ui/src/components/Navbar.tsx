import React, { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Work, Devices, GitHub, Menu as MenuIcon, Close as CloseIcon } from '@mui/icons-material';
import ThemeToggle from './ThemeToggle';

const Navbar: React.FC = () => {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const location = useLocation();

  const toggleMobileMenu = () => {
    setMobileMenuOpen(!mobileMenuOpen);
  };

  const closeMobileMenu = () => {
    setMobileMenuOpen(false);
  };

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  const navLinks = [
    { to: '/dashboard', label: 'Dashboard', icon: <Work /> },
    { to: '/jobs', label: 'Jobs', icon: <Work /> },
    { to: '/workers', label: 'Workers', icon: <Devices /> },
  ];

  return (
    <nav className="navbar">
      <Link to="/" className="navbar-brand" onClick={closeMobileMenu}>
        <img src="/logo.svg" alt="Gearr" className="navbar-logo" />
        <span className="navbar-title">Gearr</span>
      </Link>

      <div className="navbar-nav">
        {navLinks.map((link) => (
          <Link
            key={link.to}
            to={link.to}
            className={`nav-link ${isActive(link.to) ? 'active' : ''}`}
            onClick={closeMobileMenu}
          >
            {link.icon}
            <span>{link.label}</span>
          </Link>
        ))}
      </div>

      <div className="nav-actions">
        <a
          href="https://github.com/pando85/gearr"
          target="_blank"
          rel="noopener noreferrer"
          className="nav-link"
          title="GitHub"
        >
          <GitHub />
        </a>
        <ThemeToggle />
        <button className="mobile-menu-btn" onClick={toggleMobileMenu}>
          {mobileMenuOpen ? <CloseIcon /> : <MenuIcon />}
        </button>
      </div>

      <div className={`mobile-nav ${mobileMenuOpen ? 'open' : ''}`}>
        {navLinks.map((link) => (
          <Link
            key={link.to}
            to={link.to}
            className={`nav-link ${isActive(link.to) ? 'active' : ''}`}
            onClick={closeMobileMenu}
          >
            {link.icon}
            <span>{link.label}</span>
          </Link>
        ))}
      </div>
    </nav>
  );
};

export default Navbar;