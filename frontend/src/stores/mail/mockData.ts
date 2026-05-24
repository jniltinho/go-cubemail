/**
 * @file mockData.ts
 * @description Mock data structures for demonstration and fallback purposes.
 */

import type { CalEvent } from '../../types'

/**
 * Mock calendar events, keyed by day of the month.
 */
export const CAL_EVENTS: Record<number, CalEvent[]> = {
  4:  [{ t: 'Sprint kickoff', k: 'alt' }],
  6:  [{ t: 'Design review' }],
  7:  [{ t: '1:1 Helena', k: 'ghost' }],
  11: [{ t: 'All-hands' }, { t: 'Vendor call', k: 'alt' }],
  13: [{ t: 'Q3 budget prep' }],
  15: [{ t: 'Offsite planning', k: 'warn' }],
  20: [{ t: 'Standup → Atrium 2', k: 'alt' }],
  21: [{ t: '1:1 Helena' }, { t: 'Q3 budget review', k: 'warn' }],
  22: [{ t: 'Legal redlines', k: 'ghost' }],
  25: [{ t: 'Holiday — office closed', k: 'ghost' }],
  27: [{ t: 'Roadmap sync', k: 'alt' }],
  29: [{ t: 'Newsletter ships' }],
}
