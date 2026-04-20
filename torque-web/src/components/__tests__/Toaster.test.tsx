import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Toaster } from '@/components/Toaster'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

function emit(detail: { level?: string; message: string; code?: string }) {
  window.dispatchEvent(new CustomEvent('torque:toast', { detail }))
}

describe('Toaster', () => {
  it('renders a toast dispatched via CustomEvent and role=alert on error', () => {
    render(<Toaster />)
    act(() => emit({ level: 'error', message: 'Permissão negada.', code: 'PERMISSION_DENIED' }))
    const node = screen.getByRole('alert')
    expect(node).toHaveTextContent('Permissão negada.')
  })

  it('auto-dismisses after 5 seconds', () => {
    render(<Toaster />)
    act(() => emit({ level: 'info', message: 'Olá.' }))
    expect(screen.getByRole('status')).toHaveTextContent('Olá.')
    act(() => vi.advanceTimersByTime(5001))
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('caps the queue at MAX_TOASTS (4) — oldest is evicted', () => {
    render(<Toaster />)
    act(() => {
      for (let i = 0; i < 6; i++) emit({ level: 'info', message: `msg ${i}` })
    })
    const visible = screen.getAllByRole('status')
    expect(visible).toHaveLength(4)
    expect(visible[0]).toHaveTextContent('msg 2')
    expect(visible[3]).toHaveTextContent('msg 5')
  })
})
