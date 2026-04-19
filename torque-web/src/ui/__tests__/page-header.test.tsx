import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PageHeader } from '../page-header'

describe('PageHeader', () => {
  it('renders title only', () => {
    const { container } = render(<PageHeader title="Funis" />)
    expect(container).toMatchSnapshot()
  })

  it('renders with eyebrow + description + actions', () => {
    const { container } = render(
      <PageHeader
        eyebrow="Visão geral"
        title="Operação"
        description="Resumo em tempo real"
        actions={<button type="button">Atualizar</button>}
      />
    )
    expect(container).toMatchSnapshot()
  })
})
