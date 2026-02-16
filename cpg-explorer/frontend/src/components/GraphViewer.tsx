import { useEffect, useRef } from 'react';
import cytoscape from 'cytoscape';
import type { Core, ElementDefinition } from 'cytoscape';
// @ts-ignore
import dagre from 'cytoscape-dagre';
import type { Graph } from '../api/client';

// Register the dagre layout
cytoscape.use(dagre);

interface GraphViewerProps {
  graph: Graph;
  onNodeClick?: (nodeId: string) => void;
  layout?: 'dagre' | 'cose' | 'circle';
}

export const GraphViewer = ({ graph, onNodeClick, layout = 'dagre' }: GraphViewerProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<Core | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    // Convert graph data to Cytoscape format
    const elements: ElementDefinition[] = [
      ...graph.nodes.map((node) => ({
        data: {
          id: node.id,
          label: node.name,
          kind: node.kind,
          package: node.package,
          ...node.properties,
        },
      })),
      ...graph.edges.map((edge) => ({
        data: {
          id: `${edge.source}-${edge.target}`,
          source: edge.source,
          target: edge.target,
          kind: edge.kind,
          ...edge.properties,
        },
      })),
    ];

    // Initialize Cytoscape
    const cy = cytoscape({
      container: containerRef.current,
      elements,
      style: [
        {
          selector: 'node',
          style: {
            'background-color': '#3b82f6',
            'label': 'data(label)',
            'color': '#fff',
            'text-valign': 'center',
            'text-halign': 'center',
            'font-size': '10px',
            'width': '40px',
            'height': '40px',
            'border-width': 2,
            'border-color': '#1e40af',
          },
        },
        {
          selector: 'node[kind="package"]',
          style: {
            'background-color': '#10b981',
            'border-color': '#047857',
            'width': '60px',
            'height': '60px',
          },
        },
        {
          selector: 'node[kind="function"]',
          style: {
            'background-color': '#3b82f6',
            'border-color': '#1e40af',
          },
        },
        {
          selector: 'node:selected',
          style: {
            'background-color': '#f59e0b',
            'border-color': '#d97706',
            'border-width': 3,
          },
        },
        {
          selector: 'edge',
          style: {
            'width': 2,
            'line-color': '#9ca3af',
            'target-arrow-color': '#9ca3af',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            'arrow-scale': 1,
          },
        },
        {
          selector: 'edge[kind="call"]',
          style: {
            'line-color': '#3b82f6',
            'target-arrow-color': '#3b82f6',
          },
        },
        {
          selector: 'edge[kind="depends"]',
          style: {
            'line-color': '#10b981',
            'target-arrow-color': '#10b981',
          },
        },
      ],
      layout: {
        name: layout,
        ...(layout === 'dagre' ? { rankDir: 'TB' as const } : {}),
        animate: true,
        animationDuration: 500,
      } as any,
      minZoom: 0.1,
      maxZoom: 3,
    });

    // Add click handler
    if (onNodeClick) {
      cy.on('tap', 'node', (event) => {
        const node = event.target;
        onNodeClick(node.id());
      });
    }

    cyRef.current = cy;

    // Cleanup
    return () => {
      cy.destroy();
    };
  }, [graph, onNodeClick, layout]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full bg-gray-50 rounded-lg border border-gray-300"
      style={{ minHeight: '400px' }}
    />
  );
};
