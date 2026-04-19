import { useState } from 'react'
import {
  Building2,
  Users,
  KeyRound,
  Plug,
  CreditCard,
  Webhook,
  Shield,
  Bell,
  Check,
  ChevronRight,
  Search,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { PageHeader } from '@/ui/page-header'
import { Input } from '@/ui/input'
import { Avatar } from '@/ui/avatar'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { cn } from '@/lib/utils'

type SectionKey =
  | 'org'
  | 'team'
  | 'roles'
  | 'integrations'
  | 'billing'
  | 'webhooks'
  | 'security'
  | 'notifications'

const sections: { key: SectionKey; label: string; icon: LucideIcon; description: string }[] = [
  { key: 'org', label: 'Organização', icon: Building2, description: 'Identidade, domínios, fuso' },
  { key: 'team', label: 'Time', icon: Users, description: 'Usuários e especializações' },
  { key: 'roles', label: 'Papéis e permissões', icon: KeyRound, description: 'Matriz de acesso' },
  {
    key: 'integrations',
    label: 'Integrações',
    icon: Plug,
    description: 'WhatsApp, Meta, Asaas, ERP',
  },
  {
    key: 'billing',
    label: 'Plano e faturamento',
    icon: CreditCard,
    description: 'Growth · 5.000 leads/mês',
  },
  { key: 'webhooks', label: 'Webhooks', icon: Webhook, description: 'API pública e destinos' },
  { key: 'security', label: 'Segurança', icon: Shield, description: '2FA, SSO, auditoria' },
  { key: 'notifications', label: 'Notificações', icon: Bell, description: 'Canais e silêncios' },
]

export function SettingsPage() {
  const [active, setActive] = useState<SectionKey>('team')

  return (
    <div className="mx-auto max-w-[1400px] px-8">
      <PageHeader
        eyebrow="Configurações"
        title="Preferências da organização"
        description="O que você ajusta aqui vale para toda a Siderúrgica Aurora. Ações críticas pedem confirmação dupla e ficam no log de auditoria."
      />

      <div className="mt-6 grid gap-8 lg:grid-cols-[240px_1fr]">
        <nav className="space-y-0.5">
          {sections.map((s) => (
            <button
              key={s.key}
              onClick={() => setActive(s.key)}
              className={cn(
                'group flex w-full items-start gap-3 rounded-md px-3 py-2.5 text-left transition-colors',
                active === s.key ? 'bg-elevated shadow-hairline' : 'hover:bg-elevated/40'
              )}
            >
              <s.icon
                className={cn(
                  'mt-0.5 h-4 w-4 shrink-0',
                  active === s.key ? 'text-accent' : 'text-ink-dim'
                )}
                strokeWidth={1.75}
              />
              <div className="min-w-0 flex-1">
                <div
                  className={cn(
                    'text-sm',
                    active === s.key ? 'font-medium text-ink' : 'text-ink-muted'
                  )}
                >
                  {s.label}
                </div>
                <div className="text-2xs text-ink-dim">{s.description}</div>
              </div>
              {active === s.key && <ChevronRight className="mt-1 h-3 w-3 text-ink-dim" />}
            </button>
          ))}
        </nav>

        <div className="min-w-0">
          {active === 'team' && <TeamSection />}
          {active === 'org' && <OrgSection />}
          {active === 'integrations' && <IntegrationsSection />}
          {active !== 'team' && active !== 'org' && active !== 'integrations' && (
            <Placeholder label={sections.find((s) => s.key === active)!.label} />
          )}
        </div>
      </div>
    </div>
  )
}

function OrgSection() {
  return (
    <div className="space-y-6">
      <SettingsCard title="Identidade">
        <Field label="Nome fantasia">
          <Input defaultValue="Siderúrgica Aurora" />
        </Field>
        <Field label="Domínio principal">
          <Input defaultValue="aurora.ind.br" />
        </Field>
        <Field label="CNPJ">
          <Input defaultValue="12.345.678/0001-90" className="font-metric" />
        </Field>
        <Field label="Fuso horário">
          <select className="h-9 w-full rounded-md bg-elevated/60 px-3 text-sm text-ink shadow-hairline">
            <option>America/Sao_Paulo (GMT-3)</option>
          </select>
        </Field>
      </SettingsCard>

      <SettingsCard
        title="Marca"
        description="Aparece em relatórios, templates e notificações externas."
      >
        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-accent/15 font-display text-lg text-accent shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.3)]">
            SA
          </div>
          <div>
            <Button variant="secondary" size="sm">
              Enviar logo
            </Button>
            <p className="mt-1 text-2xs text-ink-dim">SVG ou PNG transparente · 256px mínimo</p>
          </div>
        </div>
      </SettingsCard>
    </div>
  )
}

