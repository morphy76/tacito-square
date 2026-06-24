import { render, screen, fireEvent } from '@testing-library/react';
import { expect, test } from 'vitest';
import TopologyView from './TopologyView';

const mockNodes = [
  { id: 'c1', name: 'Tacito Core', type: 'community' as const },
  { id: 'a1', name: 'Agent Alpha', type: 'agent' as const },
  { id: 'a2', name: 'Agent Beta', type: 'agent' as const },
];

const mockLinks = [
  { source: 'c1', target: 'a1' },
  { source: 'c1', target: 'a2' },
];

test('TopologyView renders nodes and toggles layouts correctly', () => {
  const { container } = render(
    <TopologyView
      nodes={mockNodes}
      links={mockLinks}
    />
  );

  // Assert SVG element is rendered
  const svg = container.querySelector('svg');
  expect(svg).toBeInTheDocument();

  // Initially in hub-spoke mode, should have 3 nodes (circles) and 2 links (lines)
  const circles = container.querySelectorAll('circle.node');
  expect(circles.length).toBe(3);

  const lines = container.querySelectorAll('line.link');
  expect(lines.length).toBe(2);

  // Switch to standalone mode
  const standaloneButton = screen.getByRole('button', { name: /Standalone/i });
  fireEvent.click(standaloneButton);

  // In standalone mode, there should be zero lines
  const standaloneLines = container.querySelectorAll('line.link');
  expect(standaloneLines.length).toBe(0);
});
