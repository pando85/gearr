import React, { useState } from 'react';
import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';

import LoginPage from './components/LoginPage';
import Navbar from './components/Navbar';
import JobsPage from './components/JobsPage';
import WorkersPage from './components/WorkersPage';
import Dashboard from './components/Dashboard';
import { ToastProvider } from './components/Toast';
import useMedia from './hooks/useMedia';
import { useLocalStorage } from './hooks/useLocalStorage';
import { ThemeContext, themeName, themeSetting } from './contexts/ThemeContext';
import { Theme } from './Theme';
import { useSelector } from 'react-redux';
import { RootState } from './store';
import './styles/design-system.css';

const App: React.FC = () => {
  const [token, setToken] = useState<string>('');
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [errorText, setErrorText] = useState<string>('');

  const jobs = useSelector((state: RootState) => state.jobs);

  const handleLogin = (newToken: string) => {
    setToken(newToken);
    setIsAuthenticated(true);
    setErrorText('');
  };

  const handleLogout = () => {
    setToken('');
    setIsAuthenticated(false);
  };

  const [userTheme, setUserTheme] = useLocalStorage<themeSetting>('user-prefers-color-scheme', 'auto');
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
      value={{ theme, userPreference: userTheme, setTheme: (t: themeSetting) => setUserTheme(t) }}
    >
      <Theme />
      <ToastProvider>
        <Router>
          <div className="page">
            {isAuthenticated && <Navbar />}
            {!isAuthenticated ? (
              <LoginPage onLogin={handleLogin} errorText={errorText} />
            ) : (
              <Routes>
                <Route path="/" element={<Navigate to="/dashboard" replace />} />
                <Route 
                  path="/dashboard" 
                  element={<Dashboard jobs={jobs} />} 
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
            )}
          </div>
        </Router>
      </ToastProvider>
    </ThemeContext.Provider>
  );
};

export default App;