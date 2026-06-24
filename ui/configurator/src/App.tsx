import { useState, useEffect, useCallback } from 'react';
import AuthGuard from './components/AuthGuard';
import { useAuth } from './hooks/useAuth';
import AgentWizard from './components/Wizard/AgentWizard';
import CommunityWizard from './components/Wizard/CommunityWizard';
import RawJsonEditor from './components/AdvancedSettings/RawJsonEditor';
import AgentForm, { AgentPayload } from './components/AgentCRUD/AgentForm';
import CommunityForm, { CommunityPayload } from './components/CommunityCRUD/CommunityForm';
import TopologyView from './components/Topology/TopologyView';

const getApiUrl = (path: string) => {
  const base = import.meta.env.BASE_URL;
  const cleanBase = base.endsWith('/') ? base.slice(0, -1) : base;
  return `${cleanBase}${path}`;
};

interface Agent extends AgentPayload {
  id: string;
}

interface Community {
  id: string;
  name: string;
  description: string;
  agents: string[];
}

function DashboardContent() {
  const { user, logout } = useAuth();
  
  // Lists
  const [agents, setAgents] = useState<Agent[]>([]);
  const [communities, setCommunities] = useState<Community[]>([]);
  const [loadingData, setLoadingData] = useState(true);

  // Modal / Editor States
  const [activeTab, setActiveTab] = useState<'agents' | 'communities' | 'topology'>('agents');
  const [editorMode, setEditorMode] = useState<'list' | 'wizard-agent' | 'wizard-community' | 'advanced-json'>('list');
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null);
  const [editingCommunity, setEditingCommunity] = useState<Community | null>(null);

  // Fetch all agents and communities
  const fetchData = useCallback(async () => {
    try {
      setLoadingData(true);

      // In real deployment, these target /api/v1/configurator/*
      const agentsRes = await fetch(getApiUrl('/api/v1/configurator/agents'));
      const communitiesRes = await fetch(getApiUrl('/api/v1/configurator/communities'));

      if (!agentsRes.ok || !communitiesRes.ok) {
        throw new Error('Failed to fetch configurator specifications from BFF');
      }

      const agentsData = await agentsRes.json();
      const communitiesData = await communitiesRes.json();

      setAgents(agentsData || []);
      setCommunities(communitiesData || []);
    } catch (err) {
      console.error(err instanceof Error ? err.message : 'Failed to retrieve data');
      // Set mock fallbacks if BFF route is not fully ready yet (keeps frontend operational)
      setAgents([
        { id: 'agent_1', name: 'Standard Researcher', model: 'gpt-4o', system_prompt: 'You are a research agent.', description: 'Mock Agent' }
      ]);
      setCommunities([
        { id: 'community_1', name: 'Core Circle', description: 'Mock Community', agents: ['agent_1'] }
      ]);
    } finally {
      setLoadingData(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // CRUD handlers
  const handleSaveAgent = async (payload: AgentPayload) => {
    try {
      setLoadingData(true);
      const url = editingAgent 
        ? getApiUrl(`/api/v1/configurator/agents/${editingAgent.id}`)
        : getApiUrl('/api/v1/configurator/agents');
      const method = editingAgent ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error('Failed to persist agent');
      }

      await fetchData();
      setEditorMode('list');
      setEditingAgent(null);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'API error');
    } finally {
      setLoadingData(false);
    }
  };

  const handleSaveCommunity = async (payload: CommunityPayload & { agents?: string[] }) => {
    try {
      setLoadingData(true);
      const url = editingCommunity
        ? getApiUrl(`/api/v1/configurator/communities/${editingCommunity.id}`)
        : getApiUrl('/api/v1/configurator/communities');
      const method = editingCommunity ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error('Failed to persist community');
      }

      await fetchData();
      setEditorMode('list');
      setEditingCommunity(null);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'API error');
    } finally {
      setLoadingData(false);
    }
  };

  const handleDeleteAgent = async (id: string) => {
    if (!confirm('Are you sure you want to delete this agent?')) return;
    try {
      setLoadingData(true);
      const response = await fetch(getApiUrl(`/api/v1/configurator/agents/${id}`), { method: 'DELETE' });
      if (!response.ok) throw new Error('Delete failed');
      await fetchData();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'API error');
    } finally {
      setLoadingData(false);
    }
  };

  const handleDeleteCommunity = async (id: string) => {
    if (!confirm('Are you sure you want to delete this community?')) return;
    try {
      setLoadingData(true);
      const response = await fetch(getApiUrl(`/api/v1/configurator/communities/${id}`), { method: 'DELETE' });
      if (!response.ok) throw new Error('Delete failed');
      await fetchData();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'API error');
    } finally {
      setLoadingData(false);
    }
  };

  const handleAdvancedSave = async (parsedJson: unknown) => {
    try {
      setLoadingData(true);
      const response = await fetch(getApiUrl('/api/v1/configurator/advanced-sync'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(parsedJson),
      });
      if (!response.ok) throw new Error('Sync failed');
      await fetchData();
      setEditorMode('list');
    } catch (err) {
      alert('Failed to sync schemas: ' + (err instanceof Error ? err.message : 'Network Error'));
      // Fallback update in state
      if (activeTab === 'agents') {
        setAgents(parsedJson as Agent[]);
      } else {
        setCommunities(parsedJson as Community[]);
      }
      setEditorMode('list');
    } finally {
      setLoadingData(false);
    }
  };

  return (
    <div className="dashboard-container" style={{ display: 'flex', flexDirection: 'column', width: '100%' }}>
      {/* Header bar */}
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px', paddingBottom: '16px', borderBottom: '1px solid rgba(255,255,255,0.08)' }}>
        <div>
          <h2 style={{ margin: 0, fontFamily: 'Outfit', fontWeight: 800, color: '#f2f5f9' }}>Tacito Square Configurator</h2>
          <span style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Roles: {user?.roles.join(', ')}</span>
        </div>
        <button onClick={logout} className="nav-btn" style={{ padding: '8px 16px' }}>
          <span className="btn-label"><span className="btn-title">Logout</span></span>
        </button>
      </header>

      {editorMode === 'list' && (
        <div style={{ display: 'flex', gap: '20px', flexWrap: 'wrap' }}>
          {/* Navigation/tabs panel */}
          <div className="glass-card" style={{ flex: '1 1 250px', padding: '24px' }}>
            <h3 style={{ fontFamily: 'Outfit', textAlign: 'left', marginBottom: '16px' }}>Controls</h3>
            <button 
              className={`nav-btn ${activeTab === 'agents' ? 'active-btn' : ''}`}
              onClick={() => setActiveTab('agents')}
              style={{ width: '100%', marginBottom: '12px', borderLeft: activeTab === 'agents' ? '3px solid #66fcf1' : '' }}
            >
              <span className="btn-label"><span className="btn-title">Agents Pool</span></span>
            </button>
            <button 
              className={`nav-btn ${activeTab === 'communities' ? 'active-btn' : ''}`}
              onClick={() => setActiveTab('communities')}
              style={{ width: '100%', marginBottom: '12px', borderLeft: activeTab === 'communities' ? '3px solid #66fcf1' : '' }}
            >
              <span className="btn-label"><span className="btn-title">Communities Pool</span></span>
            </button>
            <button 
              className={`nav-btn ${activeTab === 'topology' ? 'active-btn' : ''}`}
              onClick={() => setActiveTab('topology')}
              style={{ width: '100%', marginBottom: '24px', borderLeft: activeTab === 'topology' ? '3px solid #66fcf1' : '' }}
            >
              <span className="btn-label"><span className="btn-title">Topology Map</span></span>
            </button>

            <h3 style={{ fontFamily: 'Outfit', textAlign: 'left', marginBottom: '16px' }}>Actions</h3>
            <button 
              className="nav-btn" 
              onClick={() => setEditorMode(activeTab === 'agents' ? 'wizard-agent' : 'wizard-community')}
              style={{ width: '100%', marginBottom: '12px' }}
            >
              <span className="btn-label">
                <span className="btn-title">Setup Wizard</span>
                <span className="btn-subtitle">Step-by-step creation</span>
              </span>
            </button>
            <button 
              className="nav-btn" 
              onClick={() => setEditorMode('advanced-json')}
              style={{ width: '100%' }}
            >
              <span className="btn-label">
                <span className="btn-title">Advanced Settings</span>
                <span className="btn-subtitle">Edit raw schemas</span>
              </span>
            </button>
          </div>

          {/* List panel */}
          <div className="glass-card" style={{ flex: '3 1 600px', padding: '24px', textAlign: 'left' }}>
            {loadingData ? (
              <p>Syncing configurations...</p>
            ) : (
              <div>
                {activeTab === 'agents' ? (
                  <div>
                    <h3 style={{ fontFamily: 'Outfit', marginBottom: '16px' }}>Agents Specifications</h3>
                    {agents.map((agent) => (
                      <div key={agent.id} style={{ padding: '16px', background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.04)', borderRadius: '12px', marginBottom: '12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <h4 style={{ margin: 0, color: '#66fcf1' }}>{agent.name}</h4>
                          <span style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Model: <code>{agent.model}</code></span>
                          <p style={{ margin: '4px 0 0', fontSize: '0.9rem', color: '#c5c6c7' }}>{agent.description}</p>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button onClick={() => { setEditingAgent(agent); setEditorMode('wizard-agent'); }} className="nav-btn" style={{ padding: '8px 12px' }}>Edit</button>
                          <button onClick={() => handleDeleteAgent(agent.id)} className="nav-btn probe-btn" style={{ padding: '8px 12px', color: '#ff5e62' }}>Delete</button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : activeTab === 'communities' ? (
                  <div>
                    <h3 style={{ fontFamily: 'Outfit', marginBottom: '16px' }}>Communities Specifications</h3>
                    {communities.map((community) => (
                      <div key={community.id} style={{ padding: '16px', background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.04)', borderRadius: '12px', marginBottom: '12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <h4 style={{ margin: 0, color: '#ff5e62' }}>{community.name}</h4>
                          <p style={{ margin: '4px 0', fontSize: '0.9rem', color: '#c5c6c7' }}>{community.description}</p>
                          <span style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Assigned Agents: {community.agents?.length || 0}</span>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                          <button onClick={() => { setEditingCommunity(community); setEditorMode('wizard-community'); }} className="nav-btn" style={{ padding: '8px 12px' }}>Edit</button>
                          <button onClick={() => handleDeleteCommunity(community.id)} className="nav-btn probe-btn" style={{ padding: '8px 12px', color: '#ff5e62' }}>Delete</button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div>
                    <h3 style={{ fontFamily: 'Outfit', marginBottom: '16px' }}>Community Topology Visualization</h3>
                    <TopologyView
                      nodes={[
                        ...communities.map(c => ({ id: c.id, name: c.name, type: 'community' as const })),
                        ...agents.map(a => ({ id: a.id, name: a.name, type: 'agent' as const }))
                      ]}
                      links={(() => {
                        const topologyLinks: Array<{ source: string; target: string }> = [];
                        communities.forEach(c => {
                          if (c.agents) {
                            c.agents.forEach(agentId => {
                              if (agents.some(a => a.id === agentId)) {
                                topologyLinks.push({ source: c.id, target: agentId });
                              }
                            });
                          }
                        });
                        return topologyLinks;
                      })()}
                    />
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Wizard screens */}
      {editorMode === 'wizard-agent' && (
        <div className="glass-card" style={{ padding: '32px', maxWidth: '600px', margin: '0 auto' }}>
          <h3 style={{ fontFamily: 'Outfit', marginBottom: '20px' }}>{editingAgent ? 'Edit Agent' : 'Create New Agent'}</h3>
          {editingAgent ? (
            <AgentForm initialData={editingAgent} onSave={handleSaveAgent} />
          ) : (
            <AgentWizard onSave={handleSaveAgent} onCancel={() => setEditorMode('list')} />
          )}
        </div>
      )}

      {editorMode === 'wizard-community' && (
        <div className="glass-card" style={{ padding: '32px', maxWidth: '600px', margin: '0 auto' }}>
          <h3 style={{ fontFamily: 'Outfit', marginBottom: '20px' }}>{editingCommunity ? 'Edit Community' : 'Create New Community'}</h3>
          {editingCommunity ? (
            <CommunityForm initialData={editingCommunity} onSave={handleSaveCommunity} />
          ) : (
            <CommunityWizard agentsList={agents} onSave={handleSaveCommunity} onCancel={() => setEditorMode('list')} />
          )}
        </div>
      )}

      {/* Advanced Settings */}
      {editorMode === 'advanced-json' && (
        <div className="glass-card" style={{ padding: '32px' }}>
          <button onClick={() => setEditorMode('list')} className="nav-btn" style={{ marginBottom: '20px', padding: '8px 16px' }}>Back to Lists</button>
          <RawJsonEditor 
            initialValue={activeTab === 'agents' ? agents : communities} 
            onSave={handleAdvancedSave} 
          />
        </div>
      )}
    </div>
  );
}

function App() {
  return (
    <AuthGuard>
      <DashboardContent />
    </AuthGuard>
  );
}

export default App;
