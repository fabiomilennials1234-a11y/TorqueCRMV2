import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { AgentsPage } from '@/features/copilot/AgentsPage'

/**
 * AgentsPage hoje consome fixture (src/lib/seed) — o wiring real contra
 * /api/v1/agents chega na Fase C (S37). Este teste trava o shell
 * (PageHeader + botão "Novo agente") para pegar qualquer regressão
 * visual durante a Fase C.
 */
describe('AgentsPage', () => {
  it('renders the copilot hub header + primary actions', () => {
    render(<AgentsPage />)
    expect(screen.getByText('Time de agentes')).toBeDefined()
    // "Novo agente" appears once in header + once per agent card in the
    // seed fixture — assert at least one; more-than-one is acceptable.
    expect(screen.getAllByText('Novo agente').length).toBeGreaterThan(0)
    expect(screen.getByText('Biblioteca de templates')).toBeDefined()
  })
})
