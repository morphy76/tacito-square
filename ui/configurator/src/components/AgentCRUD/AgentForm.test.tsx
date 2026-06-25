import { render, screen, fireEvent } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import AgentForm from './AgentForm';

const mockOptions = {
  llm_bindings: [{ id: 'binding-1', name: 'gpt-4o' }],
  skills: [{ id: 'skill-1', name: 'Web Search' }],
  prompts: [{ id: 'prompt-1', name: 'Default Agent' }],
};

test('AgentForm validates fields and submits correct payload', () => {
  const handleSave = vi.fn();
  render(<AgentForm onSave={handleSave} options={mockOptions} />);

  const submitButton = screen.getByRole('button', { name: /Save Agent/i });

  // Try submitting empty
  fireEvent.click(submitButton);
  expect(screen.getByText(/Name is required/i)).toBeInTheDocument();
  expect(screen.getByText(/LLM Binding is required/i)).toBeInTheDocument();
  expect(handleSave).not.toHaveBeenCalled();

  // Fill in fields
  const nameInput = screen.getByLabelText(/Agent Name/i);
  const bindingSelect = screen.getByLabelText(/LLM Binding/i);
  const promptSelect = screen.getByLabelText(/Prompt Template/i);
  const descriptionInput = screen.getByLabelText(/Description/i);
  const roleInput = screen.getByLabelText(/Role/i);
  const tempInput = screen.getByLabelText(/Temperature/i);
  const tokensInput = screen.getByLabelText(/Max Tokens/i);
  const stmNsInput = screen.getByLabelText(/STM Namespace/i);
  const stmTtlInput = screen.getByLabelText(/STM TTL/i);

  // Enable LTM to render inputs
  const ltmCheckbox = screen.getByLabelText(/Enable Long-Term Memory/i);
  fireEvent.click(ltmCheckbox);

  const ltmColInput = screen.getByLabelText(/LTM Collection/i);
  const ltmDimInput = screen.getByLabelText(/LTM Dimension/i);

  fireEvent.change(nameInput, { target: { value: 'Test Agent' } });
  fireEvent.change(bindingSelect, { target: { value: 'binding-1' } });
  fireEvent.change(promptSelect, { target: { value: 'prompt-1' } });
  fireEvent.change(descriptionInput, { target: { value: 'A test agent.' } });
  fireEvent.change(roleInput, { target: { value: 'Researcher' } });
  fireEvent.change(tempInput, { target: { value: '0.8' } });
  fireEvent.change(tokensInput, { target: { value: '2048' } });
  fireEvent.change(stmNsInput, { target: { value: 'test-stm' } });
  fireEvent.change(stmTtlInput, { target: { value: '600' } });
  fireEvent.change(ltmColInput, { target: { value: 'test-ltm' } });
  fireEvent.change(ltmDimInput, { target: { value: '1536' } });

  // Select skill
  const skillCheckbox = screen.getByLabelText(/Web Search/i);
  fireEvent.click(skillCheckbox);

  // Submit filled form
  fireEvent.click(submitButton);
  expect(handleSave).toHaveBeenCalledWith({
    name: 'Test Agent',
    description: 'A test agent.',
    role: 'Researcher',
    brain: {
      llm_binding_id: 'binding-1',
      temperature: 0.8,
      max_tokens: 2048,
    },
    short_term_memory: {
      key_namespace: 'test-stm',
      ttl_seconds: 600,
    },
    long_term_memory: {
      collection_name: 'test-ltm',
      vector_dimension: 1536,
    },
    skills: ['skill-1'],
    prompt_template: 'prompt-1',
    tier: 'cpu',
    mcp_clients: [],
  });
});
