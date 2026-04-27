"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  ReactFlowProvider,
  Panel,
  useStore,
  type Node,
  type Edge,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "@dagrejs/dagre";

import { useRouter } from "next/navigation";
import { useSearchParams } from "next/navigation";
import { listAgents, listOffices, getOffice, listSchedules, listSquads } from "@/lib/api";
import { usePageRefresh } from "@/components/layout/app-shell";
import type {
  AgentResponse,
  OfficeResponse,
  ScheduleResponse,
  Squad,
} from "@/lib/types";
import { AnimatePresence, motion } from "framer-motion";
import { Check, ChevronDown } from "lucide-react";
import {
  AgentNode,
  OfficeNode,
  ScheduleNode,
} from "./node-types";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/kit/select";

/* ------------------------------------------------------------------ */
/*  Node-type registry (must be stable reference)                     */
/* ------------------------------------------------------------------ */

const nodeTypes = {
  agent: AgentNode,
  office: OfficeNode,
  schedule: ScheduleNode,
} as const;

/* ------------------------------------------------------------------ */
/*  Dagre layout helper                                               */
/* ------------------------------------------------------------------ */

const NODE_WIDTH = 200;
const NODE_HEIGHT = 80;
const GROUP_GAP = 80;

function layoutSingleGroup(nodes: Node[], edges: Edge[]): { nodes: Node[]; width: number; height: number } {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", ranksep: 80, nodesep: 50 });

  const nodeWidths = new Map<string, number>();
  for (const node of nodes) {
    // Estimate rendered width from label length (8px per char + padding)
    const label = (node.data as { label?: string })?.label || "";
    const estimated = Math.max(NODE_WIDTH, label.length * 8 + 60);
    const w = node.type === "office" ? Math.max(estimated, 220) : estimated;
    nodeWidths.set(node.id, w);
    g.setNode(node.id, { width: w, height: NODE_HEIGHT });
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  // Find leaf nodes (no outgoing edges) to rearrange into a grid
  const sources = new Set(edges.map((e) => e.source));
  const leafIds = new Set(nodes.filter((n) => !sources.has(n.id)).map((n) => n.id));
  const MAX_COLS = 2;

  const laid = nodes.map((node) => {
    const pos = g.node(node.id);
    const w = nodeWidths.get(node.id) || NODE_WIDTH;
    return { ...node, position: { x: pos.x - w / 2, y: pos.y - NODE_HEIGHT / 2 } };
  });

  // Re-layout leaf nodes into a grid if there are more than MAX_COLS
  if (leafIds.size > 1) {
    const leafNodes = laid.filter((n) => leafIds.has(n.id));
    const nonLeafNodes = laid.filter((n) => !leafIds.has(n.id));

    // Find the parent node (the one with edges to leaves) to center the grid under it
    const parentId = edges.find((e) => leafIds.has(e.target))?.source;
    const parent = parentId ? nonLeafNodes.find((n) => n.id === parentId) : null;
    const parentCenterX = parent
      ? parent.position.x + (nodeWidths.get(parent.id) || NODE_WIDTH) / 2
      : 0;

    // Find where the leaf row starts (Y position from Dagre)
    const leafY = Math.min(...leafNodes.map((n) => n.position.y));
    const cols = Math.min(MAX_COLS, leafNodes.length);

    // Sort leaves by their dagre X position to maintain relative order
    leafNodes.sort((a, b) => a.position.x - b.position.x);

    const colGap = 20;
    const rowGap = 20;
    const maxLeafW = Math.max(...leafNodes.map((n) => nodeWidths.get(n.id) || NODE_WIDTH));
    const gridWidth = cols * maxLeafW + (cols - 1) * colGap;
    const startX = parentCenterX - gridWidth / 2;

    for (let i = 0; i < leafNodes.length; i++) {
      const col = i % cols;
      const row = Math.floor(i / cols);
      leafNodes[i].position = {
        x: startX + col * (maxLeafW + colGap),
        y: leafY + row * (NODE_HEIGHT + rowGap),
      };
    }

    laid.length = 0;
    laid.push(...nonLeafNodes, ...leafNodes);
  }

  // Normalize positions to start at 0,0
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  for (const node of laid) {
    const w = nodeWidths.get(node.id) || NODE_WIDTH;
    minX = Math.min(minX, node.position.x);
    minY = Math.min(minY, node.position.y);
    maxX = Math.max(maxX, node.position.x + w);
    maxY = Math.max(maxY, node.position.y + NODE_HEIGHT);
  }

  const normalized = laid.map((n) => ({
    ...n,
    position: { x: n.position.x - minX, y: n.position.y - minY },
  }));

  return { nodes: normalized, width: maxX - minX, height: maxY - minY };
}

function applyDagreLayout(nodes: Node[], edges: Edge[], squads?: Squad[]): { nodes: Node[]; firstRowNodeIds: Set<string> } {
  const firstRowNodeIds = new Set<string>();
  // Find connected components
  const nodeIds = new Set(nodes.map((n) => n.id));
  const adj = new Map<string, Set<string>>();
  for (const id of nodeIds) adj.set(id, new Set());
  for (const edge of edges) {
    adj.get(edge.source)?.add(edge.target);
    adj.get(edge.target)?.add(edge.source);
  }

  const visited = new Set<string>();
  const components: Set<string>[] = [];
  for (const id of nodeIds) {
    if (visited.has(id)) continue;
    const component = new Set<string>();
    const stack = [id];
    while (stack.length > 0) {
      const curr = stack.pop()!;
      if (visited.has(curr)) continue;
      visited.add(curr);
      component.add(curr);
      for (const neighbor of adj.get(curr) || []) {
        if (!visited.has(neighbor)) stack.push(neighbor);
      }
    }
    components.push(component);
  }

  // Separate into connected groups (with edges) and standalone nodes
  const groups: { nodeIds: Set<string>; hasEdges: boolean }[] = [];
  const standaloneIds: string[] = [];
  for (const comp of components) {
    const hasEdges = edges.some((e) => comp.has(e.source) || comp.has(e.target));
    if (comp.size === 1 && !hasEdges) {
      standaloneIds.push([...comp][0]);
    } else {
      groups.push({ nodeIds: comp, hasEdges: true });
    }
  }

  const allNodes: Node[] = [];
  let currentY = 0;

  const ROW_GAP = 40;

  // 1. Standalone agents in a grid row at the top
  //    Layout hint: sort so same-squad nodes are consecutive in the grid
  if (standaloneIds.length > 0 && squads && squads.length > 0) {
    // Build a map: nodeId -> squad index (lower = placed earlier)
    const nodeSquadIdx = new Map<string, number>();
    squads.forEach((sq, si) => {
      sq.members.forEach((m) => {
        const nid = `agent-${m.name}`;
        if (!nodeSquadIdx.has(nid)) nodeSquadIdx.set(nid, si);
      });
    });
    standaloneIds.sort((a, b) => {
      const ai = nodeSquadIdx.get(a) ?? 999;
      const bi = nodeSquadIdx.get(b) ?? 999;
      return ai !== bi ? ai - bi : a.localeCompare(b);
    });
  }

  if (standaloneIds.length > 0) {
    const cols = Math.min(standaloneIds.length, 6);
    for (let i = 0; i < standaloneIds.length; i++) {
      const node = nodes.find((n) => n.id === standaloneIds[i])!;
      const col = i % cols;
      const row = Math.floor(i / cols);
      const placed = {
        ...node,
        position: {
          x: col * (NODE_WIDTH + 30),
          y: currentY + row * (NODE_HEIGHT + 30),
        },
      };
      allNodes.push(placed);
      firstRowNodeIds.add(placed.id);
    }
    const standaloneRows = Math.ceil(standaloneIds.length / 6);
    currentY += standaloneRows * (NODE_HEIGHT + 30) + GROUP_GAP;
  }

  // 2. Connected groups (offices) arranged in rows (max 4 per row)
  const MAX_GROUPS_PER_ROW = 4;
  let offsetX = 0;
  let rowMaxHeight = 0;
  let groupRow = 0;
  for (let i = 0; i < groups.length; i++) {
    if (i > 0 && i % MAX_GROUPS_PER_ROW === 0) {
      currentY += rowMaxHeight + ROW_GAP;
      offsetX = 0;
      rowMaxHeight = 0;
      groupRow++;
    }
    const group = groups[i];
    const groupNodes = nodes.filter((n) => group.nodeIds.has(n.id));
    const groupEdges = edges.filter((e) => group.nodeIds.has(e.source));
    const { nodes: laid, width, height } = layoutSingleGroup(groupNodes, groupEdges);
    for (const n of laid) {
      const placed = { ...n, position: { x: n.position.x + offsetX, y: n.position.y + currentY } };
      allNodes.push(placed);
      if (groupRow === 0) firstRowNodeIds.add(placed.id);
    }
    offsetX += width + GROUP_GAP;
    rowMaxHeight = Math.max(rowMaxHeight, height);
  }

  return { nodes: allNodes, firstRowNodeIds };
}

/* ------------------------------------------------------------------ */
/*  Squad color helper                                                */
/* ------------------------------------------------------------------ */

function squadColor(name: string): string {
  // Stable color from squad name hash — warm palette that contrasts on dark bg
  let h = 0;
  for (let i = 0; i < name.length; i++) {
    h = (h * 31 + name.charCodeAt(i)) >>> 0;
  }
  const hue = h % 360;
  return `hsl(${hue}, 70%, 60%)`;
}

/* ------------------------------------------------------------------ */
/*  Squad overlay: renders colored borders in ReactFlow SVG space     */
/* ------------------------------------------------------------------ */

const BORDER_PAD = 24;

interface SquadCluster {
  squadName: string;
  namespace: string;
  color: string;
  label: string;
  x: number;
  y: number;
  width: number;
  height: number;
}

function computeSquadClusters(squads: Squad[], nodes: Node[]): SquadCluster[] {
  const clusters: SquadCluster[] = [];

  for (const squad of squads) {
    const color = squadColor(squad.name);
    // Find laid-out nodes belonging to this squad
    const memberNodeIds = new Set(squad.members.map((m) => `agent-${m.name}`));
    const memberNodes = nodes.filter((n) => memberNodeIds.has(n.id));
    if (memberNodes.length === 0) continue;

    // Spatial greedy clustering: group nodes within maxDistance of each other
    const MAX_DISTANCE = 400;
    const used = new Set<string>();
    const clusterGroups: Node[][] = [];

    // Sort by position for deterministic grouping
    const sorted = [...memberNodes].sort((a, b) => a.position.x - b.position.x || a.position.y - b.position.y);

    for (const node of sorted) {
      if (used.has(node.id)) continue;
      const cluster: Node[] = [node];
      used.add(node.id);

      // Keep adding nearest unused nodes that are within maxDistance of cluster centroid
      let changed = true;
      while (changed) {
        changed = false;
        const cx = cluster.reduce((s, n) => s + n.position.x, 0) / cluster.length;
        const cy = cluster.reduce((s, n) => s + n.position.y, 0) / cluster.length;
        for (const candidate of sorted) {
          if (used.has(candidate.id)) continue;
          const dx = candidate.position.x - cx;
          const dy = candidate.position.y - cy;
          if (Math.sqrt(dx * dx + dy * dy) <= MAX_DISTANCE) {
            cluster.push(candidate);
            used.add(candidate.id);
            changed = true;
          }
        }
      }
      clusterGroups.push(cluster);
    }

    // Produce a SquadCluster rect for each group
    const totalClusters = clusterGroups.length;
    clusterGroups.forEach((clusterNodes, idx) => {
      let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
      for (const n of clusterNodes) {
        const w = (n.data as { nodeWidth?: number }).nodeWidth ?? NODE_WIDTH;
        minX = Math.min(minX, n.position.x);
        minY = Math.min(minY, n.position.y);
        maxX = Math.max(maxX, n.position.x + w);
        maxY = Math.max(maxY, n.position.y + NODE_HEIGHT);
      }

      const label = totalClusters > 1
        ? `${squad.name} (${idx + 1}/${totalClusters})`
        : squad.name;

      clusters.push({
        squadName: squad.name,
        namespace: squad.namespace,
        color,
        label,
        x: minX - BORDER_PAD,
        y: minY - BORDER_PAD,
        width: maxX - minX + BORDER_PAD * 2,
        height: maxY - minY + BORDER_PAD * 2,
      });
    });
  }

  return clusters;
}

function SquadOverlay({ squads, nodes }: { squads: Squad[]; nodes: Node[] }) {
  const router = useRouter();
  // Subscribe to viewport changes so we re-render on pan/zoom
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const transform = useStore((s: any) => s.transform as [number, number, number]);
  const [tx, ty, zoom] = transform;

  const clusters = useMemo(() => computeSquadClusters(squads, nodes), [squads, nodes]);

  if (clusters.length === 0) return null;

  // Project a flow-space rect to screen-space coords (inside the ReactFlow wrapper div)
  const project = (fx: number, fy: number) => ({
    sx: fx * zoom + tx,
    sy: fy * zoom + ty,
  });

  return (
    <Panel position="top-left" style={{ inset: 0, pointerEvents: "none" }}>
      <svg
        style={{ position: "absolute", inset: 0, width: "100%", height: "100%", overflow: "visible" }}
        // Re-render key ensures React recreates on transform change
        key={`${tx.toFixed(1)}-${ty.toFixed(1)}-${zoom.toFixed(3)}`}
      >
        {clusters.map((c: SquadCluster) => {
          const tl = project(c.x, c.y);
          const w = c.width * zoom;
          const h = c.height * zoom;
          const FONT_SIZE = Math.max(10, Math.min(13, 12 * zoom));
          const LABEL_INSET = 12 * zoom;
          const LABEL_HEIGHT = FONT_SIZE + 8;

          return (
            <g key={`${c.squadName}-${c.label}`}>
              {/* Border rect — pointer-events: none so agents underneath are draggable */}
              <rect
                x={tl.sx}
                y={tl.sy}
                width={w}
                height={h}
                rx={10 * zoom}
                ry={10 * zoom}
                fill={c.color}
                fillOpacity={0.05}
                stroke={c.color}
                strokeWidth={2}
                strokeOpacity={0.7}
                style={{ pointerEvents: "none" }}
              />
              {/* Label as foreignObject — auto-sizes to text content, only the label is clickable */}
              <foreignObject
                x={tl.sx + LABEL_INSET}
                y={tl.sy - LABEL_HEIGHT / 2}
                width={Math.max(60, w - LABEL_INSET * 2)}
                height={LABEL_HEIGHT}
                style={{ overflow: "visible" }}
              >
                <div
                  onClick={() => router.push(`/squads/${c.squadName}?namespace=${c.namespace}`)}
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    height: LABEL_HEIGHT,
                    padding: "0 8px",
                    backgroundColor: "var(--color-bg, #0d1117)",
                    color: c.color,
                    fontSize: FONT_SIZE,
                    fontWeight: 600,
                    fontFamily: "var(--font-mono, monospace)",
                    borderRadius: 4,
                    lineHeight: 1,
                    whiteSpace: "nowrap",
                    cursor: "pointer",
                    pointerEvents: "all",
                  }}
                >
                  {c.label}
                </div>
              </foreignObject>
            </g>
          );
        })}
      </svg>
    </Panel>
  );
}

