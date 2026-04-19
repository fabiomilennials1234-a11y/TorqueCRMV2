import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Separator } from '../separator'

describe('Separator', () => {
  it('renders horizontal', () => {
    const { container } = render(<Separator />)
    expect(container).toMatchSnapshot()
  })

  it('renders vertical', () => {
    const { container } = render(<Separator orientation="vertical" />)
    expect(container).toMatchSnapshot()
  })
})
