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
    <div className="contentContainer">
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
        <div className="tableContainer">
          <Navigation/>
            {!showJobTable && (
              <div className="centeredContainer">
                <div className="auth-form">
                  <form className="tokenInput" onSubmit={handleTokenSubmit}>
                    <div className="field">
                      <label className="is-label">Token</label>
                      <div className="passwordInputContainer">
                        <input
                          className="passwordInput"
                          type={showToken ? 'text' : 'password'}
                          value={token}
                          onChange={handleTokenInput}
                        />
                        <div className="passwordInputSuffix">
                          {showToken ? (
                            <VisibilityOff className="eyeIcon" onClick={handleToggleShowToken} />
                          ) : (
                            <Visibility className="eyeIcon" onClick={handleToggleShowToken} />
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
