// App.tsx

import React, { useState } from 'react';
import { BrowserRouter as Router, Route, Routes, Navigate, Link } from 'react-router-dom';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import JobTable from './JobTable';
import './App.css';

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

  return (
    <Router>
      <div className="tableContainer">
        <header className="header">
          <div className="logoContainer">
            <div className="linkContainer">
              <Link to="/" className="link">
                <img src="/logo.svg" alt="Transcoder" className="logo" />
                <div className="appNameContainer">
                  <h1 className="appName">Transcoder</h1>
                </div>
              </Link>
            </div>
          </div>
          <div className="navBar">
            <nav className="navItems">
              <Link to="/jobs" className="navItem">
                Jobs
              </Link>
            </nav>
          </div>
        </header>

        <div className="contentContainer">
          {!showJobTable && (
            <div className="centeredContainer">
              <form className="modal" onSubmit={handleTokenSubmit}>
                <p>Please enter your token:</p>
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
                <button type="submit">Submit</button>
              </form>
            </div>
          )}
          <Routes>
            <Route path="/" element={<Navigate to="/jobs" replace />} />
            <Route path="/jobs" element={<Jobs />} />
          </Routes>
        </div>
      </div>
    </Router>
  );
};

export default App;
