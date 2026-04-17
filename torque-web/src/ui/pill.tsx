import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

/**
 * Pill — compact round-shouldered tag. Hairline outlined, dimmed by default.
 * Different from Badge (uppercase, severe). Pill is for free-form tags/filters.
 */
export function Pill({
  className,
  active,
  ...props
}: HTMLAttributes<HTMLButtonElement> & { active?: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        "inline-flex items-center gap-1.5 h-6 rounded-full px-2.5 text-[0.75rem] font-medium",
        "shadow-hairline transition-colors whitespace-nowrap",
        active
          ? "bg-accent/15 text-accent shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.4)]"
          : "text-ink-muted hover:text-ink hover:bg-elevated/60",
        className,
      )}
      {...props}
    />
  );
}
