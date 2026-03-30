import { forwardRef } from "react";
import { cn } from "@/lib/utils";

type ButtonVariant = "primary" | "secondary" | "ghost" | "destructive" | "magic";
type ButtonSize = "sm" | "md" | "lg" | "icon";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
}

const variantStyles: Record<ButtonVariant, string> = {
  primary: [
    "bg-[var(--color-brand-blue)]/80 backdrop-blur-md text-white",
    "border border-white/15",
    "shadow-[0_0_0_1px_rgba(63,133,217,0.3),0_2px_8px_rgba(0,0,0,0.2)]",
    "hover:bg-[var(--color-brand-blue)]/95",
    "hover:border-white/25",
    "hover:shadow-[0_0_0_1px_rgba(63,133,217,0.5),0_0_16px_rgba(63,133,217,0.2),0_4px_12px_rgba(0,0,0,0.2)]",
    "active:scale-[0.97] active:bg-[var(--color-brand-blue)]",
  ].join(" "),
  secondary: [
    "bg-white/[0.06] backdrop-blur-md text-[var(--color-text-secondary)]",
    "border border-white/10",
    "shadow-[0_1px_3px_rgba(0,0,0,0.15)]",
    "hover:bg-white/[0.10] hover:text-[var(--color-text)]",
    "hover:border-white/15",
    "hover:shadow-[0_0_0_1px_rgba(255,255,255,0.08),0_2px_8px_rgba(0,0,0,0.15)]",
    "active:scale-[0.97] active:bg-white/[0.12]",
  ].join(" "),
  ghost: [
    "text-[var(--color-text-secondary)] bg-transparent",
    "hover:text-[var(--color-text)] hover:bg-white/[0.06]",
    "active:bg-white/[0.08] active:scale-[0.97]",
  ].join(" "),
  destructive: [
    "bg-red-500/10 backdrop-blur-md text-red-400",
    "border border-red-400/15",
    "shadow-[0_1px_3px_rgba(0,0,0,0.15)]",
    "hover:bg-red-500/20 hover:border-red-400/25",
    "hover:shadow-[0_0_0_1px_rgba(248,113,113,0.2),0_2px_8px_rgba(0,0,0,0.15)]",
    "active:scale-[0.97]",
  ].join(" "),
  magic: [
    "bg-gradient-to-r from-[var(--color-brand-blue)]/80 to-[var(--color-brand-violet)]/80 backdrop-blur-md text-white",
    "border border-white/15",
    "shadow-[0_0_0_1px_rgba(139,92,246,0.3),0_2px_8px_rgba(0,0,0,0.2)]",
    "hover:from-[var(--color-brand-blue)]/95 hover:to-[var(--color-brand-violet)]/95",
    "hover:border-white/25",
    "hover:shadow-[0_0_0_1px_rgba(139,92,246,0.5),0_0_16px_rgba(139,92,246,0.15),0_0_16px_rgba(63,133,217,0.1),0_4px_12px_rgba(0,0,0,0.2)]",
    "active:scale-[0.97]",
  ].join(" "),
};

const sizeStyles: Record<ButtonSize, string> = {
  sm: "h-7 px-3 text-[13px] gap-1.5 rounded-full",
  md: "h-8 px-4 text-[13px] gap-2 rounded-full",
  lg: "h-9 px-5 text-sm gap-2 rounded-full",
  icon: "h-7 w-7 rounded-full justify-center p-0",
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "primary", size = "md", children, ...props }, ref) => (
    <button
      ref={ref}
      className={cn(
        "inline-flex items-center justify-center font-medium cursor-pointer select-none outline-none",
        "transition-all duration-200 ease-out",
        "focus-visible:ring-2 focus-visible:ring-[var(--color-brand-blue)]/40 focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--color-bg)]",
        "disabled:opacity-40 disabled:cursor-not-allowed disabled:pointer-events-none",
        variantStyles[variant],
        sizeStyles[size],
        className
      )}
      {...props}
    >
      {children}
    </button>
  )
);
Button.displayName = "Button";
