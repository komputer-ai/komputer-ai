"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  ReactFlowProvider,
  type Node,
  type Edge,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "@dagrejs/dagre";

import { listAgents, listOffices, getOffice, listSchedules } from "@/lib/api";
import { usePageRefresh } from "@/components/layout/app-shell";
import type {
  AgentResponse,
  OfficeResponse,
  ScheduleResponse,
} from "@/lib/types";
import { AnimatePresence, motion } from "framer-motion";
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

function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
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

  // 1. Standalone agents in a grid row at the top
  if (standaloneIds.length > 0) {
    const cols = Math.min(standaloneIds.length, 6);
    for (let i = 0; i < standaloneIds.length; i++) {
      const node = nodes.find((n) => n.id === standaloneIds[i])!;
      const col = i % cols;
      const row = Math.floor(i / cols);
      allNodes.push({
        ...node,
        position: {
          x: col * (NODE_WIDTH + 30),
          y: currentY + row * (NODE_HEIGHT + 30),
        },
      });
    }
    const standaloneRows = Math.ceil(standaloneIds.length / 6);
    currentY += standaloneRows * (NODE_HEIGHT + 30) + GROUP_GAP;
  }

  // 2. Connected groups (offices) arranged in a row below standalone agents
  let offsetX = 0;
  for (const group of groups) {
    const groupNodes = nodes.filter((n) => group.nodeIds.has(n.id));
    const groupEdges = edges.filter((e) => group.nodeIds.has(e.source));
    const { nodes: laid, width } = layoutSingleGroup(groupNodes, groupEdges);
    for (const n of laid) {
      allNodes.push({ ...n, position: { x: n.position.x + offsetX, y: n.position.y + currentY } });
    }
    offsetX += width + GROUP_GAP;
  }

  return allNodes;
}

/* ------------------------------------------------------------------ */
/*  Build nodes + edges from API data                                 */
/* ------------------------------------------------------------------ */

function buildGraph(
  agents: AgentResponse[],
  offices: OfficeResponse[],
  schedules: ScheduleResponse[]
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

  // Apply layout
  const laidOutNodes = applyDagreLayout(nodes, edges);
  return { nodes: laidOutNodes, edges };
}

/* ------------------------------------------------------------------ */
/*  Inner component (uses React Flow hooks)                           */
/* ------------------------------------------------------------------ */

function TopologyGraphInner({
  initialNodes,
  initialEdges,
}: {
  initialNodes: Node[];
  initialEdges: Edge[];
}) {
  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, , onEdgesChange] = useEdgesState(initialEdges);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      nodeTypes={nodeTypes}
      fitView
      proOptions={{ hideAttribution: true }}
      minZoom={0.2}
      maxZoom={2}
    >
      <Background color="var(--color-border)" gap={24} size={1} />
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [nsFilter, setNsFilter] = useState<string>(() => {
    if (typeof window !== "undefined") return new URLSearchParams(window.location.search).get("ns") || "all";
    return "all";
  });
  const [officeFilter, setOfficeFilter] = useState<string>(() => {
    if (typeof window !== "undefined") return new URLSearchParams(window.location.search).get("office") || "all";
    return "all";
  });

  // Sync filters to URL
  useEffect(() => {
    const params = new URLSearchParams();
    if (nsFilter !== "all") params.set("ns", nsFilter);
    if (officeFilter !== "all") params.set("office", officeFilter);
    const qs = params.toString();
    const url = qs ? `${window.location.pathname}?${qs}` : window.location.pathname;
    window.history.replaceState(null, "", url);
  }, [nsFilter, officeFilter]);

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, officesRes, schedulesRes] = await Promise.all([
        listAgents(),
        listOffices(),
        listSchedules(),
      ]);
      // List endpoint doesn't include members — fetch each office individually.
      const officeList = officesRes.offices || [];
      const detailedOffices = await Promise.all(
        officeList.map((o) => getOffice(o.name, o.namespace).catch(() => o))
      );
      setAllAgents(agentsRes.agents || []);
      setAllOffices(detailedOffices);
      setAllSchedules(schedulesRes.schedules || []);
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

    if (nsFilter !== "all") {
      agents = agents.filter((a) => a.namespace === nsFilter);
      offices = offices.filter((o) => o.namespace === nsFilter);
      schedules = schedules.filter((s) => s.namespace === nsFilter);
    }

    if (officeFilter !== "all") {
      const office = offices.find((o) => o.name === officeFilter);
      if (office) {
        const memberNames = new Set([office.manager, ...(office.members || []).map((m) => m.name)]);
        agents = agents.filter((a) => memberNames.has(a.name));
        offices = [office];
        schedules = [];
      }
    }

    if (agents.length === 0 && offices.length === 0 && schedules.length === 0) {
      return null;
    }

    return buildGraph(agents, offices, schedules);
  }, [allAgents, allOffices, allSchedules, nsFilter, officeFilter]);

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
          <Select value={officeFilter} onValueChange={setOfficeFilter}>
            <SelectTrigger className="w-56 !text-[11px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              {officeNames.map((name) => (
                <SelectItem key={name} value={name}>{name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
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
              key={`${nsFilter}-${officeFilter}`}
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
                />
              </ReactFlowProvider>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
