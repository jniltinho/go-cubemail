# Spec Delta

## Purpose

Covers how a theme is chosen: the public list of themes, the server's default, each user's saved choice, the
Appearance settings, and how the choice survives reloads, sign-ins and other browsers.

## ADDED Requirements

### Requirement: Public theme list
The system SHALL serve the theme registry and the server's default theme at `GET /api/v1/themes` without a session,
as `{"default": "<id>", "themes": [{"id": "...", "label": "...", "scheme": "light|dark"}]}` in registry order.

#### Scenario: List before signing in
- **WHEN** a client without a session requests `GET /api/v1/themes`
- **THEN** the response is `200` with every registered theme in registry order and the default theme's id

### Requirement: Server default theme
The configuration key `ui.theme` SHALL set the default theme for users who have not chosen one and for the login
screen. A value that is not a registered id SHALL fall back to `larry`.

#### Scenario: Configured default
- **WHEN** `ui.theme` is `outlook` and a user who never chose a theme signs in
- **THEN** the interface uses the Outlook theme

#### Scenario: Unknown default
- **WHEN** `ui.theme` is `nope`
- **THEN** the server starts, logs a warning that names the value and the valid ids, and serves `larry` as the default

### Requirement: Per-user theme setting
`GET /api/v1/settings` SHALL return the user's `theme`: a registered id, or an empty string when the user follows
the server default. `PUT /api/v1/settings` SHALL accept an optional `theme` and store it for that user only.

#### Scenario: Save a theme
- **WHEN** a signed-in user sends `PUT /api/v1/settings` with `{"theme": "apple-mail"}`
- **THEN** the response is `200` with `theme` `apple-mail`, and a later `GET /api/v1/settings` by the same user
  returns `apple-mail`

#### Scenario: Follow the server default
- **WHEN** a user sends `PUT /api/v1/settings` with `{"theme": ""}`
- **THEN** the stored choice is cleared and `GET /api/v1/settings` returns an empty `theme`

#### Scenario: Unknown theme
- **WHEN** a user sends `PUT /api/v1/settings` with `{"theme": "nope"}`
- **THEN** the response is `400` with an error that names the value, none of the user's settings change, and no
  settings record is created for a user who had none

#### Scenario: Request without the field
- **WHEN** a user sends `PUT /api/v1/settings` without `theme`
- **THEN** the user's stored theme is unchanged

#### Scenario: Other users are unaffected
- **WHEN** one user saves a theme
- **THEN** another user's `GET /api/v1/settings` returns that other user's own value

#### Scenario: Stored theme no longer exists
- **WHEN** a user's stored theme id is not in the registry any more
- **THEN** `GET /api/v1/settings` returns an empty `theme` and the user sees the server default

### Requirement: Appearance settings
The Settings view SHALL have an Appearance tab that lists every registered theme with its label and scheme, marks
the active one, and offers "Use the server default". Choosing an entry SHALL apply it immediately and save it.

#### Scenario: Choose a theme
- **WHEN** the user opens Settings, selects Appearance and chooses "Outlook (dark)"
- **THEN** the whole interface switches to that theme without a reload, the entry is marked as active and the choice
  is saved to the user's settings

#### Scenario: Save fails
- **WHEN** the user chooses a theme and the save request fails
- **THEN** the interface and the theme remembered by the browser return to the previous theme, and an error message
  is shown

#### Scenario: Choosing while a save is in flight
- **WHEN** the user has chosen a theme and its save has not finished
- **THEN** the theme list does not accept another choice until it finishes

#### Scenario: Use the server default
- **WHEN** the user chooses "Use the server default"
- **THEN** the interface switches to the server's default theme and the user's stored choice is cleared

#### Scenario: Keyboard use
- **WHEN** the user moves through the theme list with the keyboard
- **THEN** each entry can be focused and chosen with the keyboard, and the active entry is announced as selected

### Requirement: Theme persists across reloads without a flash
The browser SHALL remember the last applied theme and apply it before the first paint of the next page load, using
only script files served by the application itself.

#### Scenario: Reload in a dark theme
- **WHEN** a user with a dark theme reloads the page
- **THEN** the first paint is already dark, and no light frame is shown before the application starts

#### Scenario: Content Security Policy is respected
- **WHEN** the page loads with the application's Content Security Policy
- **THEN** the theme is applied with no inline script and no policy violation is reported

#### Scenario: Storage unavailable
- **WHEN** the browser blocks local storage
- **THEN** the page first paints with the Larry look, switches to the server default once the theme list has loaded,
  shows no error, and applies the user's theme after sign-in

### Requirement: Theme on the login screen and across browsers
The login screen SHALL use the last theme applied in that browser, or the server default when there is none. After
sign-in, the user's saved theme SHALL replace it.

#### Scenario: First visit
- **WHEN** a browser with no remembered theme opens the login screen and the server default is `outlook`
- **THEN** the login screen first paints with the Larry look and uses the Outlook theme once the theme list has
  loaded

#### Scenario: Sign in on another browser
- **WHEN** a user who saved `graphite` signs in on a browser that remembers another theme
- **THEN** the interface switches to Graphite after sign-in and that browser now remembers Graphite

#### Scenario: Sign out
- **WHEN** a user signs out
- **THEN** the login screen keeps the theme that was active
