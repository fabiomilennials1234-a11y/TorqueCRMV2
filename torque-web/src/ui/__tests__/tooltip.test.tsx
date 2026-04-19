import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Tooltip, TooltipProvider } from '../tooltip'

describe('Tooltip', () => {
  it('renders trigger inside provider (content only on interaction)', () => {
    const { container } = render(
      <TooltipProvider>
        <Tooltip content="Atalho">
          <button type="button">A</button>
        </Tooltip>
      </TooltipProvider>
    )
    expect(container).toMatchSnapshot()
  })

  it('renders trigger with shortcut side=right', () => {
    const { container } = render(
      <TooltipProvider>
        <Tooltip content="Novo lead" shortcut="N" side="right">
          <button type="button">+</button>
        </Tooltip>
      </TooltipProvider>
    )
    expect(container).toMatchSnapshot()
  })
})
