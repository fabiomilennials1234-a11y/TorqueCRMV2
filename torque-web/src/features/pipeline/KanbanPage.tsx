import { useMemo, useState } from "react";
import { Filter, Plus, Settings2, LayoutGrid, List } from "lucide-react";
import { Button } from "@/ui/button";
import { Pill } from "@/ui/pill";
import { Kbd } from "@/ui/kbd";
import { Sheet, SheetContent } from "@/ui/sheet";
import { stageMeta, leads as seedLeads, type Lead, type Stage } from "@/lib/seed";
import { KanbanColumn } from "./KanbanColumn";
import { LeadDrawer } from "./LeadDrawer";
import { TorqueMark } from "@/shell/TorqueMark";

const stageOrder: Stage[] = [
  "novo",
  "abordado",
  "qualificado",
  "agendado",
  "proposta",
  "vendido",
  "perdido",
];

export function KanbanPage() {
  const [openLead, setOpenLead] = useState<Lead | null>(null);
  const [filter, setFilter] = useState<string | null>(null);

  const grouped = useMemo(() => {
    const filtered = filter
      ? seedLeads.filter(
          (l) =>
            l.tags.includes(filter) ||
            l.owner.name === filter ||
            l.source === filter,
        )
      : seedLeads;

    return stageOrder.reduce<Record<Stage, Lead[]>>(
      (acc, s) => {
        acc[s] = filtered.filter((l) => l.stage === s);
        return acc;
      },
      {
        novo: [], abordado: [], qualificado: [], agendado: [],
        proposta: [], vendido: [], perdido: [],
      } as Record<Stage, Lead[]>,
    );
  }, [filter]);

  const totals = useMemo(() => {
    return {
      count: seedLeads.length,
      value: seedLeads
        .filter((l) => l.stage !== "perdido")
        .reduce((s, l) => s + l.value, 0),
      won: seedLeads.filter((l) => l.stage === "vendido").length,
    };
  }, []);

  return (
    <div className="flex h-[calc(100vh-56px)] flex-col">
      {/* Toolbar */}
      <div className="shrink-0 px-8 pt-6 pb-4 shadow-hairline-b">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <div className="mb-1 flex items-center gap-2.5 text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
              <TorqueMark size={14} />
              Pipelines
              <span className="text-ink-dim">/</span>
              <span className="text-accent">WhatsApp · Qualificação</span>
            </div>
            <div className="flex items-baseline gap-4">
              <h1 className="font-display text-[1.75rem] leading-none tracking-tightest text-ink">
                WhatsApp · Qualificação
              </h1>
              <div className="font-metric text-xs text-ink-dim tabular-nums">
                {totals.count} cards · R${" "}
                {totals.value.toLocaleString("pt-BR", { maximumFractionDigits: 0 })} em
                pipeline · {totals.won} vendidos
              </div>
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            <Button variant="ghost" size="icon">
              <LayoutGrid className="h-4 w-4" strokeWidth={1.75} />
            </Button>
            <Button variant="ghost" size="icon">
              <List className="h-4 w-4" strokeWidth={1.75} />
            </Button>
            <div className="mx-2 h-5 w-px bg-hairline" />
            <Button variant="secondary" size="sm" className="gap-1.5">
              <Filter className="h-3.5 w-3.5" />
              Filtros
              <span className="ml-1 flex h-4 min-w-4 items-center justify-center rounded-xs bg-accent px-1 text-[0.625rem] font-metric text-bg">
                2
              </span>
            </Button>
            <Button variant="secondary" size="sm" className="gap-1.5">
              <Settings2 className="h-3.5 w-3.5" />
              Automações
            </Button>
            <Button variant="primary" size="sm" className="gap-1.5">
              <Plus className="h-3.5 w-3.5" />
              Novo lead <Kbd className="bg-bg/30 text-bg shadow-none">N</Kbd>
            </Button>
          </div>
        </div>

        {/* Pill filter bar */}
        <div className="mt-4 flex items-center gap-1.5 overflow-x-auto pb-1">
          <Pill active={filter === null} onClick={() => setFilter(null)}>
            Todos · {seedLeads.length}
          </Pill>
          <div className="mx-1 h-4 w-px bg-hairline" />
          <span className="text-2xs uppercase tracking-[0.12em] text-ink-dim pr-1">Tag</span>
          {["Hot", "Decisor", "BANT ok", "ICP fit", "Enterprise"].map((t) => (
            <Pill
              key={t}
              active={filter === t}
              onClick={() => setFilter(filter === t ? null : t)}
            >
              {t}
            </Pill>
          ))}
          <div className="mx-1 h-4 w-px bg-hairline" />
          <span className="text-2xs uppercase tracking-[0.12em] text-ink-dim pr-1">
            Responsável
          </span>
          {["Maíra Duarte", "Rafael Bento"].map((t) => (
            <Pill
              key={t}
              active={filter === t}
              onClick={() => setFilter(filter === t ? null : t)}
            >
              {t}
            </Pill>
          ))}
        </div>
      </div>

      {/* Board */}
      <div className="relative flex-1 min-h-0 overflow-x-auto overflow-y-hidden">
        <div className="flex h-full min-w-max gap-4 px-8 pt-5 pb-8">
          {stageOrder.map((s) => (
            <KanbanColumn
              key={s}
              stage={s}
              meta={stageMeta[s]}
              leads={grouped[s]}
              onCardClick={setOpenLead}
            />
          ))}

          {/* Add column card */}
          <button className="flex h-full w-[72px] shrink-0 flex-col items-center justify-center rounded-xl shadow-clay-1 bg-surface/30 text-ink-dim hover:shadow-clay-2 hover:bg-surface/50 hover:text-ink-muted transition-all duration-200 ease-[var(--ease-out-soft)]">
            <Plus className="h-4 w-4 mb-2" />
            <span className="text-2xs uppercase tracking-[0.14em] [writing-mode:vertical-rl]">
              Novo estágio
            </span>
          </button>
        </div>
      </div>

      <Sheet open={!!openLead} onOpenChange={(o) => !o && setOpenLead(null)}>
        {openLead && (
          <SheetContent>
            <LeadDrawer lead={openLead} />
          </SheetContent>
        )}
      </Sheet>
    </div>
  );
}
