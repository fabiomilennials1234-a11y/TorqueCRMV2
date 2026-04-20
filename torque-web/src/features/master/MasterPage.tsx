/**
 * F16 Master Admin page. Access is gated server-side by RequireMaster;
 * a non-master user hitting /master sees the page shell but every query
 * below returns 403, surfaced as toasts.
 */

import { Building2, Users, AlertTriangle } from 'lucide-react'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { friendlyMessage } from '@/api/errors'
import {
  useImpersonate,
  useMasterOrganizations,
  useSystemHealth,
  type SystemHealth as SystemHealthT,
  type MasterOrg,
} from '@/hooks/useMaster'

export function MasterPage() {
  return (
    <div className="mx-auto max-w-[1400px] px-8">
      <PageHeader
        eyebrow="Master Admin"
        title="Operações globais"
        description="Visão cross-tenant. Todas as ações aqui geram registro de auditoria."
      />
      <div className="mt-8">
        <HealthCards />
      </div>
      <div className="mt-10">
        <OrganizationsTable />
      </div>
    </div>
  )
}

function HealthCards() {
  const h = useSystemHealth()
  if (h.isLoading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-24 w-full" />)}
      </div>
    )
  }
  if (h.isError || !h.data) {
    return (
      <EmptyState
        title="Não foi possível carregar a saúde do sistema."
        description={friendlyMessage(h.error)}
      />
    )
  }
  const d: SystemHealthT = h.data
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <Metric icon={Building2} label="Orgs ativas" value={d.active_org_count} subvalue={`de ${d.org_count} totais`} />
      <Metric icon={Users} label="Usuários" value={d.user_count} subvalue={`${d.lead_count} leads`} />
      <Metric
        icon={Building2}
        label="Assinaturas"
        value={d.active_subscriptions}
        subvalue={`${d.pending_subscriptions} pendentes`}
      />
      <Metric
        icon={AlertTriangle}
        label="Ops falhas 24h"
        value={d.operations_failed_24h}
        subvalue={`${d.operations_running} em execução`}
        danger={d.operations_failed_24h > 0}
      />
    </div>
  )
}

function Metric({
  icon: Icon,
  label,
  value,
  subvalue,
  danger,
}: {
  icon: typeof Building2
  label: string
  value: number
  subvalue: string
  danger?: boolean
}) {
  return (
    <div className="rounded-lg bg-surface p-4 shadow-elev-1">
      <div className="mb-2 flex items-center gap-2 text-2xs uppercase tracking-[0.12em] text-ink-dim">
        <Icon className="h-3 w-3" strokeWidth={1.75} />
        {label}
      </div>
      <div className={`font-metric text-2xl ${danger ? 'text-warning' : 'text-ink'}`}>{value}</div>
      <div className="mt-1 text-xs text-ink-dim">{subvalue}</div>
    </div>
  )
}

function OrganizationsTable() {
  const orgs = useMasterOrganizations()
  const imp = useImpersonate()

  async function enterOrg(o: MasterOrg) {
    // Phase 1: confirm audit row is written and surface the target. The
    // follow-up sprint swaps the cookie; today we display the target in a
    // toast-style info row so the master sees the write succeeded.
    await imp.mutateAsync({ organization_id: o.id })
  }

  if (orgs.isLoading) {
    return <Skeleton className="h-64 w-full" />
  }
  if (orgs.isError || !orgs.data) {
    return <EmptyState title="Erro carregando organizações" description={friendlyMessage(orgs.error)} />
  }
  if (orgs.data.length === 0) {
    return <EmptyState title="Nenhuma organização ainda" description="Sistema em estado inicial." />
  }
  return (
    <div className="overflow-hidden rounded-lg bg-surface shadow-elev-1">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-2xs uppercase tracking-[0.12em] text-ink-dim">
            <th className="px-5 py-3 text-left font-medium">Organização</th>
            <th className="px-5 py-3 text-left font-medium">Plano</th>
            <th className="px-5 py-3 text-left font-medium">Status</th>
            <th className="px-5 py-3 text-right font-medium">Membros</th>
            <th className="px-5 py-3 text-right font-medium">Leads</th>
            <th className="px-5 py-3" />
          </tr>
        </thead>
        <tbody>
          {orgs.data.map((o, i) => (
            <tr
              key={o.id}
              className={
                'transition-colors hover:bg-elevated/40' +
                (i > 0 ? ' shadow-[inset_0_1px_0_0_hsl(var(--hairline))]' : '')
              }
            >
              <td className="px-5 py-3">
                <div className="text-sm text-ink">{o.name}</div>
                <div className="font-metric text-2xs text-ink-dim">{o.slug}</div>
              </td>
              <td className="px-5 py-3 text-ink-muted">{o.plan_id ?? '—'}</td>
              <td className="px-5 py-3">
                <Badge tone={o.payment_status === 'active' ? 'success' : 'warning'}>
                  {o.payment_status}
                </Badge>
              </td>
              <td className="px-5 py-3 text-right font-metric text-ink">{o.member_count}</td>
              <td className="px-5 py-3 text-right font-metric text-ink">{o.lead_count}</td>
              <td className="px-5 py-3 text-right">
                <Button
                  variant="ghost"
                  size="xs"
                  disabled={imp.isPending}
                  onClick={() => void enterOrg(o)}
                >
                  Entrar como admin
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
