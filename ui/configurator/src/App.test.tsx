import { render, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import App from './App';

// Mock the useAuth hook for App tests
vi.mock('./hooks/useAuth', () => ({
  useAuth: () => ({
    user: { name: 'Admin User', email: 'admin@tacito.local', roles: ['keeper-admin'] },
    loading: false,
    authenticated: true,
    roles: ['keeper-admin'],
    error: null,
    logout: vi.fn(),
  }),
}));

test('renders app header', () => {
  render(<App />);
  const headerElement = screen.getByText(/Tacito Square Configurator/i);
  expect(headerElement).toBeInTheDocument();
});
