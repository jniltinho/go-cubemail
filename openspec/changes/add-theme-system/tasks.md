## 1. Contract (no visible change)

- [ ] 1.1 Write `frontend/scripts/style-dump.js`: a console snippet that returns, for every element of the current
      screen, its color, background, border colors, radius, box-shadow, outline color and font family as JSON.
- [ ] 1.2 Start the app against the real mailbox and save the baseline dumps of: login, mail list with a message
      open (HTML and plain text), composer, contacts, calendar, settings (both tabs), a confirm dialog, a toast and
      the folder context menu.
- [ ] 1.3 Inventory every color use (literals, palette utilities, `white`/`black`, arbitrary shadows, `var(--accent*)`)
      in `frontend/src` and assign each to a role of design D1; add a role only where a use has no fitting one.
- [ ] 1.4 Rewrite the `@theme` and `:root` blocks of `frontend/src/style.css` as the contract of D1 (`@theme static`),
      with Larry's values as defaults, and group the roles with one comment line per group.
- [ ] 1.5 Rename the existing utilities in every component (`text-ink` → `text-fg`, `text-ink-sub` → `text-fg-muted`,
      `text-ink-mute` → `text-fg-subtle`, `bg-panel-2` → `bg-surface-2`, `bg-app-bg` → `bg-page`) and remove the old
      names from the contract.
- [ ] 1.6 Assign each `accent`, `accent-2`, `accent-bar`, `accent-soft` utility and `var(--accent*)` use to the role
      of its job (header, action, selection, link, focus or accent).

## 2. Components on roles (no visible change)

- [ ] 2.1 Replace the literals in the component classes of `style.css` (`.tbtn`, `.side-item`, `.mail-row`,
      `.checkbox`, `.kebab-menu`, `.kmi`, `.attachments`, `.modal-wrap`, `.composer-*`, `.rte-*`, `.search-box`,
      `.cal-*`, `.login-*`, `.kbd`, scrollbars) with roles.
- [ ] 2.2 Delete the dead classes `.gw-tweaks-*`, `.gw-tweak-*` and `.composer-rich .tox*`.
- [ ] 2.3 Replace the literals and `white`/`black` utilities in the components: `AppBar`, `AppSidebar`, `AppToolbar`,
      `MailList`, `ReadingPane`, `ComposerModal`, `RichTextToolbar` (chrome only), `LoginView`,
      `EnvelopeIllustration`, `SourceViewer`, `ContactsPane`, `ContactModal`, `CalendarPane`, `CalendarSidebar`,
      `EventEditorModal`, `SettingsPane`, `DialogModal`, `ToastContainer`, `SpinnerIcon`, `FolderRow`.
- [ ] 2.4 Put the message paper on roles: the `srcdoc` page style in `ReadingPane.vue` and `.rte-content` use
      `--color-paper`, `--color-on-paper` and `--color-paper-link` (the iframe does not inherit the page's custom properties, so the
      `srcdoc` string gets the resolved values, recomputed when the theme changes).
- [ ] 2.5 Move `extColor()` in `frontend/src/utils/helpers.ts` to role utilities.
- [ ] 2.6 Remove `* { border-radius: 0 !important }`; add `--radius-control`, `--radius-surface`, `--radius-row`,
      `--row-inset-x` and `--selection-mark-w`; use them in the component classes; replace or remove each of the 28
      `rounded*` utilities.
- [ ] 2.7 Remove `applyAccent()`, the `accent` value and its `watchEffect` from the mail store, and the store's
      export of `accent`.
- [ ] 2.8 Repeat the dumps of 1.2 and compare with the baseline: no difference. Run `npm run check` (format, lint,
      type-check) and `npm run build`.

## 3. Registry and API

- [ ] 3.1 Add `internal/theme/theme.go`: the ordered registry (id, label, scheme), `List`, `ByID` (falls back to the
      first entry) and `Valid`; table test for lookup and fallback.
- [ ] 3.2 Validate `ui.theme` in `internal/config/config.go`: an unknown id logs a warning with the valid ids and
      becomes `larry`; test both cases.
- [ ] 3.3 Add the public `GET /api/v1/themes` (`internal/handler/themes.go`, wired in `handler.go` and
      `routes.go` next to `/version`) with Swagger annotations; handler test for the payload and for access without
      a session.
- [ ] 3.4 Add `Theme string` to `model.UserSettings` (already in the AutoMigrate list of `cmd/migrate.go`); confirm
      `go-cubemail migrate` adds the column on SQLite and that existing rows read as empty.
- [ ] 3.5 Add `theme` to `settingsRequest` and `settingsResponse` in `internal/handler/settings.go`: validate
      against the registry before any field is applied (unknown → `400`, nothing saved), `""` clears, a stored id
      that is no longer registered is returned as `""`; handler tests for each scenario of the
      `theme-preference` spec, including isolation between two users.
- [ ] 3.6 Regenerate the Swagger docs (`make swagger`) and run `make test`.

## 4. Applying and choosing a theme

- [ ] 4.1 Add `frontend/src/utils/theme.ts`: `applyTheme(theme)` sets `data-theme` and `color-scheme` on the root,
      updates the `color-scheme` meta and caches `{id, scheme}` under `gorc_theme`; `resolveTheme(id, registry,
      defaultId)` returns the registered theme or the default.
