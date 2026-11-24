## 1. Contract (no visible change)

- [x] 1.1 Write `frontend/scripts/style-dump.js`: a console snippet that returns as JSON, for every element of the
      current screen and for its `::before`/`::after`, the color, background, border colors, radius, box-shadow,
      outline color, font family and the SVG `fill`, `stroke` and `stop-color`.
- [ ] 1.2 Start the app against the real mailbox and save the baseline dumps of: login, the loading screen, mail list
      with a message open (HTML and plain text), composer (rich text and HTML source), contacts, calendar with an
      event, settings (both tabs), a confirm dialog, a toast and the folder context menu.
- [x] 1.3 Inventory every color use (literals, palette utilities such as `gray-*`, `white`/`black`, arbitrary shadows,
      `var(--accent*)`) in `frontend/src` and assign each to a role of design D1; add a role only where a use has no
      fitting one.
- [x] 1.4 Rewrite the `@theme` and `:root` blocks of `frontend/src/style.css` as the contract of D1 (`@theme static`),
      with Larry's values as defaults, and group the roles with one comment line per group.
- [x] 1.5 Rename the existing utilities in every component (`text-ink` → `text-fg`, `text-ink-sub` → `text-fg-muted`,
      `text-ink-mute` → `text-fg-subtle`, `bg-panel-2` → `bg-surface-2`, `bg-app-bg` → `bg-page`) and remove the old
      names from the contract.
- [x] 1.6 Assign each `accent`, `accent-2`, `accent-bar`, `accent-soft` utility and `var(--accent*)` use to the role
      of its job (header, action, selection, link, focus or accent).

## 2. Components on roles (no visible change)

- [x] 2.1 Replace the literals in the component classes of `style.css` (`.tbtn`, `.side-item`, `.mail-row`,
      `.checkbox`, `.kebab-menu`, `.kmi`, `.attachments`, `.modal-wrap`, `.composer-*`, `.rte-*`, `.search-box`,
      `.cal-*`, `.login-*`, `.kbd`, scrollbars) with roles.
- [x] 2.2 Delete the dead classes `.gw-tweaks-*`, `.gw-tweak-*` and `.composer-rich .tox*`.
- [x] 2.3 Replace the literals and the `white`/`black`/`gray-*` utilities in `App.vue` (loading screen) and in the
      components: `AppBar`, `AppSidebar`, `AppToolbar`, `MailList`, `ReadingPane`, `ComposerModal` (chrome only: the
      reply quote markup keeps its literals, D10), `editor/RichTextToolbar` (chrome only), `editor/TipTapEditor`
      (old text tokens, the white source textarea → paper roles), `LoginView`, `EnvelopeIllustration`,
      `SourceViewer`, `ContactsPane`, `ContactModal`, `CalendarPane`, `CalendarSidebar`, `EventEditorModal`,
      `SettingsPane`, `DialogModal`, `ToastContainer`, `SpinnerIcon`, `FolderRow`.
- [x] 2.4 Put the message paper on roles: the `srcdoc` page style in `ReadingPane.vue` and `.rte-content` use
      `--color-paper`, `--color-on-paper` and `--color-paper-link` (the iframe does not inherit the page's custom
      properties, so the `srcdoc` string gets the resolved values, recomputed when the theme changes).
