import React, { useState, useEffect } from 'react';

interface RawJsonEditorProps {
  initialValue: unknown;
  onSave: (val: unknown) => void;
  activeSchema: string;
  onSchemaChange: (schema: string) => void;
}

export default function RawJsonEditor({ initialValue, onSave, activeSchema, onSchemaChange }: RawJsonEditorProps) {
  const [jsonText, setJsonText] = useState<string>(() => JSON.stringify(initialValue, null, 2));
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setJsonText(JSON.stringify(initialValue, null, 2));
    setError(null);
  }, [initialValue]);

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

      <div style={{ marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '10px' }}>
        <label htmlFor="schema-selector" style={{ fontWeight: 600, color: '#f2f5f9' }}>Schema Collection:</label>
        <select
          id="schema-selector"
          value={activeSchema}
          onChange={(e) => onSchemaChange(e.target.value)}
          style={{
            padding: '8px 12px',
            borderRadius: '8px',
            backgroundColor: '#151821',
            color: '#f2f5f9',
            border: '1px solid rgba(255,255,255,0.08)',
            cursor: 'pointer',
          }}
        >
          <option value="agents">Agents</option>
          <option value="communities">Communities</option>
          <option value="llm-bindings">LLM Bindings (Brains)</option>
          <option value="prompts">Prompt Templates</option>
          <option value="skills">Skills</option>
          <option value="mcp-servers">MCP Servers</option>
        </select>
      </div>

      <textarea
        placeholder="Enter raw JSON configuration"
        value={jsonText}
        onChange={handleChange}
        className={`json-textarea ${error ? 'has-error' : ''}`}
        rows={18}
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