/* ------------------------------------------------------------------ */
/*  Build nodes + edges from API data                                 */
/* ------------------------------------------------------------------ */

function buildGraph(
  agents: AgentResponse[],
  offices: OfficeResponse[],
  schedules: ScheduleResponse[],
  squads?: Squad[]
) {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  // Track which agents belong to an office (to avoid duplication issues)
  const agentSet = new Set(agents.map((a) => a.name));

  // Track which agent nodes have been added (to avoid duplicates)
  const addedAgentNodes = new Set<string>();

  // Agent nodes (live agents)
  for (const agent of agents) {
    addedAgentNodes.add(agent.name);
    nodes.push({
      id: `agent-${agent.name}`,
      type: "agent",
      position: { x: 0, y: 0 },
      data: {
        label: agent.name,
        status: agent.status,
        model: agent.model,
        namespace: agent.namespace,
        taskStatus: agent.taskStatus,
        lifecycle: agent.lifecycle,
        totalCostUSD: agent.totalCostUSD,
      },
    });
  }

  // Office nodes
  for (const office of offices) {
    nodes.push({
      id: `office-${office.name}`,
      type: "office",
      position: { x: 0, y: 0 },
      data: {
        label: office.name,
        phase: office.phase,
        agentCount: office.totalAgents,
        namespace: office.namespace,
        manager: office.manager,
        totalCostUSD: office.totalCostUSD,
      },
    });

    // Ensure manager agent node exists (even if deleted)
    if (office.manager && !addedAgentNodes.has(office.manager)) {
      addedAgentNodes.add(office.manager);
      nodes.push({
        id: `agent-${office.manager}`,
        type: "agent",
        position: { x: 0, y: 0 },
        data: {
          label: office.manager,
          status: "Deleted",
          model: "",
        },
      });
    }

    // Edge: office -> manager agent
    if (office.manager) {
      edges.push({
        id: `e-office-${office.name}-manager-${office.manager}`,
        source: `office-${office.name}`,
        target: `agent-${office.manager}`,
        style: { stroke: "#3f85d9", strokeWidth: 2 },
        animated: false,
      });
    }

    // Worker agents — create nodes for deleted ones, always draw edges
    for (const member of office.members || []) {
      if (member.role === "worker") {
        if (!addedAgentNodes.has(member.name)) {
          addedAgentNodes.add(member.name);
          nodes.push({
            id: `agent-${member.name}`,
            type: "agent",
            position: { x: 0, y: 0 },
            data: {
              label: member.name,
              status: "Deleted",
              model: "",
            },
          });
        }
        edges.push({
          id: `e-manager-${office.manager}-worker-${member.name}`,
          source: `agent-${office.manager}`,
          target: `agent-${member.name}`,
          style: { stroke: "#7c6bc4", strokeWidth: 1.5 },
          animated: false,
        });
      }
    }
  }

  // Schedule nodes + edges
  for (const schedule of schedules) {
    nodes.push({
      id: `schedule-${schedule.name}`,
      type: "schedule",
      position: { x: 0, y: 0 },
      data: {
        label: schedule.name,
        cron: schedule.schedule,
        phase: schedule.phase,
        namespace: schedule.namespace,
        agentName: schedule.agentName,
        totalCostUSD: schedule.totalCostUSD,
        runCount: schedule.runCount,
      },
    });

    if (schedule.agentName && agentSet.has(schedule.agentName)) {
      edges.push({
        id: `e-schedule-${schedule.name}-agent-${schedule.agentName}`,
        source: `schedule-${schedule.name}`,
        target: `agent-${schedule.agentName}`,
        style: { stroke: "#9775d6", strokeWidth: 1.5, strokeDasharray: "6 3" },
        animated: false,
      });
    }
  }

  // Compute which nodes have incoming/outgoing connections + pass layout width
  const hasIncoming = new Set(edges.map((e) => e.target));
  const hasOutgoing = new Set(edges.map((e) => e.source));
  for (const node of nodes) {
    const label = (node.data as { label?: string })?.label || "";
    const w = node.type === "office"
      ? Math.max(220, label.length * 8 + 80)
      : Math.max(160, label.length * 8 + 60);
    node.data = {
      ...node.data,
      hasIncoming: hasIncoming.has(node.id),
      hasOutgoing: hasOutgoing.has(node.id),
      nodeWidth: w,
    };
  }

  // Apply layout (pass squads for positional hints)
  const { nodes: laidOutNodes, firstRowNodeIds } = applyDagreLayout(nodes, edges, squads);
  return { nodes: laidOutNodes, edges, firstRowNodeIds };
}

