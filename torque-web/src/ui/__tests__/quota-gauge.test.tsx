import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { QuotaGauge } from '../quota-gauge'

describe('QuotaGauge', () => {
  it('renders bar 0%', () => {
    const { container } = render(<QuotaGauge current={0} limit={100} label="Leads" />)
    expect(container).toMatchSnapshot()
  })

  it('renders bar 50%', () => {
    const { container } = render(<QuotaGauge current={50} limit={100} label="Leads" />)
    expect(container).toMatchSnapshot()
  })

  it('renders bar 100%', () => {
    const { container } = render(<QuotaGauge current={100} limit={100} label="Leads" />)
    expect(container).toMatchSnapshot()
  })

  it('renders ring variant at 72%', () => {
    const { container } = render(<QuotaGauge current={72} limit={100} variant="ring" />)
    expect(container).toMatchSnapshot()
  })

  it('renders inline locked', () => {
    const { container } = render(
      <QuotaGauge current={8} limit={10} variant="inline" canAdd={false} />
    )
    expect(container).toMatchSnapshot()
  })
})
