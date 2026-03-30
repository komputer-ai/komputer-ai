"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  ResponsiveContainer,
  Tooltip,
} from "recharts";
import type { AgentResponse } from "@/lib/types";

interface TopSpendersChartProps {
  agents: AgentResponse[];
}

export function TopSpendersChart({ agents }: TopSpendersChartProps) {
  const data = agents
    .map((a) => ({
      name: a.name,
      cost: parseFloat(a.totalCostUSD || "0"),
    }))
    .filter((d) => d.cost > 0)
    .sort((a, b) => b.cost - a.cost)
    .slice(0, 5);

  if (data.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
        No agents with cost data yet.
      </p>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={data.length * 48 + 24}>
      <BarChart
        data={data}
        layout="vertical"
        margin={{ top: 4, right: 24, bottom: 4, left: 0 }}
        barCategoryGap="20%"
      >
        <XAxis
          type="number"
          stroke="#8899A6"
          tick={{ fill: "#8899A6", fontSize: 12 }}
          tickFormatter={(v: number) => `$${v.toFixed(2)}`}
          axisLine={false}
          tickLine={false}
        />
        <YAxis
          type="category"
          dataKey="name"
          stroke="#8899A6"
          tick={{ fill: "#8899A6", fontSize: 12 }}
          width={120}
          axisLine={false}
          tickLine={false}
        />
        <Tooltip
          cursor={{ fill: "rgba(43,181,178,0.08)" }}
          contentStyle={{
            backgroundColor: "#243040",
            border: "1px solid #2D3F50",
            borderRadius: 6,
            color: "#F0F4F8",
            fontSize: 12,
          }}
          formatter={(value) => [`$${Number(value).toFixed(4)}`, "Cost"]}
        />
        <Bar dataKey="cost" fill="#3f85d9" radius={[0, 4, 4, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
