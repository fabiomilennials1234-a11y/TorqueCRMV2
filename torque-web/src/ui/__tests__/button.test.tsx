import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Button } from '../button'

describe('Button', () => {
  it('renders default (secondary/md)', () => {
    const { container } = render(<Button>Save</Button>)
    expect(container).toMatchSnapshot()
  })

  it('renders primary variant', () => {
    const { container } = render(<Button variant="primary">Save</Button>)
    expect(container).toMatchSnapshot()
  })

  it('renders ghost icon size', () => {
    const { container } = render(<Button variant="ghost" size="icon" aria-label="settings" />)
    expect(container).toMatchSnapshot()
  })

  it('renders disabled state', () => {
    const { container } = render(<Button disabled>Save</Button>)
    expect(container).toMatchSnapshot()
  })

  it('renders all sizes', () => {
    const sizes = ['xs', 'sm', 'md', 'lg'] as const
    const { container } = render(
      <div>
        {sizes.map((s) => (
          <Button key={s} size={s}>
            {s}
          </Button>
        ))}
      </div>
    )
    expect(container).toMatchSnapshot()
  })
})
