import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Badge } from '../badge'

describe('Badge', () => {
  it('renders default', () => {
    const { container } = render(<Badge>Novo</Badge>)
    expect(container).toMatchSnapshot()
  })

  it('renders all tones', () => {
    const tones = ['neutral', 'accent', 'info', 'success', 'warning', 'danger'] as const
    const { container } = render(
      <div>
        {tones.map((t) => (
          <Badge key={t} tone={t}>
            {t}
          </Badge>
        ))}
      </div>
    )
    expect(container).toMatchSnapshot()
  })
})
