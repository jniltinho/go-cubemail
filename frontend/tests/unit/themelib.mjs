// Helpers of the theme tests: the role contract of style.css, the theme files, the role types, the color lint of
// the components and the WCAG contrast of the roles. Adapted from ad-manager's tests/unit/themelib.mjs.
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import postcss from 'postcss'

export const SRC = fileURLToPath(new URL('../../src/', import.meta.url))
export const ROOT = fileURLToPath(new URL('../../../', import.meta.url))
export const read = (p) => readFileSync(p, 'utf8')

export function files(dir = SRC, out = []) {
  for (const f of readdirSync(dir)) {
    const p = join(dir, f)
    if (statSync(p).isDirectory()) files(p, out)
    else if (/\.(vue|css|ts)$/.test(f)) out.push(p)
  }
  return out
}

// ── contract ────────────────────────────────────────────────────────────────────────────────────────────────
// The contract is every custom property declared in styles/contract.css inside the @theme block or the :root rule.
export const CONTRACT = 'styles/contract.css'
export function contract(css = read(join(SRC, CONTRACT))) {
  const root = postcss.parse(css)
  const values = new Map()
  const nodes = new Set()
  root.each((n) => {
    const isTheme = n.type === 'atrule' && n.name === 'theme'
    const isParams = n.type === 'rule' && n.selector === ':root'
    if (!isTheme && !isParams) return
    nodes.add(n)
    n.walkDecls((d) => d.prop.startsWith('--') && values.set(d.prop, d.value.trim()))
  })
  return { values, nodes, root }
}

