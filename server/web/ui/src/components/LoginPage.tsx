import React, { useState } from 'react';
import { Visibility, VisibilityOff, ErrorOutline } from '@mui/icons-material';
import '../styles/LoginPage.css';

interface LoginPageProps {
  onLogin: (token: string) => void;
  errorText: string;
}

const LoginPage: React.FC<LoginPageProps> = ({ onLogin, errorText }) => {
  const [token, setToken] = useState<string>('');
  const [showToken, setShowToken] = useState<boolean>(false);

  const handleTokenInput = (event: React.ChangeEvent<HTMLInputElement>) => {
    setToken(event.target.value);
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (token) {
      onLogin(token);
    }
  };

  const toggleShowToken = () => {
    setShowToken((prev) => !prev);
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-header">
          <div className="login-logo">
            <img src="/logo.svg" alt="Gearr" />
            <span>Gearr</span>
          </div>
          <div className="login-subtitle">Video Encoding Management</div>
        </div>
        
        <form className="login-body" onSubmit={handleSubmit}>
          {errorText && (
            <div className="login-error animate-slide-down">
              <ErrorOutline className="login-error-icon" />
              <div className="login-error-content">
                <div className="login-error-title">Login Failed</div>
                <div className="login-error-message">{errorText}</div>
              </div>
            </div>
          )}
          
          <div className="login-field">
            <label className="login-label" htmlFor="token">
              Authentication Token
            </label>
            <div className="login-input-wrapper">
              <input
                id="token"
                className="login-input"
                type={showToken ? 'text' : 'password'}
                value={token}
                onChange={handleTokenInput}
                placeholder="Enter your token"
                autoComplete="off"
              />
              <button
                type="button"
                className="login-toggle-visibility"
                onClick={toggleShowToken}
                tabIndex={-1}
              >
                {showToken ? <VisibilityOff /> : <Visibility />}
              </button>
            </div>
          </div>
          
          <button className="login-submit" type="submit">
            Sign In
          </button>
        </form>
        
        <div className="login-footer">
          <p className="login-footer-text">
            Need help?{' '}
            <a 
              href="https://github.com/pando85/gearr" 
              target="_blank" 
              rel="noopener noreferrer"
              className="login-footer-link"
            >
              Documentation
            </a>
          </p>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;