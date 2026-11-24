# Themes

The webmail ships with ten themes. Each user picks one in **Settings → Appearance**; the choice is saved to the
account, so it follows the user to other browsers. The administrator sets the default with `ui.theme`.

| Theme | Id | Scheme | Reference |
|---|---|---|---|
| Larry | `larry` | light | Roundcube's Larry skin: the look the application always had. The default. |
| Outlook | `outlook` | light | inspired by Outlook on the web: Fluent 2 neutrals and brand blue, 4 px corners |
| Outlook (dark) | `outlook-dark` | dark | the same on Fluent's dark grays |
| Apple Mail | `apple-mail` | light | inspired by Mail on macOS: system colors, light toolbar, inset rounded selection |
| Apple Mail (dark) | `apple-mail-dark` | dark | the same on near-black surfaces |
| Breeze | `breeze` | light | KDE Breeze, ported from ad-manager |
| Breeze (dark) | `breeze-dark` | dark | KDE Breeze Dark, ported from ad-manager |
| Graphite (dark) | `graphite` | dark | Grafana 11's dark palette with the orange accent, ported from ad-manager |
| High contrast (dark) | `high-contrast` | dark | accessibility: black, white and one yellow accent, 3 px focus ring |
| Windows XP | `windows-xp` | light | the Luna blue style: gradient title bars, Start-green action, XP push buttons, Tahoma |

**Every theme meets WCAG AAA**: normal text 7:1, large text 4.5:1 and interface components 3:1, in every pairing
listed in `frontend/src/themes/contrast.json`. A reference color that fell short was moved in lightness only; the
theme file says what it was.

Outlook, Apple Mail and Windows XP are "inspired by" looks with **no affiliation** to Microsoft or Apple: they use
public system colors and system font stacks only. No logo, icon or font of those products is used, and the
application keeps its own brand mark in every theme.

## Adding a theme

```bash
cd frontend
npm run theme:new -- nord "Nord" dark --from graphite   # id, label, light|dark, optional theme to start from
```

That creates `src/themes/nord.css` with every role filled in (from the theme given with `--from`, or from Larry)
and regenerates the server's registry. Then:

1. Edit the values in the new file. Name the reference look, and every value changed for contrast, in its header.
2. `npm test` checks the file's structure, the role types and the contrast of every pairing; fix what it reports.
3. `make frontend build` and the theme is in the list of Settings → Appearance.

Nothing else has to be touched: the stylesheet loads every file of `src/themes/`, and the list the server serves is
generated from those files. After renaming a theme, or changing its label, scheme or order by hand, run
`npm run themes` (or `make themes`) to regenerate the registry; the tests fail while it is out of date.

If a look needs something the roles cannot express, add a role to the contract with Larry's value as its default
(every other theme is then unaffected) rather than a component rule inside a theme file.

## How it is organized

```
frontend/src/
├── style.css                  entry: Tailwind, then the files below
├── styles/
│   ├── contract.css           the role contract; its values are the Larry theme
│   ├── base.css               document defaults, large-screen scaling, scrollbars, keyboard focus
│   └── components/            one file per group of component classes
│       ├── buttons.css  sidebar.css  mail-list.css  popups.css  header.css
│       └── composer.css  editor.css  calendar.css  login.css
├── themes/
│   ├── <id>.css               one theme per file: a single :root[data-theme='<id>'] block of role values
│   └── contrast.json          the pairings every theme must meet
└── utils/theme.ts             applies a theme to the document
frontend/scripts/themes.mjs    `npm run theme:new`, `npm run themes`
internal/theme/registry_gen.go the server's list of themes, generated from the files above
```

- **The contract** (`styles/contract.css`) declares every role once: one `@theme static` block (colors, corner
  radii, fonts) and one `:root` rule (shadows, relief, gradient fills, the selected row's shape, the theme's
  identity). Its values are Larry's, so Larry needs no theme file.
- **A theme file** holds only `:root[data-theme='<id>'] { --role: value; }`: no other selector, no component rule,
  no role the contract does not declare. Its identity is three roles: `--theme-label`, `--theme-scheme`
  (`light` or `dark`) and `--theme-order` (position in the list).
- **Components use roles only**: utilities such as `bg-surface`, `text-fg-muted`, `border-line`, `rounded-control`,
  or `var(--color-…)` in a stylesheet. A literal color, a palette utility (`bg-white`, `text-red-500`), a
  `rounded-*` utility other than the three role ones, or a `shadow-*` utility fails the tests.
- **Applying a theme** sets `data-theme` and `color-scheme` on `<html>` (`utils/theme.ts`, the only writer).
  `public/theme-boot.js` does the same from the browser's cache before the first paint, so a reload in a dark theme
  shows no light frame. It is a file, not an inline script, because the Content Security Policy is
  `script-src 'self'`.

