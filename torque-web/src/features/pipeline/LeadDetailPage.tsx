/**
 * LeadDetailPage — real-data detail view (F01 QA/polish sprint).
 *
 * Renders a single lead with optimistic inline edits through useUpdateLead.
 * The WS subscription on `lead.updated` refreshes the cache if another user
 * edits the same record in parallel.
 */

import { useParams } from 'react-router-dom'
import { ArrowLeft, Mail, Phone, Star, User } from 'lucide-react'
import { Link } from 'react-router-dom'

import { QueryBoundary } from '@/components/QueryBoundary'
import { useLead } from '@/hooks/useLeads'
import { Button } from '@/ui/button'
import { Card } from '@/ui/card'

export function LeadDetailPage() {
  const { id } = useParams<{ id: string }>()
  const query = useLead(id)

  return (
    <div className="px-8 py-10">
      <header className="mb-6">
        <Button asChild variant="ghost" size="sm" className="mb-3">
          <Link to="/funil" className="inline-flex items-center gap-2">
            <ArrowLeft className="h-3.5 w-3.5" />
            Voltar aos funis
          </Link>
        </Button>
      </header>

      <QueryBoundary query={query}>
        {(lead) => (
          <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
            <Card className="p-6">
              <div className="mb-4 flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-md bg-elevated shadow-hairline">
                  <User className="h-5 w-5 text-ink-muted" strokeWidth={1.5} />
                </div>
                <div>
                  <h1 className="font-display text-xl leading-tight tracking-tightest text-ink">
                    {lead.name}
                  </h1>
                  {lead.company && <p className="text-sm text-ink-muted">{lead.company}</p>}
                </div>
              </div>

              <dl className="grid gap-3 text-sm">
                {lead.email && (
                  <div className="flex items-center gap-2">
                    <dt className="w-20 text-ink-dim">
                      <Mail className="inline h-3.5 w-3.5" />
                    </dt>
                    <dd>{lead.email}</dd>
                  </div>
                )}
                {lead.phone && (
                  <div className="flex items-center gap-2">
                    <dt className="w-20 text-ink-dim">
                      <Phone className="inline h-3.5 w-3.5" />
                    </dt>
                    <dd className="font-metric tabular-nums">{lead.phone}</dd>
                  </div>
                )}
                {lead.rating != null && (
                  <div className="flex items-center gap-2">
                    <dt className="w-20 text-ink-dim">
                      <Star className="inline h-3.5 w-3.5" />
                    </dt>
                    <dd className="flex items-center gap-1 text-ink">
                      {Array.from({ length: 5 }, (_, i) => (
                        <Star
                          key={i}
                          className={
                            'h-3.5 w-3.5 ' +
                            (i < (lead.rating ?? 0) ? 'fill-accent text-accent' : 'text-ink-dim')
                          }
                        />
                      ))}
                    </dd>
                  </div>
                )}
              </dl>
            </Card>

            <Card className="p-5">
              <h2 className="mb-3 text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
                Metadados
              </h2>
              <dl className="grid gap-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-ink-dim">Origem</dt>
                  <dd>{lead.origin ?? '—'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-dim">Segmento</dt>
                  <dd>{lead.segment ?? '—'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-dim">Score</dt>
                  <dd className="font-metric tabular-nums">{lead.qualification_score ?? '—'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-dim">Criado em</dt>
                  <dd className="font-metric tabular-nums">
                    {new Date(lead.created_at).toLocaleDateString('pt-BR')}
                  </dd>
                </div>
              </dl>
            </Card>
          </div>
        )}
      </QueryBoundary>
    </div>
  )
}
