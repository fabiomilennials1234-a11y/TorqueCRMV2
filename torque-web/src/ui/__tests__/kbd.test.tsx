import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Kbd } from '../kbd'

describe('Kbd', () => {
  it('renders single key', () => {
    const { container } = render(<Kbd>N</Kbd>)
    expect(container).toMatchSnapshot()
  })

  it('renders chord', () => {
    const { container } = render(
      <>
        <Kbd>⌘</Kbd>
        <Kbd>K</Kbd>
      </>
    )
    expect(container).toMatchSnapshot()
  })
})
