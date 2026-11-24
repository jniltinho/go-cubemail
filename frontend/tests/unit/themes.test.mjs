// Theme tests: structure and role types of the theme files, color lint of the components, registry and CSS in
// step, dark themes complete, the WCAG contrast matrix and the helpers that deal with content colors.
import { join } from 'node:path'
import { describe, expect, test } from 'vite-plus/test'
import {
  SRC,
  colorOf,
  contract,
  contrastOf,
  files,
  lint,
  lintSources,
  luminance,
  parseColor,
  read,
  registry,
  themeFile,
  typeErrors,
} from './themelib.mjs'
import { inkOn } from '../../src/utils/helpers.ts'

const { values } = contract()
const known = new Set(values.keys())
const themeFiles = files(join(SRC, 'themes')).filter((f) => f.endsWith('.css'))
const blocks = new Map(themeFiles.flatMap((f) => [...themeFile(read(f), known).blocks]))
const merged = (id) => new Map([...values, ...(blocks.get(id) || [])])
const themes = registry()
const matrix = JSON.parse(read(join(SRC, 'themes/contrast.json')))

// Colors that are content, not interface (design D10). An entry that matches nothing fails the test below.
const ALLOW = [
  {
    file: 'components/editor/RichTextToolbar.vue',
    pattern: /^\s*'#[0-9a-fA-F]{6}',?$/,
    reason: 'text-color and highlight swatches: colors the user applies to the document',
  },
  {
    file: 'components/ComposerModal.vue',
    pattern: /<(p|blockquote) style="/,
    reason: 'quote markup sent in the reply: the recipient has none of our roles, so the colors are literal',
  },
  {
    file: 'utils/helpers.ts',
    pattern: /DEFAULT_CAL_COLOR = |'#000000'|'#ffffff'/,
    reason: 'calendar colors are user data; the ink over them is black or white by contrast, not by theme',
  },
]

describe('theme files', () => {
  test('only role declarations under :root[data-theme]', () => {
    for (const f of themeFiles) expect(themeFile(read(f), known).errors, f).toEqual([])
  })
  test('the structure rule catches component rules, unknown roles and plain properties', () => {
    const bad = themeFile(
      `.mail-row { color: red } :root[data-theme='a'] { --color-nope: 1; border-radius: 2px; --color-fg: #000 }`,
      known,
    )
    expect(bad.errors.length).toBe(3)
    expect(bad.blocks.get('a').get('--color-fg')).toBe('#000')
  })
  test('every role of every theme has the right type', () => {
    expect(typeErrors(values), 'contract').toEqual([])
    for (const { id } of themes) expect(typeErrors(merged(id)), id).toEqual([])
  })
  test('role types catch a gradient in a color role, a color in a radius role and a bad shadow', () => {
    const bad = new Map([
      ...values,
      ['--color-fg', 'linear-gradient(red, blue)'],
      ['--radius-control', '#fff'],
      ['--shadow-menu', 'red'],
    ])
    expect(typeErrors(bad).map((e) => e.split(':')[0])).toEqual(
      expect.arrayContaining(['--color-fg', '--radius-control', '--shadow-menu']),
    )
  })
})

