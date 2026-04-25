import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { Quota } from '@/hooks/useQuotas'
import { QuotaMeter } from '@/ui/quota-meter'

function q(partial: Partial<Quota>): Quota {
  return {
    resource: 'leads',
    plan_base: 100,
    purchased_addons: 0,
    admin_adjustment: 0,
    current_usage: 50,
    effective_limit: 100,
    remaining: 50,
    ...partial,
  }
}

describe('QuotaMeter', () => {
  it('renders nothing when quota is undefined', () => {
    const { container } = render(<QuotaMeter quota={undefined} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders nothing when effective_limit <= 0 (unconfigured)', () => {
    const { container } = render(<QuotaMeter quota={q({ effective_limit: 0, remaining: 0 })} />)
    expect(container.firstChild).toBeNull()
  })

  it('shows percentage + current / limit copy', () => {
    render(<QuotaMeter quota={q({ current_usage: 42, effective_limit: 100, remaining: 58 })} />)
    expect(screen.getByText('42%')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
    expect(screen.getByText(/\/ 100/)).toBeInTheDocument()
  })

  it('clamps percentage to 100 when over cap', () => {
    render(<QuotaMeter quota={q({ current_usage: 120, effective_limit: 100, remaining: 0 })} />)
    expect(screen.getByText('100%')).toBeInTheDocument()
  })

  it('exposes ARIA meter role + values', () => {
    render(<QuotaMeter quota={q({})} label="Leads do plano" />)
    const meter = screen.getByRole('meter', { name: 'Leads do plano' })
    expect(meter).toHaveAttribute('aria-valuemin', '0')
    expect(meter).toHaveAttribute('aria-valuemax', '100')
    expect(meter).toHaveAttribute('aria-valuenow', '50')
  })

  it('uses a custom label when provided', () => {
    render(<QuotaMeter quota={q({})} label="Meu limite" />)
    expect(screen.getByText(/Meu limite/)).toBeInTheDocument()
  })
})
