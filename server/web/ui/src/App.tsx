import React, { useState } from 'react';
import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
import Alert from 'react-bootstrap/Alert';
import { Visibility, VisibilityOff, ErrorOutline } from '@mui/icons-material';

import JobTable from './JobTable';
import WorkerTable from './WorkerTable';
import useMedia from './hooks/useMedia';
import Navigation from './Navbar';
import { ThemeContext, themeName, themeSetting } from './contexts/ThemeContext';
import { useLocalStorage } from './hooks/useLocalStorage';
import { Theme, themeLocalStorageKey } from './Theme';


const App: React.FC = () => {
  const [token, setToken] = useState<string>('');
  const [showToken, setShowToken] = useState<boolean>(false);
  const [showJobTable, setShowJobTable] = useState<boolean>(false);
  const [errorText, setErrorText] = useState<string>("");

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
      {showJobTable && <JobTable token={token} setShowJobTable={setShowJobTable} setErrorText={setErrorText} />}
    </div>
  );

  const Workers: React.FC = () => (
    <div className="content-container">
      {showJobTable && <WorkerTable token={token} setShowJobTable={setShowJobTable} setErrorText={setErrorText} />}
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
          <Navigation />
          {!showJobTable && (
            <div className="centered-container">
              <div className="auth-form">
                <form className="token-input" onSubmit={handleTokenSubmit}>
                  {errorText && (
                    <Alert key="danger" variant="danger" >
                      <div className="d-flex align-items-center">
                        <div className="mr-2">
                          <ErrorOutline />
                        </div>
                        <div>
                          <span>Login failed</span>
                          <div>{errorText}</div>

                        </div>
                      </div>
                    </Alert>
                  )}
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
          {showJobTable && (
          <Routes>
            <Route path="/" element={<Navigate to="/jobs" replace />} />
            <Route path="/jobs" element={<Jobs />} />
            <Route path="/workers" element={<Workers />} />
          </Routes>
          )}
        </div>
      </Router>
    </ThemeContext.Provider >

  );
};

export default App;
