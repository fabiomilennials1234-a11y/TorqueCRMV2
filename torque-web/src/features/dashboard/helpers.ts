import { kpis, activity, leads } from "@/lib/seed";
import { stageMeta, type Stage } from "@/lib/seed";

export { kpis, activity, leads };

export function formatShort(stage: Stage) {
  return stageMeta[stage].label;
}
