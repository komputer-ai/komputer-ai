import { CircleHelp } from "lucide-react";
import { Button } from "@/components/kit/button";

type EmptyStateProps = {
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
};

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <CircleHelp className="mb-4 size-10 text-[var(--color-text-secondary)]" />
      <h3 className="mb-1 text-base font-medium text-[var(--color-text)]">
        {title}
      </h3>
      <p className="mb-4 max-w-sm text-sm text-[var(--color-text-secondary)]">
        {description}
      </p>
      {action && (
        <Button variant="secondary" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  );
}
