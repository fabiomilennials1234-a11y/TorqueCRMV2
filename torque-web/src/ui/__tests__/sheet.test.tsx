import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Sheet, SheetContent, SheetTrigger } from '../sheet'

describe('Sheet', () => {
  it('renders closed trigger', () => {
    const { container } = render(
      <Sheet>
        <SheetTrigger asChild>
          <button type="button">Abrir</button>
        </SheetTrigger>
      </Sheet>
    )
    expect(container).toMatchSnapshot()
  })

  it('renders open with title + description', () => {
    const { baseElement } = render(
      <Sheet open>
        <SheetContent title="Lead" description="Detalhes do lead">
          corpo do sheet
        </SheetContent>
      </Sheet>
    )
    expect(baseElement).toMatchSnapshot()
  })
})
