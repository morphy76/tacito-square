import { render, screen, fireEvent } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import AgentForm from './AgentForm';

test('AgentForm validates fields and submits correct payload', () => {
  const handleSave = vi.fn();
  render(<AgentForm onSave={handleSave} />);

  const submitButton = screen.getByRole('button', { name: /Save Agent/i });

  // Try submitting empty
  fireEvent.click(submitButton);
  expect(screen.getByText(/Name is required/i)).toBeInTheDocument();
  expect(screen.getByText(/Model is required/i)).toBeInTheDocument();
  expect(handleSave).not.toHaveBeenCalled();

  // Fill in fields
  const nameInput = screen.getByLabelText(/Agent Name/i);
  const modelSelect = screen.getByLabelText(/Model/i);
  const promptTextarea = screen.getByLabelText(/System Prompt/i);

  fireEvent.change(nameInput, { target: { value: 'Test Agent' } });
  fireEvent.change(modelSelect, { target: { value: 'gpt-4o' } });
  fireEvent.change(promptTextarea, { target: { value: 'You are a test agent.' } });

  // Submit filled form
  fireEvent.click(submitButton);
  expect(handleSave).toHaveBeenCalledWith({
    name: 'Test Agent',
    model: 'gpt-4o',
    system_prompt: 'You are a test agent.',
    description: '',
  });
});
