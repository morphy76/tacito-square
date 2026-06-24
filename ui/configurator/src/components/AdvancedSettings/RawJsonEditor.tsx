import React, { useState } from 'react';

interface RawJsonEditorProps {
  initialValue: unknown;
  onSave: (val: unknown) => void;
}

export default function RawJsonEditor({ initialValue, onSave }: RawJsonEditorProps) {
  const [jsonText, setJsonText] = useState<string>(() => JSON.stringify(initialValue, null, 2));
  const [error, setError] = useState<string | null>(null);

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    setJsonText(value);

    try {
      if (value.trim() === '') {
        setError('JSON cannot be empty');
        return;
      }
      JSON.parse(value);
      setError(null);
    } catch (err) {
      setError('Invalid JSON format: ' + (err instanceof Error ? err.message : 'Syntax Error'));
    }
  };

  const handleSaveClick = () => {
    if (error) return;
    try {
      const parsed = JSON.parse(jsonText);
      onSave(parsed);
    } catch {
      setError('Invalid JSON format');
    }
  };

  return (
    <div className="raw-json-editor">
      <h3>Advanced Raw JSON Configuration</h3>
      <textarea
        placeholder="Enter raw JSON configuration"
        value={jsonText}
        onChange={handleChange}
        className={`json-textarea ${error ? 'has-error' : ''}`}
        rows={15}
        cols={50}
        style={{
          width: '100%',
          fontFamily: 'monospace',
          padding: '12px',
          borderRadius: '8px',
          backgroundColor: '#151821',
          color: '#f2f5f9',
          border: error ? '1px solid #ff5e62' : '1px solid rgba(255,255,255,0.08)',
        }}
      />
      {error && (
        <div className="error-message" style={{ color: '#ff5e62', marginTop: '8px', fontSize: '0.9rem', textAlign: 'left' }}>
          {error}
        </div>
      )}
      <button
        onClick={handleSaveClick}
        disabled={!!error}
        className="nav-btn save-btn"
        style={{ marginTop: '16px', opacity: error ? 0.5 : 1, cursor: error ? 'not-allowed' : 'pointer' }}
      >
        <span className="btn-label">
          <span className="btn-title">Save Changes</span>
          <span className="btn-subtitle">Apply raw configuration</span>
        </span>
        <span className="btn-arrow">→</span>
      </button>
    </div>
  );
}
