/**
 * FunisHubPage — the entry to F01. Lists every pipe in the tenant with a
 * quick summary (kind, position, default flag) and a link into the Kanban
 * view of each. Real data, no fixtures.
 */

import { Link } from 'react-router-dom'
import { ArrowRight, Layers, Star } from 'lucide-react'

import { QueryBoundary } from '@/components/QueryBoundary'
import { usePipes } from '@/hooks/usePipes'
import { useQuota } from '@/hooks/useQuotas'
import { Card } from '@/ui/card'
import { Pill } from '@/ui/pill'
import { QuotaMeter } from '@/ui/quota-meter'

export function FunisHubPage() {
  const pipes = usePipes()
  const leadsQuota = useQuota('leads')

  return (
    <div className="px-8 py-10">
      <header className="mb-8">
        <div className="mb-2 inline-flex items-center gap-2 text-2xs font-medium uppercase tracking-[0.16em] text-ink-muted">
          <Layers className="h-3.5 w-3.5" />
          Funis
        </div>
        <h1 className="font-display text-[2rem] leading-[1.1] tracking-tightest text-ink">
          Escolha um funil para começar
        </h1>
        <p className="mt-2 max-w-xl text-sm text-ink-muted">
          Cada funil é um pipeline independente: WhatsApp para qualificar, confirmação para reuniões,
          propostas para fechamento. Mova leads entre etapas e o histórico fica registrado.
        </p>
        {leadsQuota.data && (
          <div className="mt-4 max-w-xs">
            <QuotaMeter quota={leadsQuota.data} label="Leads do plano" />
          </div>
        )}
      </header>

      <QueryBoundary
        query={pipes}
        isEmpty={(ps) => ps.length === 0}
        emptyFallback={
          <Card className="p-8 text-center">
            <p className="text-sm text-ink-muted">
              Sua organização ainda não tem funis ativos. Fale com um administrador.
            </p>
          </Card>
        }
      >
        {(items) => (
          <ul className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {items.map((p) => (
              <li key={p.id}>
                <Link
                  to={`/pipe/${p.id}`}
                  className="group block rounded-md shadow-hairline transition hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim)/0.5)]"
                >
                  <Card className="h-full p-5">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="mb-2 inline-flex items-center gap-2 text-2xs uppercase tracking-[0.14em] text-ink-dim">
                          {p.kind}
                          {p.is_default && (
                            <Pill>
                              <Star className="mr-1 h-3 w-3" aria-hidden="true" />
                              Padrão
                            </Pill>
                          )}
                        </div>
                        <h2 className="font-display text-lg leading-tight tracking-tightest text-ink">
                          {p.name}
                        </h2>
                      </div>
                      <ArrowRight className="h-4 w-4 text-ink-dim transition group-hover:translate-x-0.5 group-hover:text-ink" />
                    </div>
                  </Card>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </QueryBoundary>
    </div>
  )
}