describe('components use roles only', () => {
  const used = new Set()
  const findings = lintSources(ALLOW, used)
  test('no literal color, palette utility, radius or shadow outside the contract', () => {
    expect(findings).toEqual([])
  })
  test('no stale allowlist entry', () => {
    expect(ALLOW.filter((a) => !used.has(a)).map((a) => a.file)).toEqual([])
  })
  test('the lint catches each kind of leak', () => {
    const rules = (text) => lint(text).map((h) => h.rule)
    expect(rules('<div class="bg-white text-[#1B3A6B]">')).toEqual(['literal color', 'palette utility'])
    expect(rules('<p class="text-red-500 hover:bg-gray-50/40">')).toEqual(['palette utility', 'palette utility'])
    expect(rules('<p class="rounded-lg shadow-md rounded-control">')).toEqual(['radius utility', 'shadow utility'])
    expect(rules('.x { border-radius: 4px; color: rgba(0, 0, 0, 0.5) }')).toEqual(['literal color', 'literal radius'])
    expect(rules('.x { border-radius: var(--radius-row); color: var(--accent-2) }')).toEqual(['old token'])
    expect(rules('<p class="rounded-surface sh-menu text-fg bg-surface/80">')).toEqual([])
  })
  test('the quote markup of a reply references no role', () => {
    const quote = read(join(SRC, 'components/ComposerModal.vue'))
      .split('\n')
      .filter((l) => /<(p|blockquote) style="/.test(l))
    expect(quote.length).toBeGreaterThan(0)
    for (const l of quote) expect(l).not.toMatch(/var\(--/)
  })
})

describe('registry', () => {
  test('the registry is read from internal/theme/theme.go', () => {
    expect(themes.map((t) => t.id)).toEqual([
      'larry',
      'outlook',
      'outlook-dark',
      'apple-mail',
      'apple-mail-dark',
      'breeze',
      'breeze-dark',
      'graphite',
      'high-contrast',
    ])
  })
  test('every registered theme but the first has one block, and every block is registered', () => {
    expect(new Set(blocks.keys())).toEqual(new Set(themes.slice(1).map((t) => t.id)))
    expect(blocks.has(themes[0].id), 'the default theme is the contract itself').toBe(false)
  })
  test("each theme's scheme matches the lightness of its page", () => {
    for (const { id, scheme } of themes) {
      const roles = merged(id)
      const dark = luminance(colorOf('surface', roles)) < luminance(colorOf('fg', roles))
      expect(dark ? 'dark' : 'light', id).toBe(scheme)
    }
  })
  test('a dark theme declares every color role', () => {
    const colorRoles = [...known].filter((p) => p.startsWith('--color-'))
    for (const { id, scheme } of themes) {
      if (scheme !== 'dark') continue
      expect(
        colorRoles.filter((p) => !blocks.get(id)?.has(p)),
        id,
      ).toEqual([])
    }
  })
  test('every theme keeps the message paper light', () => {
    for (const { id } of themes) {
      const roles = merged(id)
      expect(luminance(colorOf('paper', roles)), id).toBeGreaterThan(0.8)
      expect(luminance(colorOf('on-paper', roles)), id).toBeLessThan(0.1)
    }
  })
})

describe('contrast', () => {
  const need = (size, aaa) => (size === 'normal' ? (aaa ? 7 : 4.5) : size === 'large' ? (aaa ? 4.5 : 3) : 3)
  const failures = (id, roles = merged(id)) => {
    const aaa = matrix.aaa.includes(id)
    return matrix.recipes
      .filter((r) => aaa || !r.aaaOnly)
      .map((r) => ({ r, got: contrastOf(r, roles) }))
      .filter(({ r, got }) => got < need(r.size, aaa))
      .map(({ r, got }) => `${id} ${r.id}: ${got.toFixed(2)}:1, needs ${need(r.size, aaa)}:1`)
  }
  test('every theme meets every recipe', () => {
    expect(themes.flatMap(({ id }) => failures(id))).toEqual([])
  })
  test('a failing pairing is reported with theme, recipe and ratio', () => {
    const roles = new Map([...values, ['--color-fg-muted', '#999999']])
    expect(failures('larry', roles)).toContain('larry muted/surface: 2.85:1, needs 4.5:1')
  })
  test('translucent colors are composited over their backgrounds', () => {
    const roles = new Map([
      ['--color-a', 'rgba(0, 0, 0, 0.5)'],
      ['--color-b', '#ffffff'],
    ])
    expect(contrastOf({ fg: 'a', bg: ['b'] }, roles)).toBeCloseTo(3.98, 1)
    expect(parseColor('rgb(255 0 0 / 50%)')).toEqual([255, 0, 0, 0.5])
    expect(parseColor('linear-gradient(red, blue)')).toBe(null)
  })
})

describe('ink over user colors', () => {
  test('black on light calendar colors, white on dark ones', () => {
    expect(inkOn('#ffffff')).toBe('#000000')
    expect(inkOn('#fff9c4')).toBe('#000000') // pale yellow
    expect(inkOn('#1B3A6B')).toBe('#ffffff') // navy
    expect(inkOn('#3788d8')).toBe('#000000') // the default calendar blue: 5.6:1 with black, 3.7:1 with white
    expect(inkOn('not a color')).toBe('#ffffff')
  })
})
