"use client";

import { createContext, useContext } from "react";

export interface AgentTemplate {
  name: string;
  instructions: string;
  model: string;
  lifecycle: string;
  role?: "manager" | "worker";
  secrets?: Record<string, string>;
}

export interface CreateAgentModalContextValue {
  openWithTemplate: (template: AgentTemplate) => void;
}

export const CreateAgentModalContext =
  createContext<CreateAgentModalContextValue | null>(null);

export function useCreateAgentModal() {
  const ctx = useContext(CreateAgentModalContext);
  if (!ctx) {
    throw new Error(
      "useCreateAgentModal must be used within CreateAgentModalContext.Provider"
    );
  }
  return ctx;
}
