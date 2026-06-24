import React, { useState } from 'react';

export interface AgentPayload {
  name: string;
  model: string;
  system_prompt: string;
  description: string;
}

interface AgentFormProps {
  initialData?: AgentPayload;
  onSave: (payload: AgentPayload) => void;
}

export default function AgentForm({ initialData, onSave }: AgentFormProps) {
  const [name, setName] = useState(initialData?.name || '');
  const [model, setModel] = useState(initialData?.model || '');
  const [prompt, setPrompt] = useState(initialData?.system_prompt || '');
  const [description, setDescription] = useState(initialData?.description || '');
  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const newErrors: Record<string, string> = {};

    if (!name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (!model) {
      newErrors.model = 'Model is required';
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setErrors({});
    onSave({
      name,
      model,
      system_prompt: prompt,
      description,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="agent-form" style={{ width: '100%', textAlign: 'left' }}>
      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-name" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Agent Name</label>
        <input
          id="agent-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          style={{
            width: '100%',
            padding: '10px',
            borderRadius: '8px',
            backgroundColor: '#151821',
            color: '#f2f5f9',
            border: errors.name ? '1px solid #ff5e62' : '1px solid rgba(255,255,255,0.08)',
          }}
        />
        {errors.name && <span style={{ color: '#ff5e62', fontSize: '0.85rem' }}>{errors.name}</span>}
      </div>

      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-model" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Model</label>
        <select
          id="agent-model"
          value={model}
          onChange={(e) => setModel(e.target.value)}
          style={{
            width: '100%',
            padding: '10px',
            borderRadius: '8px',
            backgroundColor: '#151821',
            color: '#f2f5f9',
            border: errors.model ? '1px solid #ff5e62' : '1px solid rgba(255,255,255,0.08)',
          }}
        >
          <option value="">Select a model</option>
          <option value="gpt-4o">gpt-4o</option>
          <option value="gpt-4-turbo">gpt-4-turbo</option>
          <option value="gpt-3.5-turbo">gpt-3.5-turbo</option>
          <option value="claude-3-opus">claude-3-opus</option>
        </select>
        {errors.model && <span style={{ color: '#ff5e62', fontSize: '0.85rem' }}>{errors.model}</span>}
      </div>

      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-description" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Description</label>
        <input
          id="agent-description"
          type="text"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          style={{
            width: '100%',
            padding: '10px',
            borderRadius: '8px',
            backgroundColor: '#151821',
            color: '#f2f5f9',
            border: '1px solid rgba(255,255,255,0.08)',
          }}
        />
      </div>

      <div style={{ marginBottom: '24px' }}>
        <label htmlFor="agent-prompt" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>System Prompt</label>
        <textarea
          id="agent-prompt"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          rows={6}
          style={{
            width: '100%',
            padding: '10px',
            borderRadius: '8px',
            backgroundColor: '#151821',
            color: '#f2f5f9',
            border: '1px solid rgba(255,255,255,0.08)',
            fontFamily: 'monospace',
          }}
        />
      </div>

      <button type="submit" className="nav-btn submit-btn" style={{ width: '100%' }}>
        <span className="btn-label">
          <span className="btn-title">Save Agent</span>
          <span className="btn-subtitle">Commit config changes</span>
        </span>
        <span className="btn-arrow">→</span>
      </button>
    </form>
  );
}
