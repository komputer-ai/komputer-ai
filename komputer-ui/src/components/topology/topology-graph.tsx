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

import { listAgents, listOffices, listSchedules } from "@/lib/api";
import type {
  AgentResponse,
  OfficeResponse,
  ScheduleResponse,
} from "@/lib/types";
import {
  AgentNode,
  OfficeNode,
  ScheduleNode,
} from "./node-types";

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

function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", ranksep: 80, nodesep: 50 });

  for (const node of nodes) {
    g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  return nodes.map((node) => {
    const pos = g.node(node.id);
    return {
      ...node,
      position: {
        x: pos.x - NODE_WIDTH / 2,
        y: pos.y - NODE_HEIGHT / 2,
      },
    };
  });
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
      },
    });

    // Edge: office -> manager agent
    if (office.manager && agentSet.has(office.manager)) {
      edges.push({
        id: `e-office-${office.name}-manager-${office.manager}`,
        source: `office-${office.name}`,
        target: `agent-${office.manager}`,
        style: { stroke: "#3f85d9", strokeWidth: 2 },
        animated: false,
      });
    }

    // Edges: manager -> worker agents
    for (const member of office.members || []) {
      if (member.role === "worker" && agentSet.has(member.name)) {
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

  // Agent nodes
  for (const agent of agents) {
    nodes.push({
      id: `agent-${agent.name}`,
      type: "agent",
      position: { x: 0, y: 0 },
      data: {
        label: agent.name,
        status: agent.status,
        model: agent.model,
      },
    });
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
  const [graphData, setGraphData] = useState<{
    nodes: Node[];
    edges: Edge[];
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, officesRes, schedulesRes] = await Promise.all([
        listAgents(),
        listOffices(),
        listSchedules(),
      ]);

      const { nodes, edges } = buildGraph(
        agentsRes.agents || [],
        officesRes.offices || [],
        schedulesRes.schedules || []
      );

      setGraphData({ nodes, edges });
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

  if (!graphData || graphData.nodes.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-sm text-[var(--color-text-secondary)]">
          No agents, offices, or schedules found.
        </p>
      </div>
    );
  }

  return (
    <ReactFlowProvider>
      <TopologyGraphInner
        initialNodes={graphData.nodes}
        initialEdges={graphData.edges}
      />
    </ReactFlowProvider>
  );
}
