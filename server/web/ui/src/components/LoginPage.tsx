import React, { useState } from 'react';
import { Visibility, VisibilityOff, ErrorOutline } from '@mui/icons-material';

interface LoginPageProps {
  onLogin: (token: string) => void;
  errorText: string;
}

const LoginPage: React.FC<LoginPageProps> = ({ onLogin, errorText }) => {
  const [token, setToken] = useState<string>('');
  const [showToken, setShowToken] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  const handleTokenInput = (event: React.ChangeEvent<HTMLInputElement>) => {
    setToken(event.target.value);
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!token.trim()) return;
    
    setIsLoading(true);
    onLogin(token.trim());
    setTimeout(() => setIsLoading(false), 500);
  };

  const toggleShowToken = () => {
    setShowToken((prev) => !prev);
  };

  return (
    <div className="auth-page">
      <div className="auth-container">
        <div className="auth-card">
          <div className="auth-header">
            <div className="auth-logo">
              <img src="/logo.svg" alt="Gearr" />
              <span className="auth-logo-text">Gearr</span>
            </div>
            <p className="auth-subtitle">
              Video transcoding management dashboard
            </p>
          </div>

          {errorText && (
            <div className="auth-error">
              <ErrorOutline className="auth-error-icon" />
              <div className="auth-error-content">
                <div className="auth-error-title">Authentication Failed</div>
                <div className="auth-error-message">{errorText}</div>
              </div>
            </div>
          )}

          <form className="auth-form" onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="label" htmlFor="token">
                Access Token
              </label>
              <div className="token-field">
                <div className="token-input-wrapper">
                  <input
                    id="token"
                    className="token-input"
                    type={showToken ? 'text' : 'password'}
                    value={token}
                    onChange={handleTokenInput}
                    placeholder="Enter your access token"
                    autoComplete="off"
                    autoFocus
                  />
                  <button
                    type="button"
                    className="token-toggle-btn"
                    onClick={toggleShowToken}
                    tabIndex={-1}
                  >
                    {showToken ? <VisibilityOff /> : <Visibility />}
                  </button>
                </div>
              </div>
            </div>

            <button
              type="submit"
              className="auth-submit"
              disabled={!token.trim() || isLoading}
            >
              {isLoading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>

          <div className="auth-footer">
            <p className="auth-footer-text">
              Need help?{' '}
              <a
                href="https://github.com/pando85/gearr"
                target="_blank"
                rel="noopener noreferrer"
                className="auth-footer-link"
              >
                View Documentation
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;