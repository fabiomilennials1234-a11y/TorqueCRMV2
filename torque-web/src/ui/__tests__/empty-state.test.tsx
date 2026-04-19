import { render } from '@testing-library/react'
import { Inbox } from 'lucide-react'
import { describe, expect, it } from 'vitest'
import { EmptyState } from '../empty-state'

describe('EmptyState', () => {
  it('renders title only', () => {
    const { container } = render(<EmptyState title="Nenhum lead" />)
    expect(container).toMatchSnapshot()
  })

  it('renders with icon + description + action', () => {
    const { container } = render(
      <EmptyState
        icon={Inbox}
        title="Inbox vazia"
        description="Você respondeu todas as conversas — hora de prospectar."
        action={<button type="button">Criar lead</button>}
      />
    )
    expect(container).toMatchSnapshot()
  })
})
