import { useState, useMemo } from 'react';
import { computeLayout, NodePosition, Link } from './layouts';

interface TopologyViewProps {
  nodes: Array<{ id: string; name: string; type: 'community' | 'agent' }>;
  links: Link[];
}

export default function TopologyView({ nodes, links }: TopologyViewProps) {
  const [layoutType, setLayoutType] = useState<'standalone' | 'hub-spoke' | 'serialized'>('hub-spoke');
  const [selectedNode, setSelectedNode] = useState<NodePosition | null>(null);

  const width = 800;
  const height = 500;

  // Calculate coordinates based on current layout type
  const positionedNodes = useMemo(() => {
    return computeLayout(nodes, links, layoutType, width, height);
  }, [nodes, links, layoutType]);

  // Map nodes by id for link coordinate lookup
  const nodeMap = useMemo(() => {
    const map = new Map<string, NodePosition>();
    positionedNodes.forEach(n => map.set(n.id, n));
    return map;
  }, [positionedNodes]);

  // Compute lines coordinates
  const activeLinks = useMemo(() => {
    if (layoutType === 'standalone') return [];
    
    return links
      .map(link => {
        const sourceNode = nodeMap.get(link.source);
        const targetNode = nodeMap.get(link.target);
        if (sourceNode && targetNode) {
          return {
            id: `${link.source}-${link.target}`,
            x1: sourceNode.x,
            y1: sourceNode.y,
            x2: targetNode.x,
            y2: targetNode.y,
          };
        }
        return null;
      })
      .filter((l): l is NonNullable<typeof l> => l !== null);
  }, [links, nodeMap, layoutType]);

  return (
    <div className="topology-visualizer" style={{ display: 'flex', flexDirection: 'column', width: '100%', gap: '16px' }}>
      {/* Layout Toolbar */}
      <div className="layout-toolbar" style={{ display: 'flex', gap: '8px', justifyContent: 'flex-start' }}>
        <button
          className={`nav-btn ${layoutType === 'hub-spoke' ? 'active-btn' : ''}`}
          onClick={() => { setLayoutType('hub-spoke'); setSelectedNode(null); }}
          style={{ padding: '8px 16px', borderBottom: layoutType === 'hub-spoke' ? '2px solid #66fcf1' : '' }}
        >
          <span className="btn-label"><span className="btn-title">Hub-Spoke</span></span>
        </button>
        <button
          className={`nav-btn ${layoutType === 'standalone' ? 'active-btn' : ''}`}
          onClick={() => { setLayoutType('standalone'); setSelectedNode(null); }}
          style={{ padding: '8px 16px', borderBottom: layoutType === 'standalone' ? '2px solid #66fcf1' : '' }}
        >
          <span className="btn-label"><span className="btn-title">Standalone</span></span>
        </button>
        <button
          className={`nav-btn ${layoutType === 'serialized' ? 'active-btn' : ''}`}
          onClick={() => { setLayoutType('serialized'); setSelectedNode(null); }}
          style={{ padding: '8px 16px', borderBottom: layoutType === 'serialized' ? '2px solid #66fcf1' : '' }}
        >
          <span className="btn-label"><span className="btn-title">Serialized</span></span>
        </button>
      </div>

      <div style={{ display: 'flex', gap: '20px', flexWrap: 'wrap' }}>
        {/* SVG Canvas */}
        <div 
          className="glass-card svg-card" 
          style={{ 
            flex: '3 1 500px', 
            padding: '12px', 
            backgroundColor: '#0b0c10', 
            border: '1px solid rgba(255,255,255,0.08)',
            position: 'relative'
          }}
        >
          <svg 
            width="100%" 
            height="100%" 
            viewBox={`0 0 ${width} ${height}`} 
            style={{ minHeight: '400px', backgroundColor: '#0d0e12', borderRadius: '12px' }}
          >
            {/* Draw Links */}
            {activeLinks.map(link => (
              <line
                key={link.id}
                x1={link.x1}
                y1={link.y1}
                x2={link.x2}
                y2={link.y2}
                className="link"
                stroke="rgba(102, 252, 241, 0.25)"
                strokeWidth={2}
                style={{ transition: 'all 0.5s ease' }}
              />
            ))}

            {/* Draw Nodes */}
            {positionedNodes.map(node => {
              const isCommunity = node.type === 'community';
              const fill = isCommunity ? '#c82c3c' : '#1e3c72';
              const stroke = isCommunity ? '#ff5e62' : '#66fcf1';
              const glow = isCommunity ? 'rgba(255, 94, 98, 0.4)' : 'rgba(102, 252, 241, 0.4)';
              const isSelected = selectedNode?.id === node.id;

              return (
                <g 
                  key={node.id} 
                  transform={`translate(${node.x}, ${node.y})`}
                  style={{ cursor: 'pointer', transition: 'transform 0.5s ease' }}
                  onClick={() => setSelectedNode(node)}
                >
                  <circle
                    r={isCommunity ? 24 : 16}
                    fill={fill}
                    stroke={isSelected ? '#ffffff' : stroke}
                    strokeWidth={isSelected ? 3 : 2}
                    className="node"
                    style={{
                      transition: 'all 0.3s ease',
                      filter: `drop-shadow(0 0 8px ${glow})`
                    }}
                  />
                  <text
                    y={isCommunity ? 40 : 32}
                    textAnchor="middle"
                    fill="#f2f5f9"
                    style={{ fontSize: '0.85rem', fontFamily: 'Outfit', fontWeight: 600 }}
                  >
                    {node.name}
                  </text>
                  <text
                    y={0}
                    dy=".3em"
                    textAnchor="middle"
                    fill="#ffffff"
                    style={{ fontSize: '0.75rem', fontFamily: 'monospace', pointerEvents: 'none' }}
                  >
                    {isCommunity ? 'C' : 'A'}
                  </text>
                </g>
              );
            })}
          </svg>
        </div>

        {/* Info panel */}
        <div className="glass-card details-drawer" style={{ flex: '1 1 250px', padding: '24px', textAlign: 'left', minHeight: '400px' }}>
          <h3 style={{ fontFamily: 'Outfit', margin: '0 0 16px' }}>Node Specifications</h3>
          {selectedNode ? (
            <div>
              <div 
                style={{ 
                  padding: '8px 12px', 
                  borderRadius: '16px', 
                  background: selectedNode.type === 'community' ? 'rgba(200, 44, 60, 0.15)' : 'rgba(30, 60, 114, 0.15)',
                  border: selectedNode.type === 'community' ? '1px solid rgba(200, 44, 60, 0.3)' : '1px solid rgba(102, 252, 241, 0.3)',
                  marginBottom: '16px',
                  display: 'inline-block',
                  fontSize: '0.8rem',
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  color: selectedNode.type === 'community' ? '#ff5e62' : '#66fcf1'
                }}
              >
                {selectedNode.type}
              </div>
              <h4 style={{ margin: '0 0 8px', fontFamily: 'Outfit', fontSize: '1.2rem' }}>{selectedNode.name}</h4>
              <p style={{ fontSize: '0.85rem', color: '#94a3b8', margin: '0 0 12px' }}>ID: <code>{selectedNode.id}</code></p>
              
              <div style={{ marginTop: '20px', borderTop: '1px solid rgba(255,255,255,0.08)', paddingTop: '16px' }}>
                <p style={{ fontSize: '0.9rem', color: '#c5c6c7' }}>
                  Coordinates: <code>X: {Math.round(selectedNode.x)}, Y: {Math.round(selectedNode.y)}</code>
                </p>
              </div>
            </div>
          ) : (
            <p style={{ color: '#94a3b8', fontSize: '0.9rem' }}>Click on a node in the graph to view details.</p>
          )}
        </div>
      </div>
    </div>
  );
}
