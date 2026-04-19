import { useState } from 'react'
import {
  Play,
  Save,
  Undo2,
  Redo2,
  Grid3x3,
  Sparkles,
  Zap,
  GitBranch,
  Clock,
  Send,
  Tag,
  ArrowRight,
  MessagesSquare,
  Database,
  Webhook,
  UserPlus,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/ui/button'
import { Badge } from '@/ui/badge'
import { workflowNodes, workflowEdges } from '@/lib/seed'
import { cn } from '@/lib/utils'

type NodeKind = 'trigger' | 'condition' | 'action' | 'wait'

const nodeStyle: Record<NodeKind, { label: string; icon: LucideIcon; color: string }> = {
  trigger: { label: 'Gatilho', icon: Zap, color: 'hsl(var(--accent))' },
  condition: { label: 'Condição', icon: GitBranch, color: 'hsl(var(--info))' },
  action: { label: 'Ação', icon: Send, color: 'hsl(var(--success))' },
  wait: { label: 'Esperar', icon: Clock, color: 'hsl(var(--warning))' },
}

export function WorkflowBuilderPage() {
  const [selected, setSelected] = useState<string | null>('n2')

  return (
    <div className="flex h-[calc(100vh-56px)] min-h-0 flex-col">
      {/* Top bar */}
      <div className="flex shrink-0 items-center justify-between gap-4 bg-bg px-8 py-4 shadow-hairline-b">
        <div>
          <div className="mb-1 flex items-center gap-2 text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
            Fluxos · <span className="text-accent">v4 · Rascunho</span>
          </div>
          <div className="flex items-baseline gap-3">
            <h1 className="font-display text-[1.5rem] leading-none tracking-tightest text-ink">
              Aquecer após 72h sem resposta
            </h1>
            <Badge tone="warning">não publicado</Badge>
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          <Button variant="ghost" size="icon">
            <Undo2 className="h-4 w-4" strokeWidth={1.75} />
          </Button>
          <Button variant="ghost" size="icon">
            <Redo2 className="h-4 w-4" strokeWidth={1.75} />
          </Button>
          <div className="mx-2 h-5 w-px bg-hairline" />
          <Button variant="secondary" size="sm" className="gap-1.5">
            <Play className="h-3.5 w-3.5" />
            Testar com lead
          </Button>
          <Button variant="secondary" size="sm" className="gap-1.5">
            <Save className="h-3.5 w-3.5" />
            Salvar rascunho
          </Button>
          <Button variant="primary" size="sm">
            Publicar
          </Button>
        </div>
      </div>

      <div className="grid min-h-0 flex-1 grid-cols-[260px_1fr_320px]">
        {/* Node palette */}
        <aside className="min-h-0 overflow-y-auto bg-surface p-4 shadow-[inset_-1px_0_0_0_hsl(var(--hairline))]">
          <h3 className="mb-3 text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
            Paleta
          </h3>
          <PaletteGroup
            label="Gatilhos"
            items={[
              { label: 'Lead criado', icon: UserPlus },
              { label: 'Mensagem recebida', icon: MessagesSquare },
              { label: 'Webhook externo', icon: Webhook },
              { label: 'Stage alterado', icon: ArrowRight },
            ]}
          />
          <PaletteGroup
            label="Controle"
            items={[
              { label: 'Condição', icon: GitBranch },
              { label: 'Esperar tempo', icon: Clock },
              { label: 'Esperar resposta', icon: MessagesSquare },
              { label: 'Divisão A/B', icon: Grid3x3 },
            ]}
          />
          <PaletteGroup
            label="Ações"
            items={[
              { label: 'Enviar mensagem', icon: Send },
              { label: 'Adicionar tag', icon: Tag },
              { label: 'Atribuir responsável', icon: UserPlus },
              { label: 'Gravar no banco', icon: Database },
              { label: 'Chamar Copilot', icon: Sparkles },
            ]}
          />
        </aside>

        {/* Canvas */}
        <main className="relative min-h-0 overflow-auto bg-bg">
          <div
            className="absolute inset-0 bg-grid opacity-30"
            style={{ backgroundSize: '32px 32px' }}
          />
          <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_40%,hsl(var(--bg))_100%)]" />
          <div className="relative min-h-[600px] min-w-[1100px] p-8">
            <svg className="pointer-events-none absolute inset-0 h-full w-full">
              {workflowEdges.map((e, i) => {
                const from = workflowNodes.find((n) => n.id === e.from)!
                const to = workflowNodes.find((n) => n.id === e.to)!
                const x1 = from.x + 200 + 32
                const y1 = from.y + 36 + 32
                const x2 = to.x + 32
                const y2 = to.y + 36 + 32
                const mx = (x1 + x2) / 2
                return (
                  <g key={i}>
                    <path
                      d={`M ${x1} ${y1} C ${mx} ${y1} ${mx} ${y2} ${x2} ${y2}`}
                      fill="none"
                      stroke="hsl(var(--hairline))"
                      strokeWidth="1.5"
                    />
                    {/* arrowhead */}
                    <circle cx={x2} cy={y2} r="2.5" fill="hsl(var(--ink-dim))" />
                    {e.label && (
                      <g transform={`translate(${mx - 14},${(y1 + y2) / 2 - 8})`}>
                        <rect
                          x="0"
                          y="0"
                          width="28"
                          height="16"
                          rx="3"
                          fill="hsl(var(--elevated))"
                          stroke="hsl(var(--hairline))"
                        />
                        <text
                          x="14"
                          y="11"
                          fill="hsl(var(--ink-muted))"
                          fontSize="9"
                          fontFamily="JetBrains Mono"
                          textAnchor="middle"
                        >
                          {e.label}
                        </text>
                      </g>
                    )}
                  </g>
                )
              })}
            </svg>

            {workflowNodes.map((n) => (
              <WorkflowNode
                key={n.id}
                id={n.id}
                kind={n.type as NodeKind}
                title={n.title}
                note={n.note}
                x={n.x + 32}
                y={n.y + 32}
                active={selected === n.id}
                onClick={() => setSelected(n.id)}
              />
            ))}
          </div>
        </main>

        {/* Inspector */}
        <aside className="min-h-0 overflow-y-auto bg-surface p-5 shadow-[inset_1px_0_0_0_hsl(var(--hairline))]">
          {selected && <Inspector nodeId={selected} />}
        </aside>
      </div>
    </div>
  )
}

