# TASK-M7.2-T5: Community Topology Visualization Component

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T5                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/`)                               |
| Status      | VERIFIED                                                           |
| Depends On  | SPEC-FR-M7.2, TASK-M7.2-T1, TASK-M7.2-T2                           |

## Objective

Create a custom topology visualization component using interactive SVGs to represent agent-community layouts. The visualization must support multiple coordinate layouts: **standalone** units (isolated agent nodes), **hub-spoke** networks (a central community node with spokes pointing to assigned agents), and extensible layout formats representing **serialized** future topologies (linear or chained nodes). It must support interactive node details, layout selection via a toolbar, hover/focus effects, and animations matching the Piazza Tacito styling.

## Files

| File | Action |
|------|--------|
| `ui/configurator/src/components/Topology/TopologyView.tsx` | NEW |
| `ui/configurator/src/components/Topology/TopologyView.test.tsx` | NEW |
| `ui/configurator/src/components/Topology/layouts.ts` | NEW |

## RED Phase

1. **Topology Render Assertions**:
   - Create `ui/configurator/src/components/Topology/TopologyView.test.tsx` with Vitest.
   - Assert the following rendering conditions based on a mock topology configuration:
     - **Nodes and Edges**: Verify that the correct number of SVG circles (nodes) and lines (edges) are generated based on a mock payload (e.g. 1 community, 2 assigned agents, 1 unassigned agent).
     - **Standalone layout**: Verify that for a "standalone" configuration, nodes are arranged in a layout where no connection lines (edges) are rendered between the elements.
     - **Hub-spoke layout**: Verify that for a "hub-spoke" configuration, lines are rendered between the coordinates of the community node and each assigned agent spoke.
     - **Layout Switcher**: Verify that clicking the toolbar layout buttons modifies the active layout mode state.
   - Run Vitest tests (`npm run test`) inside `ui/configurator` and verify they fail (RED).

## GREEN Phase

1. **Implement Coordinates Calculator (`layouts.ts`)**:
   - Write functions that compute coordinate lists `(x, y)` based on nodes data and active layout:
     - `standalone`: positions all agents in a grid or simple horizontal array.
     - `hub-spoke`: computes a circular path of coordinates centered around a community node.
     - `serialized`: positions agents in a sequential line representing chained processing.

2. **Implement `TopologyView` SVG Component**:
   - Create `TopologyView.tsx` with responsive `<svg>` container.
   - Map calculated node coordinates to `<g>` elements containing `<circle>`, icons, and text labels.
   - Map connections to `<line>` or `<path>` elements.
   - Style elements with smooth CSS transition effects (`transition: all 0.3s ease`).
   - Add hover states to nodes (e.g. changing glow colors using variables like `--color-porphyry-glow` or `--color-steel-light`).
   - Implement node click callbacks displaying detailed configurations in a drawer/side-modal.
   - Add a control toolbar enabling users to toggle between standalone, hub-spoke, and serialized layouts.

3. **Verify tests**:
   - Run the frontend tests and confirm they pass (GREEN).

## REFACTOR Phase

- Optimize rendering of connection lines (ensure they start/end exactly at circle borders or use appropriate offsets).
- Confirm SVG canvas scales properly in smaller cards or when resized dynamically.
- Clean up any raw mathematical equations in coordinates calculators to preserve clarity and structure.
