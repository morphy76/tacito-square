import { useState } from 'react';
import { AgentPayload, WizardOptions } from '../AgentCRUD/AgentForm';
import { getApiUrl } from '../../utils/api';

interface AgentWizardProps {
  options: WizardOptions | null;
  onSave: (payload: AgentPayload) => void;
  onCancel: () => void;
  onRefreshOptions?: () => Promise<void>;
}

export default function AgentWizard({ options, onSave, onCancel, onRefreshOptions }: AgentWizardProps) {
  const [step, setStep] = useState(1);
  
  // Step 1: Metadata
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [role, setRole] = useState('');
  
  // Step 2: Brain & Prompts
  const [binding, setBinding] = useState('');
  const [temp, setTemp] = useState(0.7);
  const [tokens, setTokens] = useState(1024);
  const [promptTemplate, setPromptTemplate] = useState('');
  
  // Inline Brain creation
  const [showNewBrainForm, setShowNewBrainForm] = useState(false);
  const [newBrainName, setNewBrainName] = useState('');
  const [newBrainProvider, setNewBrainProvider] = useState('openai');
  const [newBrainBaseURL, setNewBrainBaseURL] = useState('');
  const [newBrainModel, setNewBrainModel] = useState('');

  // Inline Prompt creation
  const [showNewPromptForm, setShowNewPromptForm] = useState(false);
  const [newPromptName, setNewPromptName] = useState('');
  const [newPromptContent, setNewPromptContent] = useState('');

  // Step 3: Memory & Capabilities
  const [stmNs, setStmNs] = useState('agent');
  const [stmTtl, setStmTtl] = useState(3600);
  const [ltmEnabled, setLtmEnabled] = useState(true);
  const [ltmCol, setLtmCol] = useState('memory');
  const [ltmDim, setLtmDim] = useState(1536);
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [tier, setTier] = useState('cpu');
  const mcpClients: string[] = [];

  // Inline Skill creation
  const [showNewSkillForm, setShowNewSkillForm] = useState(false);
  const [newSkillName, setNewSkillName] = useState('');
  const [newSkillDesc, setNewSkillDesc] = useState('');
  const [newSkillContent, setNewSkillContent] = useState('');

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

  const handleCreateBrain = async () => {
    if (!newBrainName || !newBrainProvider || !newBrainModel) {
      setError('Brain name, provider, and default model are required');
      return;
    }
    try {
      const res = await fetch(getApiUrl('/api/v1/configurator/llm-bindings'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: newBrainName,
          provider: newBrainProvider,
          api_base_url: newBrainBaseURL || undefined,
          default_model: newBrainModel,
          api_key_secret_ref: 'placeholder',
        }),
      });
      if (!res.ok) throw new Error('Failed to create brain');
      const data = await res.json();
      if (onRefreshOptions) {
        await onRefreshOptions();
      }
      setBinding(data.id);
      setShowNewBrainForm(false);
      setNewBrainName('');
      setNewBrainBaseURL('');
      setNewBrainModel('');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error creating brain');
    }
  };

  const handleCreatePrompt = async () => {
    if (!newPromptName || !newPromptContent) {
      setError('Prompt name and content are required');
      return;
    }
    try {
      const res = await fetch(getApiUrl('/api/v1/configurator/prompts'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: newPromptName,
          content: newPromptContent,
          status: 'active',
        }),
      });
      if (!res.ok) throw new Error('Failed to create prompt template');
      const data = await res.json();
      if (onRefreshOptions) {
        await onRefreshOptions();
      }
      setPromptTemplate(data.id);
      setShowNewPromptForm(false);
      setNewPromptName('');
      setNewPromptContent('');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error creating prompt');
    }
  };

  const handlePromptUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (event) => {
      setNewPromptContent(event.target?.result as string || '');
    };
    reader.readAsText(file);
  };

  const handleCreateSkill = async () => {
    if (!newSkillName || !newSkillContent) {
      setError('Skill name and content are required');
      return;
    }
    try {
      const res = await fetch(getApiUrl('/api/v1/configurator/skills'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: newSkillName,
          description: newSkillDesc,
          content: newSkillContent,
        }),
      });
      if (!res.ok) throw new Error('Failed to create skill');
      const data = await res.json();
      if (onRefreshOptions) {
        await onRefreshOptions();
      }
      setSelectedSkills(prev => [...prev, data.id]);
      setShowNewSkillForm(false);
      setNewSkillName('');
      setNewSkillDesc('');
      setNewSkillContent('');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error creating skill');
    }
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
      long_term_memory: ltmEnabled ? {
        collection_name: ltmCol,
        vector_dimension: Number(ltmDim),
      } : {
        collection_name: '',
        vector_dimension: 0,
      },
      skills: selectedSkills,
      prompt_template: promptTemplate,
      tier,
      mcp_clients: mcpClients.map(id => ({ client_id: id })),
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
            <label htmlFor="agent-wizard-name" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Agent Name</label>
            <input
              id="agent-wizard-name"
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
            <label htmlFor="agent-wizard-description" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Description</label>
            <input
              id="agent-wizard-description"
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
            <label htmlFor="agent-wizard-role" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Role</label>
            <input
              id="agent-wizard-role"
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
            <label htmlFor="agent-wizard-binding" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LLM Binding</label>
            <div style={{ display: 'flex', gap: '8px' }}>
              <select
                id="agent-wizard-binding"
                value={binding}
                onChange={(e) => setBinding(e.target.value)}
                style={{
                  flex: 1,
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
              <button
                type="button"
                onClick={() => setShowNewBrainForm(!showNewBrainForm)}
                className="nav-btn"
                style={{ padding: '0 12px', fontSize: '0.9rem' }}
              >
                {showNewBrainForm ? 'Cancel' : '+ New Brain'}
              </button>
            </div>
          </div>

          {showNewBrainForm && (
            <div style={{ border: '1px solid rgba(255,255,255,0.08)', borderRadius: '8px', padding: '16px', marginBottom: '16px', backgroundColor: '#1e2230' }}>
              <h5 style={{ marginTop: 0, marginBottom: '12px', color: '#66fcf1' }}>Create New Brain (LLM Binding)</h5>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Name</label>
                <input
                  type="text"
                  value={newBrainName}
                  onChange={(e) => setNewBrainName(e.target.value)}
                  placeholder="e.g. My OpenAI Binding"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
                />
              </div>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Provider</label>
                <select
                  value={newBrainProvider}
                  onChange={(e) => setNewBrainProvider(e.target.value)}
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
                >
                  <option value="openai">openai</option>
                  <option value="anthropic">anthropic</option>
                  <option value="local">local</option>
                </select>
              </div>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>API Base URL</label>
                <input
                  type="text"
                  value={newBrainBaseURL}
                  onChange={(e) => setNewBrainBaseURL(e.target.value)}
                  placeholder="https://api.openai.com/v1"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
                />
              </div>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Default Model</label>
                <input
                  type="text"
                  value={newBrainModel}
                  onChange={(e) => setNewBrainModel(e.target.value)}
                  placeholder="gpt-4o"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
                />
              </div>
              <button
                type="button"
                onClick={handleCreateBrain}
                className="nav-btn"
                style={{ width: '100%', padding: '8px' }}
              >
                Save Brain
              </button>
            </div>
          )}

          <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
            <div style={{ flex: 1 }}>
              <label htmlFor="agent-wizard-temp" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Temperature</label>
              <input
                id="agent-wizard-temp"
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
              <label htmlFor="agent-wizard-tokens" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Max Tokens</label>
              <input
                id="agent-wizard-tokens"
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
            <label htmlFor="agent-wizard-prompt" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Prompt Template</label>
            <div style={{ display: 'flex', gap: '8px' }}>
              <select
                id="agent-wizard-prompt"
                value={promptTemplate}
                onChange={(e) => setPromptTemplate(e.target.value)}
                style={{
                  flex: 1,
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
              <button
                type="button"
                onClick={() => setShowNewPromptForm(!showNewPromptForm)}
                className="nav-btn"
                style={{ padding: '0 12px', fontSize: '0.9rem' }}
              >
                {showNewPromptForm ? 'Cancel' : '+ New Prompt'}
              </button>
            </div>
          </div>

          {showNewPromptForm && (
            <div style={{ border: '1px solid rgba(255,255,255,0.08)', borderRadius: '8px', padding: '16px', marginBottom: '16px', backgroundColor: '#1e2230' }}>
              <h5 style={{ marginTop: 0, marginBottom: '12px', color: '#66fcf1' }}>Create/Upload Prompt Template</h5>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Name</label>
                <input
                  type="text"
                  value={newPromptName}
                  onChange={(e) => setNewPromptName(e.target.value)}
                  placeholder="e.g. Code Analyst Prompt"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
                />
              </div>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Content</label>
                <textarea
                  value={newPromptContent}
                  onChange={(e) => setNewPromptContent(e.target.value)}
                  placeholder="Write prompt instructions..."
                  rows={4}
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)', fontFamily: 'monospace' }}
                />
              </div>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Upload Prompt File</label>
                <input
                  type="file"
                  accept=".txt,.md"
                  onChange={handlePromptUpload}
                  style={{ color: '#94a3b8', fontSize: '0.85rem' }}
                />
              </div>
              <button
                type="button"
                onClick={handleCreatePrompt}
                className="nav-btn"
                style={{ width: '100%', padding: '8px' }}
              >
                Save Prompt
              </button>
            </div>
          )}

          <div style={{ marginBottom: '16px' }}>
            <label htmlFor="agent-wizard-tier" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Deployment Tier</label>
            <select
              id="agent-wizard-tier"
              value={tier}
              onChange={(e) => setTier(e.target.value)}
              style={{
                width: '100%',
                padding: '10px',
                borderRadius: '8px',
                backgroundColor: '#151821',
                color: '#f2f5f9',
                border: '1px solid rgba(255,255,255,0.08)',
              }}
            >
              <option value="cpu">cpu</option>
              <option value="gpu">gpu</option>
              <option value="high-memory">high-memory</option>
            </select>
          </div>
        </div>
      )}

      {step === 3 && (
        <div className="wizard-step">
          <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
            <div style={{ flex: 1 }}>
              <label htmlFor="agent-wizard-stm-ns" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>STM Namespace</label>
              <input
                id="agent-wizard-stm-ns"
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
              <label htmlFor="agent-wizard-stm-ttl" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>STM TTL (seconds)</label>
              <input
                id="agent-wizard-stm-ttl"
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

          <div style={{ marginBottom: '16px', display: 'flex', alignItems: 'center' }}>
            <input
              id="agent-wizard-ltm-enabled"
              type="checkbox"
              checked={ltmEnabled}
              onChange={(e) => setLtmEnabled(e.target.checked)}
              style={{ marginRight: '8px' }}
            />
            <label htmlFor="agent-wizard-ltm-enabled" style={{ fontWeight: 600, cursor: 'pointer' }}>Enable Long-Term Memory (LTM)</label>
          </div>

          {ltmEnabled && (
            <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
              <div style={{ flex: 1 }}>
                <label htmlFor="agent-wizard-ltm-col" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LTM Collection</label>
                <input
                  id="agent-wizard-ltm-col"
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
                <label htmlFor="agent-wizard-ltm-dim" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LTM Dimension</label>
                <input
                  id="agent-wizard-ltm-dim"
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
          )}

          <div style={{ marginBottom: '16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
              <label style={{ fontWeight: 600 }}>Capabilities (Skills)</label>
              <button
                type="button"
                onClick={() => setShowNewSkillForm(!showNewSkillForm)}
                className="nav-btn"
                style={{ padding: '4px 10px', fontSize: '0.8rem' }}
              >
                {showNewSkillForm ? 'Cancel' : '+ Create Skill'}
              </button>
            </div>

            {showNewSkillForm && (
              <div style={{ border: '1px solid rgba(255,255,255,0.08)', borderRadius: '8px', padding: '16px', marginBottom: '16px', backgroundColor: '#1e2230' }}>
                <h5 style={{ marginTop: 0, marginBottom: '12px', color: '#66fcf1' }}>Create New Skill</h5>
                <div style={{ marginBottom: '12px' }}>
                  <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Name</label>
                  <input
                    type="text"
                    value={newSkillName}
                    onChange={(e) => setNewSkillName(e.target.value)}
                    placeholder="e.g. WebSearcher"
                    style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
                  />
                </div>
                <div style={{ marginBottom: '12px' }}>
                  <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Description</label>
                  <input
                    type="text"
                    value={newSkillDesc}
                    onChange={(e) => setNewSkillDesc(e.target.value)}
                    placeholder="Describe what the skill does..."
                    style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
                  />
                </div>
                <div style={{ marginBottom: '12px' }}>
                  <label style={{ display: 'block', marginBottom: '4px', fontSize: '0.85rem' }}>Content</label>
                  <textarea
                    value={newSkillContent}
                    onChange={(e) => setNewSkillContent(e.target.value)}
                    placeholder="Skill code/instructions..."
                    rows={3}
                    style={{ width: '100%', padding: '8px', borderRadius: '4px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)', fontFamily: 'monospace' }}
                  />
                </div>
                <button
                  type="button"
                  onClick={handleCreateSkill}
                  className="nav-btn"
                  style={{ width: '100%', padding: '8px' }}
                >
                  Save Skill
                </button>
              </div>
            )}

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

          <div style={{ marginBottom: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
              <input
                id="agent-wizard-mcp-enabled"
                type="checkbox"
                checked={false}
                disabled
                style={{ marginRight: '8px' }}
              />
              <label htmlFor="agent-wizard-mcp-enabled" style={{ fontWeight: 600, color: '#94a3b8' }}>Enable MCP Client Connections (Disabled)</label>
            </div>
            <span style={{ display: 'block', color: '#ff5e62', fontSize: '0.8rem', marginBottom: '8px' }}>
              ⚠️ MCP client integration is currently disabled.
            </span>
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
