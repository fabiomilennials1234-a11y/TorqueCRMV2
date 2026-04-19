import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Input } from '../input'

describe('Input', () => {
  it('renders default', () => {
    const { container } = render(<Input placeholder="email" />)
    expect(container).toMatchSnapshot()
  })

  it('renders disabled', () => {
    const { container } = render(<Input placeholder="email" disabled />)
    expect(container).toMatchSnapshot()
  })

  it('renders email type with value', () => {
    const { container } = render(<Input type="email" defaultValue="a@b.co" />)
    expect(container).toMatchSnapshot()
  })
})