/* ------------------------------------------------------------------ */
/*  Inner component (uses React Flow hooks)                           */
/* ------------------------------------------------------------------ */

function TopologyGraphInner({
  initialNodes,
  initialEdges,
  firstRowNodeIds,
  squads,
}: {
  initialNodes: Node[];
  initialEdges: Edge[];
  firstRowNodeIds: Set<string>;
  squads: Squad[];
}) {
  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, , onEdgesChange] = useEdgesState(initialEdges);

  const fitViewOptions = useMemo(() => ({
    nodes: initialNodes.filter((n) => firstRowNodeIds.has(n.id)).map((n) => ({ id: n.id })),
    padding: 0.15,
  }), [initialNodes, firstRowNodeIds]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      nodeTypes={nodeTypes}
      fitView
      fitViewOptions={fitViewOptions}
      proOptions={{ hideAttribution: true }}
      minZoom={0.2}
      maxZoom={2}
    >
      <Background color="var(--color-border)" gap={24} size={1} />
      {squads.length > 0 && <SquadOverlay squads={squads} nodes={nodes} />}
      <Controls
        className="!bg-[var(--color-surface)] !border-[var(--color-border)] !shadow-lg [&>button]:!bg-[var(--color-surface)] [&>button]:!border-[var(--color-border)] [&>button]:!text-[var(--color-text)] [&>button:hover]:!bg-[var(--color-bg)]"
      />
      <MiniMap
        nodeColor={(node) => node.id.startsWith("schedule-") ? "#8B5CF6" : "#3f85d9"}
        maskColor="rgba(26,35,50,0.8)"
        className="!bg-[var(--color-surface)] !border-[var(--color-border)]"
      />
    </ReactFlow>
  );
}

