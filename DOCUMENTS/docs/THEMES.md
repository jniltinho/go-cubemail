# Themes

The webmail ships with nine themes. Each user picks one in **Settings → Appearance**; the choice is saved to the
account, so it follows the user to other browsers. The administrator sets the default with `ui.theme`.

| Theme | Id | Scheme | Reference |
|---|---|---|---|
| Larry | `larry` | light | Roundcube's Larry skin: the look the application always had. The default. |
| Outlook | `outlook` | light | inspired by Outlook on the web: Fluent 2 neutrals, brand blue `#0F6CBD`, 4 px corners |
| Outlook (dark) | `outlook-dark` | dark | the same on Fluent's dark grays, lighter blue `#479EF5` |
| Apple Mail | `apple-mail` | light | inspired by Mail on macOS: system colors, light toolbar, inset rounded selection |
| Apple Mail (dark) | `apple-mail-dark` | dark | the same on near-black surfaces, system blue `#0A84FF` |
| Breeze | `breeze` | light | KDE Breeze, ported from ad-manager |
| Breeze (dark) | `breeze-dark` | dark | KDE Breeze Dark, ported from ad-manager |
| Graphite (dark) | `graphite` | dark | Grafana 11's dark palette with the orange accent, ported from ad-manager |
| High contrast (dark) | `high-contrast` | dark | accessibility: black, white and one yellow accent, 3 px focus ring, WCAG AAA |

Outlook and Apple Mail are "inspired by" looks with **no affiliation** to Microsoft or Apple: they use public system
colors and system font stacks only. No logo, icon or font of those products is used, and the application keeps its
own brand mark in every theme.

## How a theme works

A theme is a set of values for named **roles**. Nothing else changes between themes: the layout, the components and
the markup are the same.

