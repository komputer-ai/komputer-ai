"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ReactFlow,
  Background,
  useNodesState,
  useEdgesState,
  useReactFlow,
  ReactFlowProvider,
  type Node,
  type Edge,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "@dagrejs/dagre";
import { ExternalLink, Network } from "lucide-react";
import Link from "next/link";
import { motion } from "framer-motion";

import { listAgents, listOffices, getOffice, listSchedules } from "@/lib/api";
import type { AgentResponse, OfficeResponse, ScheduleResponse } from "@/lib/types";
import { AgentNode, OfficeNode, ScheduleNode } from "@/components/topology/node-types";

const nodeTypes = { agent: AgentNode, office: OfficeNode, schedule: ScheduleNode } as const;

const NODE_WIDTH = 200;
const NODE_HEIGHT = 80;

function layoutNodes(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", ranksep: 80, nodesep: 50 });

  const nodeWidths = new Map<string, number>();
  for (const node of nodes) {
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

  // Find leaf nodes (no outgoing edges) to rearrange into a 2-col grid
  const sources = new Set(edges.map((e) => e.source));
  const leafIds = new Set(nodes.filter((n) => !sources.has(n.id)).map((n) => n.id));
  const MAX_COLS = 2;

  const laid = nodes.map((node) => {
    const pos = g.node(node.id);
    const w = nodeWidths.get(node.id) || NODE_WIDTH;
    return { ...node, position: { x: pos.x - w / 2, y: pos.y - NODE_HEIGHT / 2 } };
  });

  if (leafIds.size > 1) {
    const leafNodes = laid.filter((n) => leafIds.has(n.id));
    const nonLeafNodes = laid.filter((n) => !leafIds.has(n.id));

    const parentId = edges.find((e) => leafIds.has(e.target))?.source;
    const parent = parentId ? nonLeafNodes.find((n) => n.id === parentId) : null;
    const parentCenterX = parent
      ? parent.position.x + (nodeWidths.get(parent.id) || NODE_WIDTH) / 2
      : 0;

    const leafY = Math.min(...leafNodes.map((n) => n.position.y));
    const cols = Math.min(MAX_COLS, leafNodes.length);
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

  return laid;
}

function buildAgentGraph(
  agentName: string,
  agents: AgentResponse[],
  offices: OfficeResponse[],
  schedules: ScheduleResponse[]
) {
  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const addedAgents = new Set<string>();

  // Find offices this agent belongs to (as manager or member)
  const relatedOffices = offices.filter(
    (o) => o.manager === agentName || o.members?.some((m) => m.name === agentName)
  );

  // Find schedules targeting this agent
  const relatedSchedules = schedules.filter((s) => s.agentName === agentName);

  // If agent has no relations, return empty
  if (relatedOffices.length === 0 && relatedSchedules.length === 0) {
    return { nodes: [], edges: [], officeNames: [] };
  }

  const agentMap = new Map(agents.map((a) => [a.name, a]));

  function addAgentNode(name: string) {
    if (addedAgents.has(name)) return;
    addedAgents.add(name);
    const a = agentMap.get(name);
    nodes.push({
      id: `agent-${name}`,
      type: "agent",
      position: { x: 0, y: 0 },
      data: {
        label: name,
        status: a?.status ?? "Deleted",
        model: a?.model ?? "",
        namespace: a?.namespace,
        taskStatus: a?.taskStatus,
        lifecycle: a?.lifecycle,
        totalCostUSD: a?.totalCostUSD,
      },
    });
  }

  for (const office of relatedOffices) {
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

    // Manager
    if (office.manager) {
      addAgentNode(office.manager);
      edges.push({
        id: `e-office-${office.name}-manager-${office.manager}`,
        source: `office-${office.name}`,
        target: `agent-${office.manager}`,
        style: { stroke: "#3f85d9", strokeWidth: 2 },
      });
    }

    // Workers
    for (const member of office.members || []) {
      if (member.role === "worker") {
        addAgentNode(member.name);
        edges.push({
          id: `e-manager-${office.manager}-worker-${member.name}`,
          source: `agent-${office.manager}`,
          target: `agent-${member.name}`,
          style: { stroke: "#7c6bc4", strokeWidth: 1.5 },
        });
      }
    }
  }

  // Schedules
  for (const schedule of relatedSchedules) {
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
    addAgentNode(agentName);
    edges.push({
      id: `e-schedule-${schedule.name}-agent-${agentName}`,
      source: `schedule-${schedule.name}`,
      target: `agent-${agentName}`,
      style: { stroke: "#9775d6", strokeWidth: 1.5, strokeDasharray: "6 3" },
    });
  }

  // Compute incoming/outgoing + widths
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

  const laidOut = layoutNodes(nodes, edges);
  const officeNames = relatedOffices.map((o) => o.name);
  return { nodes: laidOut, edges, officeNames };
}

function AgentTopologyInner({ nodes: initialNodes, edges: initialEdges, focusNodeId }: { nodes: Node[]; edges: Edge[]; focusNodeId: string }) {
  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, , onEdgesChange] = useEdgesState(initialEdges);
  const { setCenter } = useReactFlow();
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (initialized) return;
    const focusNode = initialNodes.find((n) => n.id === focusNodeId);
    if (!focusNode) return;
    const w = (focusNode.data as { nodeWidth?: number })?.nodeWidth || NODE_WIDTH;
    // Small delay to let ReactFlow measure the viewport
    const timer = setTimeout(() => {
      setCenter(focusNode.position.x + w / 2, focusNode.position.y + NODE_HEIGHT / 2, { zoom: 1.1, duration: 300 });
      setInitialized(true);
    }, 50);
    return () => clearTimeout(timer);
  }, [initialized, initialNodes, focusNodeId, setCenter]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      nodeTypes={nodeTypes}
      proOptions={{ hideAttribution: true }}
      minZoom={0.3}
      maxZoom={1.5}
      panOnDrag
      zoomOnScroll={false}
    >
      <Background color="var(--color-border)" gap={24} size={1} />
    </ReactFlow>
  );
}