### Roles

| Group | Roles (`--color-` unless noted) |
|---|---|
| Identity | `--theme-label`, `--theme-scheme`, `--theme-order` |
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
| Depth and relief | `--shadow-menu`, `--shadow-dialog`, `--shadow-toast`, `--shadow-card`, `--shadow-focus`, `--shadow-focus-header`, `--shadow-tab-on`, `--shadow-action`, `--shadow-action-2`, `--shadow-field` |
| Gradient fills | `--bg-header`, `--bg-tab-on`, `--bg-action`, `--bg-action-hover`, `--bg-action-2`, `--bg-action-2-hover`, `--bg-strip`, `--bg-backdrop` |
| Shape | `--radius-control`, `--radius-surface`, `--radius-row`, `--row-inset-x`, `--selection-mark-w` |
| Type | `--font-sans`, `--font-mono` |

A role describes a job, not a color: `--color-action` is "the fill of a primary button", whatever its hue. An `on-*`
role is the ink drawn over the role it names. A gradient fill is drawn over the color role of the same name and
defaults to `none`; only Windows XP uses them. The color role stays the value the contrast is measured on, and a
test checks the gradient's stops as well.

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

## Large screens

The interface is drawn for viewports of about 1280 to 1920 px (Full HD). Wider viewports scale it as a whole
(`styles/base.css`): 1.25 from 2400 px (QHD at 100%), 1.6 from 3200 px (4K at 100%). A 4K monitor at 150% system
scaling reports 2560 px and gets 1.25. Sizes written in viewport units divide by `--ui-zoom`.

## What the tests guarantee

`frontend/tests/unit/themes.test.mjs` (run by `npm test` / `make test-frontend`) reads source files only:

- a theme file holds one theme, is named after its id, declares its identity and contains nothing but role
  declarations of the contract;
- a color role holds a color, a shadow role a shadow, a radius role a length, a fill role a gradient or `none`;
- components hold no literal color, palette utility, radius or shadow outside the contract (except the allowlist,
  whose stale entries fail);
- the generated registry is up to date with the theme files, and each theme's scheme matches its colors;
- a dark theme declares every color role, and every theme keeps the paper light;
- **contrast**: every pairing of `contrast.json` at the AAA level in every theme, with translucent colors
  composited before measuring;
- **gradients**: between 20% and 80% of its height, every stop of a gradient fill keeps its text at 7:1 (the top
  and bottom fifth may hold a highlight or a shade).

`frontend/scripts/style-dump.js` is a console snippet that dumps the computed colors, radii, shadows and fonts of
every element of a screen, to compare a screen before and after a stylesheet change.

## Server side

- `GET /api/v1/themes` (public): `{"default": "<id>", "themes": [{"id", "label", "scheme"}, …]}`.
- `ui.theme` in `config.toml`: the default for users who have not chosen a theme, and for the login screen. An id
  that is not registered logs a warning at start and falls back to `larry`.
- `GET` / `PUT /api/v1/settings` carry `theme`: a registered id, or `""` to follow the server default. An unknown id
  is rejected with `400`. The value lives in `user_settings.theme`, added by `go-cubemail migrate`.
- The list is `internal/theme/registry_gen.go`, generated; do not edit it by hand.

Order of precedence in the browser: the signed-in user's saved theme, else the theme this browser remembers (login
screen), else the server default. Signing out keeps the remembered theme.

## Changes to the Larry look

Moving Larry onto roles kept its colors as they rendered (its lighter and darker blues used to be derived at run
time from the navy; they are now fixed values in the contract). These are the deliberate differences:

- **AAA contrast**, each the smallest change in lightness, with the original value noted beside the role: muted
  and faint text are darker (faint text was 3.1:1), links and the hover of the primary button are a darker blue,
  the outlines of fields and of secondary buttons are darker (they were 1.5:1; components need 3:1), and the star
  and the warning color are a darker gold.
- **Two styles that were written but never applied now show**: the current view's tab in the app bar is
  highlighted with its mark, and the calendar's sidebar has the sidebar background.
- **Keyboard focus**: every control shows a focus ring when reached with the keyboard (buttons had none); a mouse
  click leaves none.
- **Event labels** are black or white by the calendar color's luminance instead of always white.
- **The active folder's icon and menu button** take the active row's ink.
- **One-off colors were folded into roles**: dialog overlays share one tint, the "Decline" button of an invitation
  is the danger color, delete buttons are the danger color instead of a brighter red, and a disabled delete button
  looks disabled.
- **Large screens** are scaled up (see above) instead of showing the 13 px interface at its natural size.
- The runtime accent color hook (`applyAccent`, which no control could change) is gone; a theme is the only source
  of a color.
