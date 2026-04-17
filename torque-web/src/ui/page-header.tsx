import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function PageHeader({
  eyebrow,
  title,
  description,
  actions,
  className,
}: {
  eyebrow?: string;
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  className?: string;
}) {
  return (
    <header
      className={cn(
        "flex items-start justify-between gap-6 pt-8 pb-6",
        "shadow-hairline-b",
        className,
      )}
    >
      <div className="min-w-0">
        {eyebrow && (
          <div className="text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim mb-2">
            {eyebrow}
          </div>
        )}
        <h1 className="font-display text-[2rem] leading-[1.05] tracking-tightest text-ink">
          {title}
        </h1>
        {description && (
          <p className="mt-2 max-w-2xl text-[0.9375rem] leading-relaxed text-ink-muted">
            {description}
          </p>
        )}
      </div>
      {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
    </header>
  );
}
