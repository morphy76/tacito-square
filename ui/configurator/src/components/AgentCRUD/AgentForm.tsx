import React, { useState } from 'react';

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
}

export interface WizardOptions {
  llm_bindings: Array<{ id: string; name: string }>;
  skills: Array<{ id: string; name: string }>;
  prompts: Array<{ id: string; name: string }>;
}

interface AgentFormProps {
  initialData?: AgentPayload;
  options: WizardOptions | null;
  onSave: (payload: AgentPayload) => void;
}

export default function AgentForm({ initialData, options, onSave }: AgentFormProps) {
  const [name, setName] = useState(initialData?.name || '');
  const [description, setDescription] = useState(initialData?.description || '');
  const [role, setRole] = useState(initialData?.role || '');
  const [binding, setBinding] = useState(initialData?.brain?.llm_binding_id || '');
  const [temp, setTemp] = useState(initialData?.brain?.temperature ?? 0.7);
  const [tokens, setTokens] = useState(initialData?.brain?.max_tokens ?? 1024);
  const [stmNs, setStmNs] = useState(initialData?.short_term_memory?.key_namespace || 'agent');
  const [stmTtl, setStmTtl] = useState(initialData?.short_term_memory?.ttl_seconds ?? 3600);
  const [ltmCol, setLtmCol] = useState(initialData?.long_term_memory?.collection_name || 'memory');
  const [ltmDim, setLtmDim] = useState(initialData?.long_term_memory?.vector_dimension ?? 1536);
  const [promptTemplate, setPromptTemplate] = useState(initialData?.prompt_template || '');
  const [selectedSkills, setSelectedSkills] = useState<string[]>(initialData?.skills || []);

  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleToggleSkill = (skillId: string) => {
    setSelectedSkills((prev) =>
      prev.includes(skillId) ? prev.filter((id) => id !== skillId) : [...prev, skillId]
    );
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
        <select
          id="agent-binding"
          value={binding}
          onChange={(e) => setBinding(e.target.value)}
          style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: errors.binding ? '1px solid #ff5e62' : '1px solid rgba(255,255,255,0.08)' }}
        >
          <option value="">Select a binding</option>
          {bindingsList.map(b => (
            <option key={b.id} value={b.id}>{b.name}</option>
          ))}
        </select>
        {errors.binding && <span style={{ color: '#ff5e62', fontSize: '0.85rem' }}>{errors.binding}</span>}
      </div>

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
        <select
          id="agent-prompt-template"
          value={promptTemplate}
          onChange={(e) => setPromptTemplate(e.target.value)}
          style={{ width: '100%', padding: '10px', borderRadius: '8px', backgroundColor: '#151821', color: '#f2f5f9', border: '1px solid rgba(255,255,255,0.08)' }}
        >
          <option value="">Select a prompt template</option>
          {promptsList.map(p => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
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

      <h4 style={{ fontFamily: 'Outfit', color: '#66fcf1', borderBottom: '1px solid rgba(255,255,255,0.08)', paddingBottom: '8px', marginBottom: '16px', marginTop: '24px' }}>Capabilities (Skills)</h4>
      
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
