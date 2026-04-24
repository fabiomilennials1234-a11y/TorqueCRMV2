import { describe, expect, it } from 'vitest'

import { queryKeys } from '../queryKeys'

describe('queryKeys', () => {
  it('returns stable root keys', () => {
    expect(queryKeys.leads.all()).toEqual(['leads'])
    expect(queryKeys.operations.all()).toEqual(['operations'])
  })

  it('list keys include filters only when provided', () => {
    expect(queryKeys.leads.list()).toEqual(['leads', 'list'])
    expect(queryKeys.leads.list({ stage: 'qualif' })).toEqual([
      'leads',
      'list',
      { stage: 'qualif' },
    ])
  })

  it('detail keys scope by id', () => {
    expect(queryKeys.leads.detail('abc')).toEqual(['leads', 'detail', 'abc'])
    expect(queryKeys.operations.detail('op-1')).toEqual(['operations', 'detail', 'op-1'])
  })

  it('pipe entries nest under pipe id', () => {
    expect(queryKeys.pipes.entries('p1')).toEqual(['pipes', 'p1', 'entries'])
    expect(queryKeys.pipes.entries('p1', { stage: 'new' })).toEqual([
      'pipes',
      'p1',
      'entries',
      { stage: 'new' },
    ])
  })

  it('detail and list keys never collide', () => {
    // Same-domain disambiguation via the second segment.
    expect(queryKeys.leads.list()).not.toEqual(queryKeys.leads.detail('list'))
  })
})
