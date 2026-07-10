import { render, screen } from '@testing-library/react';
import { expect, test, vi, beforeEach, afterEach, describe } from 'vitest';
import AuthGuard from './AuthGuard';
import { useAuth } from '../hooks/useAuth';

// Mock the useAuth hook
vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

describe('AuthGuard Component', () => {
  const originalLocation = window.location;

  beforeEach(() => {
    // Mock window.location
    const windowMock = window as any;
    delete windowMock.location;
    windowMock.location = {
      ...originalLocation,
      href: '',
    };
  });

  afterEach(() => {
    const windowMock = window as any;
    windowMock.location = originalLocation;
    vi.restoreAllMocks();
  });

  test('renders loading state', () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      loading: true,
      authenticated: false,
      roles: [],
      error: null,
      logout: vi.fn(),
      refetch: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Protected Content</div>
      </AuthGuard>
    );

    expect(screen.getByTestId('auth-loading')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  test('redirects unauthenticated user to login', () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      loading: false,
      authenticated: false,
      roles: [],
      error: new Error('Unauthorized'),
      logout: vi.fn(),
      refetch: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Protected Content</div>
      </AuthGuard>
    );

    expect(window.location.href).toBe('/ui/api/v1/auth/login');
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  test('shows access denied for authenticated user without correct roles', () => {
    vi.mocked(useAuth).mockReturnValue({
      user: { id: '1', name: 'Viewer', email: 'viewer@tacito.local', roles: ['keeper-viewer'], tenant_id: 't1' },
      loading: false,
      authenticated: true,
      roles: ['keeper-viewer'],
      error: null,
      logout: vi.fn(),
      refetch: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Protected Content</div>
      </AuthGuard>
    );

    expect(screen.getByText(/Access Denied/i)).toBeInTheDocument();
    expect(screen.getByText(/You do not have the required permissions/i)).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  test('renders children for authorized user with keeper-admin role', () => {
    vi.mocked(useAuth).mockReturnValue({
      user: { id: '1', name: 'Admin', email: 'admin@tacito.local', roles: ['keeper-admin'], tenant_id: 't1' },
      loading: false,
      authenticated: true,
      roles: ['keeper-admin'],
      error: null,
      logout: vi.fn(),
      refetch: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Protected Content</div>
      </AuthGuard>
    );

    expect(screen.getByText('Protected Content')).toBeInTheDocument();
    expect(screen.queryByText(/Access Denied/i)).not.toBeInTheDocument();
  });

  test('renders children for authorized user with agent-spawner role', () => {
    vi.mocked(useAuth).mockReturnValue({
      user: { id: '1', name: 'Spawner', email: 'spawner@tacito.local', roles: ['agent-spawner'], tenant_id: 't1' },
      loading: false,
      authenticated: true,
      roles: ['agent-spawner'],
      error: null,
      logout: vi.fn(),
      refetch: vi.fn(),
    });

    render(
      <AuthGuard>
        <div>Protected Content</div>
      </AuthGuard>
    );

    expect(screen.getByText('Protected Content')).toBeInTheDocument();
    expect(screen.queryByText(/Access Denied/i)).not.toBeInTheDocument();
  });
});
