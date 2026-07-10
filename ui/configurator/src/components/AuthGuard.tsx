import React, { useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';

interface AuthGuardProps {
  children: React.ReactNode;
}

export default function AuthGuard({ children }: AuthGuardProps) {
  const { loading, authenticated, roles, error, logout } = useAuth();

  useEffect(() => {
    if (!loading && (!authenticated || error)) {
      const base = import.meta.env.BASE_URL;
      const cleanBase = base.endsWith('/') ? base.slice(0, -1) : base;
      window.location.href = `${cleanBase}/api/v1/auth/login`;
    }
  }, [loading, authenticated, error]);

  if (loading) {
    return (
      <div data-testid="auth-loading" className="auth-loading-container">
        <div className="spinner"></div>
        <p>Verifying authentication session...</p>
      </div>
    );
  }

  if (!authenticated || error) {
    return null; // Will redirect via useEffect
  }

  const hasRequiredRole = roles.includes('keeper-admin') || roles.includes('agent-spawner');

  if (!hasRequiredRole) {
    return (
      <div className="access-denied-container">
        <div className="glass-card access-denied-card">
          <div className="monument-header">
            <div className="fountain-outer-ring"></div>
            <div className="fountain-inner-ring">
              <div className="steel-mast-core"></div>
            </div>
          </div>
          <h1>Access Denied</h1>
          <p className="subtitle">Unauthorized Access Attempt</p>
          <p className="description">
            You do not have the required permissions to view the administration panel. 
            This view is restricted to administrators and spawners only.
          </p>
          <button className="nav-btn logout-action-btn" onClick={logout}>
            <span className="btn-label">
              <span className="btn-title">Logout</span>
              <span className="btn-subtitle">Terminate session</span>
            </span>
            <span className="btn-arrow">→</span>
          </button>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