- [x] 2.5 Move `extColor()` in `frontend/src/utils/helpers.ts` to role utilities.
- [x] 2.6 Remove the three radius resets (`* { border-radius: 0 !important }`, `button, input, select, textarea
      { border-radius: 0 }` and `.rte-toolbar .tbtn`'s `!important`); add `--radius-control`, `--radius-surface`,
      `--radius-row`, `--row-inset-x` and `--selection-mark-w`; use them in the component classes and on native
      controls; replace or remove each of the 28 `rounded*` utilities.
- [x] 2.7 Remove `applyAccent()`, the `accent` value and its `watchEffect` from the mail store, and the store's
      export of `accent`.
- [x] 2.8 Repeat the dumps of 1.2 and compare with the baseline: no difference. Run `npm run check` (format, lint,
      type-check) and `npm run build`.

## 3. Visible fixes to the base look

- [x] 3.1 Add the single `:focus-visible` recipe of D14 to `style.css` (buttons, fields, checkboxes, list rows, menu
      items, tabs) on `--shadow-focus`, and remove the per-field `focus:ring-*` utilities (`SettingsPane`,
      `CalendarSidebar`, `EventEditorModal`, `ContactModal`, `DialogModal`'s scoped rule).
- [x] 3.2 Draw event labels in black or white by the calendar color's luminance (one helper in
      `frontend/src/utils/helpers.ts`, used by `CalendarPane.vue` and `.cal-evt`); unit test with white, pale yellow
      and navy.

## 4. Registry and API

- [x] 4.1 Add `internal/theme/theme.go`: the ordered registry (id, label, scheme), `List`, `ByID` (falls back to the
      first entry) and `Valid`; table test for lookup and fallback.
- [x] 4.2 Validate `ui.theme` in `internal/config/config.go`: an unknown id logs a warning with the valid ids and
      becomes `larry`; test both cases.
- [x] 4.3 Add the public `GET /api/v1/themes` (`internal/handler/themes.go`, wired in `handler.go` and
      `routes.go` next to `/version`) with Swagger annotations; handler test for the payload and for access without
      a session.
- [x] 4.4 Add `Theme string` to `model.UserSettings` with **no** `default:` tag (design D6; the model is already in
      the AutoMigrate list of `cmd/migrate.go`); confirm `go-cubemail migrate` adds the column on SQLite and that
      existing rows read as empty.
- [x] 4.5 Add `theme` to `settingsRequest` and `settingsResponse` in `internal/handler/settings.go`: bind and
      validate the body before loading the user's settings (unknown → `400`, nothing saved or created), `""` clears,
      a stored id that is no longer registered is returned as `""`; handler tests for each scenario of the
      `theme-preference` spec, including isolation between two users, an invalid request from a user with no
      settings row, and a save that sets and then clears the theme through `SettingsRepo.Save`.
- [x] 4.6 Regenerate the Swagger docs (`make swagger`) and run `make test`.

## 5. Applying and choosing a theme

- [x] 5.1 Add `frontend/src/utils/theme.ts`: `applyTheme(theme)` sets `data-theme` and `color-scheme` on the root,
      updates the `color-scheme` meta and caches `{id, scheme}` under `gorc_theme`; `resolveTheme(id, registry,
      defaultId)` returns the registered theme or the default.
- [x] 5.2 Add `frontend/public/theme-boot.js` (classic script, `try`/`catch`, reads `gorc_theme`, sets the two
      attributes) and load it in `<head>` of `frontend/index.html` before the stylesheet; add the `color-scheme`
      meta. Confirm in the built `web/dist/index.html` that the script tag survives the build, that the Go static
      handler serves `/theme-boot.js` (not under `/assets/`, so not cached as immutable) and that the browser reports
      no CSP violation.
- [x] 5.3 Load `/api/v1/themes` in `checkSession()` (`frontend/src/stores/auth.ts`) and keep the registry and the
      default in the store; on the login screen apply the cached theme or the default.
- [x] 5.4 Add `theme` to `UserSettings` in `frontend/src/types.ts`, to `DEFAULT_SETTINGS` in `settingsActions.ts` and
      to the `prefs` initializer in `SettingsPane.vue`; after `fetchSettings()` apply the user's theme (or the
      default when empty).
- [x] 5.5 Add a theme save to `frontend/src/stores/mail/settingsActions.ts` that returns whether it succeeded (the
      existing `saveSettings()` swallows the error).
- [x] 5.6 Add the Appearance tab to `SettingsPane.vue`: the list of themes with label and scheme tag, the active
      entry marked, "Use the server default", keyboard-operable (`role="radiogroup"`); choosing applies at once and
      saves; the list is disabled while the save is in flight; on failure the previous theme and cache are restored
      and an error toast is shown.
- [x] 5.7 Update the CSP comment in `internal/server/middleware/security_headers.go` (it mentions the accent
      picker); the policy itself does not change.

## 6. Themes and theme tests

- [x] 6.1 Add `frontend/tests/unit/themelib.mjs` (contract parser, theme-file parser, role types, color parsing,
      compositing and WCAG ratio), adapted from ad-manager's `tests/unit/themelib.mjs`.
- [x] 6.2 Add `frontend/src/themes/contrast.json`: the recipes (text, focus and component roles over the backgrounds
      where they appear: page, surface, surface-2, row-hover, selection, header, tab-on, sidebar, field, action,
      action-2, danger, status fills, paper) and the `aaa` list with `high-contrast`.
- [x] 6.3 Add `frontend/tests/unit/themes.test.mjs` (imports from `vite-plus/test`): theme-file structure, role
      types, component lint with the allowlist (composer swatches, reply quote markup, calendar colors) and its
      stale-entry check, registry parity and scheme, complete dark themes, contrast, and a check that the reply
      quote markup references no custom property. Include one negative case per rule so each check is known to
      catch its failure.
- [x] 6.4 Add the `test` script (`vp test`) to `frontend/package.json` and a `test-frontend` target (`npm test`) to
      the `Makefile`.
- [x] 6.5 Fix Larry's failing pairings (known: `--color-fg-subtle`) by the smallest lightness change, noting the
      original value beside the role.
