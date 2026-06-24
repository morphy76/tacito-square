import { useState } from 'react';
import { CommunityPayload } from '../CommunityCRUD/CommunityForm';

interface CommunityWizardProps {
  agentsList: Array<{ id: string; name: string }>;
  onSave: (payload: CommunityPayload & { agents: string[] }) => void;
  onCancel: () => void;
}

export default function CommunityWizard({ agentsList, onSave, onCancel }: CommunityWizardProps) {
  const [step, setStep] = useState(1);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  const handleNext = () => {
    if (step === 1 && !name.trim()) {
      setError('Community Name is required');
      return;
    }
    setError(null);
    setStep((prev) => prev + 1);
  };

  const handleBack = () => {
    setError(null);
    setStep((prev) => prev - 1);
  };

  const handleToggleAgent = (agentId: string) => {
    setSelectedAgents((prev) =>
      prev.includes(agentId) ? prev.filter((id) => id !== agentId) : [...prev, agentId]
    );
  };

  const handleFinish = () => {
    onSave({
      name,
      description,
      agents: selectedAgents,
    });
  };

  return (
    <div className="wizard-container" style={{ width: '100%', textAlign: 'left' }}>
      <div className="stepper" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '24px' }}>
        <div style={{ fontWeight: step === 1 ? 'bold' : 'normal', color: step === 1 ? '#66fcf1' : '#94a3b8' }}>1. Metadata</div>
        <div style={{ fontWeight: step === 2 ? 'bold' : 'normal', color: step === 2 ? '#66fcf1' : '#94a3b8' }}>2. Assign Agents</div>
      </div>

      {error && (
        <div style={{ color: '#ff5e62', marginBottom: '16px', fontSize: '0.9rem' }}>
          {error}
        </div>
      )}

      {step === 1 && (
        <div className="wizard-step">
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Community Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
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
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Description</label>
            <textarea
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
        </div>
      )}

      {step === 2 && (
        <div className="wizard-step">
          <label style={{ display: 'block', marginBottom: '12px', fontWeight: 600 }}>Select Agents for this Community</label>
          <div className="agents-checkbox-list" style={{ maxHeight: '200px', overflowY: 'auto' }}>
            {agentsList.map((agent) => (
              <div key={agent.id} style={{ display: 'flex', alignItems: 'center', marginBottom: '10px' }}>
                <input
                  type="checkbox"
                  id={`checkbox-${agent.id}`}
                  checked={selectedAgents.includes(agent.id)}
                  onChange={() => handleToggleAgent(agent.id)}
                  style={{ marginRight: '10px', width: '18px', height: '18px', cursor: 'pointer' }}
                />
                <label htmlFor={`checkbox-${agent.id}`} style={{ cursor: 'pointer' }}>{agent.name}</label>
              </div>
            ))}
            {agentsList.length === 0 && <p style={{ color: '#94a3b8' }}>No agents available to assign.</p>}
          </div>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '24px' }}>
        <button
          onClick={step === 1 ? onCancel : handleBack}
          className="nav-btn"
          style={{ width: '45%' }}
        >
          <span className="btn-label">
            <span className="btn-title">{step === 1 ? 'Cancel' : 'Back'}</span>
          </span>
        </button>
        <button
          onClick={step === 2 ? handleFinish : handleNext}
          className="nav-btn"
          style={{ width: '45%' }}
        >
          <span className="btn-label">
            <span className="btn-title">{step === 2 ? 'Finish' : 'Next'}</span>
          </span>
          <span className="btn-arrow">→</span>
        </button>
      </div>
    </div>
  );
}