// ── theme files ─────────────────────────────────────────────────────────────────────────────────────────────
const THEME_SELECTOR = /^:root\[data-theme=(['"])([a-z0-9-]+)\1\]$/

// themeFile parses one themes/*.css: blocks by theme id, and the structure errors (anything that is not a
// :root[data-theme='id'] rule declaring known roles).
export function themeFile(css, known) {
  const blocks = new Map()
  const errors = []
  postcss.parse(css).each((n) => {
    if (n.type === 'comment') return
    const m = n.type === 'rule' && n.selector.match(THEME_SELECTOR)
    if (!m)
      return errors.push(`line ${n.source.start.line}: only :root[data-theme='id'] { --role: value } rules are allowed`)
    if (blocks.has(m[2])) errors.push(`line ${n.source.start.line}: theme ${m[2]} has more than one block`)
    const decls = blocks.get(m[2]) || new Map()
    n.each((d) => {
      if (d.type === 'comment') return
      if (d.type !== 'decl' || !d.prop.startsWith('--'))
        return errors.push(`line ${d.source.start.line}: ${d.prop || d.type} is not a role`)
      if (!known.has(d.prop)) return errors.push(`line ${d.source.start.line}: unknown role ${d.prop}`)
      decls.set(d.prop, d.value.trim())
    })
    blocks.set(m[2], decls)
  })
  return { blocks, errors }
}

// ── colors ──────────────────────────────────────────────────────────────────────────────────────────────────
const NAMED = { white: [255, 255, 255, 1], black: [0, 0, 0, 1], transparent: [0, 0, 0, 0] }

// resolve replaces var(--role) by the role's value in `roles` (a Map), following chains.
export function resolve(value, roles, depth = 0) {
  if (depth > 8) throw new Error(`var() chain too deep in ${value}`)
  return value.replace(/var\((--[a-z0-9-]+)\)/g, (_, name) => {
    if (!roles.has(name)) throw new Error(`unknown role ${name}`)
    return resolve(roles.get(name), roles, depth + 1)
  })
}

// parseColor returns [r, g, b, a] (0-255, alpha 0-1) or null when the text is not one color.
export function parseColor(text) {
  const v = text.trim().toLowerCase()
  if (NAMED[v]) return NAMED[v]
  let m = v.match(/^#([0-9a-f]{3,8})$/)
  if (m && [3, 4, 6, 8].includes(m[1].length)) {
    const h = m[1].length <= 4 ? m[1].replace(/./g, '$&$&') : m[1]
    const n = (i) => parseInt(h.slice(i, i + 2), 16)
    return [n(0), n(2), n(4), h.length === 8 ? n(6) / 255 : 1]
  }
  m = v.match(/^rgba?\(\s*([\d.]+)[\s,]+([\d.]+)[\s,]+([\d.]+)(?:\s*[,/]\s*([\d.]+%?))?\s*\)$/)
  if (m) {
    const a = m[4] === undefined ? 1 : m[4].endsWith('%') ? parseFloat(m[4]) / 100 : parseFloat(m[4])
    return [Number(m[1]), Number(m[2]), Number(m[3]), a]
  }
  return null
}

// over composites a color with alpha over an opaque one.
export const over = ([r, g, b, a], [R, G, B]) => [r * a + R * (1 - a), g * a + G * (1 - a), b * a + B * (1 - a), 1]

export function luminance([r, g, b]) {
  const lin = (c) => (c / 255 <= 0.04045 ? c / 255 / 12.92 : ((c / 255 + 0.055) / 1.055) ** 2.4)
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
}

export function ratio(a, b) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

// colorOf resolves a role to a color; throws when the role is missing or is not a color.
export function colorOf(name, roles) {
  const prop = name.startsWith('--') ? name : `--color-${name}`
  if (!roles.has(prop)) throw new Error(`unknown role ${prop}`)
  const c = parseColor(resolve(roles.get(prop), roles))
  if (!c) throw new Error(`${prop} is not a color: ${roles.get(prop)}`)
  return c
}

// contrastOf measures a recipe {fg, bg: [top … bottom]}: the backgrounds are composited bottom-up (the bottom one
// over white if it is translucent), then the text over the result.
export function contrastOf(recipe, roles) {
  let back = [255, 255, 255, 1]
  for (const name of [...recipe.bg].reverse()) back = over(colorOf(name, roles), back)
  return ratio(over(colorOf(recipe.fg, roles), back), back)
}

// ── role types ──────────────────────────────────────────────────────────────────────────────────────────────
const LENGTH = /^(0|-?[\d.]+(px|rem|em))$/
const SHADOW_PART = /^(inset\s+)?(-?[\d.]+(px)?\s+){2,4}\S.*$|^(-?[\d.]+(px)?\s+){2,4}\S.*\s+inset$/
// splitTop splits on commas that are not inside parentheses.
export function splitTop(value) {
  const out = []
  let depth = 0
  let cur = ''
  for (const ch of value) {
    if (ch === '(') depth++
    if (ch === ')') depth--
    if (ch === ',' && depth === 0) {
      out.push(cur.trim())
      cur = ''
    } else cur += ch
  }
  out.push(cur.trim())
  return out
}

function shadowError(value) {
  if (value === 'none') return null
  for (const part of splitTop(value)) {
    if (!SHADOW_PART.test(part)) return `not a shadow: ${part}`
    const color = part.replace(/^inset\s+|\s+inset$/g, '').replace(/^(-?[\d.]+(px)?\s+){2,4}/, '')
    if (!parseColor(color)) return `shadow color is not a color: ${color}`
  }
  return null
}

// typeErrors checks every role of a merged theme: a color role holds a color, a shadow role a shadow, a radius or
// size role a length, a background role gradients or none, a font role a font list.
export function typeErrors(roles) {
  const errors = []
  for (const [prop, raw] of roles) {
    let value
    try {
      value = resolve(raw, roles)
    } catch (e) {
      errors.push(`${prop}: ${e.message}`)
      continue
    }
    let err = null
    if (prop.startsWith('--color-')) err = parseColor(value) ? null : `not a color: ${value}`
    else if (prop.startsWith('--shadow-')) err = shadowError(value)
    else if (prop.startsWith('--radius-') || prop === '--row-inset-x' || prop === '--selection-mark-w')
      err = LENGTH.test(value) ? null : `not a length: ${value}`
    else if (prop.startsWith('--bg-'))
      err =
        value === 'none' || splitTop(value).every((p) => /^(repeating-)?(linear|radial)-gradient\(.+\)$/s.test(p))
          ? null
          : `not a background image: ${value}`
    else if (prop.startsWith('--font-')) err = value.length ? null : 'empty font list'
    else if (prop === '--theme-label') err = /^(['"]).+\1$/.test(value) ? null : `not a quoted label: ${value}`
    else if (prop === '--theme-scheme') err = /^(light|dark)$/.test(value) ? null : `not light or dark: ${value}`
    else if (prop === '--theme-order') err = /^\d+$/.test(value) ? null : `not a whole number: ${value}`
    else continue // Tailwind's own theme keys
    if (err) errors.push(`${prop}: ${err}`)
  }
  return errors
}

// ── component lint ──────────────────────────────────────────────────────────────────────────────────────────
const PALETTE =
  'slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose'
const UTIL =
  'bg|text|border|border-[tlrbxy]|from|to|via|ring|outline|fill|stroke|divide|decoration|accent|caret|placeholder|shadow'
// [rule, pattern, applies to stylesheets]: utility names are only looked for in components, where a class list can
// hold them; a stylesheet's prose ("a shadow role") must not trip them.
const RULES = [
  ['literal color', /#[0-9a-fA-F]{3,8}\b|\b(?:rgba?|hsla?|oklch|oklab)\([^)]*\)/g, true],
  [
    'palette utility',
    new RegExp(`(?<![\\w-])(?:${UTIL})-(?:(?:${PALETTE})-\\d{2,3}|white|black)(?:/\\d+)?(?![\\w-])`, 'g'),
    false,
  ],
  ['radius utility', /(?<![\w-])rounded(?!-(?:control|surface|row)(?![\w-]))(?:-[a-z0-9[\]]+)*(?![\w-])/g, false],
  ['shadow utility', /(?<![\w-])shadow(?:-(?:2xs|xs|sm|md|lg|xl|2xl|inner|\[[^\]]*\]))?(?![\w-])/g, false],
  [
    'literal radius',
    /border-radius:(?!\s*(?:var\(--radius-(?:control|surface|row)\)|0|inherit)\s*[;}\n])[^;}\n]+/g,
    true,
  ],
  ['old token', /var\(--(?:accent(?:-[a-z0-9]+)*|row-selected|color-(?:ink|panel|app-bg|success)[a-z0-9-]*)\)/g, true],
]

// lint returns the findings of one source text as {line, rule, text}. `skip` is a list of [start, end] offsets
// that are not linted (the contract in style.css); `stylesheet` leaves the utility rules out.
export function lint(text, skip = [], stylesheet = false) {
  const found = []
  for (const [rule, re, inCss] of RULES) {
    if (stylesheet && !inCss) continue
    for (const m of text.matchAll(re)) {
      if (skip.some(([a, b]) => m.index >= a && m.index < b)) continue
      found.push({ line: text.slice(0, m.index).split('\n').length, rule, text: m[0] })
    }
  }
  return found.sort((a, b) => a.line - b.line)
}

// lintSources lints every component source: all of src except the theme files and the contract. Findings matched by an allowlist entry {file, pattern, reason} are dropped; `used` collects the entries
// that matched something.
export function lintSources(allow, used = new Set()) {
  const findings = []
  for (const f of files()) {
    const rel = relative(SRC, f)
    if (rel.startsWith('themes/') || rel === CONTRACT) continue
    const text = read(f)
    for (const hit of lint(text, [], rel.endsWith('.css'))) {
      const line = text.split('\n')[hit.line - 1]
      const entry = allow.find((a) => a.file === rel && a.pattern.test(line))
      if (entry) used.add(entry)
      else findings.push(`${rel}:${hit.line}: ${hit.rule}: ${hit.text}`)
    }
  }
  return findings
}

// ── registry ────────────────────────────────────────────────────────────────────────────────────────────────
// registry reads the ordered theme list of internal/theme/registry_gen.go (generated by scripts/themes.mjs).
export function registry(go = read(join(ROOT, 'internal/theme/registry_gen.go'))) {
  return [...go.matchAll(/\{ID: "([^"]+)", Label: "([^"]+)", Scheme: "(\w+)"\}/g)].map((m) => ({
    id: m[1],
    label: m[2],
    scheme: m[3],
  }))
}

// ── gradients ───────────────────────────────────────────────────────────────────────────────────────────────
// gradientStops returns the color stops of a linear-gradient value as [{color, at}] (at in percent; stops without
// a position are spread evenly). `none` has no stops.
export function gradientStops(value) {
  const m = value.trim().match(/^linear-gradient\((.*)\)$/s)
  if (!m) return []
  const parts = splitTop(m[1]).filter((p) => !/^(to |[\d.-]+(deg|turn|rad))/.test(p))
  return parts.map((p, i) => {
    const pos = p.match(/\s([\d.]+)%$/)
    const color = parseColor(pos ? p.slice(0, pos.index) : p)
    return { color, at: pos ? Number(pos[1]) : (i / Math.max(parts.length - 1, 1)) * 100 }
  })
}