- [x] 6.6 `frontend/src/themes/outlook.css`: `outlook` and `outlook-dark` (Fluent 2 neutrals and brand blue, 4 px
      radius); header comment with the reference and every value changed for contrast.
- [x] 6.7 `frontend/src/themes/apple-mail.css`: `apple-mail` and `apple-mail-dark` (macOS system colors, starting
      from ad-manager's `crystal.css` values; 6/10/6 px radii, inset selected row, `-apple-system` stack).
- [x] 6.8 `frontend/src/themes/high-contrast.css`: `high-contrast`, ported from ad-manager's `high-contrast.css`
      (3 px focus ring through `--shadow-focus`, no depth shadows); note the upstream file and commit.
- [x] 6.9 `frontend/src/themes/breeze.css` (`breeze`, `breeze-dark`) and `frontend/src/themes/graphite.css`
      (`graphite`), ported from ad-manager's files of the same names.
- [x] 6.10 Import the five files in `style.css` and add the eight entries to the registry in the spec's order; run
      `npm test` and `npm run check` until both pass.

## 7. Verification in the running app

- [ ] 7.1 Build (`make frontend build`), run `go-cubemail migrate`, start on localhost against the real mailbox and,
      in each of the nine themes, walk: login, mail list (read, unread, starred, selected, hover, multi-select),
      reading pane (HTML, plain text, attachments, calendar invitation), composer with the toolbar's menus and the
      HTML-source mode, contacts with its modal, calendar with the event editor, settings (three tabs),
      confirm/prompt dialogs, each toast kind, the folder context menu and drag-over state, the source viewer.
- [ ] 7.2 In every dark theme look for leftovers of a light assumption: white boxes, dark icons on dark, native
      controls and scrollbars that stayed light.
- [ ] 7.3 Walk each screen with the keyboard in `larry` and `high-contrast`: every control shows the focus
      indicator; a mouse click leaves none.
- [ ] 7.4 Send a reply from a dark theme to the test mailbox and confirm the received HTML has the usual quote
      colors and no custom property.
- [ ] 7.5 Check reload in a dark theme (no light first frame), sign-out and sign-in, a second browser profile, and
      storage blocked.
- [x] 7.6 Check `ui.theme = "outlook"` with a fresh user and with a user who already chose a theme.

## 8. Documentation

- [x] 8.1 Write `DOCUMENTS/docs/THEMES.md`: the role contract by group, the theme file format, how to add a theme
      (file, import, registry line, contrast run), the catalog table with references, and the "no affiliation" note.
- [x] 8.2 Update `DOCUMENTS/docs/SDD.md`: section 3 (`ui.theme` values), 4 (`frontend/src/themes/`,
      `internal/theme/`), 6 (`GET /api/v1/themes`), 7 (`UserSettings.Theme`), 9.5 (roles instead of the accent
      tokens), 11.1 (themes instead of the accent picker) and 12 (CSP note).
- [x] 8.3 Update `config.toml.example` (valid `ui.theme` ids), the feature list in `README.md` and the migrate step
      in the upgrade notes.

## Verification notes

What was checked while implementing, and what is still open in the unticked tasks:

- **1.2 / 2.8** The computed-style dumps were taken on localhost against a seeded in-memory IMAP mailbox, not the
  real one, on: login, mail list with an HTML, a plain-text and an attachment message open, composer (rich text),
  contacts, calendar and both settings tabs. Not dumped: the loading screen, the composer's HTML-source mode, a
  calendar with an event, a confirm dialog, a toast and the folder context menu. The only differences from the
  pre-change build are the ones listed under "Changes to the Larry look" in `DOCUMENTS/docs/THEMES.md`.
- **7.1 / 7.2** An in-page audit measured the rendered contrast of every text and icon on seven screens (mail with
  a message open, contacts, calendar, the three settings tabs, composer) in each of the nine themes; it found the
  active folder's icon (fixed) and the star in the light themes (the documented `aaaOnly` exception). The mail view
  was looked at in Outlook, Outlook dark, Apple Mail, Apple Mail dark and High contrast, the composer in Graphite
  and the login screen in Outlook dark. Not walked by hand: dialogs, toasts, the folder context menu and its
  drag-over state, the source viewer, a calendar invitation, the event editor, and hover states.
- **7.3** Keyboard focus was checked on the header and the toolbar in High contrast (3 px yellow ring, drawn inside
  the control on the header; none after a mouse click). Larry and the other screens were not walked.
- **7.4** Not done: the test setup had no SMTP server. The unit test covers that the quote markup holds no role.
- **7.5** Checked: reload in a dark theme starts dark, sign-out keeps the theme, sign-in applies the user's saved
  theme over the cached one. Not checked: a second browser profile and blocked storage.
