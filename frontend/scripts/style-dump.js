// Computed-style dump, for checking that a change to the stylesheet or to a theme leaves a screen as it was.
// Paste into the browser console on the screen to check; it returns a map from each element's position in the
// document (and its ::before / ::after) to its colors, corner radii, shadow, outline, font and SVG paint.
// Take one dump before the change and one after, on the same screen in the same state, and compare them:
//   const before = styleDump()   // … apply the change, reload, return to the same screen …
//   styleDiff(before, styleDump())
// The dump sees the resting state only; hover, focus and drag states have to be checked by hand.
const STYLE_DUMP_PROPS = [
  'color',
  'background-color',
  'border-top-color',
  'border-right-color',
  'border-bottom-color',
  'border-left-color',
  'border-top-left-radius',
  'border-top-right-radius',
  'border-bottom-right-radius',
  'border-bottom-left-radius',
  'box-shadow',
  'outline-color',
  'font-family',
  'fill',
  'stroke',
  'stop-color',
]

function styleDump(root = document.body) {
  const position = (el) => {
    const parts = []
    for (let e = el; e && e !== root; e = e.parentElement) {
      parts.unshift(`${e.tagName.toLowerCase()}:${[...e.parentElement.children].indexOf(e)}`)
    }
    return parts.join('>')
  }
  const dump = {}
  for (const el of root.querySelectorAll('*')) {
    for (const pseudo of ['', '::before', '::after']) {
      const style = getComputedStyle(el, pseudo || null)
      if (pseudo && style.content === 'none') continue
      dump[position(el) + pseudo] = STYLE_DUMP_PROPS.map((p) => style.getPropertyValue(p)).join(' | ')
    }
  }
  return dump
}

// Lists what differs between two dumps as { position: { property: [before, after] } }.
function styleDiff(before, after) {
  const diff = {}
  for (const key of new Set([...Object.keys(before), ...Object.keys(after)])) {
    if (before[key] === after[key]) continue
    if (before[key] === undefined || after[key] === undefined) {
      diff[key] = before[key] === undefined ? 'only after' : 'only before'
      continue
    }
    const a = before[key].split(' | ')
    const b = after[key].split(' | ')
    diff[key] = Object.fromEntries(STYLE_DUMP_PROPS.map((p, i) => [p, [a[i], b[i]]]).filter(([, [x, y]]) => x !== y))
  }
  return diff
}

// make both callable from the console after pasting the file
globalThis.styleDump = styleDump
globalThis.styleDiff = styleDiff
