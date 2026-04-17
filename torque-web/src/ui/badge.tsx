import { cva, type VariantProps } from "class-variance-authority";
import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

const badge = cva(
  "inline-flex items-center gap-1.5 rounded-xs px-1.5 py-0.5 text-2xs font-medium uppercase tracking-[0.08em] whitespace-nowrap",
  {
    variants: {
      tone: {
        neutral: "bg-elevated text-ink-muted shadow-hairline",
        accent: "bg-accent/10 text-accent shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.3)]",
        success: "bg-success/10 text-success shadow-[inset_0_0_0_1px_hsl(var(--success)/0.3)]",
        warning: "bg-warning/10 text-warning shadow-[inset_0_0_0_1px_hsl(var(--warning)/0.3)]",
        danger: "bg-danger/10 text-danger shadow-[inset_0_0_0_1px_hsl(var(--danger)/0.3)]",
        info: "bg-info/10 text-info shadow-[inset_0_0_0_1px_hsl(var(--info)/0.3)]",
        solid: "bg-ink text-bg",
      },
    },
    defaultVariants: { tone: "neutral" },
  },
);

export interface BadgeProps
  extends HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badge> {}

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badge({ tone }), className)} {...props} />;
}
