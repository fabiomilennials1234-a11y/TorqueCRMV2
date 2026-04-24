import { useEffect, useId, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Building2,
  Users,
  KeyRound,
  Plug,
  CreditCard,
  Webhook,
  Shield,
  Bell,
  ChevronRight,
  Search,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { PageHeader } from '@/ui/page-header'
import { Input } from '@/ui/input'
import { Avatar } from '@/ui/avatar'
import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Skeleton } from '@/ui/skeleton'
import { EmptyState } from '@/ui/empty-state'
import { friendlyMessage } from '@/api/errors'
import {
  useAddMember,
  useDeactivateMember,
  useMembers,
  useUpdateMember,
  type MemberRole,
  type TeamMember,
} from '@/hooks/useMembers'
import { useOrganization, useUpdateOrganization } from '@/hooks/useOrgSettings'
import { IntegrationsSection } from '@/features/settings/IntegrationsSection'
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

const SECTION_KEYS: SectionKey[] = [
  'org',
  'team',
  'roles',
  'integrations',
  'billing',
  'webhooks',
  'security',
  'notifications',
]

function coerceTab(v: string | null): SectionKey {
  return SECTION_KEYS.includes(v as SectionKey) ? (v as SectionKey) : 'team'
}

export function SettingsPage() {
  const [params] = useSearchParams()
  const [active, setActive] = useState<SectionKey>(() => coerceTab(params.get('tab')))
  useEffect(() => {
    const next = coerceTab(params.get('tab'))
    setActive(next)
  }, [params])

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
  const org = useOrganization()
  const update = useUpdateOrganization()
  const [draft, setDraft] = useState<{
    name: string
    legal_name: string
    cnpj: string
    timezone: string
  } | null>(null)

  // Sync draft state to server response on first success.
  const current = org.data
  if (current && draft === null) {
    setDraft({
      name: current.name,
      legal_name: current.legal_name ?? '',
      cnpj: current.cnpj ?? '',
      timezone: current.timezone ?? 'America/Sao_Paulo',
    })
  }

  if (org.isLoading || !draft) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (org.isError) {
    return (
      <EmptyState
        title="Não foi possível carregar a organização."
        description={friendlyMessage(org.error)}
        action={
          <Button variant="ghost" size="sm" onClick={() => void org.refetch()}>
            Tentar de novo
          </Button>
        }
      />
    )
  }

  const dirty =
    !current ||
    draft.name !== current.name ||
    draft.legal_name !== (current.legal_name ?? '') ||
    draft.cnpj !== (current.cnpj ?? '') ||
    draft.timezone !== (current.timezone ?? 'America/Sao_Paulo')

  return (
    <form
      className="space-y-6"
      onSubmit={(e) => {
        e.preventDefault()
        if (!dirty) return
        void update.mutateAsync({
          name: draft.name,
          legal_name: draft.legal_name,
          cnpj: draft.cnpj,
          timezone: draft.timezone,
        })
      }}
    >
      <SettingsCard title="Identidade">
        <Field label="Nome fantasia">
          <Input
            value={draft.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </Field>
        <Field label="Razão social">
          <Input
            value={draft.legal_name}
            onChange={(e) => setDraft({ ...draft, legal_name: e.target.value })}
          />
        </Field>
        <Field label="CNPJ">
          <Input
            value={draft.cnpj}
            onChange={(e) => setDraft({ ...draft, cnpj: e.target.value })}
            className="font-metric"
          />
        </Field>
        <Field label="Fuso horário">
          <select
            className="h-9 w-full rounded-md bg-elevated/60 px-3 text-sm text-ink shadow-hairline"
            value={draft.timezone}
            onChange={(e) => setDraft({ ...draft, timezone: e.target.value })}
          >
            <option value="America/Sao_Paulo">America/Sao_Paulo (GMT-3)</option>
            <option value="America/Manaus">America/Manaus (GMT-4)</option>
            <option value="America/Belem">America/Belem (GMT-3)</option>
            <option value="UTC">UTC</option>
          </select>
        </Field>
      </SettingsCard>

      <div className="flex justify-end">
        <Button variant="primary" size="sm" type="submit" disabled={!dirty || update.isPending}>
          {update.isPending ? 'Salvando…' : 'Salvar alterações'}
        </Button>
      </div>
    </form>
  )
}

function TeamSection() {
  const [query, setQuery] = useState('')
  const [showInvite, setShowInvite] = useState(false)
  const [includeInactive, setIncludeInactive] = useState(false)
  const members = useMembers(includeInactive)
  const addMember = useAddMember()

  const filtered = useMemo<TeamMember[]>(() => {
    const rows = members.data ?? []
    const q = query.trim().toLowerCase()
    if (!q) return rows
    return rows.filter(
      (m) => m.display_name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q)
    )
  }, [members.data, query])

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-dim" />
          <Input
            placeholder="Buscar membro…"
            className="pl-8"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <label className="flex items-center gap-2 text-xs text-ink-muted">
          <input
            type="checkbox"
            checked={includeInactive}
            onChange={(e) => setIncludeInactive(e.target.checked)}
          />
          Mostrar inativos
        </label>
        <div className="ml-auto">
          <Button variant="primary" size="sm" onClick={() => setShowInvite((v) => !v)}>
            Convidar membro
          </Button>
        </div>
      </div>

      {showInvite && (
        <InviteForm
          disabled={addMember.isPending}
          onCancel={() => setShowInvite(false)}
          onSubmit={async (body) => {
            await addMember.mutateAsync(body)
            setShowInvite(false)
          }}
        />
      )}

      {members.isLoading && (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      )}

      {members.isError && (
        <EmptyState
          title="Não foi possível carregar o time."
          description={friendlyMessage(members.error)}
          action={
            <Button variant="ghost" size="sm" onClick={() => void members.refetch()}>
              Tentar de novo
            </Button>
          }
        />
      )}

      {members.isSuccess && filtered.length === 0 && (
        <EmptyState
          title="Nenhum membro encontrado"
          description="Ajuste a busca ou convide alguém."
        />
      )}

      {members.isSuccess && filtered.length > 0 && (
        <div className="overflow-hidden rounded-lg bg-surface shadow-elev-1">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-2xs uppercase tracking-[0.12em] text-ink-dim">
                <th className="px-5 py-3 text-left font-medium">Membro</th>
                <th className="px-5 py-3 text-left font-medium">Papel</th>
                <th className="px-5 py-3 text-left font-medium">Status</th>
                <th className="px-5 py-3" />
              </tr>
            </thead>
            <tbody>
              {filtered.map((m, i) => (
                <MemberRow key={m.id} member={m} separator={i > 0} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function InviteForm({
  disabled,
  onSubmit,
  onCancel,
}: {
  disabled: boolean
  onSubmit: (d: { email: string; display_name: string; role: MemberRole }) => Promise<void>
  onCancel: () => void
}) {
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [role, setRole] = useState<MemberRole>('membro')
  const emailId = useId()
  const nameId = useId()
  const roleId = useId()
  const valid = email.includes('@') && displayName.trim().length > 0

  return (
    <form
      className="flex flex-wrap items-end gap-2 rounded-lg bg-surface p-4 shadow-elev-1"
      onSubmit={(e) => {
        e.preventDefault()
        if (!valid) return
        void onSubmit({ email: email.trim(), display_name: displayName.trim(), role })
      }}
    >
      <div className="min-w-[200px] flex-1">
        <label htmlFor={emailId} className="mb-1 block text-xs text-ink-muted">
          E-mail do usuário
        </label>
        <Input
          id={emailId}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          type="email"
          required
        />
      </div>
      <div className="min-w-[200px] flex-1">
        <label htmlFor={nameId} className="mb-1 block text-xs text-ink-muted">
          Nome de exibição
        </label>
        <Input
          id={nameId}
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          required
        />
      </div>
      <div className="w-36">
        <label htmlFor={roleId} className="mb-1 block text-xs text-ink-muted">
          Papel
        </label>
        <select
          id={roleId}
          className="h-9 w-full rounded-md bg-elevated/60 px-3 text-sm text-ink shadow-hairline"
          value={role}
          onChange={(e) => setRole(e.target.value as MemberRole)}
        >
          <option value="membro">Membro</option>
          <option value="admin">Admin</option>
        </select>
      </div>
      <div className="flex gap-2">
        <Button variant="ghost" size="sm" type="button" onClick={onCancel}>
          Cancelar
        </Button>
        <Button variant="primary" size="sm" type="submit" disabled={disabled || !valid}>
          {disabled ? 'Adicionando…' : 'Adicionar'}
        </Button>
      </div>
    </form>
  )
}

function MemberRow({ member, separator }: { member: TeamMember; separator: boolean }) {
  const [editing, setEditing] = useState(false)
  const update = useUpdateMember(member.id)
  const deactivate = useDeactivateMember(member.id)

  return (
    <tr
      className={cn(
        'transition-colors hover:bg-elevated/40',
        separator && 'shadow-[inset_0_1px_0_0_hsl(var(--hairline))]'
      )}
    >
      <td className="px-5 py-3">
        <div className="flex items-center gap-3">
          <Avatar size="md" fallback={initials(member.display_name)} />
          <div>
            <div className="text-sm text-ink">{member.display_name}</div>
            <div className="font-metric text-2xs text-ink-dim">{member.email}</div>
          </div>
        </div>
      </td>
      <td className="px-5 py-3">
        {editing ? (
          <select
            className="h-7 rounded-md bg-elevated/60 px-2 text-xs text-ink shadow-hairline"
            defaultValue={member.role}
            onChange={(e) => {
              const next = e.target.value as MemberRole
              void update.mutateAsync({ role: next }).finally(() => setEditing(false))
            }}
          >
            <option value="membro">Membro</option>
            <option value="admin">Admin</option>
          </select>
        ) : (
          <Badge tone={member.role === 'admin' ? 'accent' : 'neutral'}>
            {member.role === 'admin' ? 'Admin' : 'Membro'}
          </Badge>
        )}
      </td>
      <td className="px-5 py-3">
        {member.is_active ? (
          <span className="inline-flex items-center gap-1.5 text-xs text-success">
            <span className="h-1.5 w-1.5 rounded-full bg-success" />
            ativo
          </span>
        ) : (
          <span className="inline-flex items-center gap-1.5 text-xs text-ink-dim">
            <span className="h-1.5 w-1.5 rounded-full bg-ink-dim" />
            inativo
          </span>
        )}
      </td>
      <td className="px-5 py-3 text-right">
        {member.is_active && (
          <div className="flex justify-end gap-1">
            <Button variant="ghost" size="xs" onClick={() => setEditing((v) => !v)}>
              {editing ? 'Cancelar' : 'Editar'}
            </Button>
            <Button
              variant="ghost"
              size="xs"
              disabled={deactivate.isPending}
              onClick={() => void deactivate.mutateAsync()}
            >
              Desativar
            </Button>
          </div>
        )}
      </td>
    </tr>
  )
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return (parts[0] ?? '').slice(0, 2).toUpperCase()
  return ((parts[0]?.[0] ?? '') + (parts[parts.length - 1]?.[0] ?? '')).toUpperCase()
}

// IntegrationsSection is now owned by features/settings/IntegrationsSection.tsx
// (S50 — rich UI with connect modals, sync action, last-error surfacing).

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