- [ ] 4.2 Add `frontend/public/theme-boot.js` (classic script, `try`/`catch`, reads `gorc_theme`, sets the two
      attributes) and load it in `<head>` of `frontend/index.html` before the stylesheet; add the `color-scheme`
      meta. Confirm in the built `web/dist/index.html` that the script tag survives the build and that the browser
      reports no CSP violation.
- [ ] 4.3 Load `/api/v1/themes` in `checkSession()` (`frontend/src/stores/auth.ts`) and keep the registry and the
      default in the store; on the login screen apply the cached theme or the default.
- [ ] 4.4 Add `theme` to `UserSettings` in `frontend/src/types.ts` and to `DEFAULT_SETTINGS`; after
      `fetchSettings()` apply the user's theme (or the default when empty).
- [ ] 4.5 Add the Appearance tab to `SettingsPane.vue`: the list of themes with label and scheme tag, the active
      entry marked, "Use the server default", keyboard-operable (`role="radiogroup"`); choosing applies at once,
      saves with one `PUT`, and restores the previous theme with an error toast if the save fails.
- [ ] 4.6 Update the CSP comment in `internal/server/middleware/security_headers.go` (it mentions the accent
      picker); the policy itself does not change.

## 5. Themes

- [ ] 5.1 `frontend/src/themes/outlook.css`: `outlook` and `outlook-dark` (Fluent 2 neutrals and brand blue, 4 px
      radius); header comment with the reference and every value changed for contrast.
- [ ] 5.2 `frontend/src/themes/apple-mail.css`: `apple-mail` and `apple-mail-dark` (macOS system colors, starting
      from ad-manager's `crystal.css` values; 6/10/6 px radii, inset selected row, `-apple-system` stack).
- [ ] 5.3 `frontend/src/themes/breeze.css`: `breeze` and `breeze-dark`, ported from ad-manager's `breeze.css`; note
      the upstream file and commit in the header.
- [ ] 5.4 `frontend/src/themes/graphite.css`: `graphite`, ported from ad-manager's `graphite.css`.
- [ ] 5.5 `frontend/src/themes/high-contrast.css`: `high-contrast`, ported from ad-manager's `high-contrast.css`
      (3 px focus ring through `--shadow-focus`, no depth shadows).
- [ ] 5.6 Import the five files in `style.css` and add the eight entries to the registry in the spec's order.

## 6. Theme tests

- [ ] 6.1 Add `frontend/tests/unit/themelib.mjs` (contract parser, theme-file parser, role types, color parsing,
      compositing and WCAG ratio), adapted from ad-manager's `tests/unit/themelib.mjs`.
- [ ] 6.2 Add `frontend/src/themes/contrast.json`: the recipes (text and component roles over the backgrounds where
      they appear: page, surface, surface-2, row-hover, selection, header, tab-on, sidebar, field, action, action-2,
      danger, status fills, paper) and the `aaa` list with `high-contrast`.
- [ ] 6.3 Add `frontend/tests/unit/themes.test.mjs` (imports from `vite-plus/test`): theme-file structure, role types, component lint with the allowlist
      (composer swatches, calendar colors) and its stale-entry check, registry parity and scheme, complete dark
      themes, contrast. Include one negative case per rule so each check is known to catch its failure.
- [ ] 6.4 Add the `test` script (`vp test`) to `frontend/package.json` and a `test-frontend` target (`npm test`) to
      the `Makefile`; run `npm test` and `npm run check`; fix every theme value the contrast test rejects, noting the original value in the file.

## 7. Verification in the running app

- [ ] 7.1 Build (`make frontend build`), start on localhost against the real mailbox and, in each of the nine
      themes, walk: login, mail list (read, unread, starred, selected, hover, multi-select), reading pane (HTML,
      plain text, attachments, calendar invitation), composer with the toolbar's menus, contacts with its modal,
      calendar with the event editor, settings (three tabs), confirm/prompt dialogs, each toast kind, the folder
      context menu and drag-over state, the source viewer.
- [ ] 7.2 In every dark theme look for leftovers of a light assumption: white boxes, dark icons on dark, native
      controls and scrollbars that stayed light.
- [ ] 7.3 Check reload in a dark theme (no light first frame), sign-out and sign-in, a second browser profile, and
      storage blocked.
- [ ] 7.4 Check `ui.theme = "outlook"` with a fresh user and with a user who already chose a theme.

## 8. Documentation

- [ ] 8.1 Write `DOCUMENTS/docs/THEMES.md`: the role contract by group, the theme file format, how to add a theme
      (file, import, registry line, contrast run), the catalog table with references, and the "no affiliation" note.
- [ ] 8.2 Update `DOCUMENTS/docs/SDD.md`: section 3 (`ui.theme` values), 4 (`frontend/src/themes/`,
      `internal/theme/`), 6 (`GET /api/v1/themes`), 7 (`UserSettings.Theme`), 9.5 (roles instead of the accent
      tokens), 11.1 (themes instead of the accent picker) and 12 (CSP note).
- [ ] 8.3 Update `config.toml.example` (valid `ui.theme` ids) and the feature list in `README.md`.
