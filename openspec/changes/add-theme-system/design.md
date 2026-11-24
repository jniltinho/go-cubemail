## Context

go-cubemail's look is Roundcube's Larry theme, written directly into the code:

- `frontend/src/style.css` has a `@theme` block with literal values (`--color-line`, `--color-panel`, `--color-ink`,
  ...), a `:root` block with six accent variables, and about 48 literal colors inside its component classes.
- The components carry about 130 hex literals (56 of them are the composer's color swatches, which are content) and
  about 60 `white`/`black`/`gray-*` utilities; `App.vue` and `editor/TipTapEditor.vue` have some too.
- `*, *::before, *::after { border-radius: 0 !important }` makes every corner square; the 28 `rounded*` utilities in
  the components have no effect today. Two more resets sit beside it: `button, input, select, textarea
  { border-radius: 0 }` and `.rte-toolbar .tbtn { border-radius: 0 !important }`.
- Fields get their focus ring from per-element utilities (`focus:ring-1 focus:ring-accent`); `.tbtn` buttons have no
  focus style at all.
- Larry's faintest text (`--color-ink-mute`, `#8A93A1`, used for message snippets) is 3.1:1 on white, below WCAG AA.
- `applyAccent()` (`frontend/src/utils/helpers.ts`) writes the accent variables as **inline styles** on `<html>`,
  driven by an `accent` value in the mail store that no component changes. Inline styles win over any stylesheet
  rule, so this code would silently override every theme.
- The backend reads `ui.theme` from the config into `cfg.UI.Theme` and never uses it. Per-user preferences live in
  `UserSettings` behind `GET`/`PUT /api/v1/settings`.
- The CSP is `script-src 'self'` (no inline scripts) and `style-src 'self' 'unsafe-inline'`.
- HTML messages render in a sandboxed `srcdoc` iframe with a hard-coded white page; the composer's editing area is
  hard-coded white too. Plain-text messages render as normal elements of the reading pane.
- The frontend builds with Vite+ (`vp dev`, `vp build`, `vp check`), which bundles a test runner (`vp test`), but
  has no tests yet. `postcss` is already a devDependency.
- Only `go-cubemail migrate` runs AutoMigrate; `serve` does not.

The model to follow is `jniltinho/ad-manager`: a contract of role tokens in `style.css` whose defaults are the default
theme, one file per theme under `frontend/src/themes/` with nothing but `:root[data-theme='id'] { --role: value }`,
a registry in Go, and tests for file structure, component colors, registry parity and WCAG contrast.

## Goals / Non-Goals

**Goals:**

- A new theme is one CSS file plus one registry line; no component is edited.
- Larry stays the default and keeps its look; only documented contrast corrections change it.
- Outlook and Apple Mail looks, light and dark, plus four themes ported from ad-manager.
- Each user picks a theme; the choice follows the user across browsers and survives reloads without a flash.
- A theme cannot ship with unreadable text: contrast is tested, not eyeballed.
- No new runtime or build dependency.

**Non-Goals:**

- Layout or density per theme (Outlook's app rail, Apple's unified toolbar, row heights). Themes change color, type,
  corner shape and depth only; the three-column structure is the same in all of them.
- The remaining ad-manager themes, a "follow the system" light/dark switch, theme thumbnails, a flat-look switch,
  user-defined themes or a color picker.
- Recoloring message content (HTML mail, the composer's document).
- Translating theme labels (the interface is English-only).

## Decisions

### D1. Role-token contract, with ad-manager's vocabulary

The contract is the set of custom properties declared in `style.css` inside one `@theme static` block (roles that
need a Tailwind utility) and one `:root` rule (shape parameters). Their values are Larry's. Roles describe a job,
not a color.

`static` is required: checked against the installed Tailwind 4, a plain `@theme` drops every variable that no
utility or stylesheet rule references, and a role read only from script (the message paper, D9) or only by a theme
would vanish from the build. The same check confirms that a `--radius-control` theme variable yields the
`rounded-control` utility, that the bare `rounded` utility is a fixed `0.25rem` (so it cannot follow a theme), and
that opacity modifiers such as `bg-surface/40` follow the active theme through `color-mix()` in browsers that
support it and fall back to Larry's value elsewhere.

| Group | Roles |
|---|---|
| Text | `--color-fg`, `-fg-strong`, `-fg-muted`, `-fg-subtle`, `-link`, `-link-hover` |
| Page and surfaces | `--color-page`, `-surface`, `-surface-2`, `-surface-3`, `-line`, `-line-soft`, `-row-hover`, `-overlay` |
| Selection | `--color-selection`, `-on-selection`, `-selection-mark`, `-selection-line` |
| Fields | `--color-field`, `-on-field`, `-field-line`, `-field-focus`, `-focus` |
| Primary action | `--color-action`, `-action-hover`, `-action-line`, `-on-action` |
| Secondary action | `--color-action-2`, `-action-2-hover`, `-action-2-line`, `-on-action-2` |
| Danger and status | `--color-danger`, `-on-danger`, `-ok`, `-err`, `-warn`, `-ok-soft`, `-err-soft`, `-warn-soft`, `-on-ok-soft`, `-on-err-soft`, `-on-warn-soft` |
| Header (app bar) | `--color-header`, `-header-brand`, `-on-header`, `-on-header-muted`, `-header-line`, `-header-hover`, `-tab-on`, `-on-tab-on`, `-tab-mark` |
| Sidebar | `--color-sidebar`, `-on-sidebar`, `-sidebar-hover`, `-count` |
| Mail list | `--color-unread`, `-unread-mark`, `-star` |
| Accent | `--color-accent`, `-accent-strong`, `-accent-soft` |
| Message content | `--color-paper`, `-on-paper`, `-paper-link` |
| Depth | `--shadow-menu`, `--shadow-dialog`, `--shadow-toast`, `--shadow-card`, `--shadow-focus` |
| Shape | `--radius-control`, `--radius-surface`, `--radius-row`, `--row-inset-x`, `--selection-mark-w` |
| Type | `--font-sans`, `--font-mono` |

The names shared with ad-manager (`fg*`, `page`, `surface*`, `line*`, `row-hover`, `field*`, `action*`, `danger`,
`selection`, `focus`, `ok`/`err`/`warn`, `link*`, `header`, `accent*`) mean the same thing there, so a ported palette
is mostly a copy of values. The final list is fixed in task group 1 by the inventory of today's color uses: a role
is added only when a real use needs it, and a use is never forced onto a role with a different job.

Today's tokens map as follows; the old names are removed, not aliased, so no component can keep using them:

| Today | Role |
|---|---|
| `--color-ink`, `-ink-sub`, `-ink-mute` | `--color-fg`, `-fg-muted`, `-fg-subtle` |
| `--color-panel`, `-panel-2`, `-app-bg` | `--color-surface`, `-surface-2`, `-page` |
| `--color-line`, `-line-soft`, `-star`, `-danger` | unchanged names |
| `--color-success` | `--color-ok` |
| `--accent`, `--accent-2`, `--accent-bar`, `--accent-soft`, `--accent-soft-2`, `--row-selected` | split by job: `action*`, `header*`, `selection*`, `link`, `focus`, `accent*` |

The accent family is the only non-mechanical part. Larry uses one navy for the header, the primary button, links and
the selection mark; Outlook and Apple Mail use different colors for those jobs, so each accent use (61 utilities in the
components plus the `var(--accent*)` references) is assigned to the role of its job.

*Alternative considered:* keep today's names and add roles only for the literals. Less churn, but ad-manager's
palettes would need a name translation forever, and `accent` would still mean four different things.

### D2. Larry is the contract's default; tokenizing changes nothing a user can see

Tokenizing (task groups 1 and 2) lands before any second theme exists, with the rule "no computed style changes".
It is checked with a computed-style dump: a small script, run in the browser console on the same screens before and
after, writes for every element and for its `::before`/`::after` the color, background, border colors, radius,
shadow, outline, font family and the SVG `fill`, `stroke` and `stop-color`; the two dumps must be equal. This needs
no new dependency and is stricter than comparing screenshots by eye. The dump sees the resting state only; hover,
focus and drag states are checked by hand in task group 7.

Larry's contrast corrections come **after** that gate, as their own step (task 6.5): each pairing the contrast test
rejects is fixed by the smallest change in lightness, same hue, with the original value noted in the contract, as
ad-manager did for its default theme. Known so far: `--color-fg-subtle` (`#8A93A1`, 3.1:1 on the surface).

`larry` has no theme file: `data-theme="larry"` matches no block and the contract's defaults apply.

### D3. Theme files hold role values only

`frontend/src/themes/<family>.css` contains comments and `:root[data-theme='<id>'] { --role: value; }` rules, nothing
else. A family file holds the light and the dark variant (`outlook.css` has `outlook` and `outlook-dark`). All theme
files are imported by `style.css`, so every theme ships in the one stylesheet and switching is instant and offline.
Nine themes of about 70 roles add a few kilobytes, less than one lazy-loading mechanism would cost in code.

A dark theme declares every color role: inheriting a light default by accident is the usual source of unreadable
text, so the test fails on a dark theme that leaves a color role undeclared.

*Alternative considered:* theme-specific component rules (`[data-theme='apple-mail'] .mail-row { ... }`). Rejected:
it is how themes drift apart and break on the next component change. A look that the roles cannot express gets a new
role or shape parameter, available to every theme.

### D4. Radius roles replace the global reset

`* { border-radius: 0 !important }` is removed, and so are the two resets beside it (the native-control one and
`.rte-toolbar .tbtn`'s `!important`), which would otherwise keep defeating the roles. Three roles take their place,
all `0` by default:
`--radius-control` (buttons, fields, checkboxes, chips), `--radius-surface` (dialogs, menus, cards, toasts) and
`--radius-row` (list rows, sidebar items). Declared in `@theme`, they produce the utilities `rounded-control`,
`rounded-surface` and `rounded-row`. The component classes use the roles; the 28 `rounded*` utilities are each
replaced by a role utility or removed, and the lint rejects any other `rounded-*` utility or literal
`border-radius`. Two shape parameters, `--row-inset-x` and `--selection-mark-w`, let Apple Mail draw its selected
row as an inset rounded rectangle with no left bar, while Larry keeps the full-width row with the 3 px bar.

Outlook uses 4 px controls and surfaces; Apple Mail 6 px controls, 10 px surfaces and 6 px rows.

*Alternative considered:* keep every theme square, as ad-manager does. Square Outlook and Apple Mail would not read
as either; the roles cost three variables.

### D5. The registry lives in Go and is the single list

A new package `internal/theme` holds the ordered list `{ID, Label, Scheme}`; the first entry is the default of last
resort. `ByID` returns the entry or the first one and `false`. Both `internal/config` (start-up check of `ui.theme`)
and `internal/handler` (settings validation, the public endpoint) import it, which is why it is not in
`internal/handler`.

The SPA never hard-codes the list: it reads `GET /api/v1/themes`. A frontend test parses the Go file and fails
unless every registry id except the first has a `data-theme` block and every block's id is in the registry, and
unless each theme's `color-scheme` matches the lightness of its page color.

*Alternative considered:* the list in TypeScript with the server validating a duplicate. Two lists need a parity
test anyway, and the server must own validation.

### D6. Per-user preference, server default, public endpoint

- `GET /api/v1/themes` (public, no session): `{"default": "<id>", "themes": [{"id","label","scheme"}, ...]}`. The
  login screen needs it before any session exists; the content is static and not sensitive.
- `ui.theme` is the server default. An id that is not in the registry logs a warning at start and falls back to
  `larry`; the server still starts.
- `UserSettings.Theme` (string) is added by AutoMigrate when `go-cubemail migrate` runs. Empty means "follow the
  server default". The column must **not** carry a GORM `default:` tag: the settings upsert is a `Create` with
  `UpdateAll`, and GORM replaces a zero value with the tagged default on create, so a tagged column could never be
  cleared (checked in a scratch program; the same mechanism is why `compose_html = false` does not survive a save
  today, a separate bug noted under Open Questions).
- `GET /api/v1/settings` returns `theme`: the stored id, or `""` when the user follows the default or the stored id
  is no longer in the registry.
- `PUT /api/v1/settings` accepts an optional `theme`: a registry id, or `""` to follow the default. Any other value
  is `400` and nothing is saved: the handler binds and validates the body before it loads the user's settings
  (today it loads first, and loading creates the row). The field is a pointer like the others, so clients that do
  not send it are
  unaffected.

*Alternative considered:* one global theme set by the administrator, as in ad-manager. A webmail has many users with
personal taste and accessibility needs (the high-contrast theme is a per-person need); the administrator keeps
control of the default.

### D7. Applying the theme without a flash, within the CSP

`frontend/src/utils/theme.ts` has one function that sets `data-theme` and `color-scheme` on `<html>`, updates the
`color-scheme` meta and caches `{id, scheme}` in `localStorage` under `gorc_theme`. It is the only writer.

`frontend/public/theme-boot.js` is a classic (blocking) script in `<head>` that reads the same key and sets the same
two things before the stylesheet paints anything. It is an external file because the CSP forbids inline scripts;
`script-src 'self'` already allows it, so the CSP does not change. It is wrapped in `try`/`catch`: storage can be
blocked, and then the page simply starts as Larry.

Order of precedence once the app runs: the signed-in user's stored theme, else the browser's cached theme (login
screen), else the server default. On login the user's setting replaces the cache; logout keeps the cache so the
login screen stays in the user's theme on that browser.

Known limit: the very first visit of a browser to a server whose default is not `larry` paints Larry until
`/api/v1/themes` answers. Injecting the default into `index.html` on the server would remove it; left out until
someone needs it.

### D8. `applyAccent` is removed, not adapted

The `accent` store value, its `watchEffect`, `applyAccent()` and the `.gw-tweaks-*` classes are deleted. Keeping a
runtime accent would mean inline styles that defeat the theme files and derived colors that no contrast test can
see. The SDD's "accent color picker" entries are corrected to describe themes.

### D9. Message content stays on a light paper

`--color-paper`, `--color-on-paper` and `--color-paper-link` style the `srcdoc` iframe page and the composer's
editing area. Every starter theme, dark ones included, sets them to a light page with dark ink. HTML mail is
third-party content written for a white page; inverting it produces unreadable mail, and what the user composes is
sent to recipients who see it on white. The chrome around the paper (headers, attachments bar, toolbar) follows the
theme, and plain-text messages, which are rendered by the app, use the theme's surface and text roles.

*Alternative considered:* a dark paper with a color filter over the iframe. It breaks images and brand colors and is
a known source of bug reports in other clients.

### D10. Colors that are content are allowlisted, not tokenized

The component lint rejects literal colors and palette utilities outside the contract. Three kinds of color are data
and are listed, with the reason, in an allowlist next to the test:

- the composer's text-color and highlight swatches (colors the user applies to the document);
- the inline colors of the quote header and `blockquote` that `ComposerModal.vue` builds for replies: this markup is
  sent as `body_html`, and the recipient's mail client has none of our custom properties, so it must keep literal
  colors and must never reference a role;
- calendar colors chosen by the user and stored per calendar, with their literal fallback.

The file-type icon tints in `extColor()` move to status roles (`err`, `ok`, `accent`) and need no entry.

A user-chosen calendar color is also a background: event labels are drawn over it in fixed white today, which is
unreadable on a light calendar color in any theme. The label ink becomes black or white, whichever contrasts more
with the calendar color (one small helper), so the contrast guarantee does not stop at the allowlist.

An allowlist entry names a file and a pattern; an entry that no longer matches anything fails the test, so the list
cannot rot.

### D11. Tests run under Vite+'s test runner

`frontend/tests/unit/themelib.mjs` (helpers) and `frontend/tests/unit/themes.test.mjs` run with `vp test` (the
runner Vite+ already ships: imports from `vite-plus/test`) and the installed `postcss`, through `npm test` and
`make test-frontend`. This is the layout ad-manager uses, so its helpers port with few edits:

1. theme files: only `:root[data-theme]` rules declaring known roles;
2. role types: a color role holds a color, a shadow role a shadow, a radius role a length;
3. component lint: no literal color, palette utility, `rounded-*` or `border-radius` outside the contract, except
   the allowlist;
4. registry parity and scheme (D5), and every dark theme declares every color role (D3);
5. contrast: `frontend/src/themes/contrast.json` lists recipes `{id, fg, bg[], size}` (a text role over the stack of
   backgrounds where it appears). Colors with alpha are composited over the stack. Normal text needs 4.5:1, large
   text and interface components 3:1; themes listed under `aaa` need 7:1 and 4.5:1.

The tests read source files only; they need no browser and no backend.

### D12. The picker is a plain list

Settings gains an "Appearance" tab: one row per theme with its label and a light/dark tag, plus "Use the server
default". Choosing a row applies the theme at once and saves it with one `PUT`; if the save fails the previous theme
(and the cached one) is restored and a toast explains. The existing `saveSettings()` swallows errors and returns
nothing, so the tab uses a theme save that reports success or failure, and the list is disabled while a save is in
flight so two choices cannot race. No thumbnails: a faithful thumbnail needs each theme's values outside the
cascade, which the `:root`-scoped files do not offer, and the live preview is the whole application.

### D13. Outlook and Apple Mail are "inspired by"

Both use public system colors and system font stacks only (`"Segoe UI", ...` and `-apple-system, ...`): no logos,
product names in the interface beyond the theme label, icons or bundled fonts. Each theme file's header comment
names its reference and every value changed for contrast, as ad-manager's files do. The docs carry a
"no affiliation" note.

### D14. One focus recipe for every control

A single `:focus-visible` rule in `style.css` draws `--shadow-focus` on buttons, fields, checkboxes, list rows, menu
items and the theme list; the per-field `focus:ring-*` utilities are removed. Larry's value reproduces today's field
ring, so fields look the same and buttons gain a focus mark they lacked (a deliberate, visible accessibility fix,
applied after the gate of D2). High contrast sets a solid 3 px yellow ring. The contrast matrix has a recipe for
the focus color against each background a focused control sits on (3:1).

## Risks / Trade-offs

- **Tokenizing touches most components** → It lands first, alone, with the computed-style dump as its gate (D2), so
  a regression cannot hide behind a new theme.
- **Removing `!important` exposes radii the reset was hiding** (native controls, `<dialog>`, file inputs) → The lint
  covers our own CSS; the dump covers computed radius on every element, native ones included.
- **A role is missing for a look someone wants** → Add the role with Larry's value as default; existing themes are
  unaffected. The contract is expected to grow by a few roles while the first themes are written.
- **Reference colors fail contrast** (system blues on white, secondary labels) → Adjust lightness, keep hue, note the
  original in the file. The test decides, not the reference.
- **Larry cannot be both pixel-identical and AA** → Identity is the gate for tokenizing only; the corrections are a
  separate, listed step (D2).
- **Dark theme, white message paper** → Intended (D9); the reading pane's paper gets the theme's line around it so
  it reads as a sheet. A dark paper for plain-text mail already follows from the roles.
- **`localStorage` blocked or cleared** → The theme still comes from the server after login; only the pre-paint
  cache is lost.
- **Two lists drifting (Go registry, CSS blocks)** → Parity test (D5).
- **The first theme of a scheme hides a hard-coded light assumption** (an icon drawn in dark ink, a white box) →
  Task group 7 walks every screen in a dark theme before the change is called done.

## Migration Plan

1. Ship groups 1 and 2 (tokenize) on their own: no visible change, easy to revert.
2. Ship the two visible fixes to the base look (group 3) and the registry, the endpoint, the setting and the picker
   with only `larry` selectable (groups 4 and 5).
3. Add the tests, Larry's contrast corrections and the themes (group 6).

Deployment follows the usual order for a schema change: run `go-cubemail migrate` (it adds the `theme` column), then
start the new binary. Without the migration the settings endpoints fail (the upsert and the create-on-first-read
both write every column), so the migration is mandatory, as for earlier schema changes. Existing users have an
empty `theme` and keep Larry unless the administrator sets another `ui.theme`.

Rollback: an older binary ignores the extra column and the cached `gorc_theme` key; nothing has to be undone.

## Open Questions

- **Starter set.** Nine themes are proposed. Is the ported set right (`breeze`, `breeze-dark`, `graphite`,
  `high-contrast`), or should others come in the first round? The review of this proposal suggested deferring the
  three Breeze/Graphite ports: they are not needed to prove the contract, and each theme adds a full manual
  walkthrough in task group 7.
- **`compose_html` cannot be saved as `false`** (existing bug, see D6). Not caused by this change and not fixed by
  it; it should get its own fix.
- **Server-rendered default.** Is the first-visit limit in D7 acceptable for deployments whose default is not Larry?
