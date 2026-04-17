import { cva } from "class-variance-authority";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

const stepCircle = cva(
  "relative z-10 flex shrink-0 items-center justify-center rounded-full transition-[background-color,box-shadow,color] duration-[280ms] ease-[var(--ease-in-out-precise)]",
  {
    variants: {
      variant: {
        numbered: "h-8 w-8 text-xs font-medium",
        iconed: "h-8 w-8",
        minimal: "h-2.5 w-2.5",
      },
      state: {
        completed: "bg-success/15 text-success shadow-[inset_0_0_0_1px_hsl(var(--success)/0.4)]",
        current: "bg-accent/15 text-accent shadow-glow",
        future: "bg-elevated text-ink-dim shadow-hairline",
      },
    },
    compoundVariants: [
      {
        variant: "minimal",
        state: "completed",
        class: "bg-success shadow-none",
      },
      {
        variant: "minimal",
        state: "current",
        class: "bg-accent shadow-glow",
      },
      {
        variant: "minimal",
        state: "future",
        class: "bg-ink-dim/30 shadow-none",
      },
    ],
    defaultVariants: { variant: "numbered", state: "future" },
  },
);

export interface StepProgressProps {
  steps: string[];
  currentStep: number;
  completedSteps?: number[];
  variant?: "numbered" | "iconed" | "minimal";
  className?: string;
}

export function StepProgress({
  steps,
  currentStep,
  completedSteps = [],
  variant = "numbered",
  className,
}: StepProgressProps) {
  const isMinimal = variant === "minimal";

  function resolveState(index: number): "completed" | "current" | "future" {
    if (completedSteps.includes(index)) return "completed";
    if (index === currentStep) return "current";
    return "future";
  }

  return (
    <nav
      aria-label="Progresso"
      className={cn("flex w-full items-start", isMinimal && "items-center", className)}
    >
      {steps.map((label, i) => {
        const state = resolveState(i);
        const isLast = i === steps.length - 1;

        return (
          <div
            key={i}
            className={cn(
              "flex flex-1 items-center",
              isMinimal ? "items-center" : "flex-col items-center",
              isLast && "flex-none",
            )}
          >
            {/* Step circle + connector row */}
            <div className={cn("flex w-full items-center", isLast && "w-auto")}>
              <div
                role="listitem"
                aria-label={label}
                aria-current={state === "current" ? "step" : undefined}
                className={cn(stepCircle({ variant, state }))}
              >
                {!isMinimal && state === "completed" && (
                  <Check className="h-4 w-4" strokeWidth={1.75} />
                )}
                {!isMinimal && state !== "completed" && variant === "numbered" && (
                  <span className="font-metric tabular-nums">{i + 1}</span>
                )}
                {!isMinimal && state !== "completed" && variant === "iconed" && (
                  <span className="font-metric tabular-nums text-xs">{i + 1}</span>
                )}
              </div>

              {/* Connector line */}
              {!isLast && (
                <div className="relative mx-1 flex-1 h-px">

                  {/* Track */}
                  <div className="absolute inset-0 bg-hairline" />
                  {/* Fill */}
                  <div
                    className={cn(
                      "absolute inset-y-0 left-0 bg-success transition-[width] duration-[280ms] ease-[var(--ease-in-out-precise)]",
                    )}
                    style={{
                      width:
                        completedSteps.includes(i) ? "100%" : "0%",
                    }}
                  />
                </div>
              )}
            </div>

            {/* Label */}
            {!isMinimal && (
              <span
                className={cn(
                  "mt-2 text-2xs font-medium transition-colors duration-[280ms] ease-[var(--ease-in-out-precise)]",
                  state === "completed" && "text-success",
                  state === "current" && "text-ink",
                  state === "future" && "text-ink-dim",
                )}
              >
                {label}
              </span>
            )}
          </div>
        );
      })}
    </nav>
  );
}
