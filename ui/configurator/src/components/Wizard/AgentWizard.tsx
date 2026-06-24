import { useState } from 'react';
import { AgentPayload } from '../AgentCRUD/AgentForm';

interface AgentWizardProps {
  onSave: (payload: AgentPayload) => void;
  onCancel: () => void;
}

export default function AgentWizard({ onSave, onCancel }: AgentWizardProps) {
  const [step, setStep] = useState(1);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [model, setModel] = useState('');
  const [prompt, setPrompt] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleNext = () => {
    if (step === 1 && !name.trim()) {
      setError('Name is required');
      return;
    }
    if (step === 2 && !model) {
      setError('Model is required');
      return;
    }
    setError(null);
    setStep((prev) => prev + 1);
  };

  const handleBack = () => {
    setError(null);
    setStep((prev) => prev - 1);
  };

  const handleFinish = () => {
    onSave({
      name,
      description,
      model,
      system_prompt: prompt,
    });
  };

  return (
    <div className="wizard-container" style={{ width: '100%', textAlign: 'left' }}>
      <div className="stepper" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '24px' }}>
        <div style={{ fontWeight: step === 1 ? 'bold' : 'normal', color: step === 1 ? '#66fcf1' : '#94a3b8' }}>1. Metadata</div>
        <div style={{ fontWeight: step === 2 ? 'bold' : 'normal', color: step === 2 ? '#66fcf1' : '#94a3b8' }}>2. Model</div>
        <div style={{ fontWeight: step === 3 ? 'bold' : 'normal', color: step === 3 ? '#66fcf1' : '#94a3b8' }}>3. System Prompt</div>
      </div>

      {error && (
        <div style={{ color: '#ff5e62', marginBottom: '16px', fontSize: '0.9rem' }}>
          {error}
        </div>
      )}

      {step === 1 && (
        <div className="wizard-step">
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Agent Name</label>
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
            <input
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
        </div>
      )}

      {step === 2 && (
        <div className="wizard-step">
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Select Model</label>
            <select
              value={model}
              onChange={(e) => setModel(e.target.value)}
              style={{
                width: '100%',
                padding: '10px',
                borderRadius: '8px',
                backgroundColor: '#151821',
                color: '#f2f5f9',
                border: '1px solid rgba(255,255,255,0.08)',
              }}
            >
              <option value="">Select a model</option>
              <option value="gpt-4o">gpt-4o</option>
              <option value="gpt-4-turbo">gpt-4-turbo</option>
              <option value="claude-3-opus">claude-3-opus</option>
            </select>
          </div>
        </div>
      )}

      {step === 3 && (
        <div className="wizard-step">
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>System Prompt</label>
            <textarea
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
          onClick={step === 3 ? handleFinish : handleNext}
          className="nav-btn"
          style={{ width: '45%' }}
        >
          <span className="btn-label">
            <span className="btn-title">{step === 3 ? 'Finish' : 'Next'}</span>
          </span>
          <span className="btn-arrow">→</span>
        </button>
      </div>
    </div>
  );
}
