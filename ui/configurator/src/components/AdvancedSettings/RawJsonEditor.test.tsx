import { render, screen, fireEvent } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import RawJsonEditor from './RawJsonEditor';

test('RawJsonEditor validates JSON and handles save', () => {
  const handleSave = vi.fn();
  render(
    <RawJsonEditor
      initialValue={{ name: 'Test' }}
      onSave={handleSave}
    />
  );

  const textarea = screen.getByPlaceholderText(/Enter raw JSON configuration/i) as HTMLTextAreaElement;
  const saveButton = screen.getByRole('button', { name: /Save Changes/i });

  // Verify initial value is formatted correctly
  expect(JSON.parse(textarea.value)).toEqual({ name: 'Test' });

  // Type invalid JSON
  fireEvent.change(textarea, { target: { value: '{ name: "invalid" }' } }); // Missing double quotes around keys
  expect(screen.getByText(/Invalid JSON format/i)).toBeInTheDocument();
  expect(saveButton).toBeDisabled();

  // Type valid JSON
  fireEvent.change(textarea, { target: { value: '{\n  "name": "updated"\n}' } });
  expect(screen.queryByText(/Invalid JSON format/i)).not.toBeInTheDocument();
  expect(saveButton).not.toBeDisabled();

  // Click Save
  fireEvent.click(saveButton);
  expect(handleSave).toHaveBeenCalledWith({ name: 'updated' });
});