- **The contract** is in `frontend/src/style.css`: one `@theme static` block (colors, corner radii, fonts) and one
  `:root` rule (shadows, the selected row's shape, the login backdrop). Its values are Larry's, so Larry needs no
  theme file.
- **A theme file** under `frontend/src/themes/` holds only `:root[data-theme='<id>'] { --role: value; }` rules. No
  other selector, no component rule, no role the contract does not declare.
- **Components use roles only**: utilities such as `bg-surface`, `text-fg-muted`, `border-line`, `rounded-control`,
  or `var(--color-…)` in a stylesheet. A literal color, a palette utility (`bg-white`, `text-red-500`), a
  `rounded-*` utility other than the three role ones, or a `shadow-*` utility fails the tests.
- **Applying a theme** sets `data-theme` and `color-scheme` on `<html>` (`frontend/src/utils/theme.ts`, the only
  writer). `frontend/public/theme-boot.js` does the same from the browser's cache before the first paint, so a
  reload in a dark theme shows no light frame. It is a file, not an inline script, because the Content Security
  Policy is `script-src 'self'`.

### Roles

| Group | Roles (`--color-` unless noted) |
|---|---|
| Text | `fg`, `fg-strong`, `fg-muted`, `fg-subtle`, `heading`, `link`, `link-hover` |
| Page and surfaces | `page`, `backdrop` (login), `surface`, `surface-2`, `surface-3`, `line`, `line-soft`, `popup-line`, `row-hover`, `overlay`, `scroll-thumb` |
| Selection (the selected message) | `selection`, `on-selection`, `on-selection-muted`, `on-selection-subtle`, `on-selection-star`, `selection-mark`, `selection-line` |
| Fields and focus | `field`, `on-field`, `field-line`, `field-focus`, `focus` |
| Primary action | `action`, `action-hover`, `action-line`, `on-action` |
| Secondary action | `action-2`, `action-2-hover`, `action-2-line`, `action-2-hover-line`, `on-action-2` |
| Danger and status | `danger`, `on-danger`, `ok`, `err`, `warn`, `err-soft`, `on-err-soft`, `err-line` |
| Header (app bar, dialog title bars, login box) | `header`, `header-brand`, `on-header`, `on-header-muted`, `on-header-err`, `header-line`, `header-control-line`, `header-hover`, `tab-on`, `on-tab-on`, `tab-mark` |
| Sidebar | `sidebar`, `on-sidebar`, `sidebar-hover`, `sidebar-active`, `on-sidebar-active`, `count` |
| Mail list | `unread`, `unread-mark`, `star` |
| Accent | `accent`, `accent-strong`, `accent-soft`, `on-accent` |
| Source viewer | `code`, `on-code`, `on-code-muted` |
| Message content | `paper`, `paper-2`, `on-paper`, `paper-muted`, `paper-subtle`, `paper-link`, `paper-line`, `--font-paper` |
| Depth | `--shadow-menu`, `--shadow-dialog`, `--shadow-toast`, `--shadow-card`, `--shadow-focus`, `--shadow-focus-header` |
| Shape | `--radius-control`, `--radius-surface`, `--radius-row`, `--row-inset-x`, `--selection-mark-w`, `--bg-backdrop` |
| Type | `--font-sans`, `--font-mono` |

A role describes a job, not a color: `--color-action` is "the fill of a primary button", whatever its hue. An `on-*`
role is the ink drawn over the role it names.

### Message content stays on a light paper

HTML mail and the composer's document are drawn on the paper roles, and **every theme keeps the paper light with
dark ink**, dark themes included. HTML mail is third-party content written for a white page, and what a user
composes is read by recipients on white. Only the interface around the paper follows the theme. Plain-text messages
are rendered by the application and use the theme's surface and text roles.

The quote header and quote block the composer adds to a reply are sent in the message, so they use literal colors
and never a role.

### Colors that are content

Three kinds of color are data, not interface, and are exempt from the lint through an allowlist in
`frontend/tests/unit/themes.test.mjs`: the composer's text-color and highlight swatches, the reply quote markup, and
calendar colors (with `DEFAULT_CAL_COLOR`). Text drawn over a calendar's color is black or white, whichever
contrasts more (`inkOn` in `frontend/src/utils/helpers.ts`).

## Adding a theme

1. Create `frontend/src/themes/<family>.css` with one `:root[data-theme='<id>']` block per variant. A dark theme
   must declare every `--color-*` role; a light theme should too, or Larry's navy shows through. Start from the
   file closest to the look you want. Note the reference and every value changed for contrast in the header comment.
2. Import the file in `frontend/src/style.css`, next to the other themes.
3. Add one line to the registry in `internal/theme/theme.go`: `{ID: "<id>", Label: "<label>", Scheme: "light|dark"},`.
   The order of the registry is the order of the list in Settings; the first entry is the default of last resort.
4. Run the tests and fix what they report:

   ```bash
   make test-frontend   # theme files, component lint, registry parity, contrast
   make test            # registry lookup, the settings and themes endpoints
   ```

If a look needs something the roles cannot express, add a role to the contract with Larry's value as its default
(every other theme is then unaffected) rather than a component rule inside a theme file.

## What the tests guarantee

`frontend/tests/unit/themes.test.mjs` (run by `npm test` / `make test-frontend`) reads source files only:

- a theme file contains nothing but role declarations of the contract;
- a color role holds a color, a shadow role a shadow, a radius role a length;
- components hold no literal color, palette utility, radius or shadow outside the contract (except the allowlist,
  whose stale entries fail);
- the registry in Go and the theme blocks in CSS are the same set, and each theme's scheme matches its colors;
- a dark theme declares every color role, and every theme keeps the paper light;
- **contrast**: `frontend/src/themes/contrast.json` lists each text and component role over the backgrounds where it
  appears. Normal text needs 4.5:1; large text and interface components 3:1 (WCAG AA). Themes in its `aaa` list
  (`high-contrast`) need 7:1 and 4.5:1. Translucent colors are composited before measuring.

Three pairings are checked for `high-contrast` only (`aaaOnly`): field outlines, secondary-button outlines and the
star. In the other themes the control's fill or the icon's shape marks them, and the reference looks use light
outlines.

`frontend/scripts/style-dump.js` is a console snippet that dumps the computed colors, radii, shadows and fonts of
every element of a screen, to compare a screen before and after a stylesheet change.

## Server side

- `GET /api/v1/themes` (public): `{"default": "<id>", "themes": [{"id", "label", "scheme"}, …]}`.
- `ui.theme` in `config.toml`: the default for users who have not chosen a theme, and for the login screen. An id
  that is not registered logs a warning at start and falls back to `larry`.
- `GET` / `PUT /api/v1/settings` carry `theme`: a registered id, or `""` to follow the server default. An unknown id
  is rejected with `400`. The value lives in `user_settings.theme`, added by `go-cubemail migrate`.

Order of precedence in the browser: the signed-in user's saved theme, else the theme this browser remembers (login
screen), else the server default. Signing out keeps the remembered theme.

## Changes to the Larry look

Moving Larry onto roles kept its colors as they rendered (its lighter and darker blues used to be derived at run
time from the navy; they are now fixed values in the contract). These are the deliberate differences:

- **Contrast corrections**, each the smallest change in lightness: faint text `#8a93a1` → `#66707f` (it was 3.1:1),
  muted and faint text on the selected message → `#555e6d`, the warning color `#e0a40c` → `#bf8c0a`, faint text in
  the source viewer and placeholders on the paper. The original value is noted beside each role.
- **Two styles that were written but never applied now show**: the current view's tab in the app bar is
  highlighted with its mark, and the calendar's sidebar has the sidebar background.
- **Keyboard focus**: every control shows a focus ring when reached with the keyboard (buttons had none); a mouse
  click leaves none.
- **Event labels** are black or white by the calendar color's luminance instead of always white.
- **The active folder's icon and menu button** take the active row's ink.
- **One-off colors were folded into roles**: dialog overlays share one tint, the "Decline" button of an invitation
  is the danger color, delete buttons are the danger color instead of a brighter red, and a disabled delete button
  looks disabled.
- The runtime accent color hook (`applyAccent`, which no control could change) is gone; a theme is the only source
  of a color.
