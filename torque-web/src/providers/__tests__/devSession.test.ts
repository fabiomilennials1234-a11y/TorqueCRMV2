import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { DEV_SESSION, isDevAuthEnabled } from '@/providers/devSession'

describe('devSession', () => {
  beforeEach(() => {
    // `DEV` is typed as boolean in ImportMetaEnv — pass boolean.
    vi.stubEnv('DEV', true)
  })
  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('returns true when DEV and VITE_DEV_AUTH are both set', () => {
    vi.stubEnv('VITE_DEV_AUTH', '1')
    expect(isDevAuthEnabled()).toBe(true)
    vi.stubEnv('VITE_DEV_AUTH', 'true')
    expect(isDevAuthEnabled()).toBe(true)
  })

  it('returns false when VITE_DEV_AUTH is unset or off', () => {
    vi.stubEnv('VITE_DEV_AUTH', '')
    expect(isDevAuthEnabled()).toBe(false)
    vi.stubEnv('VITE_DEV_AUTH', '0')
    expect(isDevAuthEnabled()).toBe(false)
    vi.stubEnv('VITE_DEV_AUTH', 'false')
    expect(isDevAuthEnabled()).toBe(false)
  })

  it('returns false in production builds regardless of VITE_DEV_AUTH', () => {
    vi.stubEnv('DEV', false)
    vi.stubEnv('VITE_DEV_AUTH', '1')
    expect(isDevAuthEnabled()).toBe(false)
  })

  it('DEV_SESSION grants every declared feature permission', () => {
    for (const [k, v] of Object.entries(DEV_SESSION.featurePermissions)) {
      expect(v, `permission ${k} should be true`).toBe(true)
    }
    expect(DEV_SESSION.role).toBe('admin')
    expect(DEV_SESSION.user.email).toBe('marcelo@gmail.com')
  })
})
