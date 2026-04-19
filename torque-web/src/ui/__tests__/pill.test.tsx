import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Pill } from '../pill'

describe('Pill', () => {
  it('renders idle', () => {
    const { container } = render(<Pill>Todos · 24</Pill>)
    expect(container).toMatchSnapshot()
  })

  it('renders active', () => {
    const { container } = render(<Pill active>Hot</Pill>)
    expect(container).toMatchSnapshot()
  })
})
