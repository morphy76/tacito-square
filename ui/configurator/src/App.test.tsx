import { render, screen } from '@testing-library/react';
import { expect, test, vi, beforeEach } from 'vitest';
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

beforeEach(() => {
  global.fetch = vi.fn().mockImplementation((url: string) => {
    if (url.includes('/api/v1/configurator/wizard/options')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ llm_bindings: [], skills: [], prompts: [] }),
      });
    }
    if (url.includes('/api/v1/configurator/agents')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
      });
    }
    if (url.includes('/api/v1/configurator/communities')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
      });
    }
    return Promise.reject(new Error('Unknown url: ' + url));
  }) as unknown as typeof fetch;
});

test('renders app header', async () => {
  render(<App />);
  const headerElement = await screen.findByText(/Tacito Square Configurator/i);
  expect(headerElement).toBeInTheDocument();
});
