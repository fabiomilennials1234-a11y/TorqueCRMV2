import { Plus, Pause, Play, MoreHorizontal, TrendingUp, Users, Target, Clock } from "lucide-react";
import { Button } from "@/ui/button";
import { Badge } from "@/ui/badge";
import { Card, CardHeader, CardTitle, CardBody } from "@/ui/card";
import { Sparkline } from "@/ui/spark";
import { PageHeader } from "@/ui/page-header";
import { EmptyState } from "@/ui/empty-state";
import { campaigns } from "@/lib/seed";
import { formatRelative } from "@/lib/utils";

export function CampaignsPage() {
  return (
    <div className="mx-auto max-w-[1400px] px-8">
      <PageHeader
        eyebrow="Campanhas · Outbound"
        title="Cadências em execução"
        description="Fluxos conversacionais ativos que aquecem leads frios até o handoff humano."
        actions={
          <Button variant="primary" size="md" className="gap-1.5">
            <Plus className="h-4 w-4" />
            Nova campanha
          </Button>
        }
      />

      {/* Summary */}
      <div className="mt-8 grid grid-cols-1 gap-px bg-hairline rounded-lg overflow-hidden shadow-elev-1 md:grid-cols-4">
        <Metric icon={Users} label="Em campanhas" value="3.240" hint="leads enrolados" />
        <Metric icon={Target} label="Taxa de resposta" value="16.9%" hint="+3.1pp vs mês" tone="up" />
        <Metric icon={TrendingUp} label="Conversão para qualificado" value="5.2%" hint="+0.8pp" tone="up" />
        <Metric icon={Clock} label="Mensagens agendadas" value="842" hint="nas próximas 24h" />
      </div>

      <div className="mt-8 space-y-4">
        {campaigns.map((c) => (
          <CampaignRow key={c.id} c={c} />
        ))}

        <button className="group flex w-full items-center justify-center gap-2 rounded-lg shadow-[inset_0_0_0_1px_hsl(var(--hairline))] py-6 text-sm text-ink-dim hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim))] hover:text-ink-muted transition-shadow">
          <Plus className="h-4 w-4" />
          Nova campanha · partir de template ICP industrial
        </button>
      </div>
    </div>
  );
}

function Metric({
  icon: Icon,
  label,
  value,
  hint,
  tone,
}: {
  icon: React.ComponentType<{ className?: string; strokeWidth?: number }>;
  label: string;
  value: string;
  hint: string;
  tone?: "up" | "down";
}) {
  return (
    <div className="bg-surface p-5">
      <div className="flex items-center gap-2 text-2xs uppercase tracking-[0.14em] text-ink-dim">
        <Icon className="h-3 w-3" strokeWidth={1.75} />
        {label}
      </div>
      <div className="mt-3 font-display text-[1.75rem] leading-none tracking-tightest text-ink tabular-nums">
        {value}
      </div>
      <div
        className={[
          "mt-1 text-xs font-metric",
          tone === "up" ? "text-success" : "text-ink-dim",
        ].join(" ")}
      >
        {hint}
      </div>
    </div>
  );
}

function CampaignRow({ c }: { c: (typeof campaigns)[number] }) {
  const progress = c.planned > 0 ? (c.sent / c.planned) * 100 : 0;
  const responded = c.sent > 0 ? (c.replied / c.sent) * 100 : 0;

  return (
    <Card className="transition-all hover:shadow-elev-2">
      <div className="flex items-start gap-6 p-5">
        {/* Status orb */}
        <div className="relative shrink-0 mt-1">
          <div
            className={[
              "h-2.5 w-2.5 rounded-full",
              c.status === "running" && "bg-success",
              c.status === "paused" && "bg-warning",
              c.status === "draft" && "bg-ink-dim",
            ].filter(Boolean).join(" ")}
          />
          {c.status === "running" && (
            <span className="absolute -inset-1 rounded-full bg-success/20 animate-ping" />
          )}
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-3">
            <h3 className="font-display text-[1.125rem] leading-tight tracking-tightest text-ink">
              {c.name}
            </h3>
            <Badge
              tone={
                c.status === "running" ? "success" : c.status === "paused" ? "warning" : "neutral"
              }
            >
              {c.status === "running" ? "executando" : c.status === "paused" ? "pausado" : "rascunho"}
            </Badge>
          </div>
          <div className="mt-1 flex items-center gap-3 text-xs text-ink-dim">
            {c.startedAt ? (
              <span className="font-metric">
                iniciada {formatRelative(c.startedAt)}
              </span>
            ) : (
              <span>nunca executada</span>
            )}
            <span>·</span>
            <span>Canal: WhatsApp · Cadência 3-5-8</span>
          </div>

          {/* Progress track */}
          {c.status !== "draft" && (
            <div className="mt-4">
              <div className="flex items-baseline justify-between text-2xs uppercase tracking-[0.12em] text-ink-dim">
                <span>Progresso de envio</span>
                <span className="font-metric text-ink-muted">
                  {c.sent.toLocaleString("pt-BR")} / {c.planned.toLocaleString("pt-BR")}
                </span>
              </div>
              <div className="mt-1.5 h-1 overflow-hidden rounded-full bg-hairline">
                <div
                  className="h-full rounded-full bg-accent transition-all"
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>
          )}
        </div>

        {/* Stats */}
        <div className="flex items-center gap-8 shrink-0">
          <Stat label="Resposta" value={`${responded.toFixed(1)}%`} />
          <Stat label="Convertidos" value={c.converted.toString()} />
          <Sparkline data={[3, 5, 8, 6, 9, 12, 14, 11, 16, 15, 18, 20]} width={80} height={28} />
          <div className="flex items-center gap-0.5">
            <Button variant="ghost" size="icon">
              {c.status === "running" ? (
                <Pause className="h-4 w-4" />
              ) : (
                <Play className="h-4 w-4" />
              )}
            </Button>
            <Button variant="ghost" size="icon">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </Card>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-right">
      <div className="text-2xs uppercase tracking-[0.12em] text-ink-dim">
        {label}
      </div>
      <div className="font-metric text-sm text-ink tabular-nums">{value}</div>
    </div>
  );
}