function TeamSection() {
  const team = [
    {
      name: 'Fábio Milennials',
      email: 'fabio@aurora.ind.br',
      role: 'Admin',
      spec: '—',
      status: 'active',
      initials: 'FM',
    },
    {
      name: 'Rafael Bento',
      email: 'rafael@aurora.ind.br',
      role: 'Membro',
      spec: 'Closer',
      status: 'active',
      initials: 'RB',
    },
    {
      name: 'Maíra Duarte',
      email: 'maira@aurora.ind.br',
      role: 'Membro',
      spec: 'SDR',
      status: 'active',
      initials: 'MD',
    },
    {
      name: 'Tatiana Alves',
      email: 'tatiana@aurora.ind.br',
      role: 'Membro',
      spec: 'SDR',
      status: 'active',
      initials: 'TA',
    },
    {
      name: 'Pedro Lima',
      email: 'pedro@aurora.ind.br',
      role: 'Membro',
      spec: 'Prospectador',
      status: 'pending',
      initials: 'PL',
    },
  ]
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-dim" />
          <Input placeholder="Buscar membro…" className="pl-8" />
        </div>
        <div className="ml-auto flex items-center gap-2">
          <Button variant="outline" size="sm">
            Importar CSV
          </Button>
          <Button variant="primary" size="sm">
            Convidar membro
          </Button>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg bg-surface shadow-elev-1">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-2xs uppercase tracking-[0.12em] text-ink-dim">
              <th className="px-5 py-3 text-left font-medium">Membro</th>
              <th className="px-5 py-3 text-left font-medium">Papel</th>
              <th className="px-5 py-3 text-left font-medium">Especialização</th>
              <th className="px-5 py-3 text-left font-medium">Status</th>
              <th className="px-5 py-3" />
            </tr>
          </thead>
          <tbody>
            {team.map((m, i) => (
              <tr
                key={m.email}
                className={cn(
                  'transition-colors hover:bg-elevated/40',
                  i > 0 && 'shadow-[inset_0_1px_0_0_hsl(var(--hairline))]'
                )}
              >
                <td className="px-5 py-3">
                  <div className="flex items-center gap-3">
                    <Avatar size="md" fallback={m.initials} />
                    <div>
                      <div className="text-sm text-ink">{m.name}</div>
                      <div className="font-metric text-2xs text-ink-dim">{m.email}</div>
                    </div>
                  </div>
                </td>
                <td className="px-5 py-3">
                  <Badge tone={m.role === 'Admin' ? 'accent' : 'neutral'}>{m.role}</Badge>
                </td>
                <td className="px-5 py-3 text-ink-muted">{m.spec}</td>
                <td className="px-5 py-3">
                  {m.status === 'active' ? (
                    <span className="inline-flex items-center gap-1.5 text-xs text-success">
                      <span className="h-1.5 w-1.5 rounded-full bg-success" />
                      ativo
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1.5 text-xs text-warning">
                      <span className="h-1.5 w-1.5 rounded-full bg-warning" />
                      convite pendente
                    </span>
                  )}
                </td>
                <td className="px-5 py-3 text-right">
                  <Button variant="ghost" size="xs">
                    Gerenciar
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function IntegrationsSection() {
  const integrations = [
    {
      name: 'WhatsApp · Evolution API',
      desc: 'Canal conversacional principal',
      status: 'connected',
      logo: 'WA',
    },
    { name: 'Meta Ads', desc: 'Ingestão de leads Lead Ads', status: 'connected', logo: 'M' },
    { name: 'Asaas', desc: 'Cobrança e assinaturas', status: 'connected', logo: 'A' },
    { name: 'Google Calendar', desc: 'Agenda de reuniões', status: 'connected', logo: 'G' },
    { name: 'TinyERP', desc: 'Catálogo de produtos e pedidos', status: 'disconnected', logo: 'T' },
    { name: 'n8n (auto-hospedado)', desc: 'Orquestrador externo', status: 'action', logo: 'n8' },
  ]
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {integrations.map((i) => (
        <div
          key={i.name}
          className="flex items-start gap-3 rounded-lg bg-surface p-4 shadow-elev-1 transition-shadow hover:shadow-elev-2"
        >
          <div className="font-metric flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-elevated text-xs text-ink-muted shadow-hairline">
            {i.logo}
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-sm font-medium text-ink">{i.name}</div>
            <div className="text-xs text-ink-muted">{i.desc}</div>
            <div className="mt-2 flex items-center gap-2">
              {i.status === 'connected' && (
                <Badge tone="success">
                  <Check className="h-2.5 w-2.5" />
                  conectado
                </Badge>
              )}
              {i.status === 'disconnected' && <Badge tone="neutral">desconectado</Badge>}
              {i.status === 'action' && <Badge tone="warning">requer atenção</Badge>}
              <Button variant="ghost" size="xs">
                {i.status === 'connected' ? 'Gerenciar' : 'Conectar'}
              </Button>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

function Placeholder({ label }: { label: string }) {
  return (
    <div className="rounded-lg bg-surface p-10 text-center shadow-elev-1">
      <p className="text-sm text-ink-muted">
        A seção <span className="text-ink">{label}</span> ainda não tem detalhamento neste skeleton.
        Será implementada no próximo sprint com os respectivos contratos do backend Go.
      </p>
    </div>
  )
}

function SettingsCard({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: React.ReactNode
}) {
  return (
    <div className="rounded-lg bg-surface shadow-elev-1">
      <div className="px-5 py-4 shadow-hairline-b">
        <h3 className="text-sm font-medium text-ink">{title}</h3>
        {description && <p className="mt-1 text-xs text-ink-muted">{description}</p>}
      </div>
      <div className="space-y-4 p-5">{children}</div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[200px_1fr] items-center gap-4">
      <label className="text-xs text-ink-muted">{label}</label>
      <div>{children}</div>
    </div>
  )
}
