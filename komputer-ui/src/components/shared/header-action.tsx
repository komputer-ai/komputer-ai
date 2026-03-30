"use client";

import { Plus } from "lucide-react";
import { Button } from "@/components/kit/button";

interface HeaderActionProps {
  label: string;
  onClick: () => void;
}

export function HeaderAction({ label, onClick }: HeaderActionProps) {
  return (
    <Button variant="primary" size="sm" onClick={onClick}>
      <Plus className="size-3.5" />
      {label}
    </Button>
  );
}
