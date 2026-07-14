/**
 * SafeGAI Hybrid App - Main Application Component
 *
 * Provides role-based routing between User, Operator, and Maintainer views.
 * Manages authentication state and role selection.
 *
 * SAFETY UX RULES:
 * - Active critical warnings visible in ALL views
 * - Role determines which controls are available
 * - USER mode: read-only, NO ACK, NO settings
 * - OPERATOR mode: event ACK/resolve, work window
 * - MAINTAINER mode: diagnostics, I/O test, backup
 */

import React, { useState, useCallback } from 'react';
import type { Role } from './types/roles';
import type { LocalAdapter } from './adapters/localAdapter';
import { mockAdapter } from './adapters/mockAdapter';
import { UserView } from './pages/UserView';
import { OperatorView } from './pages/OperatorView';
import { MaintainerView } from './pages/MaintainerView';

// ---------------------------------------------------------------------------
// App State
// ---------------------------------------------------------------------------

interface AuthState {
  authenticated: boolean;
  role: Role;
  username: string;
  token: string;
}

// ---------------------------------------------------------------------------
// App Component
// ---------------------------------------------------------------------------

export const App: React.FC = () => {
  const [auth, setAuth] = useState<AuthState>({
    authenticated: false,
    role: 'USER',
    username: '',
    token: '',
  });

  const [loginUsername, setLoginUsername] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [loginError, setLoginError] = useState('');

  const adapter: LocalAdapter = mockAdapter;

  const handleLogin = useCallback(async () => {
    setLoginError('');
    const response = await adapter.login({
      username: loginUsername,
      password: loginPassword,
    });

    if (response.success) {
      setAuth({
        authenticated: true,
        role: response.role,
        username: loginUsername,
        token: response.token,
      });
    } else {
      setLoginError(response.error || 'Login failed');
    }
  }, [adapter, loginUsername, loginPassword]);

  const handleLogout = useCallback(async () => {
    await adapter.logout();
    setAuth({
      authenticated: false,
      role: 'USER',
      username: '',
      token: '',
    });
  }, [adapter]);

  // Login Screen
  if (!auth.authenticated) {
    return (
      <div className="app app--login" role="main" aria-label="SafeGAI Login">
        <div className="app__login-container">
          <h1 className="app__title">SafeGAI</h1>
          <p className="app__subtitle">AI Fisheye Zone Safety System</p>
          <form
            className="app__login-form"
            onSubmit={(e) => {
              e.preventDefault();
              handleLogin();
            }}
          >
            <div className="app__form-group">
              <label htmlFor="login-username">Username:</label>
              <input
                id="login-username"
                type="text"
                value={loginUsername}
                onChange={(e) => setLoginUsername(e.target.value)}
                placeholder="Enter username"
                autoComplete="username"
              />
            </div>
            <div className="app__form-group">
              <label htmlFor="login-password">Password:</label>
              <input
                id="login-password"
                type="password"
                value={loginPassword}
                onChange={(e) => setLoginPassword(e.target.value)}
                placeholder="Enter password"
                autoComplete="current-password"
              />
            </div>
            {loginError && (
              <div className="app__login-error" role="alert">
                {loginError}
              </div>
            )}
            <button className="app__btn app__btn--login" type="submit">
              Login
            </button>
          </form>
          <div className="app__login-hints">
            <p><small>Mock login hints:</small></p>
            <ul>
              <li><code>user-*</code> / any 4+ char password = USER role</li>
              <li><code>operator-*</code> / any 4+ char password = OPERATOR role</li>
              <li><code>maintainer-*</code> / any 4+ char password = MAINTAINER role</li>
            </ul>
          </div>
        </div>
      </div>
    );
  }

  // Authenticated: Show role-based view
  return (
    <div className="app app--authenticated">
      {/* Header */}
      <header className="app__header" role="banner">
        <h1 className="app__header-title">SafeGAI</h1>
        <nav className="app__header-nav" aria-label="User information">
          <span className="app__header-role">
            Role: <strong>{auth.role}</strong>
          </span>
          <span className="app__header-user">
            User: <strong>{auth.username}</strong>
          </span>
          <button
            className="app__btn app__btn--logout"
            onClick={handleLogout}
            aria-label="Logout"
          >
            Logout
          </button>
        </nav>
      </header>

      {/* Role-based content */}
      <div className="app__content">
        {auth.role === 'USER' && <UserView adapter={adapter} />}
        {auth.role === 'OPERATOR' && (
          <OperatorView adapter={adapter} username={auth.username} />
        )}
        {auth.role === 'MAINTAINER' && (
          <MaintainerView adapter={adapter} username={auth.username} />
        )}
      </div>

      {/* Footer */}
      <footer className="app__footer" role="contentinfo">
        <span>SafeGAI v0.1.0 - AI Fisheye Zone Safety 4CH</span>
        <span>Local Gateway Mode</span>
      </footer>
    </div>
  );
};

export default App;
