import { useState } from 'react';
import { CommunityPayload } from '../CommunityCRUD/CommunityForm';

interface CommunityWizardProps {
  onSave: (payload: CommunityPayload) => void;
  onCancel: () => void;
}

export default function CommunityWizard({ onSave, onCancel }: CommunityWizardProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [topology, setTopology] = useState('');
  const [configStr, setConfigStr] = useState('{}');
  const [error, setError] = useState<string | null>(null);

  const handleFinish = () => {
    if (!name.trim()) {
      setError('Community Name is required');
      return;
    }
    if (!topology) {
      setError('Topology is required');
      return;
    }

    let parsedConfig = {};
    if (configStr.trim()) {
      try {
        parsedConfig = JSON.parse(configStr);
      } catch (err) {
        setError('Invalid JSON format for Configuration');
        return;
      }
    }

    setError(null);
    onSave({
      name,
      description,
      topology,
      configuration: parsedConfig,
    });
  };

  return (
    <div className="wizard-container" style={{ width: '100%', textAlign: 'left' }}>
      {error && (
        <div style={{ color: '#ff5e62', marginBottom: '16px', fontSize: '0.9rem', padding: '8px 12px', backgroundColor: 'rgba(255, 94, 98, 0.1)', borderRadius: '8px', border: '1px solid rgba(255, 94, 98, 0.2)' }}>
          {error}
        </div>
      )}

      <div className="wizard-step">
        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Community Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Analysis Group"
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

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Topology</label>
          <select
            value={topology}
            onChange={(e) => setTopology(e.target.value)}
            style={{
              width: '100%',
              padding: '10px',
              borderRadius: '8px',
              backgroundColor: '#151821',
              color: '#f2f5f9',
              border: '1px solid rgba(255,255,255,0.08)',
            }}
          >
            <option value="">Select a topology</option>
            <option value="standalone">standalone</option>
            <option value="hub-spoke">hub-spoke</option>
          </select>
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={3}
            placeholder="Describe the purpose of this community..."
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
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Configuration (JSON)</label>
          <textarea
            value={configStr}
            onChange={(e) => setConfigStr(e.target.value)}
            rows={4}
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
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '24px' }}>
        <button
          onClick={onCancel}
          className="nav-btn"
          style={{ width: '45%' }}
        >
          <span className="btn-label">
            <span className="btn-title">Cancel</span>
          </span>
        </button>
        <button
          onClick={handleFinish}
          className="nav-btn"
          style={{ width: '45%' }}
        >
          <span className="btn-label">
            <span className="btn-title">Finish</span>
          </span>
          <span className="btn-arrow">→</span>
        </button>
      </div>
    </div>
  );
}
