import { useState } from 'react';
import { AgentPayload, WizardOptions } from '../AgentCRUD/AgentForm';

interface AgentWizardProps {
  options: WizardOptions | null;
  onSave: (payload: AgentPayload) => void;
  onCancel: () => void;
}

export default function AgentWizard({ options, onSave, onCancel }: AgentWizardProps) {
  const [step, setStep] = useState(1);
  
  // Step 1: Metadata
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [role, setRole] = useState('');
  
  // Step 2: Brain
  const [binding, setBinding] = useState('');
  const [temp, setTemp] = useState(0.7);
  const [tokens, setTokens] = useState(1024);
  const [promptTemplate, setPromptTemplate] = useState('');

  // Step 3: Memory & Capabilities
  const [stmNs, setStmNs] = useState('agent');
  const [stmTtl, setStmTtl] = useState(3600);
  const [ltmCol, setLtmCol] = useState('memory');
  const [ltmDim, setLtmDim] = useState(1536);
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);

  const [error, setError] = useState<string | null>(null);

  const handleNext = () => {
    if (step === 1) {
      if (!name.trim()) {
        setError('Agent Name is required');
        return;
      }
      setError(null);
      setStep(2);
    } else if (step === 2) {
      if (!binding) {
        setError('LLM Binding is required');
        return;
      }
      setError(null);
      setStep(3);
    }
  };

  const handleBack = () => {
    setError(null);
    setStep((prev) => prev - 1);
  };

  const handleToggleSkill = (skillId: string) => {
    setSelectedSkills((prev) =>
      prev.includes(skillId) ? prev.filter((id) => id !== skillId) : [...prev, skillId]
    );
  };

  const handleFinish = () => {
    onSave({
      name,
      description,
      role,
      brain: {
        llm_binding_id: binding,
        temperature: Number(temp),
        max_tokens: Number(tokens),
      },
      short_term_memory: {
        key_namespace: stmNs,
        ttl_seconds: Number(stmTtl),
      },
      long_term_memory: {
        collection_name: ltmCol,
        vector_dimension: Number(ltmDim),
      },
      skills: selectedSkills,
      prompt_template: promptTemplate,
    });
  };

  const bindingsList = options?.llm_bindings || [];
  const promptsList = options?.prompts || [];
  const skillsList = options?.skills || [];

  return (
    <div className="wizard-container" style={{ width: '100%', textAlign: 'left' }}>
      <div className="stepper" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '24px' }}>
        <div style={{ fontWeight: step === 1 ? 'bold' : 'normal', color: step === 1 ? '#66fcf1' : '#94a3b8', borderBottom: step === 1 ? '2px solid #66fcf1' : 'none', paddingBottom: '4px' }}>1. Metadata</div>
        <div style={{ fontWeight: step === 2 ? 'bold' : 'normal', color: step === 2 ? '#66fcf1' : '#94a3b8', borderBottom: step === 2 ? '2px solid #66fcf1' : 'none', paddingBottom: '4px' }}>2. Brain Config</div>
        <div style={{ fontWeight: step === 3 ? 'bold' : 'normal', color: step === 3 ? '#66fcf1' : '#94a3b8', borderBottom: step === 3 ? '2px solid #66fcf1' : 'none', paddingBottom: '4px' }}>3. Memory & Skills</div>
      </div>

      {error && (
        <div style={{ color: '#ff5e62', marginBottom: '16px', fontSize: '0.9rem', padding: '8px 12px', backgroundColor: 'rgba(255, 94, 98, 0.1)', borderRadius: '8px', border: '1px solid rgba(255, 94, 98, 0.2)' }}>
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
              placeholder="e.g. Research Specialist"
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
              placeholder="Describe the agent's main objective..."
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
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Role</label>
            <input
              type="text"
              value={role}
              onChange={(e) => setRole(e.target.value)}
              placeholder="e.g. Analyst"
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
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LLM Binding</label>
            <select
              value={binding}
              onChange={(e) => setBinding(e.target.value)}
              style={{
                width: '100%',
                padding: '10px',
                borderRadius: '8px',
                backgroundColor: '#151821',
                color: '#f2f5f9',
                border: '1px solid rgba(255,255,255,0.08)',
              }}
            >
              <option value="">Select a binding</option>
              {bindingsList.map(b => (
                <option key={b.id} value={b.id}>{b.name}</option>
              ))}
            </select>
          </div>

          <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Temperature</label>
              <input
                type="number"
                step="0.1"
                min="0"
                max="2"
                value={temp}
                onChange={(e) => setTemp(Number(e.target.value))}
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
            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Max Tokens</label>
              <input
                type="number"
                value={tokens}
                onChange={(e) => setTokens(Number(e.target.value))}
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

          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Prompt Template</label>
            <select
              value={promptTemplate}
              onChange={(e) => setPromptTemplate(e.target.value)}
              style={{
                width: '100%',
                padding: '10px',
                borderRadius: '8px',
                backgroundColor: '#151821',
                color: '#f2f5f9',
                border: '1px solid rgba(255,255,255,0.08)',
              }}
            >
              <option value="">Select a prompt template</option>
              {promptsList.map(p => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          </div>
        </div>
      )}

      {step === 3 && (
        <div className="wizard-step">
          <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>STM Namespace</label>
              <input
                type="text"
                value={stmNs}
                onChange={(e) => setStmNs(e.target.value)}
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
            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>STM TTL (seconds)</label>
              <input
                type="number"
                value={stmTtl}
                onChange={(e) => setStmTtl(Number(e.target.value))}
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

          <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LTM Collection</label>
              <input
                type="text"
                value={ltmCol}
                onChange={(e) => setLtmCol(e.target.value)}
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
            <div style={{ flex: 1 }}>
              <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LTM Dimension</label>
              <input
                type="number"
                value={ltmDim}
                onChange={(e) => setLtmDim(Number(e.target.value))}
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

          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Capabilities (Skills)</label>
            <div style={{ maxHeight: '120px', overflowY: 'auto', border: '1px solid rgba(255,255,255,0.08)', borderRadius: '8px', padding: '10px', backgroundColor: '#151821' }}>
              {skillsList.length === 0 ? (
                <span style={{ color: '#94a3b8', fontSize: '0.9rem' }}>No skills available</span>
              ) : (
                skillsList.map(s => (
                  <div key={s.id} style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
                    <input
                      id={"wizard-skill-" + s.id}
                      type="checkbox"
                      checked={selectedSkills.includes(s.id)}
                      onChange={() => handleToggleSkill(s.id)}
                      style={{ marginRight: '8px' }}
                    />
                    <label htmlFor={"wizard-skill-" + s.id} style={{ color: '#f2f5f9', fontSize: '0.9rem', cursor: 'pointer' }}>{s.name}</label>
                  </div>
                ))
              )}
            </div>
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
