import { describe, it, expect } from 'vitest'
import { filterListings, sampleListings } from './market'

describe('market filtering', () => {
  it('filters by text search', () => {
    const out = filterListings(sampleListings, 'bike')
    expect(out.length).toBe(1)
    expect(out[0].title).toContain('Bike')
  })
})
