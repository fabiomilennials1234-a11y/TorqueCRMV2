import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Avatar } from '../avatar'

describe('Avatar', () => {
  it('renders with fallback initials', () => {
    const { container } = render(<Avatar fallback="FM" />)
    expect(container).toMatchSnapshot()
  })

  it('renders large size with ring', () => {
    const { container } = render(<Avatar fallback="RB" size="lg" ring />)
    expect(container).toMatchSnapshot()
  })
})
