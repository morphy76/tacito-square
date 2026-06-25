import React, { useState } from 'react';
import { getApiUrl } from '../../App';

export interface AgentPayload {
  name: string;
  description: string;
  role: string;
  brain: {
    llm_binding_id: string;
    temperature: number;
    max_tokens: number;
  };
  short_term_memory: {
    key_namespace: string;
    ttl_seconds: number;
  };
  long_term_memory: {
    collection_name: string;
    vector_dimension: number;
  };
  skills: string[];
  prompt_template: string;
  tier?: string;
  mcp_clients?: Array<{ client_id: string }>;
}

export interface WizardOptions {
  llm_bindings: Array<{ id: string; name: string }>;
  skills: Array<{ id: string; name: string }>;
  prompts: Array<{ id: string; name: string }>;
  mcp_servers: Array<{ id: string; name: string }>;
}

interface AgentFormProps {
  initialData?: AgentPayload;
  options: WizardOptions | null;
  onSave: (payload: AgentPayload) => void;
  onRefreshOptions?: () => Promise<void>;
}

export default function AgentForm({ initialData, options, onSave, onRefreshOptions }: AgentFormProps) {
  const [name, setName] = useState(initialData?.name || '');
  const [description, setDescription] = useState(initialData?.description || '');
  const [role, setRole] = useState(initialData?.role || '');
  const [binding, setBinding] = useState(initialData?.brain?.llm_binding_id || '');
  const [temp, setTemp] = useState(initialData?.brain?.temperature ?? 0.7);
  const [tokens, setTokens] = useState(initialData?.brain?.max_tokens ?? 1024);
  const [stmNs, setStmNs] = useState(initialData?.short_term_memory?.key_namespace || 'agent');
  const [stmTtl, setStmTtl] = useState(initialData?.short_term_memory?.ttl_seconds ?? 3600);
  
  const [ltmEnabled, setLtmEnabled] = useState(() => {
    return !!(initialData?.long_term_memory?.collection_name && initialData?.long_term_memory?.vector_dimension);
  });
  const [ltmCol, setLtmCol] = useState(initialData?.long_term_memory?.collection_name || 'memory');
  const [ltmDim, setLtmDim] = useState(initialData?.long_term_memory?.vector_dimension ?? 1536);
  
  const [promptTemplate, setPromptTemplate] = useState(initialData?.prompt_template || '');
  const [selectedSkills, setSelectedSkills] = useState<string[]>(initialData?.skills || []);
  const [tier, setTier] = useState(initialData?.tier || 'cpu');
  const [mcpClients, setMcpClients] = useState<string[]>(() => {
    return initialData?.mcp_clients?.map(c => c.client_id) || [];
  });

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

  // Inline Skill creation
  const [showNewSkillForm, setShowNewSkillForm] = useState(false);
  const [newSkillName, setNewSkillName] = useState('');
  const [newSkillDesc, setNewSkillDesc] = useState('');
  const [newSkillContent, setNewSkillContent] = useState('');

  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleToggleSkill = (skillId: string) => {
    setSelectedSkills((prev) =>
      prev.includes(skillId) ? prev.filter((id) => id !== skillId) : [...prev, skillId]
    );
  };

  const handleCreateBrain = async () => {
    if (!newBrainName || !newBrainProvider || !newBrainModel) {
      setErrors(prev => ({ ...prev, brainForm: 'Brain name, provider, and default model are required' }));
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
      setErrors(prev => {
        const { brainForm, ...rest } = prev;
        return rest;
      });
    } catch (err) {
      setErrors(prev => ({ ...prev, brainForm: err instanceof Error ? err.message : 'Error' }));
    }
  };

  const handleCreatePrompt = async () => {
    if (!newPromptName || !newPromptContent) {
      setErrors(prev => ({ ...prev, promptForm: 'Prompt name and content are required' }));
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
      setErrors(prev => {
        const { promptForm, ...rest } = prev;
        return rest;
      });
    } catch (err) {
      setErrors(prev => ({ ...prev, promptForm: err instanceof Error ? err.message : 'Error' }));
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
      setErrors(prev => ({ ...prev, skillForm: 'Skill name and content are required' }));
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
      setErrors(prev => {
        const { skillForm, ...rest } = prev;
        return rest;
      });
    } catch (err) {
      setErrors(prev => ({ ...prev, skillForm: err instanceof Error ? err.message : 'Error' }));
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const newErrors: Record<string, string> = {};

    if (!name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (!binding) {
      newErrors.binding = 'LLM Binding is required';
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setErrors({});
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
    <form onSubmit={handleSubmit} className="agent-form" style={{ width: '100%', textAlign: 'left' }}>
      <h4 style={{ fontFamily: 'Outfit', color: '#66fcf1', borderBottom: '1px solid rgba(255,255,255,0.08)', paddingBottom: '8px', marginBottom: '16px' }}>Metadata</h4>
      
      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-name" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Agent Name</label>
        <input
          id="agent-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: errors.name ? '1px solid #ff5e62' : '1px solid rgba(255,255,255,0.08)' }}
        />
        {errors.name && <span style={{ color: '#ff5e62', fontSize: '0.85rem' }}>{errors.name}</span>}
      </div>

      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-description" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Description</label>
        <input
          id="agent-description"
          type="text"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
        />
      </div>

      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-role" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Role</label>
        <input
          id="agent-role"
          type="text"
          value={role}
          onChange={(e) => setRole(e.target.value)}
          style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
        />
      </div>

      <h4 style={{ fontFamily: 'Outfit', color: '#66fcf1', borderBottom: '1px solid rgba(255,255,255,0.08)', paddingBottom: '8px', marginBottom: '16px', marginTop: '24px' }}>Brain Configuration</h4>

      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-binding" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LLM Binding</label>
        <div style={{ display: 'flex', gap: '8px' }}>
          <select
            id="agent-binding"
            value={binding}
            onChange={(e) => setBinding(e.target.value)}
            style={{ flex: 1, padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: errors.binding ? '1px solid #ff5e62' : '1px solid rgba(255,255,255,0.08)' }}
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
        {errors.binding && <span style={{ color: '#ff5e62', fontSize: '0.85rem' }}>{errors.binding}</span>}
      </div>

      {showNewBrainForm && (
        <div style={{ border: '1px solid rgba(255,255,255,0.08)', borderRadius: '8px', padding: '16px', marginBottom: '16px', backgroundColor: '#1e2230' }}>
          <h5 style={{ marginTop: 0, marginBottom: '12px', color: '#66fcf1' }}>Create New Brain (LLM Binding)</h5>
          {errors.brainForm && <div style={{ color: '#ff5e62', fontSize: '0.85rem', marginBottom: '8px' }}>{errors.brainForm}</div>}
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
          <label htmlFor="agent-temp" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Temperature</label>
          <input
            id="agent-temp"
            type="number"
            step="0.1"
            min="0"
            max="2"
            value={temp}
            onChange={(e) => setTemp(Number(e.target.value))}
            style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
          />
        </div>
        <div style={{ flex: 1 }}>
          <label htmlFor="agent-tokens" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Max Tokens</label>
          <input
            id="agent-tokens"
            type="number"
            value={tokens}
            onChange={(e) => setTokens(Number(e.target.value))}
            style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
          />
        </div>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <label htmlFor="agent-prompt-template" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Prompt Template</label>
        <div style={{ display: 'flex', gap: '8px' }}>
          <select
            id="agent-prompt-template"
            value={promptTemplate}
            onChange={(e) => setPromptTemplate(e.target.value)}
            style={{ flex: 1, padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
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
          {errors.promptForm && <div style={{ color: '#ff5e62', fontSize: '0.85rem', marginBottom: '8px' }}>{errors.promptForm}</div>}
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
        <label htmlFor="agent-tier" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>Deployment Tier</label>
        <select
          id="agent-tier"
          value={tier}
          onChange={(e) => setTier(e.target.value)}
          style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
        >
          <option value="cpu">cpu</option>
          <option value="gpu">gpu</option>
          <option value="high-memory">high-memory</option>
        </select>
      </div>

      <h4 style={{ fontFamily: 'Outfit', color: '#66fcf1', borderBottom: '1px solid rgba(255,255,255,0.08)', paddingBottom: '8px', marginBottom: '16px', marginTop: '24px' }}>Memory Systems</h4>

      <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
        <div style={{ flex: 1 }}>
          <label htmlFor="agent-stm-ns" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>STM Namespace</label>
          <input
            id="agent-stm-ns"
            type="text"
            value={stmNs}
            onChange={(e) => setStmNs(e.target.value)}
            style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
          />
        </div>
        <div style={{ flex: 1 }}>
          <label htmlFor="agent-stm-ttl" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>STM TTL</label>
          <input
            id="agent-stm-ttl"
            type="number"
            value={stmTtl}
            onChange={(e) => setStmTtl(Number(e.target.value))}
            style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
          />
        </div>
      </div>

      <div style={{ marginBottom: '16px', display: 'flex', alignItems: 'center' }}>
        <input
          id="agent-ltm-enabled"
          type="checkbox"
          checked={ltmEnabled}
          onChange={(e) => setLtmEnabled(e.target.checked)}
          style={{ marginRight: '8px' }}
        />
        <label htmlFor="agent-ltm-enabled" style={{ fontWeight: 600, cursor: 'pointer' }}>Enable Long-Term Memory (LTM)</label>
      </div>

      {ltmEnabled && (
        <div style={{ display: 'flex', gap: '16px', marginBottom: '16px' }}>
          <div style={{ flex: 1 }}>
            <label htmlFor="agent-ltm-col" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LTM Collection</label>
            <input
              id="agent-ltm-col"
              type="text"
              value={ltmCol}
              onChange={(e) => setLtmCol(e.target.value)}
              style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
            />
          </div>
          <div style={{ flex: 1 }}>
            <label htmlFor="agent-ltm-dim" style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>LTM Dimension</label>
            <input
              id="agent-ltm-dim"
              type="number"
              value={ltmDim}
              onChange={(e) => setLtmDim(Number(e.target.value))}
              style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
            />
          </div>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px', marginTop: '24px' }}>
        <h4 style={{ fontFamily: 'Outfit', color: '#66fcf1', borderBottom: '1px solid rgba(255,255,255,0.08)', paddingBottom: '8px', margin: 0, flex: 1 }}>Capabilities (Skills)</h4>
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
          {errors.skillForm && <div style={{ color: '#ff5e62', fontSize: '0.85rem', marginBottom: '8px' }}>{errors.skillForm}</div>}
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
      
      <div style={{ marginBottom: '24px', maxHeight: '150px', overflowY: 'auto', border: '1px solid rgba(255,255,255,0.08)', borderRadius: '8px', padding: '10px', backgroundColor: '#151821' }}>
        {skillsList.length === 0 ? (
          <span style={{ color: '#94a3b8', fontSize: '0.9rem' }}>No skills available</span>
        ) : (
          skillsList.map(s => (
            <div key={s.id} style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
              <input
                id={"skill-" + s.id}
                type="checkbox"
                checked={selectedSkills.includes(s.id)}
                onChange={() => handleToggleSkill(s.id)}
                style={{ marginRight: '8px' }}
              />
              <label htmlFor={"skill-" + s.id} style={{ color: '#f2f5f9', fontSize: '0.9rem', cursor: 'pointer' }}>{s.name}</label>
            </div>
          ))
        )}
      </div>

      <div style={{ marginBottom: '16px', marginTop: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
          <input
            id="agent-mcp-enabled"
            type="checkbox"
            checked={false}
            disabled
            style={{ marginRight: '8px' }}
          />
          <label htmlFor="agent-mcp-enabled" style={{ fontWeight: 600, color: '#94a3b8' }}>Enable MCP Client Connections (Disabled)</label>
        </div>
        <span style={{ display: 'block', color: '#ff5e62', fontSize: '0.8rem', marginBottom: '8px' }}>
          ⚠️ MCP client integration is currently disabled.
        </span>
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
