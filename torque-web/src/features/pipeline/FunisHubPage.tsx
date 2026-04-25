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
        <div className="text-2xs text-ink-muted mb-2 inline-flex items-center gap-2 font-medium tracking-[0.16em] uppercase">
          <Layers className="h-3.5 w-3.5" />
          Funis
        </div>
        <h1 className="font-display tracking-tightest text-ink text-[2rem] leading-[1.1]">
          Escolha um funil para começar
        </h1>
        <p className="text-ink-muted mt-2 max-w-xl text-sm">
          Cada funil é um pipeline independente: WhatsApp para qualificar, confirmação para
          reuniões, propostas para fechamento. Mova leads entre etapas e o histórico fica
          registrado.
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
            <p className="text-ink-muted text-sm">
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
                  className="group shadow-hairline block rounded-md transition hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim)/0.5)]"
                >
                  <Card className="h-full p-5">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="text-2xs text-ink-dim mb-2 inline-flex items-center gap-2 tracking-[0.14em] uppercase">
                          {p.kind}
                          {p.is_default && (
                            <Pill>
                              <Star className="mr-1 h-3 w-3" aria-hidden="true" />
                              Padrão
                            </Pill>
                          )}
                        </div>
                        <h2 className="font-display tracking-tightest text-ink text-lg leading-tight">
                          {p.name}
                        </h2>
                      </div>
                      <ArrowRight className="text-ink-dim group-hover:text-ink h-4 w-4 transition group-hover:translate-x-0.5" />
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
