import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Sparkline } from '../spark'

describe('Sparkline', () => {
  it('renders default', () => {
    const { container } = render(<Sparkline data={[1, 3, 2, 5, 4, 6, 8]} />)
    expect(container).toMatchSnapshot()
  })

  it('renders custom size', () => {
    const { container } = render(<Sparkline data={[10, 20, 15]} width={80} height={20} />)
    expect(container).toMatchSnapshot()
  })

  it('returns null with < 2 points', () => {
    const { container } = render(<Sparkline data={[5]} />)
    expect(container.firstChild).toBeNull()
  })
})
