import { describe, expect, it } from 'vitest'

import { cn, formatCompact, formatCurrency, formatRelative, initials } from '@/lib/utils'

describe('lib/utils', () => {
  describe('cn', () => {
    it('merges clsx-style inputs', () => {
      expect(cn('a', 'b')).toContain('a')
      expect(cn('a', 'b')).toContain('b')
    })

    it('dedupes conflicting tailwind classes via twMerge', () => {
      expect(cn('p-2', 'p-4')).toBe('p-4')
    })

    it('drops falsy values', () => {
      const result = cn('a', false && 'b', null, undefined, 'c')
      expect(result).toContain('a')
      expect(result).toContain('c')
      expect(result).not.toContain('b')
    })
  })

  describe('initials', () => {
    it('takes first letter of up to `max` words uppercased', () => {
      expect(initials('Ana Julia')).toBe('AJ')
      expect(initials('Marco Antonio Silva')).toBe('MA')
      expect(initials('Marco Antonio Silva', 3)).toBe('MAS')
    })

    it('collapses extra whitespace', () => {
      expect(initials('  Ana   Julia  ')).toBe('AJ')
    })

    it('handles empty or whitespace-only input', () => {
      expect(initials('')).toBe('')
      expect(initials('   ')).toBe('')
    })
  })

  describe('formatRelative', () => {
    const anchor = new Date('2026-04-20T12:00:00Z')

    it('returns "agora" under 60s', () => {
      expect(formatRelative(new Date(anchor.getTime() - 30 * 1000), anchor)).toBe('agora')
    })

    it('returns minutes up to 1h', () => {
      expect(formatRelative(new Date(anchor.getTime() - 30 * 60 * 1000), anchor)).toBe('30m')
    })

    it('returns hours up to 24h', () => {
      expect(formatRelative(new Date(anchor.getTime() - 5 * 3600 * 1000), anchor)).toBe('5h')
    })

    it('returns days up to 7', () => {
      expect(formatRelative(new Date(anchor.getTime() - 3 * 86400 * 1000), anchor)).toBe('3d')
    })

    it('falls back to locale date past a week', () => {
      const old = new Date('2026-03-01T12:00:00Z')
      const out = formatRelative(old, anchor)
      // Locale format includes day digits; "01" always present.
      expect(out).toMatch(/\d{2}/)
    })
  })

  describe('formatCurrency', () => {
    it('formats BRL without decimals by default', () => {
      const out = formatCurrency(150000)
      expect(out).toMatch(/R\$/)
      expect(out).toMatch(/150\.000/)
    })

    it('accepts override currency', () => {
      const out = formatCurrency(1234, 'USD')
      expect(out).toMatch(/US\$|\$/)
    })
  })

  describe('formatCompact', () => {
    it('renders 1.5k for 1500', () => {
      const out = formatCompact(1500)
      expect(out.toLowerCase()).toMatch(/mil|k/)
    })

    it('keeps small numbers intact', () => {
      expect(formatCompact(42)).toBe('42')
    })
  })
})
