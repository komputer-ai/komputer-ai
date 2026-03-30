"use client";

import { createContext, useContext, useState } from "react";
import { motion } from "framer-motion";
import { cn } from "@/lib/utils";

interface TabsContextType {
  value: string;
  onChange: (value: string) => void;
}

const TabsContext = createContext<TabsContextType | null>(null);

function useTabsContext() {
  const ctx = useContext(TabsContext);
  if (!ctx) throw new Error("Tabs components must be used within <Tabs>");
  return ctx;
}

interface TabsProps {
  defaultValue: string;
  value?: string;
  onValueChange?: (value: string) => void;
  children: React.ReactNode;
  className?: string;
}

export function Tabs({ defaultValue, value: controlledValue, onValueChange, children, className }: TabsProps) {
  const [internalValue, setInternalValue] = useState(defaultValue);
  const value = controlledValue ?? internalValue;
  const onChange = onValueChange ?? setInternalValue;

  return (
    <TabsContext.Provider value={{ value, onChange }}>
      <div className={className}>{children}</div>
    </TabsContext.Provider>
  );
}

export function TabsList({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex items-center gap-1 border-b border-[var(--color-border)] pb-px",
        className
      )}
      {...props}
    />
  );
}

interface TabsTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  value: string;
}

export function TabsTrigger({ value: tabValue, className, children, ...props }: TabsTriggerProps) {
  const { value, onChange } = useTabsContext();
  const isActive = value === tabValue;

  return (
    <button
      type="button"
      className={cn(
        "relative px-3 py-2 text-sm font-medium transition-colors cursor-pointer",
        isActive
          ? "text-[var(--color-text)]"
          : "text-[var(--color-text-secondary)] hover:text-[var(--color-text)]",
        className
      )}
      onClick={() => onChange(tabValue)}
      {...props}
    >
      {children}
      {isActive && (
        <motion.div
          className="absolute bottom-0 left-0 right-0 h-[3px] bg-[var(--color-brand-blue)] rounded-full shadow-[0_1px_4px_var(--color-brand-blue-glow)]"
          layoutId="tab-indicator"
          transition={{ duration: 0.2, ease: "easeInOut" }}
        />
      )}
    </button>
  );
}

interface TabsContentProps {
  value: string;
  className?: string;
  children?: React.ReactNode;
}

export function TabsContent({ value: tabValue, className, children }: TabsContentProps) {
  const { value } = useTabsContext();
  if (value !== tabValue) return null;

  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.15 }}
    >
      {children}
    </motion.div>
  );
}
