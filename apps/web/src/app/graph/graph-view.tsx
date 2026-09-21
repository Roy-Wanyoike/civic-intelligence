'use client';

// GraphExplorer — the interactive force-directed graph visualization
// behind /graph (ENG-I1, Wave 9).
//
// The component is the platform's signature differentiator: a visual
// relationship explorer where citizens can trace how Bills, Acts,
// Institutions, People, Constitution Articles, and Government borrowing
// are connected. No civic intelligence platform in Africa offers a graph
// explorer of this kind.
//
// Architecture:
//   - d3-force drives the layout physics (forceManyBody, forceLink,
//     forceCenter, forceCollide). The simulation runs in a useEffect
//     that re-creates the forces whenever the node/edge set changes.
//   - SVG renders the simulation state on every tick. We use SVG (not
//     canvas) so each node + edge is a real DOM element that can carry
//     a <title>, hover state, and click handler — accessibility gate.
//   - Zoom/pan: a wrapping <g> with a transform that we mutate via
//     wheel + drag handlers. The simulation runs in "graph space"; the
//     transform maps graph space to screen space.
//   - Node selection: clicking a node calls /api/v1/graph/node/{id} to
//     load the detail panel.
//   - Search: a debounced call to /api/v1/graph/search?q=...
//   - Find path: two node pickers + a call to /api/v1/graph/paths.
//   - Export PNG: html2canvas captures the SVG container. SVG must be
//     serialized to a data URL because html2canvas does not traverse
//     SVG natively — we render the SVG to a canvas via the browser's
//     XMLSerializer + Image, then html2canvas captures the canvas.

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Search,
  ZoomIn,
  ZoomOut,
  Maximize2,
  Download,
  Table as TableIcon,
  Network as NetworkIcon,
  List as ListIcon,
  X,
  ArrowRight,
  ExternalLink,
  AlertCircle,
  Loader2,
} from 'lucide-react';
import {
  forceCenter,
  forceCollide,
  forceLink,
  forceManyBody,
  forceSimulation,
  type Simulation,
} from 'd3-force';
import {
  EDGE_TYPE_META,
  NODE_TYPE_META,
  findGraphPath,
  getGraphNode,
  getGraphRelationships,
  searchGraphNodes,
  type GraphEdge,
  type GraphEdgeType,
  type GraphNode,
  type GraphNodeType,
  type GraphSummary,
} from '@/lib/graph-api';

// SimNode is a d3-force node augmented with our display fields. d3-force
// mutates `x` and `y` on every tick; we keep `vx` and `vy` for completeness
// (d3 sets them too).
interface SimNode extends GraphNode {
  x: number;
  y: number;
  vx: number;
  vy: number;
  fx?: number | null;
  fy?: number | null;
}

// SimLink is a d3-force link. The source/target are typed as string on
// GraphEdge (the wire format), but d3-force mutates them to SimNode
// references after the simulation starts. We declare the type as a union
// so both shapes are valid; the rendering code normalises back to SimNode
// via the `simNodes` lookup.
interface SimLink {
  source: string | SimNode | number;
  target: string | SimNode | number;
  type: GraphEdgeType;
  properties?: Record<string, unknown>;
  index: number;
}

type ViewMode = 'graph' | 'table' | 'list';

interface GraphExplorerProps {
  summary: GraphSummary | null;
}

const DEFAULT_CENTER_NODE = 'admin-uhuru-kenyatta';
const DEFAULT_DEPTH: 1 | 2 | 3 = 2;

