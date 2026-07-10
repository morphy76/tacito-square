import { render, screen, fireEvent } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import CommunityForm from './CommunityForm';

test('CommunityForm validates fields and submits correct payload', () => {
  const handleSave = vi.fn();
  render(<CommunityForm onSave={handleSave} />);

  const submitButton = screen.getByRole('button', { name: /Save Community/i });

  // Try submitting empty
  fireEvent.click(submitButton);
  expect(screen.getByText(/Community Name is required/i)).toBeInTheDocument();
  expect(screen.getByText(/Topology is required/i)).toBeInTheDocument();
  expect(handleSave).not.toHaveBeenCalled();

  // Fill in fields
  const nameInput = screen.getByLabelText(/Community Name/i);
  const descriptionInput = screen.getByLabelText(/Description/i);
  const topologySelect = screen.getByLabelText(/Topology/i);
  const configInput = screen.getByLabelText(/Configuration/i);

  fireEvent.change(nameInput, { target: { value: 'Test Community' } });
  fireEvent.change(descriptionInput, { target: { value: 'A sample community.' } });
  fireEvent.change(topologySelect, { target: { value: 'standalone' } });
  fireEvent.change(configInput, { target: { value: '{"key": "value"}' } });

  // Submit filled form
  fireEvent.click(submitButton);
  expect(handleSave).toHaveBeenCalledWith({
    name: 'Test Community',
    description: 'A sample community.',
    topology: 'standalone',
    configuration: { key: 'value' },
  });
});