/* ------------------------------------------------------------------ */
/*  Exported component                                                */
/* ------------------------------------------------------------------ */

export function TopologyGraph() {
  const [allAgents, setAllAgents] = useState<AgentResponse[]>([]);
  const [allOffices, setAllOffices] = useState<OfficeResponse[]>([]);
  const [allSchedules, setAllSchedules] = useState<ScheduleResponse[]>([]);
  const [allSquads, setAllSquads] = useState<Squad[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const searchParams = useSearchParams();
  const [nsFilter, setNsFilter] = useState<string>(searchParams.get("ns") || "all");
  const [officeFilter, setOfficeFilter] = useState<string[]>(() => {
    const val = searchParams.get("office");
    return val ? val.split(",") : [];
  });
  const [officeDropdownOpen, setOfficeDropdownOpen] = useState(false);
  const officeDropdownRef = useRef<HTMLDivElement>(null);

  // Sync filters to URL
  useEffect(() => {
    const params = new URLSearchParams();
    if (nsFilter !== "all") params.set("ns", nsFilter);
    if (officeFilter.length > 0) params.set("office", officeFilter.join(","));
    const qs = params.toString();
    const url = qs ? `${window.location.pathname}?${qs}` : window.location.pathname;
    window.history.replaceState(null, "", url);
  }, [nsFilter, officeFilter]);

  // Close office dropdown on click outside (capture phase so React Flow doesn't swallow it)
  useEffect(() => {
    if (!officeDropdownOpen) return;
    function handleClick(e: PointerEvent) {
      if (officeDropdownRef.current && !officeDropdownRef.current.contains(e.target as globalThis.Node)) {
        setOfficeDropdownOpen(false);
      }
    }
    document.addEventListener("pointerdown", handleClick, true);
    return () => document.removeEventListener("pointerdown", handleClick, true);
  }, [officeDropdownOpen]);

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, officesRes, schedulesRes, squadsRes] = await Promise.all([
        listAgents(),
        listOffices(),
        listSchedules(),
        listSquads().catch(() => ({ squads: [] })),
      ]);
      // List endpoint doesn't include members — fetch each office individually.
      const officeList = officesRes.offices || [];
      const detailedOffices = await Promise.all(
        officeList.map((o) => getOffice(o.name, o.namespace).catch(() => o))
      );
      setAllAgents(agentsRes.agents || []);
      setAllOffices(detailedOffices);
      setAllSchedules(schedulesRes.schedules || []);
      setAllSquads(squadsRes.squads || []);
      setError(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to load topology data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  usePageRefresh(fetchData);

  // Derive unique namespaces and office names for filter options
  const namespaces = useMemo(() => {
    const ns = new Set<string>();
    allAgents.forEach((a) => ns.add(a.namespace));
    allOffices.forEach((o) => ns.add(o.namespace));
    allSchedules.forEach((s) => ns.add(s.namespace));
    return Array.from(ns).sort();
  }, [allAgents, allOffices, allSchedules]);

  const officeNames = useMemo(() => {
    return allOffices.map((o) => o.name).sort();
  }, [allOffices]);

  // Apply filters
  const graphData = useMemo(() => {
    let agents = allAgents;
    let offices = allOffices;
    let schedules = allSchedules;
    let squads = allSquads;

    if (nsFilter !== "all") {
      agents = agents.filter((a) => a.namespace === nsFilter);
      offices = offices.filter((o) => o.namespace === nsFilter);
      schedules = schedules.filter((s) => s.namespace === nsFilter);
      squads = squads.filter((sq) => sq.namespace === nsFilter);
    }

    if (officeFilter.length > 0) {
      const filterSet = new Set(officeFilter);
      const selectedOffices = offices.filter((o) => filterSet.has(o.name));
      if (selectedOffices.length > 0) {
        const memberNames = new Set<string>();
        for (const office of selectedOffices) {
          if (office.manager) memberNames.add(office.manager);
          for (const m of office.members || []) memberNames.add(m.name);
        }
        agents = agents.filter((a) => memberNames.has(a.name));
        offices = selectedOffices;
        schedules = [];
      }
    }

    if (agents.length === 0 && offices.length === 0 && schedules.length === 0) {
      return null;
    }

    return { ...buildGraph(agents, offices, schedules, squads), squads };
  }, [allAgents, allOffices, allSchedules, allSquads, nsFilter, officeFilter]);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--color-brand-blue)] border-t-transparent" />
          <p className="text-sm text-[var(--color-text-secondary)]">
            Loading topology...
          </p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="rounded-lg border border-red-400/20 bg-red-400/5 p-4 text-sm text-red-400">
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Filter bar */}
      <div className="shrink-0 flex items-center justify-end gap-4 px-4 py-2 border-b border-[var(--color-border)] bg-[var(--color-bg-subtle)]">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-[var(--color-text-secondary)]">Namespace</span>
          <Select value={nsFilter} onValueChange={setNsFilter}>
            <SelectTrigger className="w-36 !text-[11px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              {namespaces.map((ns) => (
                <SelectItem key={ns} value={ns}>{ns}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-[var(--color-text-secondary)]">Office</span>
          <div className="relative" ref={officeDropdownRef}>
            <button
              type="button"
              className="flex items-center justify-between w-56 h-8 px-3 rounded-[var(--radius-sm)] text-[11px] font-[family-name:var(--font-mono)] bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)] shadow-[inset_0_2px_4px_rgba(0,0,0,0.2)] transition-all duration-150 cursor-pointer hover:border-[var(--color-border-hover)] focus:outline-none focus:border-[var(--color-brand-blue)]/60"
              onClick={() => setOfficeDropdownOpen(!officeDropdownOpen)}
            >
              <span className="truncate">
                {officeFilter.length === 0 ? "All" : officeFilter.length === 1 ? officeFilter[0] : `${officeFilter.length} selected`}
              </span>
              <ChevronDown className={`h-4 w-4 text-[var(--color-text-muted)] transition-transform duration-200 ${officeDropdownOpen ? "rotate-180" : ""}`} />
            </button>
            <AnimatePresence>
              {officeDropdownOpen && (
                <motion.div
                  className="absolute right-0 z-50 w-56 mt-1 py-1 rounded-[var(--radius-md)] bg-[var(--color-surface-raised)] border border-[var(--color-border)] shadow-[0_8px_32px_rgba(0,0,0,0.4),0_2px_8px_rgba(0,0,0,0.2)] overflow-y-auto max-h-60"
                  initial={{ opacity: 0, y: -4, scale: 0.98 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: -4, scale: 0.98 }}
                  transition={{ duration: 0.12, ease: "easeOut" }}
                >
                  <div
                    className={`flex items-center justify-between px-3 py-2 text-sm cursor-pointer transition-colors hover:bg-[var(--color-surface-hover)] ${officeFilter.length === 0 ? "text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"}`}
                    onClick={() => setOfficeFilter([])}
                  >
                    <span>All</span>
                    {officeFilter.length === 0 && <Check className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />}
                  </div>
                  {officeNames.map((name) => {
                    const selected = officeFilter.includes(name);
                    return (
                      <div
                        key={name}
                        className={`flex items-center justify-between px-3 py-2 text-sm cursor-pointer transition-colors hover:bg-[var(--color-surface-hover)] ${selected ? "text-[var(--color-brand-blue)]" : "text-[var(--color-text)]"}`}
                        onClick={() => {
                          setOfficeFilter((prev) =>
                            selected ? prev.filter((n) => n !== name) : [...prev, name]
                          );
                        }}
                      >
                        <span className="truncate">{name}</span>
                        {selected && <Check className="h-4 w-4 shrink-0 text-[var(--color-brand-blue)]" />}
                      </div>
                    );
                  })}
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </div>

      {/* Graph */}
      <div className="flex-1">
        <AnimatePresence mode="wait">
          {!graphData || graphData.nodes.length === 0 ? (
            <motion.div
              key="empty"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="flex h-full items-center justify-center"
            >
              <p className="text-sm text-[var(--color-text-secondary)]">
                No agents, offices, or schedules found.
              </p>
            </motion.div>
          ) : (
            <motion.div
              key={`${nsFilter}-${officeFilter.join(",")}`}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.25 }}
              className="h-full"
            >
              <ReactFlowProvider>
                <TopologyGraphInner
                  initialNodes={graphData.nodes}
                  initialEdges={graphData.edges}
                  firstRowNodeIds={graphData.firstRowNodeIds}
                  squads={graphData.squads}
                />
              </ReactFlowProvider>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Squad-scoped topology — embedded on the squad details page        */
/* ------------------------------------------------------------------ */

export function SquadTopologyGraph({ squad, agents }: { squad: Squad; agents: AgentResponse[] }) {
  const memberNames = useMemo(() => new Set(squad.members.map((m) => m.name)), [squad.members]);
  const filteredAgents = useMemo(
    () => agents.filter((a) => memberNames.has(a.name) && a.namespace === squad.namespace),
    [agents, memberNames, squad.namespace],
  );

  const graphData = useMemo(() => {
    const built = buildGraph(filteredAgents, [], [], [squad]);
    return { ...built, squads: [squad] };
  }, [filteredAgents, squad]);

  if (filteredAgents.length === 0) {
    return (
      <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] p-4 text-sm text-[var(--color-text-secondary)]">
        No member agents to display.
      </div>
    );
  }

  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden h-[420px]">
      <ReactFlowProvider>
        <TopologyGraphInner
          initialNodes={graphData.nodes}
          initialEdges={graphData.edges}
          firstRowNodeIds={graphData.firstRowNodeIds}
          squads={[squad]}
        />
      </ReactFlowProvider>
    </div>
  );
}
