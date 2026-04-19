import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ScoreMeter } from '../score-meter'

describe('ScoreMeter', () => {
  it('renders cold band (value < 30)', () => {
    const { container } = render(<ScoreMeter value={18} />)
    expect(container).toMatchSnapshot()
  })

  it('renders warm band', () => {
    const { container } = render(<ScoreMeter value={55} />)
    expect(container).toMatchSnapshot()
  })

  it('renders accent band', () => {
    const { container } = render(<ScoreMeter value={72} />)
    expect(container).toMatchSnapshot()
  })

  it('renders hot band (>=85)', () => {
    const { container } = render(<ScoreMeter value={91} />)
    expect(container).toMatchSnapshot()
  })

  it('clamps above 100', () => {
    const { container } = render(<ScoreMeter value={200} showLabel={false} />)
    expect(container).toMatchSnapshot()
  })
})
