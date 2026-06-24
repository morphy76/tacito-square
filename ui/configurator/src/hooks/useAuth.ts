import { useState, useEffect, useCallback } from 'react';

export interface User {
  id: string;
  name: string;
  email: string;
  roles: string[];
  tenant_id: string;
}

export function useAuth() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [authenticated, setAuthenticated] = useState<boolean>(false);
  const [roles, setRoles] = useState<string[]>([]);
  const [error, setError] = useState<Error | null>(null);

  const fetchUser = useCallback(async () => {
    try {
      setLoading(true);
      const base = import.meta.env.BASE_URL;
      const cleanBase = base.endsWith('/') ? base.slice(0, -1) : base;
      const response = await fetch(`${cleanBase}/api/v1/auth/me`, {
        headers: {
          'Accept': 'application/json',
        },
      });

      if (response.status === 401) {
        setAuthenticated(false);
        setUser(null);
        setRoles([]);
        setError(new Error('Unauthorized'));
      } else if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      } else {
        const data: User = await response.json();
        setUser(data);
        setAuthenticated(true);
        setRoles(data.roles || []);
        setError(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch user'));
      setAuthenticated(false);
      setUser(null);
      setRoles([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUser();
  }, [fetchUser]);

  const logout = useCallback(async () => {
    try {
      setLoading(true);
      const base = import.meta.env.BASE_URL;
      const cleanBase = base.endsWith('/') ? base.slice(0, -1) : base;
      const response = await fetch(`${cleanBase}/api/v1/auth/logout`, {
        method: 'POST',
      });
      if (response.ok) {
        // Redirect browser to trigger Keycloak end-session redirect
        window.location.href = `${cleanBase}/api/v1/auth/login`;
      }
    } catch (err) {
      console.error('Logout failed', err);
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    user,
    loading,
    authenticated,
    roles,
    error,
    logout,
    refetch: fetchUser,
  };
}
