import { describe, expect, it } from 'vitest'
import { maskApiKey } from '@/utils/maskApiKey'

describe('maskApiKey', () => {
  it('abbreviates long keys to a stable prefix and suffix', () => {
    expect(maskApiKey('sk-abcdefghijklmnopqrstuvwxyz0123456789')).toBe('sk-abc...6789')
  })

  it('keeps short keys recognizable without exposing the full value', () => {
    expect(maskApiKey('sk-short')).toBe('sk-s***')
  })

  it('handles empty values', () => {
    expect(maskApiKey('')).toBe('')
  })
})