export function AgentTopology({ agentName, agentNs }: { agentName: string; agentNs?: string }) {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [offices, setOffices] = useState<OfficeResponse[]>([]);
  const [schedules, setSchedules] = useState<ScheduleResponse[]>([]);
  const [loaded, setLoaded] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, officesRes, schedulesRes] = await Promise.all([
        listAgents(),
        listOffices(),
        listSchedules(),
      ]);
      const officeList = officesRes.offices || [];
      const detailed = await Promise.all(
        officeList.map((o) => getOffice(o.name, o.namespace).catch(() => o))
      );
      setAgents(agentsRes.agents || []);
      setOffices(detailed);
      setSchedules(schedulesRes.schedules || []);
    } catch {
      // silent
    } finally {
      setLoaded(true);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const { nodes, edges, officeNames } = useMemo(
    () => loaded ? buildAgentGraph(agentName, agents, offices, schedules) : { nodes: [], edges: [], officeNames: [] },
    [loaded, agentName, agents, offices, schedules]
  );

  const topologyHref = useMemo(() => {
    const params = new URLSearchParams();
    if (officeNames.length > 0) params.set("office", officeNames.join(","));
    const qs = params.toString();
    return qs ? `/topology?${qs}` : "/topology";
  }, [officeNames]);

  if (!loaded || nodes.length === 0) return null;

  return (
    <motion.div
      className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden transition-colors duration-150 hover:border-[var(--color-border-hover)]"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: "easeOut", delay: 0.15 }}
    >
      <div className="px-5 pt-4 pb-2 flex items-center gap-2">
        <Network className="h-3.5 w-3.5 text-[var(--color-text-muted)]" />
        <h3 className="text-[11px] uppercase tracking-wider font-semibold text-[var(--color-text-muted)]">Topology</h3>
        <Link
          href={topologyHref}
          className="ml-auto flex items-center gap-1 text-[11px] text-[var(--color-text-secondary)] hover:text-[var(--color-brand-blue)] transition-colors"
        >
          View in Topology
          <ExternalLink className="h-3 w-3" />
        </Link>
      </div>
      <div className="h-[320px]">
        <ReactFlowProvider>
          <AgentTopologyInner nodes={nodes} edges={edges} focusNodeId={`agent-${agentName}`} />
        </ReactFlowProvider>
      </div>
    </motion.div>
  );
}
