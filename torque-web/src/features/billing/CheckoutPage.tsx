/**
 * F14 CheckoutPage.
 *
 * Flow: no subscription → show plan picker (stub: single plan) → start
 * checkout → provider returns PIX payload → display QR for the user.
 * Active sub → display current period + cancel option. The actual state
 * transitions happen via provider webhook; the page reacts to
 * `subscription.*` WS events.
 */

import { useState } from 'react'

import { Button } from '@/ui/button'
import { Badge } from '@/ui/badge'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { friendlyMessage } from '@/api/errors'
import {
  useCancelSubscription,
  useStartCheckout,
  useSubscription,
  type Subscription,
} from '@/hooks/useBilling'

const PLAN = { id: 'growth', label: 'Growth', monthlyCents: 19900 }

export function CheckoutPage() {
  const sub = useSubscription()

  return (
    <div className="mx-auto max-w-[900px] px-8">
      <PageHeader
        eyebrow="Plano e faturamento"
        title="Assinatura"
        description="Gerencie sua assinatura do Torque. Pagamentos via PIX, confirmados automaticamente."
      />

      {sub.isLoading && (
        <div className="mt-6 space-y-2">
          <Skeleton className="h-8 w-1/2" />
          <Skeleton className="h-32 w-full" />
        </div>
      )}

      {sub.isError && (
        <EmptyState
          title="Não foi possível carregar a assinatura."
          description={friendlyMessage(sub.error)}
          action={
            <Button variant="ghost" size="sm" onClick={() => void sub.refetch()}>
              Tentar de novo
            </Button>
          }
        />
      )}

      {sub.isSuccess && !sub.data && <NoSubscription />}
      {sub.isSuccess && sub.data && <ActiveSubscription subscription={sub.data} />}
    </div>
  )
}

function NoSubscription() {
  const start = useStartCheckout()
  const [startedId, setStartedId] = useState<string | null>(null)

  async function handleStart() {
    const sub = await start.mutateAsync({
      plan_id: PLAN.id,
      amount_cents: PLAN.monthlyCents,
      currency: 'BRL',
    })
    setStartedId(sub.id)
  }

  return (
    <div className="mt-6 rounded-lg bg-surface p-8 shadow-elev-1">
      <h2 className="font-display text-xl text-ink">Você ainda não tem uma assinatura ativa</h2>
      <p className="mt-2 text-sm text-ink-muted">
        Escolha um plano e gere a cobrança PIX. Após o pagamento, sua organização é
        habilitada automaticamente.
      </p>

      <div className="mt-6 flex items-center justify-between rounded-md bg-elevated/40 px-5 py-4 shadow-hairline">
        <div>
          <div className="text-sm text-ink">{PLAN.label}</div>
          <div className="text-xs text-ink-muted">Assinatura mensal</div>
        </div>
        <div className="font-metric text-lg text-ink">{formatMoney(PLAN.monthlyCents, 'BRL')}</div>
      </div>

      <div className="mt-6 flex gap-2">
        <Button
          variant="primary"
          size="md"
          disabled={start.isPending}
          onClick={() => void handleStart()}
        >
          {start.isPending ? 'Gerando cobrança…' : 'Iniciar checkout'}
        </Button>
      </div>

      {startedId && (
        <p className="mt-4 text-xs text-ink-dim">
          Cobrança {startedId} gerada. Aguardando confirmação do pagamento.
        </p>
      )}
    </div>
  )
}

function ActiveSubscription({ subscription }: { subscription: Subscription }) {
  const cancel = useCancelSubscription()
  return (
    <div className="mt-6 space-y-4">
      <div className="flex items-center gap-3">
        <h2 className="font-display text-xl text-ink">{subscription.plan_id}</h2>
        <StatusBadge status={subscription.status} />
      </div>

      <div className="rounded-lg bg-surface p-6 shadow-elev-1">
        <Row label="Valor" value={formatMoney(subscription.amount_cents, subscription.currency)} />
        <Row label="Provedor" value={subscription.provider} />
        {subscription.current_period_end && (
          <Row label="Renovação" value={formatDate(subscription.current_period_end)} />
        )}
        {subscription.pix_expires_at && subscription.status === 'pending' && (
          <Row label="PIX expira em" value={formatDate(subscription.pix_expires_at)} />
        )}
      </div>

      {subscription.status === 'pending' && subscription.pix_qr_code && (
        <div className="rounded-lg bg-surface p-6 shadow-elev-1">
          <h3 className="font-display text-lg text-ink">Pague via PIX</h3>
          <p className="mt-1 text-xs text-ink-muted">
            Copie o código abaixo ou escaneie o QR no app do seu banco.
          </p>
          <pre className="mt-3 max-w-full overflow-x-auto rounded-md bg-elevated/40 p-3 font-mono text-xs text-ink">
            {subscription.pix_qr_code}
          </pre>
        </div>
      )}

      {(subscription.status === 'active' || subscription.status === 'past_due') && (
        <Button
          variant="ghost"
          size="sm"
          disabled={cancel.isPending}
          onClick={() => void cancel.mutateAsync()}
        >
          Cancelar assinatura
        </Button>
      )}
    </div>
  )
}

function StatusBadge({ status }: { status: Subscription['status'] }) {
  const toneMap: Record<Subscription['status'], 'accent' | 'success' | 'warning' | 'neutral'> = {
    pending: 'warning',
    active: 'success',
    past_due: 'warning',
    cancelled: 'neutral',
    expired: 'neutral',
  }
  const label: Record<Subscription['status'], string> = {
    pending: 'Aguardando pagamento',
    active: 'Ativa',
    past_due: 'Em atraso',
    cancelled: 'Cancelada',
    expired: 'Expirada',
  }
  return <Badge tone={toneMap[status]}>{label[status]}</Badge>
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-ink-muted">{label}</span>
      <span className="font-metric text-ink">{value}</span>
    </div>
  )
}

function formatMoney(cents: number, currency: string) {
  try {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(cents / 100)
  } catch {
    return `${(cents / 100).toFixed(2)} ${currency}`
  }
}

function formatDate(iso: string) {
  const d = new Date(iso)
  return d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}
