// App.tsx

import React, { useState } from 'react';
import JobTable from './JobTable';
import useMedia from './hooks/useMedia';
import Navigation from './Navbar';

import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
import { ThemeContext, themeName, themeSetting } from './contexts/ThemeContext';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import { useLocalStorage } from './hooks/useLocalStorage';
import { Theme, themeLocalStorageKey } from './Theme';

const App: React.FC = () => {
  const [token, setToken] = useState<string>('');
  const [showToken, setShowToken] = useState<boolean>(false);
  const [showJobTable, setShowJobTable] = useState<boolean>(false);

  const handleTokenInput = (event: React.ChangeEvent<HTMLInputElement>) => {
    setToken(event.target.value);
  };

  const handleTokenSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (token) {
      setShowJobTable(true);
    }
  };

  const handleToggleShowToken = () => {
    setShowToken((prevShowToken) => !prevShowToken);
  };

  const Jobs: React.FC = () => (
    <div className="content-container">
      {showJobTable && <JobTable token={token} setShowJobTable={setShowJobTable} />}
    </div>
  );

  const [userTheme, setUserTheme] = useLocalStorage<themeSetting>(themeLocalStorageKey, 'auto');
  const browserHasThemes = useMedia('(prefers-color-scheme)');
  const browserWantsDarkTheme = useMedia('(prefers-color-scheme: dark)');

  let theme: themeName;
  if (userTheme !== 'auto') {
    theme = userTheme;
  } else {
    theme = browserHasThemes ? (browserWantsDarkTheme ? 'dark' : 'light') : 'light';
  }

  return (
    <ThemeContext.Provider
      value={{ theme: theme, userPreference: userTheme, setTheme: (t: themeSetting) => setUserTheme(t) }}
    >
      <Theme />
      <Router>
        <div className="page">
          <Navigation/>
            {!showJobTable && (
              <div className="centered-container">
                <div className="auth-form">
                  <form className="token-input" onSubmit={handleTokenSubmit}>
                    <div className="field">
                      <label className="is-label">Token</label>
                      <div className="password-input-container">
                        <input
                          className="password-input"
                          type={showToken ? 'text' : 'password'}
                          value={token}
                          onChange={handleTokenInput}
                        />
                        <div className="password-input-suffix">
                          {showToken ? (
                            <VisibilityOff className="eye-icon" onClick={handleToggleShowToken} />
                          ) : (
                            <Visibility className="eye-icon" onClick={handleToggleShowToken} />
                          )}
                        </div>
                      </div>
                    </div>
                    <button className="btn btn-primary is-label" type="submit">Sign In</button>
                  </form>
                </div>
              </div>
            )}
            <Routes>
              <Route path="/" element={<Navigate to="/jobs" replace />} />
              <Route path="/jobs" element={<Jobs />} />
            </Routes>
          </div>
      </Router>
    </ThemeContext.Provider >

  );
};

export default App;