function PaletteGroup({
  label,
  items,
}: {
  label: string
  items: { label: string; icon: LucideIcon }[]
}) {
  return (
    <div className="mb-5">
      <div className="mb-2 text-2xs uppercase tracking-[0.12em] text-ink-dim">{label}</div>
      <div className="space-y-1">
        {items.map((it) => {
          const Icon = it.icon
          return (
            <button
              key={it.label}
              className="group flex w-full cursor-grab items-center gap-2 rounded-sm bg-elevated/50 px-2.5 py-2 text-[0.8125rem] text-ink-muted shadow-hairline transition-colors hover:bg-elevated hover:text-ink active:cursor-grabbing"
            >
              <Icon
                className="h-3.5 w-3.5 text-ink-dim group-hover:text-accent"
                strokeWidth={1.75}
              />
              <span className="flex-1 text-left">{it.label}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

function WorkflowNode({
  kind,
  title,
  note,
  x,
  y,
  active,
  onClick,
}: {
  id: string
  kind: NodeKind
  title: string
  note?: string | undefined
  x: number
  y: number
  active: boolean
  onClick: () => void
}) {
  const meta = nodeStyle[kind]
  const Icon = meta.icon
  return (
    <button
      onClick={onClick}
      className={cn(
        'absolute flex w-[200px] flex-col items-start rounded-md bg-elevated p-3 text-left transition-all',
        'shadow-elev-1 hover:shadow-elev-2',
        active
          ? 'shadow-[0_0_0_1px_hsl(var(--accent)),0_10px_30px_-10px_hsl(var(--accent)/0.4)]'
          : 'shadow-hairline'
      )}
      style={{ left: x, top: y }}
    >
      <div className="flex items-center gap-2">
        <div
          className="flex h-5 w-5 items-center justify-center rounded-xs"
          style={{ backgroundColor: `${meta.color}22`, color: meta.color }}
        >
          <Icon className="h-3 w-3" strokeWidth={2} />
        </div>
        <span
          className="text-2xs font-medium uppercase tracking-[0.12em]"
          style={{ color: meta.color }}
        >
          {meta.label}
        </span>
      </div>
      <div className="mt-2 text-[0.875rem] font-medium leading-tight text-ink">{title}</div>
      {note && <div className="mt-1 text-2xs text-ink-dim">{note}</div>}
      {/* IO pins */}
      <span className="absolute -left-1.5 top-1/2 h-3 w-3 -translate-y-1/2 rounded-full bg-elevated shadow-hairline" />
      <span className="absolute -right-1.5 top-1/2 h-3 w-3 -translate-y-1/2 rounded-full bg-elevated shadow-hairline" />
    </button>
  )
}

function Inspector({ nodeId }: { nodeId: string }) {
  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-display text-lg tracking-tightest text-ink">Inspector</h3>
        <Badge tone="info">Condição</Badge>
      </div>

      <div className="mb-5 rounded-md bg-elevated/60 p-3 shadow-hairline">
        <div className="text-2xs uppercase tracking-[0.12em] text-ink-dim">Nó selecionado</div>
        <div className="font-metric mt-1 text-xs text-ink">{nodeId}</div>
      </div>

      <FieldGroup label="Avaliar">
        <select className="h-9 w-full appearance-none rounded-md bg-elevated/60 px-3 text-sm text-ink shadow-hairline">
          <option>Lead — Score de qualificação</option>
          <option>Lead — Stage atual</option>
          <option>Lead — Tempo no stage</option>
          <option>Conversa — Última mensagem</option>
        </select>
      </FieldGroup>

      <FieldGroup label="Operador">
        <div className="flex gap-1">
          {['=', '≥', '>', '<', '≠'].map((op) => (
            <button
              key={op}
              className={cn(
                'font-metric flex h-8 w-8 items-center justify-center rounded-sm text-sm',
                op === '≥'
                  ? 'bg-accent/15 text-accent shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.4)]'
                  : 'bg-elevated/60 text-ink-muted shadow-hairline hover:text-ink'
              )}
            >
              {op}
            </button>
          ))}
        </div>
      </FieldGroup>

      <FieldGroup label="Valor">
        <input
          defaultValue="60"
          className="font-metric h-9 w-full rounded-md bg-elevated/60 px-3 text-sm text-ink shadow-hairline focus-visible:shadow-[inset_0_0_0_1px_hsl(var(--accent))]"
        />
      </FieldGroup>

      <FieldGroup label="Quando falso, desviar para">
        <select className="h-9 w-full rounded-md bg-elevated/60 px-3 text-sm text-ink shadow-hairline">
          <option>Esperar 20min (n4)</option>
          <option>Finalizar execução</option>
        </select>
      </FieldGroup>

      <div className="mt-6 rounded-md bg-accent/5 p-3 shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.2)]">
        <div className="mb-1.5 flex items-center gap-1.5 text-2xs font-medium uppercase tracking-[0.14em] text-accent">
          <Sparkles className="h-3 w-3" />
          Sugestão IA
        </div>
        <p className="text-xs leading-relaxed text-ink">
          Leads com score entre 60 e 70 convertem 18% mais quando recebem uma mensagem de
          humanização antes do handoff. Quer adicionar esse passo?
        </p>
        <Button size="xs" variant="ghost" className="mt-2 px-0 text-accent">
          Adicionar passo →
        </Button>
      </div>
    </div>
  )
}

function FieldGroup({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="mb-4">
      <label className="mb-1.5 block text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
        {label}
      </label>
      {children}
    </div>
  )
}
