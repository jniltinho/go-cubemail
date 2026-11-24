## Why

The webmail has one hard-wired look (Larry: navy, square corners). Its colors are spread over three places: a
`@theme` block of literal values in `frontend/src/style.css`, about 48 literals inside that file's component classes
and about 130 hex literals plus 60 `white`/`black` utilities in the components. Nothing can be re-skinned without
editing components, there is no dark look, and the "accent picker" the SDD describes is dead code: `applyAccent()`
recolors six variables from a store value no component can change.

The sibling project `ad-manager` already solved this with a role-token contract, one CSS file per theme, a theme
registry in Go and tests that keep every theme legible. This change brings that model to go-cubemail and ships the
two looks mail users know best, Outlook and Apple Mail.

## What Changes

- **Role-token contract.** `frontend/src/style.css` declares every color, radius, shadow and font the interface uses
  as a named role (`--color-fg`, `--color-surface`, `--color-header`, `--color-selection`, `--radius-control`, ...).
  The contract's default values are Larry's, so Larry is the default theme and looks exactly as it does today.
  Role names follow ad-manager's vocabulary where the two interfaces overlap, so its palettes port with few edits;
  mail-only roles are added (sidebar, unread mark, star, message paper).
- **Components use roles only.** Literal colors and `white`/`black` utilities leave the components and the component
  classes of `style.css`. A short, justified allowlist keeps the colors that are content, not interface: the
  composer's text and highlight swatches, user-chosen calendar colors and the message paper.
- **One file per theme family** under `frontend/src/themes/`, holding only `:root[data-theme='id'] { --role: value }`
  blocks: no selectors, no component rules.
- **Starter catalog of nine themes:** `larry` (default), `outlook`, `outlook-dark`, `apple-mail`, `apple-mail-dark`,
  and four ported from ad-manager: `breeze`, `breeze-dark`, `graphite` (dark) and `high-contrast` (dark, WCAG AAA).
  Outlook and Apple Mail are "inspired by": no logos, trademarks, proprietary fonts or icons.
- **Corner radius becomes three roles** (control, surface, row) that default to `0`, replacing the global
  `border-radius: 0 !important` reset. Larry stays square; Outlook and Apple Mail round their corners.
- **Theme registry in Go** (id, label, scheme) as the single list, served by a new public endpoint
  `GET /api/v1/themes` together with the server default (`ui.theme`, already read from the config and unused today).
- **Per-user theme preference:** a new `theme` field in `GET`/`PUT /api/v1/settings`, stored in a new
  `UserSettings.Theme` column, validated against the registry. An empty value follows the server default.
- **Appearance tab in Settings:** the list of themes; choosing one applies it at once and saves it.
- **No theme flash on reload:** the last theme used in the browser is cached and applied by a small script file in
  `<head>` before first paint (an external file: the CSP forbids inline scripts).
- **BREAKING (internal only):** `applyAccent()`, the `accent` store value and the unused `.gw-tweaks-*` classes are
  removed; the theme owns the accent. No user-visible feature is lost, because no component could change the accent.
- **Message content keeps a light "paper"** in every theme: the HTML message iframe and the composer's editing area
  are not recolored, so third-party HTML and outgoing mail look as their authors intended.
- **Theme tests** with no new dependency (Vite+'s `vp test` and the `postcss` already installed): theme-file structure,
  component color lint, registry and CSS in step, role types and a WCAG contrast matrix (AA for every theme, AAA for
  `high-contrast`).
- **Docs:** a new `DOCUMENTS/docs/THEMES.md` (contract, how to add a theme) and corrections to the SDD.

Out of scope, each a follow-up of one CSS file once the contract exists: the other ad-manager themes (Violet, Aurora,
Crystal, Cone, Classic blue/gray, Grafana, RabbitMQ, Matrix, Winamp, Pink Panther), a "follow the system" light/dark
switch, per-theme layout or density changes, theme thumbnails in the picker, a flat-look switch and user-defined
themes.

## Capabilities

### New Capabilities

- `theming`: the role-token contract, the theme file format, how a theme is applied to the page, the radius roles,
  the light paper for message content and the checks that keep every theme legible.
- `theme-catalog`: the themes that ship, one requirement per theme, with its scheme, reference look and contrast
  level.
- `theme-preference`: the registry endpoint, the server default, the per-user setting, the Appearance tab and how
  the choice survives reloads, logins and other devices.

### Modified Capabilities

None. The repository has no specs yet; this change adds the first three.

## Impact

- **Frontend, new:** `frontend/src/themes/{outlook,apple-mail,breeze,graphite,high-contrast}.css`,
  `frontend/src/themes/contrast.json`, `frontend/src/utils/theme.ts`, `frontend/public/theme-boot.js`,
  `frontend/tests/unit/{themelib,themes.test}.mjs`, `frontend/scripts/style-dump.js`.
- **Frontend, changed:** `frontend/src/style.css` (contract, component classes, radius reset, dead classes removed),
  `frontend/index.html` (boot script, `color-scheme` meta), `frontend/src/main.ts`, every component with a literal
  color (`RichTextToolbar`, `ReadingPane`, `LoginView`, `AppBar`, `EnvelopeIllustration`, `ComposerModal`,
  `SourceViewer`, `ContactsPane`, `ContactModal`, `CalendarPane`, `CalendarSidebar`, `ToastContainer`,
  `DialogModal`, `SpinnerIcon`) and every component using the renamed utilities, `SettingsPane.vue` (Appearance
  tab), `frontend/src/utils/helpers.ts` (`applyAccent` removed, `extColor` on roles),
  `frontend/src/stores/mail/index.ts` (`accent` removed), `frontend/src/stores/mail/settingsActions.ts`,
  `frontend/src/stores/auth.ts`, `frontend/src/types.ts`, `frontend/package.json` (`test` script).
- **Backend, new:** `internal/theme/theme.go` (registry) with its test, `internal/handler/themes.go`.
- **Backend, changed:** `internal/model/user_settings.go` (`Theme` column, added by AutoMigrate),
  `internal/handler/settings.go`, `internal/handler/handler.go`, `internal/server/routes.go`,
  `internal/config/config.go` (validate `ui.theme` at start), `internal/server/middleware/security_headers.go`
  (comment only: the CSP itself does not change), `docs/` (regenerated Swagger).
- **Build:** `Makefile` (`test-frontend` target).
- **Docs:** `DOCUMENTS/docs/THEMES.md` (new), `DOCUMENTS/docs/SDD.md` (sections 3, 4, 6, 7, 9.5, 11.1, 12),
  `config.toml.example`, `README.md`.
- **API:** one new public route; one new optional field in the settings payload. Existing clients keep working.
- **Data:** one new column, empty by default, added by `go-cubemail migrate` (required before the new binary
  serves); existing rows read as "follow the server default". No data migration.
- **Dependencies:** none added.
