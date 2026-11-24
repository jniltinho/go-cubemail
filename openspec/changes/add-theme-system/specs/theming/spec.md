# Spec Delta

## Purpose

Defines how the webmail's look is described and applied: a contract of named roles for color, shape, depth and type,
theme files that only give values to those roles, and the checks that keep every theme legible.

## ADDED Requirements

### Requirement: Role-token contract
The interface SHALL take every color, corner radius, shadow and font family from a named role declared in one
contract in the main stylesheet. The contract's default values SHALL be the Larry theme's values.

#### Scenario: No theme selected
- **WHEN** the page loads with no theme attribute on the root element
- **THEN** the interface renders with the contract's defaults, which is the Larry look

#### Scenario: Default theme selected
- **WHEN** the root element's theme attribute is `larry`
- **THEN** the interface renders exactly as it does with no theme attribute

### Requirement: Components use roles only
Components and component classes SHALL NOT contain literal colors, color-palette utilities, literal corner radii or
radius utilities other than the role ones. Colors that are user content SHALL be exempt only through an allowlist
that names the file, the pattern and the reason.

#### Scenario: Literal color in a component
- **WHEN** a component gains a literal color such as `#1B3A6B` or a utility such as `bg-white`
- **THEN** the theme tests fail and name the file and the line

#### Scenario: Content colors are allowed
- **WHEN** the theme tests run over the composer's text-color swatches, the user-chosen calendar colors and the
  quote markup built for replies
- **THEN** those colors are accepted because the allowlist names them

#### Scenario: Stale allowlist entry
- **WHEN** an allowlist entry no longer matches anything in its file
- **THEN** the theme tests fail and name the entry

### Requirement: Theme file format
A theme SHALL be defined only by rules of the form `:root[data-theme='<id>']` that declare roles of the contract.
A theme file SHALL NOT contain any other selector, any property that is not a role, or a role the contract does not
declare.

#### Scenario: Component rule in a theme file
- **WHEN** a theme file contains a rule such as `.mail-row { color: red }`
- **THEN** the theme tests fail and name the file and the line

#### Scenario: Unknown role
- **WHEN** a theme file declares `--color-nope`
- **THEN** the theme tests fail and name the role

#### Scenario: Wrong kind of value
- **WHEN** a theme gives a gradient to a color role or a color to a radius role
- **THEN** the theme tests fail and name the role and the theme

### Requirement: Theme applied to the whole document
The active theme SHALL be set as the `data-theme` attribute of the root element, and the root's `color-scheme` SHALL
be the theme's scheme (`light` or `dark`), so that native controls and scrollbars follow the theme.

#### Scenario: Switch to a dark theme
- **WHEN** the active theme changes to a theme whose scheme is `dark`
- **THEN** the root element has that theme's id in `data-theme`, its `color-scheme` is `dark`, and every view,
  dialog, menu and toast uses the theme's roles without a reload

#### Scenario: Unknown theme id
- **WHEN** the theme to apply has an id that is not in the theme registry
- **THEN** the default theme is applied instead

### Requirement: Corner radius roles
Corner radius SHALL come from three roles: control (buttons, fields, checkboxes, chips), surface (dialogs, menus,
cards, toasts) and row (list rows and sidebar items). All three SHALL default to `0`.

#### Scenario: Larry stays square
- **WHEN** the Larry theme is active
- **THEN** every element of the interface has a computed corner radius of `0`

#### Scenario: A theme rounds its corners
- **WHEN** a theme sets the control radius to `4px`
- **THEN** buttons, fields and checkboxes have 4 px corners and elements of the other two groups keep theirs

### Requirement: Light paper for message content
HTML message bodies and the composer's editing area SHALL be drawn on the paper roles, and every theme SHALL set the
paper to a light background with dark text. The interface around the paper SHALL follow the theme.

#### Scenario: HTML message in a dark theme
- **WHEN** a dark theme is active and the user opens an HTML message
- **THEN** the message body is shown with a light background and its own colors, and the message headers, the
  attachments bar and the toolbar use the dark theme's roles

#### Scenario: Composing in a dark theme
- **WHEN** a dark theme is active and the user writes a message
- **THEN** the editing area, in rich-text and in HTML-source mode, has a light background with dark text, and the
  composer's header fields and toolbar use the dark theme's roles

#### Scenario: Plain-text message in a dark theme
- **WHEN** a dark theme is active and the user opens a message that has only a plain-text body
- **THEN** the text is shown with the theme's surface and text roles

### Requirement: Outgoing mail carries no theme
Markup the application adds to an outgoing message (the quote header and quote block of a reply or forward) SHALL
use fixed colors and SHALL NOT reference a role, so the message looks the same to the recipient whatever theme the
sender uses.

#### Scenario: Reply written in a dark theme
- **WHEN** a user with a dark theme replies to a message and sends it
- **THEN** the HTML body that is sent contains no reference to a custom property and the same quote colors as a
  reply sent with the Larry theme

### Requirement: Text over user-chosen colors
Text drawn over a color the user chose (a calendar's color behind an event label) SHALL use black or white ink,
whichever has the higher contrast with that color.

#### Scenario: Light calendar color
- **WHEN** a calendar's color is white or pale yellow and one of its events is shown
- **THEN** the event's label is drawn in black

#### Scenario: Dark calendar color
- **WHEN** a calendar's color is navy and one of its events is shown
- **THEN** the event's label is drawn in white

### Requirement: Visible keyboard focus
Every control that can take keyboard focus (buttons, fields, checkboxes, list rows, menu items, tabs) SHALL show a
focus indicator drawn from the theme's focus roles when it is focused with the keyboard, with at least 3:1 contrast
against the background it sits on.

#### Scenario: Tab to a toolbar button
- **WHEN** the user moves focus to a toolbar button with the Tab key, in any theme
- **THEN** the button shows the theme's focus indicator

#### Scenario: Mouse click
- **WHEN** the user clicks a button with the mouse
- **THEN** no focus indicator stays on it

### Requirement: Contrast of every theme
Every theme SHALL meet WCAG AA contrast in each listed pairing of a text or component role with the backgrounds it
appears on: 4.5:1 for normal text and 3:1 for large text and interface components. Themes marked as enhanced SHALL
meet 7:1 for normal text and 4.5:1 for large text.

#### Scenario: A theme fails a pairing
- **WHEN** a theme's muted text reaches only 3.9:1 on the surface role
- **THEN** the theme tests fail and name the theme, the pairing and the measured ratio

#### Scenario: Translucent colors
- **WHEN** a role's value has transparency
- **THEN** its contrast is measured after compositing it over the backgrounds of the pairing

### Requirement: Dark themes are complete
A theme whose scheme is `dark` SHALL declare every color role of the contract, so that no light default shows
through.

#### Scenario: Missing role in a dark theme
- **WHEN** a dark theme does not declare one of the contract's color roles
- **THEN** the theme tests fail and name the theme and the role

### Requirement: No runtime color overrides
The application SHALL NOT set role values from script. The accent recoloring that wrote role values as inline styles
on the root element is removed; a theme file is the only source of a role's value.

#### Scenario: Root element carries no inline role values
- **WHEN** the application is running with any theme
- **THEN** the root element's inline style declares no custom property