export function GraphExplorer({ summary }: GraphExplorerProps) {
  // === Graph state ===
  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [edges, setEdges] = useState<GraphEdge[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // === Selection state ===
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [selectedNodeNeighbours, setSelectedNodeNeighbours] = useState<{
    nodes: GraphNode[];
    edges: GraphEdge[];
  }>({ nodes: [], edges: [] });

  // === Search state ===
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<GraphNode[]>([]);
  const [searchOpen, setSearchOpen] = useState(false);

  // === Depth + view mode ===
  const [depth, setDepth] = useState<1 | 2 | 3>(DEFAULT_DEPTH);
  const [viewMode, setViewMode] = useState<ViewMode>('graph');

  // === Path finder ===
  const [pathMode, setPathMode] = useState(false);
  const [pathFrom, setPathFrom] = useState<string>('');
  const [pathTo, setPathTo] = useState<string>('');
  const [pathNodes, setPathNodes] = useState<GraphNode[]>([]);
  const [pathEdges, setPathEdges] = useState<GraphEdge[]>([]);
  const [pathError, setPathError] = useState<string | null>(null);

  // === Zoom/pan ===
  const [transform, setTransform] = useState({ x: 0, y: 0, k: 1 });
  const svgRef = useRef<SVGSVGElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const isPanning = useRef(false);
  const panStart = useRef({ x: 0, y: 0, tx: 0, ty: 0 });

  // === d3 simulation ===
  // The simulation is held in a ref so we can stop + restart it without
  // re-creating it on every render. We re-create it when the node/edge
  // set changes (the previous simulation's nodes are no longer valid).
  const simRef = useRef<Simulation<SimNode, undefined> | null>(null);
  const [simTick, setSimTick] = useState(0); // bump to trigger re-render on tick
  const simNodesRef = useRef<SimNode[]>([]);
  const simLinksRef = useRef<SimLink[]>([]);

  // === Initial load: fetch the centre node's neighbourhood ===
  const loadNeighbourhood = useCallback(
    async (id: string, d: 1 | 2 | 3) => {
      setLoading(true);
      setError(null);
      try {
        const resp = await getGraphRelationships(id, d);
        setNodes(resp.nodes);
        setEdges(resp.edges);
        setSelectedNodeId(id);
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load graph');
        setNodes([]);
        setEdges([]);
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    loadNeighbourhood(DEFAULT_CENTER_NODE, DEFAULT_DEPTH);
  }, [loadNeighbourhood]);

  // === Build d3 simulation whenever nodes/edges change ===
  useEffect(() => {
    if (nodes.length === 0) {
      simRef.current = null;
      simNodesRef.current = [];
      simLinksRef.current = [];
      return;
    }

    // Materialize SimNode[] — preserve x/y from the previous run if the
    // node existed before so the layout doesn't jump on every refocus.
    const prev = new Map(simNodesRef.current.map((n) => [n.id, n]));
    const simNodes: SimNode[] = nodes.map((n) => {
      const p = prev.get(n.id);
      const cx = (containerRef.current?.clientWidth ?? 800) / 2;
      const cy = (containerRef.current?.clientHeight ?? 600) / 2;
      return {
        ...n,
        x: p?.x ?? cx + (Math.random() - 0.5) * 200,
        y: p?.y ?? cy + (Math.random() - 0.5) * 200,
        vx: 0,
        vy: 0,
        fx: p?.fx ?? null,
        fy: p?.fy ?? null,
      };
    });
    const idToIndex = new Map(simNodes.map((n, i) => [n.id, i]));
    const simLinks: SimLink[] = edges.map((e, i) => ({
      ...e,
      source: idToIndex.get(e.source) ?? 0,
      target: idToIndex.get(e.target) ?? 0,
      index: i,
    }));
    simNodesRef.current = simNodes;
    simLinksRef.current = simLinks;

    const cx = (containerRef.current?.clientWidth ?? 800) / 2;
    const cy = (containerRef.current?.clientHeight ?? 600) / 2;
    const sim = forceSimulation<SimNode>(simNodes)
      .force('charge', forceManyBody().strength(-300))
      .force(
        'link',
        forceLink<SimNode, SimLink>(simLinks)
          .id((d) => d.id)
          .distance(80)
          .strength(0.3),
      )
      .force('center', forceCenter(cx, cy))
      .force('collide', forceCollide<SimNode>().radius(28))
      .alpha(1)
      .alphaDecay(0.04)
      .on('tick', () => setSimTick((t) => (t + 1) % 1_000_000));
    simRef.current = sim;
    return () => {
      sim.stop();
    };
  }, [nodes, edges]);

  // === Selection side-panel: load node detail when a node is clicked ===
  useEffect(() => {
    if (!selectedNodeId) {
      setSelectedNode(null);
      setSelectedNodeNeighbours({ nodes: [], edges: [] });
      return;
    }
    let cancelled = false;
    getGraphNode(selectedNodeId)
      .then((resp) => {
        if (cancelled) return;
        setSelectedNode(resp.nodes[0] ?? null);
        setSelectedNodeNeighbours({
          nodes: resp.nodes.slice(1),
          edges: resp.edges,
        });
      })
      .catch((e) => {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : 'Failed to load node detail');
      });
    return () => {
      cancelled = true;
    };
  }, [selectedNodeId]);

  // === Search: debounced ===
  useEffect(() => {
    if (!searchQuery.trim()) {
      setSearchResults([]);
      return;
    }
    const handle = setTimeout(async () => {
      try {
        const resp = await searchGraphNodes(searchQuery, 10);
        setSearchResults(resp.nodes);
      } catch {
        setSearchResults([]);
      }
    }, 250);
    return () => clearTimeout(handle);
  }, [searchQuery]);

  // === Path finder ===
  const findPath = useCallback(async () => {
    if (!pathFrom || !pathTo) {
      setPathError('Select both a starting node and a target node.');
      return;
    }
    setLoading(true);
    setPathError(null);
    try {
      const resp = await findGraphPath(pathFrom, pathTo);
      setPathNodes(resp.nodes);
      setPathEdges(resp.edges);
      // Also drive the main graph view to show the path.
      setNodes(resp.nodes);
      setEdges(resp.edges);
    } catch (e) {
      setPathError(
        e instanceof Error ? e.message : 'No path found between the selected nodes.',
      );
      setPathNodes([]);
      setPathEdges([]);
    } finally {
      setLoading(false);
    }
  }, [pathFrom, pathTo]);

  // === Zoom/pan handlers ===
  const onWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault();
    const delta = -e.deltaY * 0.001;
    setTransform((t) => {
      const k = Math.min(4, Math.max(0.25, t.k * (1 + delta)));
      return { ...t, k };
    });
  }, []);

  const startPan = useCallback((e: React.MouseEvent) => {
    // Only start panning when the user drags the background (not a node).
    if (e.target !== e.currentTarget && (e.target as SVGElement).tagName !== 'rect') return;
    isPanning.current = true;
    panStart.current = {
      x: e.clientX,
      y: e.clientY,
      tx: transform.x,
      ty: transform.y,
    };
  }, [transform.x, transform.y]);

  const onPan = useCallback((e: React.MouseEvent) => {
    if (!isPanning.current) return;
    const dx = e.clientX - panStart.current.x;
    const dy = e.clientY - panStart.current.y;
    setTransform((t) => ({ ...t, x: panStart.current.tx + dx, y: panStart.current.ty + dy }));
  }, []);

  const endPan = useCallback(() => {
    isPanning.current = false;
  }, []);

  const zoomIn = useCallback(
    () => setTransform((t) => ({ ...t, k: Math.min(4, t.k * 1.25) })),
    [],
  );
  const zoomOut = useCallback(
    () => setTransform((t) => ({ ...t, k: Math.max(0.25, t.k / 1.25) })),
    [],
  );
  const zoomReset = useCallback(
    () => setTransform({ x: 0, y: 0, k: 1 }),
    [],
  );

  // === Node drag (within simulation) ===
  const draggingNode = useRef<SimNode | null>(null);
  const startNodeDrag = useCallback(
    (e: React.MouseEvent, nodeId: string) => {
      e.stopPropagation();
      const node = simNodesRef.current.find((n) => n.id === nodeId);
      if (!node) return;
      draggingNode.current = node;
      node.fx = node.x;
      node.fy = node.y;
      simRef.current?.alphaTarget(0.3).restart();
    },
    [],
  );
  const onNodeDrag = useCallback((e: React.MouseEvent) => {
    if (!draggingNode.current || !svgRef.current) return;
    const rect = svgRef.current.getBoundingClientRect();
    const x = (e.clientX - rect.left - transform.x) / transform.k;
    const y = (e.clientY - rect.top - transform.y) / transform.k;
    draggingNode.current.fx = x;
    draggingNode.current.fy = y;
  }, [transform.x, transform.y, transform.k]);
  const endNodeDrag = useCallback(() => {
    if (draggingNode.current) {
      draggingNode.current.fx = null;
      draggingNode.current.fy = null;
      draggingNode.current = null;
      simRef.current?.alphaTarget(0);
    }
    isPanning.current = false;
  }, []);

  // === Export PNG (html2canvas) ===
  const exportPNG = useCallback(async () => {
    if (!containerRef.current || !svgRef.current) return;
    try {
      const { default: html2canvas } = await import('html2canvas');
      // Serialize the SVG to a data URL, render it on a canvas, then
      // capture that canvas with html2canvas (html2canvas does not
      // traverse SVG natively).
      const svg = svgRef.current;
      const xml = new XMLSerializer().serializeToString(svg);
      const svgDataUrl = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(xml);
      const img = new Image();
      img.crossOrigin = 'anonymous';
      await new Promise<void>((resolve, reject) => {
        img.onload = () => resolve();
        img.onerror = () => reject(new Error('failed to render SVG to image'));
        img.src = svgDataUrl;
      });
      const canvas = document.createElement('canvas');
      const w = containerRef.current.clientWidth;
      const h = containerRef.current.clientHeight;
      canvas.width = w * 2; // 2x for retina
      canvas.height = h * 2;
      const ctx = canvas.getContext('2d');
      if (!ctx) throw new Error('no 2d context');
      ctx.fillStyle = '#ffffff';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
      // Wrap the canvas in a div so html2canvas can capture it.
      const wrapper = document.createElement('div');
      wrapper.style.position = 'fixed';
      wrapper.style.left = '-9999px';
      wrapper.style.top = '0';
      wrapper.style.width = `${w}px`;
      wrapper.style.height = `${h}px`;
      wrapper.appendChild(canvas);
      document.body.appendChild(wrapper);
      await html2canvas(wrapper, {
        backgroundColor: '#ffffff',
        scale: 2,
        logging: false,
      });
      document.body.removeChild(wrapper);
      // Trigger download.
      const link = document.createElement('a');
      link.download = 'civic-knowledge-graph.png';
      link.href = canvas.toDataURL('image/png');
      link.click();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to export PNG');
    }
  }, []);

  // === Memoized derived state ===
  const pathEdgeSet = useMemo(() => {
    const s = new Set<string>();
    for (const e of pathEdges) s.add(`${e.source}|${e.target}|${e.type}`);
    return s;
  }, [pathEdges]);

  const selectedNeighbourIds = useMemo(() => {
    if (!selectedNodeId) return new Set<string>();
    const s = new Set<string>();
    for (const e of selectedNodeNeighbours.edges) {
      if (e.source === selectedNodeId) s.add(e.target);
      if (e.target === selectedNodeId) s.add(e.source);
    }
    return s;
  }, [selectedNodeId, selectedNodeNeighbours.edges]);

  // === Render ===
  return (
    <div className="space-y-4">
      {/* === Toolbar === */}
      <div className="flex flex-wrap items-center gap-3 rounded-lg border border-civic-border bg-civic-paper p-3">
        {/* Search */}
        <div className="relative min-w-[16rem] flex-1">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-civic-stone"
            aria-hidden="true"
          />
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setSearchOpen(true);
            }}
            onFocus={() => setSearchOpen(true)}
            onBlur={() => setTimeout(() => setSearchOpen(false), 150)}
            placeholder="Search nodes by name (e.g. 'uhuru', 'data protection', 'parliament')"
            aria-label="Search graph nodes"
            className="w-full rounded-md border border-civic-border bg-civic-mist py-2 pl-9 pr-3 text-sm focus:border-civic-leaf focus:outline-none"
          />
          {searchOpen && searchResults.length > 0 ? (
            <ul
              role="listbox"
              aria-label="Search results"
              className="absolute z-30 mt-1 max-h-80 w-full overflow-auto rounded-md border border-civic-border bg-civic-paper shadow-lg"
            >
              {searchResults.map((n) => (
                <li key={n.id} role="option" aria-selected="false">
                  <button
                    type="button"
                    onMouseDown={(e) => {
                      e.preventDefault();
                      loadNeighbourhood(n.id, depth);
                      setSelectedNodeId(n.id);
                      setSearchOpen(false);
                      setViewMode('graph');
                    }}
                    className="flex w-full items-start gap-2 px-3 py-2 text-left text-sm hover:bg-civic-mist"
                  >
                    <TypeBadge type={n.type} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium text-civic-ink">
                        {n.label}
                      </span>
                      <span className="block truncate text-[11px] text-civic-stone">
                        {n.id}
                      </span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>

        {/* Depth selector */}
        <label className="flex items-center gap-2 text-xs text-civic-stone">
          <span className="font-semibold uppercase tracking-wide">Depth</span>
          <select
            value={depth}
            onChange={(e) => {
              const d = Number(e.target.value) as 1 | 2 | 3;
              setDepth(d);
              if (selectedNodeId) loadNeighbourhood(selectedNodeId, d);
            }}
            className="rounded-md border border-civic-border bg-civic-mist px-2 py-1.5 text-sm focus:border-civic-leaf focus:outline-none"
            aria-label="Relationship depth"
          >
            <option value={1}>1 hop</option>
            <option value={2}>2 hops</option>
            <option value={3}>3 hops</option>
          </select>
        </label>

        {/* View toggle */}
        <div
          role="group"
          aria-label="View mode"
          className="flex overflow-hidden rounded-md border border-civic-border"
        >
          <ViewModeButton
            active={viewMode === 'graph'}
            onClick={() => setViewMode('graph')}
            label="Graph"
            icon={NetworkIcon}
          />
          <ViewModeButton
            active={viewMode === 'table'}
            onClick={() => setViewMode('table')}
            label="Table"
            icon={TableIcon}
          />
          <ViewModeButton
            active={viewMode === 'list'}
            onClick={() => setViewMode('list')}
            label="List"
            icon={ListIcon}
          />
        </div>

        {/* Find path toggle */}
        <button
          type="button"
          onClick={() => setPathMode((v) => !v)}
          className={`inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm font-medium transition ${
            pathMode
              ? 'border-civic-leaf bg-civic-leaf text-civic-paper'
              : 'border-civic-border bg-civic-paper text-civic-ink hover:border-civic-leaf'
          }`}
          aria-pressed={pathMode}
        >
          <ArrowRight className="h-4 w-4" aria-hidden="true" />
          Find Path
        </button>

        {/* Export PNG */}
        <button
          type="button"
          onClick={exportPNG}
          className="inline-flex items-center gap-1.5 rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-sm font-medium text-civic-ink hover:border-civic-leaf"
          aria-label="Export graph as PNG"
        >
          <Download className="h-4 w-4" aria-hidden="true" />
          PNG
        </button>
      </div>

      {/* === Path finder panel === */}
      {pathMode ? (
        <div className="rounded-lg border border-civic-border bg-civic-mist p-4">
          <h2 className="font-serif text-sm font-semibold text-civic-forest">
            Find shortest path between two nodes
          </h2>
          <p className="mt-1 text-xs text-civic-stone">
            Search for two nodes by label. The graph will highlight the
            shortest connection between them.
          </p>
          <div className="mt-3 grid items-end gap-3 sm:grid-cols-[1fr_1fr_auto]">
            <PathNodePicker
              label="From"
              value={pathFrom}
              onChange={setPathFrom}
              placeholder="e.g. ke-bill-constitution-2008"
            />
            <PathNodePicker
              label="To"
              value={pathTo}
              onChange={setPathTo}
              placeholder="e.g. president-mwai-kibaki"
            />
            <button
              type="button"
              onClick={findPath}
              disabled={loading || !pathFrom || !pathTo}
              className="inline-flex items-center gap-1.5 rounded-md bg-civic-forest px-4 py-2 text-sm font-medium text-civic-paper hover:bg-civic-leaf disabled:opacity-50"
            >
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <ArrowRight className="h-4 w-4" aria-hidden="true" />
              )}
              Find
            </button>
          </div>
          {pathError ? (
            <p className="mt-2 flex items-center gap-1.5 text-xs text-rose-700">
              <AlertCircle className="h-3.5 w-3.5" aria-hidden="true" />
              {pathError}
            </p>
          ) : null}
          {pathNodes.length > 0 ? (
            <ol className="mt-3 flex flex-wrap items-center gap-1 text-xs">
              {pathNodes.map((n, i) => (
                <li key={n.id} className="flex items-center gap-1">
                  <span
                    className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 ${
                      NODE_TYPE_META[n.type]?.badge ?? 'bg-civic-mist text-civic-ink'
                    }`}
                  >
                    {n.label}
                  </span>
                  {i < pathNodes.length - 1 ? (
                    <ArrowRight
                      className="h-3 w-3 text-civic-stone"
                      aria-hidden="true"
                    />
                  ) : null}
                </li>
              ))}
            </ol>
          ) : null}
        </div>
      ) : null}

      {/* === Error banner === */}
      {error ? (
        <div
          role="alert"
          className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-sm text-rose-800"
        >
          {error}
        </div>
      ) : null}

      {/* === Main view === */}
      <div className="grid gap-4 lg:grid-cols-[1fr_22rem]">
        <div
          ref={containerRef}
          className="relative h-[60vh] min-h-[420px] overflow-hidden rounded-lg border border-civic-border bg-civic-paper lg:h-[72vh]"
        >
          {loading ? (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-civic-paper/80 backdrop-blur-sm">
              <Loader2 className="h-6 w-6 animate-spin text-civic-forest" aria-hidden="true" />
              <span className="sr-only">Loading graph…</span>
            </div>
          ) : null}

          {viewMode === 'graph' ? (
            <GraphView
              svgRef={svgRef}
              simNodes={simNodesRef.current}
              simLinks={simLinksRef.current}
              simTick={simTick}
              transform={transform}
              selectedNodeId={selectedNodeId}
              selectedNeighbourIds={selectedNeighbourIds}
              pathEdgeSet={pathEdgeSet}
              onWheel={onWheel}
              onMouseDown={startPan}
              onMouseMove={(e) => {
                onPan(e);
                onNodeDrag(e);
              }}
              onMouseUp={() => {
                endPan();
                endNodeDrag();
              }}
              onMouseLeave={() => {
                endPan();
                endNodeDrag();
              }}
              onNodeMouseDown={startNodeDrag}
              onNodeClick={(id) => setSelectedNodeId(id)}
            />
          ) : viewMode === 'table' ? (
            <TableView nodes={nodes} edges={edges} onNodeClick={setSelectedNodeId} />
          ) : (
            <MobileListView
              nodes={nodes}
              edges={edges}
              onNodeClick={(id) => {
                setSelectedNodeId(id);
                loadNeighbourhood(id, depth);
              }}
            />
          )}

          {/* Zoom controls */}
          {viewMode === 'graph' ? (
            <div className="absolute bottom-3 right-3 flex flex-col gap-1">
              <button
                type="button"
                onClick={zoomIn}
                aria-label="Zoom in"
                className="inline-flex h-8 w-8 items-center justify-center rounded-md border border-civic-border bg-civic-paper text-civic-ink shadow-sm hover:border-civic-leaf"
              >
                <ZoomIn className="h-4 w-4" aria-hidden="true" />
              </button>
              <button
                type="button"
                onClick={zoomOut}
                aria-label="Zoom out"
                className="inline-flex h-8 w-8 items-center justify-center rounded-md border border-civic-border bg-civic-paper text-civic-ink shadow-sm hover:border-civic-leaf"
              >
                <ZoomOut className="h-4 w-4" aria-hidden="true" />
              </button>
              <button
                type="button"
                onClick={zoomReset}
                aria-label="Reset zoom"
                className="inline-flex h-8 w-8 items-center justify-center rounded-md border border-civic-border bg-civic-paper text-civic-ink shadow-sm hover:border-civic-leaf"
              >
                <Maximize2 className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          ) : null}

          {/* Legend */}
          {viewMode === 'graph' ? <Legend /> : null}
        </div>

        {/* === Side panel: node detail === */}
        <NodeDetailPanel
          node={selectedNode}
          neighbours={selectedNodeNeighbours.nodes}
          edges={selectedNodeNeighbours.edges}
          onClose={() => setSelectedNodeId(null)}
          onNeighbourClick={(id) => {
            setSelectedNodeId(id);
            loadNeighbourhood(id, depth);
            setViewMode('graph');
          }}
        />
      </div>

      {/* === Summary footnote === */}
      {summary ? (
        <p className="text-xs text-civic-stone">
          {summary.disclaimer}
        </p>
      ) : null}
    </div>
  );
}

// --- Sub-components ---

function ViewModeButton({
  active,
  onClick,
  label,
  icon: Icon,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
  icon: typeof NetworkIcon;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium transition ${
        active
          ? 'bg-civic-forest text-civic-paper'
          : 'bg-civic-paper text-civic-ink hover:bg-civic-mist'
      }`}
    >
      <Icon className="h-4 w-4" aria-hidden="true" />
      <span className="hidden sm:inline">{label}</span>
    </button>
  );
}

function TypeBadge({ type }: { type: GraphNodeType }) {
  const meta = NODE_TYPE_META[type];
  if (!meta) return null;
  return (
    <span
      className={`inline-flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full text-[9px] font-semibold uppercase ${meta.badge}`}
      title={meta.label}
    >
      {meta.label[0]}
    </span>
  );
}

function GraphView({
  svgRef,
  simNodes,
  simLinks,
  simTick,
  transform,
  selectedNodeId,
  selectedNeighbourIds,
  pathEdgeSet,
  onWheel,
  onMouseDown,
  onMouseMove,
  onMouseUp,
  onMouseLeave,
  onNodeMouseDown,
  onNodeClick,
}: {
  svgRef: React.MutableRefObject<SVGSVGElement | null>;
  simNodes: SimNode[];
  simLinks: SimLink[];
  simTick: number;
  transform: { x: number; y: number; k: number };
  selectedNodeId: string | null;
  selectedNeighbourIds: Set<string>;
  pathEdgeSet: Set<string>;
  onWheel: (e: React.WheelEvent) => void;
  onMouseDown: (e: React.MouseEvent) => void;
  onMouseMove: (e: React.MouseEvent) => void;
  onMouseUp: () => void;
  onMouseLeave: () => void;
  onNodeMouseDown: (e: React.MouseEvent, id: string) => void;
  onNodeClick: (id: string) => void;
}) {
  // simTick is read implicitly by reading simNodes[].x/y — we include it
  // in the dependency so React re-renders on every simulation tick.
  void simTick;

  return (
    <svg
      ref={svgRef}
      className="h-full w-full touch-none select-none"
      role="img"
      aria-label="Interactive force-directed graph of civic entities"
      onWheel={onWheel}
      onMouseDown={onMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={onMouseUp}
      onMouseLeave={onMouseLeave}
    >
      <rect
        x={0}
        y={0}
        width="100%"
        height="100%"
        fill="transparent"
        className="cursor-grab active:cursor-grabbing"
      />
      <g transform={`translate(${transform.x},${transform.y}) scale(${transform.k})`}>
        {/* Edges */}
        <g stroke="#c8d3cc" fill="none">
          {simLinks.map((link, i) => {
            const src =
              typeof link.source === 'number'
                ? simNodes[link.source]
                : typeof link.source === 'string'
                  ? simNodes.find((n) => n.id === link.source)
                  : (link.source as SimNode);
            const tgt =
              typeof link.target === 'number'
                ? simNodes[link.target]
                : typeof link.target === 'string'
                  ? simNodes.find((n) => n.id === link.target)
                  : (link.target as SimNode);
            if (!src || !tgt) return null;
            const key = `${src.id}|${tgt.id}|${link.type}`;
            const isPath = pathEdgeSet.has(key);
            const isHighlighted =
              selectedNodeId != null &&
              (src.id === selectedNodeId || tgt.id === selectedNodeId);
            const meta = EDGE_TYPE_META[link.type as GraphEdgeType];
            return (
              <line
                key={`${src.id}-${tgt.id}-${link.type}-${i}`}
                x1={src.x}
                y1={src.y}
                x2={tgt.x}
                y2={tgt.y}
                stroke={isPath ? '#0c3b2e' : meta?.colour ?? '#c8d3cc'}
                strokeWidth={isPath ? 3 : isHighlighted ? 2 : 1}
                strokeOpacity={isPath ? 1 : isHighlighted ? 0.8 : 0.4}
              >
                <title>{`${src.label} ${meta?.label ?? link.type} ${tgt.label}`}</title>
              </line>
            );
          })}
        </g>
        {/* Nodes */}
        <g>
          {simNodes.map((n) => {
            const meta = NODE_TYPE_META[n.type as GraphNodeType];
            if (!meta) return null;
            const isSelected = n.id === selectedNodeId;
            const isNeighbour = selectedNeighbourIds.has(n.id);
            const dim = selectedNodeId != null && !isSelected && !isNeighbour;
            return (
              <g
                key={n.id}
                transform={`translate(${n.x},${n.y})`}
                className="cursor-pointer"
                onMouseDown={(e) => onNodeMouseDown(e, n.id)}
                onClick={(e) => {
                  e.stopPropagation();
                  onNodeClick(n.id);
                }}
                opacity={dim ? 0.4 : 1}
              >
                <circle
                  r={isSelected ? 14 : 10}
                  fill={meta.fill}
                  stroke={isSelected ? '#0c3b2e' : meta.stroke}
                  strokeWidth={isSelected ? 3 : 2}
                />
                <title>{`${meta.label}: ${n.label}`}</title>
                <text
                  y={22}
                  textAnchor="middle"
                  className="pointer-events-none fill-civic-ink"
                  style={{ fontSize: '11px', fontWeight: 500 }}
                >
                  {n.label.length > 30 ? n.label.slice(0, 28) + '…' : n.label}
                </text>
              </g>
            );
          })}
        </g>
      </g>
    </svg>
  );
}

function TableView({
  nodes,
  edges,
  onNodeClick,
}: {
  nodes: GraphNode[];
  edges: GraphEdge[];
  onNodeClick: (id: string) => void;
}) {
  return (
    <div className="h-full overflow-auto">
      <div className="grid gap-4 p-3 md:grid-cols-2">
        <section aria-label="Nodes">
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-civic-stone">
            Nodes ({nodes.length})
          </h3>
          <table className="w-full text-left text-sm">
            <thead className="bg-civic-mist text-xs uppercase text-civic-stone">
              <tr>
                <th className="px-2 py-1.5">Type</th>
                <th className="px-2 py-1.5">Label</th>
                <th className="px-2 py-1.5">ID</th>
              </tr>
            </thead>
            <tbody>
              {nodes.map((n) => (
                <tr
                  key={n.id}
                  onClick={() => onNodeClick(n.id)}
                  className="cursor-pointer border-b border-civic-border hover:bg-civic-mist"
                >
                  <td className="px-2 py-1.5">
                    <TypeBadge type={n.type as GraphNodeType} />
                  </td>
                  <td className="px-2 py-1.5 text-civic-ink">{n.label}</td>
                  <td className="px-2 py-1.5 font-mono text-xs text-civic-stone">{n.id}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
        <section aria-label="Edges">
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-civic-stone">
            Edges ({edges.length})
          </h3>
          <table className="w-full text-left text-sm">
            <thead className="bg-civic-mist text-xs uppercase text-civic-stone">
              <tr>
                <th className="px-2 py-1.5">Source</th>
                <th className="px-2 py-1.5">Type</th>
                <th className="px-2 py-1.5">Target</th>
              </tr>
            </thead>
            <tbody>
              {edges.map((e, i) => (
                <tr key={`${e.source}-${e.target}-${e.type}-${i}`} className="border-b border-civic-border">
                  <td className="px-2 py-1.5 font-mono text-xs text-civic-stone">{e.source}</td>
                  <td className="px-2 py-1.5 text-civic-ink">{e.type}</td>
                  <td className="px-2 py-1.5 font-mono text-xs text-civic-stone">{e.target}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      </div>
    </div>
  );
}

function MobileListView({
  nodes,
  edges,
  onNodeClick,
}: {
  nodes: GraphNode[];
  edges: GraphEdge[];
  onNodeClick: (id: string) => void;
}) {
  // For each node, find its direct neighbours.
  const neighbourMap = useMemo(() => {
    const m = new Map<string, { node: GraphNode; edge: GraphEdge }[]>();
    for (const e of edges) {
      const srcNode = nodes.find((n) => n.id === e.source);
      const tgtNode = nodes.find((n) => n.id === e.target);
      if (!srcNode || !tgtNode) continue;
      if (!m.has(e.source)) m.set(e.source, []);
      if (!m.has(e.target)) m.set(e.target, []);
      m.get(e.source)!.push({ node: tgtNode, edge: e });
      m.get(e.target)!.push({ node: srcNode, edge: e });
    }
    return m;
  }, [nodes, edges]);

  return (
    <div className="h-full overflow-auto p-3">
      <ul className="space-y-2">
        {nodes.map((n) => {
          const neighbours = neighbourMap.get(n.id) ?? [];
          return (
            <MobileListRow
              key={n.id}
              node={n}
              neighbours={neighbours}
              onNodeClick={onNodeClick}
            />
          );
        })}
      </ul>
    </div>
  );
}

function MobileListRow({
  node,
  neighbours,
  onNodeClick,
}: {
  node: GraphNode;
  neighbours: { node: GraphNode; edge: GraphEdge }[];
  onNodeClick: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  return (
    <li className="rounded-lg border border-civic-border bg-civic-paper p-3">
      <div className="flex items-start gap-2">
        <TypeBadge type={node.type as GraphNodeType} />
        <button
          type="button"
          onClick={() => onNodeClick(node.id)}
          className="min-w-0 flex-1 text-left"
        >
          <span className="block truncate font-medium text-civic-ink">{node.label}</span>
          <span className="block truncate text-[11px] text-civic-stone">{node.id}</span>
        </button>
        {neighbours.length > 0 ? (
          <button
            type="button"
            onClick={() => setExpanded((v) => !v)}
            aria-expanded={expanded}
            className="rounded-md border border-civic-border px-2 py-1 text-xs text-civic-ink hover:border-civic-leaf"
          >
            {expanded ? 'Hide' : 'Expand'} ({neighbours.length})
          </button>
        ) : null}
      </div>
      {expanded && neighbours.length > 0 ? (
        <ul className="mt-2 space-y-1 border-t border-civic-border pt-2">
          {neighbours.map(({ node: nb, edge }) => (
            <li key={`${nb.id}-${edge.type}`} className="text-xs">
              <button
                type="button"
                onClick={() => onNodeClick(nb.id)}
                className="flex w-full items-center gap-1.5 rounded px-1 py-0.5 text-left hover:bg-civic-mist"
              >
                <span className="font-mono text-civic-stone">{edge.type}</span>
                <ArrowRight className="h-3 w-3 text-civic-stone" aria-hidden="true" />
                <TypeBadge type={nb.type as GraphNodeType} />
                <span className="truncate text-civic-ink">{nb.label}</span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </li>
  );
}

function NodeDetailPanel({
  node,
  neighbours,
  edges,
  onClose,
  onNeighbourClick,
}: {
  node: GraphNode | null;
  neighbours: GraphNode[];
  edges: GraphEdge[];
  onClose: () => void;
  onNeighbourClick: (id: string) => void;
}) {
  if (!node) {
    return (
      <aside
        className="hidden rounded-lg border border-civic-border bg-civic-paper p-4 text-sm text-civic-stone lg:block"
        aria-label="Node detail"
      >
        <p>Click a node to see its details and direct relationships.</p>
      </aside>
    );
  }
  const meta = NODE_TYPE_META[node.type as GraphNodeType];
  const sourceURL = (node.properties?.source_url as string | undefined) ?? '';
  return (
    <aside
      className="flex h-full max-h-[72vh] flex-col rounded-lg border border-civic-border bg-civic-paper"
      aria-label="Node detail"
    >
      <header className="flex items-start justify-between gap-2 border-b border-civic-border p-3">
        <div className="min-w-0 flex-1">
          {meta ? (
            <span
              className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${meta.badge}`}
            >
              {meta.label}
            </span>
          ) : null}
          <h2 className="mt-1.5 font-serif text-base font-semibold text-civic-ink">
            {node.label}
          </h2>
          <p className="mt-0.5 font-mono text-[11px] text-civic-stone">{node.id}</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close detail panel"
          className="inline-flex h-7 w-7 items-center justify-center rounded-md text-civic-stone hover:bg-civic-mist"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      </header>
      <div className="flex-1 overflow-auto p-3 text-sm">
        {/* Properties */}
        {node.properties && Object.keys(node.properties).length > 0 ? (
          <dl className="space-y-1.5">
            {Object.entries(node.properties).map(([k, v]) => (
              <div key={k} className="grid grid-cols-[8rem_1fr] gap-2 text-xs">
                <dt className="font-semibold uppercase tracking-wide text-civic-stone">
                  {k.replace(/_/g, ' ')}
                </dt>
                <dd className="break-words text-civic-ink">
                  {formatPropertyValue(v)}
                </dd>
              </div>
            ))}
          </dl>
        ) : null}

        {sourceURL ? (
          <a
            href={sourceURL}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 inline-flex items-center gap-1 text-xs text-civic-leaf hover:underline"
          >
            <ExternalLink className="h-3 w-3" aria-hidden="true" />
            View authoritative source
          </a>
        ) : null}

        {/* Relationships */}
        <h3 className="mt-4 mb-1 text-xs font-semibold uppercase tracking-wide text-civic-stone">
          Relationships ({neighbours.length})
        </h3>
        <ul className="space-y-1">
          {edges.map((e, i) => {
            const neighbourId = e.source === node.id ? e.target : e.source;
            const neighbour = neighbours.find((n) => n.id === neighbourId);
            if (!neighbour) return null;
            const direction = e.source === node.id ? '→' : '←';
            return (
              <li key={`${e.source}-${e.target}-${e.type}-${i}`}>
                <button
                  type="button"
                  onClick={() => onNeighbourClick(neighbour.id)}
                  className="flex w-full items-start gap-1.5 rounded px-1 py-1 text-left text-xs hover:bg-civic-mist"
                >
                  <span className="font-mono text-civic-stone">{e.type}</span>
                  <span className="text-civic-stone">{direction}</span>
                  <TypeBadge type={neighbour.type as GraphNodeType} />
                  <span className="min-w-0 flex-1 truncate text-civic-ink">
                    {neighbour.label}
                  </span>
                </button>
              </li>
            );
          })}
          {neighbours.length === 0 ? (
            <li className="px-1 py-1 text-xs text-civic-stone">
              No direct relationships.
            </li>
          ) : null}
        </ul>
      </div>
    </aside>
  );
}

function PathNodePicker({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
}) {
  const [q, setQ] = useState('');
  const [results, setResults] = useState<GraphNode[]>([]);
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!q.trim()) {
      setResults([]);
      return;
    }
    const h = setTimeout(async () => {
      try {
        const resp = await searchGraphNodes(q, 10);
        setResults(resp.nodes);
      } catch {
        setResults([]);
      }
    }, 250);
    return () => clearTimeout(h);
  }, [q]);

  return (
    <div className="relative">
      <label className="block text-[11px] font-semibold uppercase tracking-wide text-civic-stone">
        {label}
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={() => {
          setQ(value);
          setOpen(true);
        }}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        onInput={(e) => {
          setQ((e.target as HTMLInputElement).value);
          setOpen(true);
        }}
        placeholder={placeholder}
        className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-sm focus:border-civic-leaf focus:outline-none"
        aria-label={`${label} node`}
      />
      {open && results.length > 0 ? (
        <ul
          role="listbox"
          aria-label={`${label} search results`}
          className="absolute z-30 mt-1 max-h-60 w-full overflow-auto rounded-md border border-civic-border bg-civic-paper shadow-lg"
        >
          {results.map((n) => (
            <li key={n.id} role="option" aria-selected={value === n.id}>
              <button
                type="button"
                onMouseDown={(e) => {
                  e.preventDefault();
                  onChange(n.id);
                  setQ(n.label);
                  setOpen(false);
                }}
                className="flex w-full items-start gap-2 px-3 py-2 text-left text-sm hover:bg-civic-mist"
              >
                <TypeBadge type={n.type as GraphNodeType} />
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-medium text-civic-ink">{n.label}</span>
                  <span className="block truncate text-[11px] text-civic-stone">{n.id}</span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function Legend() {
  return (
    <details className="absolute left-3 top-3 max-w-xs rounded-md border border-civic-border bg-civic-paper/95 p-2 text-xs shadow-sm">
      <summary className="cursor-pointer font-semibold text-civic-forest">
        Legend
      </summary>
      <div className="mt-2 space-y-1.5">
        <p className="font-semibold uppercase text-civic-stone">Nodes</p>
        <ul className="grid grid-cols-2 gap-x-2 gap-y-1">
          {Object.entries(NODE_TYPE_META).map(([type, meta]) => (
            <li key={type} className="flex items-center gap-1.5">
              <span
                className="inline-block h-3 w-3 rounded-full"
                style={{ backgroundColor: meta.fill, border: `1.5px solid ${meta.stroke}` }}
                aria-hidden="true"
              />
              <span className="text-civic-ink">{meta.label}</span>
            </li>
          ))}
        </ul>
      </div>
    </details>
  );
}

// --- Helpers ---

function formatPropertyValue(v: unknown): string {
  if (v == null) return '';
  if (typeof v === 'string') return v;
  if (typeof v === 'number' || typeof v === 'boolean') return String(v);
  try {
    return JSON.stringify(v);
  } catch {
    return String(v);
  }
}
