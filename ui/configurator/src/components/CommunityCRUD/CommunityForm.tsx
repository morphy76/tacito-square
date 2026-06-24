import React, { useState } from 'react';

export interface CommunityPayload {
  name: string;
  description: string;
}

interface CommunityFormProps {
  initialData?: CommunityPayload;
  onSave: (payload: CommunityPayload) => void;
}

export default function CommunityForm({ initialData, onSave }: CommunityFormProps) {
  const [name, setName] = useState(initialData?.name || '');
  const [description, setDescription] = useState(initialData?.description || '');
  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const newErrors: Record<string, string> = {};

    if (!name.trim()) {
      newErrors.name = 'Community Name is required';
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setErrors({});
    onSave({
      name,
      description,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="community-form" style={{ width: '100%', textAlign: 'left' }}>
      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="community-name" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Community Name</label>
        <input
          id="community-name"
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

      <div style={{ marginBottom: '24px' }}>
        <label htmlFor="community-description" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Description</label>
        <textarea
          id="community-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={4}
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

      <button type="submit" className="nav-btn submit-btn" style={{ width: '100%' }}>
        <span className="btn-label">
          <span className="btn-title">Save Community</span>
          <span className="btn-subtitle">Commit config changes</span>
        </span>
        <span className="btn-arrow">→</span>
      </button>
    </form>
  );
}
