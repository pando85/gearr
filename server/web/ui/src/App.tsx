import React, { useState } from 'react';
import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import LoginPage from './components/LoginPage';
import Dashboard from './components/Dashboard';
import JobsPage from './components/JobsPage';
import WorkersPage from './components/WorkersPage';
import { ThemeContext, themeName, themeSetting } from './contexts/ThemeContext';
import { useLocalStorage } from './hooks/useLocalStorage';
import useMedia from './hooks/useMedia';
import { Theme, themeLocalStorageKey } from './components/Theme';

const App: React.FC = () => {
  const [token, setToken] = useState<string>('');
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [errorText, setErrorText] = useState<string>('');

  const handleLogin = (inputToken: string) => {
    setToken(inputToken);
    setIsAuthenticated(true);
    setErrorText('');
  };

  const handleLogout = () => {
    setToken('');
    setIsAuthenticated(false);
  };

  const [userTheme, setUserTheme] = useLocalStorage<themeSetting>(themeLocalStorageKey, 'auto');
  const browserHasThemes = useMedia('(prefers-color-scheme)');
  const browserWantsDarkTheme = useMedia('(prefers-color-scheme: dark)');

  let theme: themeName;
  if (userTheme !== 'auto') {
    theme = userTheme;
  } else {
    theme = browserHasThemes ? (browserWantsDarkTheme ? 'dark' : 'light') : 'light';
  }

  if (!isAuthenticated) {
    return (
      <ThemeContext.Provider
        value={{
          theme: theme,
          userPreference: userTheme,
          setTheme: (t: themeSetting) => setUserTheme(t),
        }}
      >
        <Theme />
        <LoginPage onLogin={handleLogin} errorText={errorText} />
      </ThemeContext.Provider>
    );
  }

  return (
    <ThemeContext.Provider
      value={{
        theme: theme,
        userPreference: userTheme,
        setTheme: (t: themeSetting) => setUserTheme(t),
      }}
    >
      <Theme />
      <Router>
        <Layout>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route
              path="/dashboard"
              element={<Dashboard token={token} />}
            />
            <Route
              path="/jobs"
              element={
                <JobsPage
                  token={token}
                  setShowJobTable={setIsAuthenticated}
                  setErrorText={setErrorText}
                />
              }
            />
            <Route
              path="/workers"
              element={
                <WorkersPage
                  token={token}
                  setShowJobTable={setIsAuthenticated}
                  setErrorText={setErrorText}
                />
              }
            />
          </Routes>
        </Layout>
      </Router>
    </ThemeContext.Provider>
  );
};

export default App;