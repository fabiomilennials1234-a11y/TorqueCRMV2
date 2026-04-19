import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ChannelBadge } from '../channel-badge'

describe('ChannelBadge', () => {
  it('renders icon-only for each channel', () => {
    const channels = ['whatsapp', 'messenger', 'instagram', 'sz_chat'] as const
    const { container } = render(
      <div>
        {channels.map((c) => (
          <ChannelBadge key={c} channel={c} />
        ))}
      </div>
    )
    expect(container).toMatchSnapshot()
  })

  it('renders labeled variants', () => {
    const channels = ['whatsapp', 'messenger', 'instagram', 'sz_chat'] as const
    const { container } = render(
      <div>
        {channels.map((c) => (
          <ChannelBadge key={c} channel={c} variant="labeled" />
        ))}
      </div>
    )
    expect(container).toMatchSnapshot()
  })

  it('renders dot + muted', () => {
    const { container } = render(
      <div>
        <ChannelBadge channel="whatsapp" variant="dot" />
        <ChannelBadge channel="whatsapp" muted />
      </div>
    )
    expect(container).toMatchSnapshot()
  })
})
