/**
 * F08 Campanhas — CampaignDetailPage (S46).
 *
 * Rota /campaigns/:id. Mostra stats detalhadas + lista de recipients com
 * status individual. Ações contextuais (Lançar/Pausar/Retomar/Cancelar)
 * em função do estado atual.
 */

import { ArrowLeft, Ban, Pause, Play } from 'lucide-react'
import { Link, Navigate, useParams } from 'react-router-dom'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { friendlyMessage } from '@/api/errors'
import {
  useCampaign,
  useCancelCampaign,
  usePauseCampaign,
  useRecipients,
  useResumeCampaign,
  type Campaign,
  type CampaignStatus,
} from '@/hooks/useCampaigns'

const statusTone: Record<CampaignStatus, 'success' | 'danger' | 'neutral'> = {
  draft: 'neutral',
  scheduled: 'neutral',
  running: 'success',
  paused: 'neutral',
  completed: 'success',
  cancelled: 'danger',
}

export function CampaignDetailPage() {
  const { id } = useParams<{ id: string }>()
  if (!id) return <Navigate to="/campaigns" replace />
  return <CampaignDetailInner campaignId={id} />
}

function CampaignDetailInner({ campaignId }: { campaignId: string }) {
  const query = useCampaign(campaignId)
  const recipients = useRecipients(campaignId)

  if (query.isLoading) return <DetailSkeleton />
  if (query.isError || !query.data) {
    return (
      <div className="mx-auto max-w-md p-12 text-center">
        <p className="text-sm text-danger">{friendlyMessage(query.error)}</p>
        <Link to="/campaigns" className="mt-4 inline-block text-sm text-accent underline">
          Voltar
        </Link>
      </div>
    )
  }

  const campaign = query.data

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <Link
        to="/campaigns"
        className="mb-3 inline-flex items-center gap-1.5 text-xs text-ink-dim hover:text-ink-muted"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Campanhas
      </Link>
      <PageHeader
        eyebrow="Campanha"
        title={campaign.name}
        description={campaign.description ?? undefined}
        actions={<CampaignActions campaign={campaign} />}
      />

      <div className="mt-6 grid grid-cols-2 gap-3 md:grid-cols-5">
        <StatCard label="Status" value={campaign.status} tone={statusTone[campaign.status]} />
        <NumberCard label="Enfileirados" value={campaign.stats_queued} />
        <NumberCard label="Enviados" value={campaign.stats_sent} />
        <NumberCard label="Falhas" value={campaign.stats_failed} danger />
        <NumberCard label="Pulados" value={campaign.stats_skipped} />
      </div>

      <section className="mt-8">
        <h2 className="mb-3 text-sm font-medium text-ink">Destinatários</h2>
        {recipients.isLoading ? (
          <Skeleton className="h-24" />
        ) : (recipients.data ?? []).length === 0 ? (
          <EmptyState
            title="Nenhum destinatário ainda"
            description="Lance a campanha para enfileirar leads."
          />
        ) : (
          <ul className="space-y-1.5">
            {recipients.data!.map((r) => (
              <li
                key={r.id}
                className="flex items-center justify-between rounded-md bg-elevated/30 px-3 py-2 text-xs"
              >
                <span className="font-mono text-ink">{r.lead_id.slice(0, 8)}</span>
                <div className="flex items-center gap-2">
                  <Badge tone={recipientTone(r.status)}>{r.status}</Badge>
                  {r.sent_at && (
                    <span className="text-2xs text-ink-dim">
                      {new Date(r.sent_at).toLocaleString('pt-BR')}
                    </span>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

function recipientTone(status: string): 'success' | 'danger' | 'neutral' {
  switch (status) {
    case 'sent':
      return 'success'
    case 'failed':
      return 'danger'
    default:
      return 'neutral'
  }
}

function CampaignActions({ campaign }: { campaign: Campaign }) {
  const pause = usePauseCampaign(campaign.id)
  const resume = useResumeCampaign(campaign.id)
  const cancel = useCancelCampaign(campaign.id)

  return (
    <div className="flex items-center gap-2">
      {campaign.status === 'running' && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void pause.mutateAsync()}
          disabled={pause.isPending}
        >
          <Pause className="mr-1 h-3.5 w-3.5" />
          Pausar
        </Button>
      )}
      {campaign.status === 'paused' && (
        <Button
          type="button"
          variant="primary"
          size="sm"
          onClick={() => void resume.mutateAsync()}
          disabled={resume.isPending}
        >
          <Play className="mr-1 h-3.5 w-3.5" />
          Retomar
        </Button>
      )}
      {['draft', 'scheduled', 'running', 'paused'].includes(campaign.status) && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => void cancel.mutateAsync()}
          disabled={cancel.isPending}
        >
          <Ban className="mr-1 h-3.5 w-3.5" />
          Cancelar
        </Button>
      )}
    </div>
  )
}

function StatCard({
  label,
  value,
  tone,
}: {
  label: string
  value: string
  tone: 'success' | 'danger' | 'neutral'
}) {
  return (
    <div className="rounded-lg bg-surface p-3 shadow-elev-1">
      <div className="text-2xs uppercase tracking-wide text-ink-dim">{label}</div>
      <div className="mt-2">
        <Badge tone={tone}>{value}</Badge>
      </div>
    </div>
  )
}

function NumberCard({ label, value, danger }: { label: string; value: number; danger?: boolean }) {
  return (
    <div className="rounded-lg bg-surface p-3 shadow-elev-1">
      <div className="text-2xs uppercase tracking-wide text-ink-dim">{label}</div>
      <div
        className={
          'font-fraunces mt-2 text-2xl ' + (danger && value > 0 ? 'text-danger' : 'text-ink')
        }
      >
        {value.toLocaleString('pt-BR')}
      </div>
    </div>
  )
}

function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-5xl space-y-4 px-8 py-8">
      <Skeleton className="h-8 w-64" />
      <div className="grid grid-cols-5 gap-3">
        <Skeleton className="h-20" />
        <Skeleton className="h-20" />
        <Skeleton className="h-20" />
        <Skeleton className="h-20" />
        <Skeleton className="h-20" />
      </div>
      <Skeleton className="h-48 w-full" />
    </div>
  )
}
