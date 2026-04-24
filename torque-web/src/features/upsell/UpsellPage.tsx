/**
 * F12 Upsell — UpsellPage (S48).
 *
 * Oportunidades de expansão: lista leads recentes que o vendedor pode
 * abordar com produto complementar. V8 tem uma materialized view para
 * isso; S48 entrega a experiência da tela consumindo o hook de leads
 * já existente (filtro rico + view dedicada ficam para quando um
 * tenant pedir critérios mais ricos que "recentes").
 */

import { useLeads, type Lead } from '@/hooks/useLeads'

import { Badge } from '@/ui/badge'
import { EmptyState } from '@/ui/empty-state'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'

export function UpsellPage() {
  const query = useLeads({ page_size: 50 })

  return (
    <div className="mx-auto max-w-5xl px-8 py-8">
      <PageHeader
        eyebrow="Expansão"
        title="Oportunidades de upsell"
        description="Leads recentes elegíveis para ofertas complementares."
      />
      {query.isLoading ? (
        <Skeleton className="mt-6 h-40" />
      ) : query.items.length === 0 ? (
        <EmptyState
          title="Nenhuma oportunidade ainda"
          description="Leads com interação recente aparecerão aqui."
        />
      ) : (
        <ul className="mt-6 grid grid-cols-1 gap-2 md:grid-cols-2">
          {query.items.map((lead: Lead) => (
            <li key={lead.id} className="rounded-lg bg-surface p-3 shadow-elev-1">
              <div className="flex items-center gap-2">
                <span className="font-medium text-ink">{lead.name}</span>
                {lead.segment && <Badge tone="neutral">{lead.segment}</Badge>}
                {lead.origin && <Badge tone="neutral">{lead.origin}</Badge>}
              </div>
              {lead.company && <div className="mt-1 text-2xs text-ink-dim">{lead.company}</div>}
              <div className="mt-1 text-2xs text-ink-dim">
                Atualizado: {new Date(lead.updated_at).toLocaleString('pt-BR')}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
