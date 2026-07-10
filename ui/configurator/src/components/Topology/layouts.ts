export interface NodePosition {
  id: string;
  name: string;
  type: 'community' | 'agent';
  x: number;
  y: number;
}

export interface Link {
  source: string;
  target: string;
}

export function computeLayout(
  nodes: Array<{ id: string; name: string; type: 'community' | 'agent' }>,
  _links: Link[],
  layoutType: 'standalone' | 'hub-spoke' | 'serialized',
  width: number = 800,
  height: number = 500
): NodePosition[] {
  const centerX = width / 2;
  const centerY = height / 2;

  if (layoutType === 'hub-spoke') {
    // Find the community node (hub)
    const communityNode = nodes.find(n => n.type === 'community');
    const agentNodes = nodes.filter(n => n.type === 'agent');

    const result: NodePosition[] = [];
    
    if (communityNode) {
      result.push({ ...communityNode, x: centerX, y: centerY });
    }

    const radius = Math.min(width, height) * 0.35;
    agentNodes.forEach((node, index) => {
      const angle = (index * 2 * Math.PI) / agentNodes.length;
      result.push({
        ...node,
        x: centerX + radius * Math.cos(angle),
        y: centerY + radius * Math.sin(angle),
      });
    });

    return result;
  }

  if (layoutType === 'serialized') {
    // Horizontal linear chain
    const result: NodePosition[] = [];
    const spacing = width / (nodes.length + 1);

    nodes.forEach((node, index) => {
      result.push({
        ...node,
        x: spacing * (index + 1),
        y: centerY,
      });
    });

    return result;
  }

  // Fallback: standalone (grid layout)
  const result: NodePosition[] = [];
  const cols = Math.ceil(Math.sqrt(nodes.length));
  const rows = Math.ceil(nodes.length / cols);
  const spacingX = width / (cols + 1);
  const spacingY = height / (rows + 1);

  nodes.forEach((node, index) => {
    const col = index % cols;
    const row = Math.floor(index / cols);
    result.push({
      ...node,
      x: spacingX * (col + 1),
      y: spacingY * (row + 1),
    });
  });

  return result;
}
