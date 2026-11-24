# Spec Delta

## Purpose

Lists the themes that ship with the webmail. Each theme has an id, a label, a scheme, a reference look and a
contrast level; adding or changing a shipped theme is a change to this capability.

## ADDED Requirements

### Requirement: Theme registry
The system SHALL keep one ordered registry of themes, each with an id, a label and a scheme (`light` or `dark`).
The registry SHALL contain, in this order: `larry`, `outlook`, `outlook-dark`, `apple-mail`, `apple-mail-dark`,
`breeze`, `breeze-dark`, `graphite`, `high-contrast`.

#### Scenario: Registry and stylesheet agree
- **WHEN** the theme tests compare the registry with the theme files
- **THEN** every registry id except `larry` has exactly one theme block, and every theme block's id is in the registry

#### Scenario: Scheme matches the page
- **WHEN** the theme tests read a theme whose scheme is `dark`
- **THEN** its page color is darker than its text color, and the opposite holds for a `light` theme

### Requirement: Larry theme
The registry SHALL include `larry` ("Larry", light) as the first entry: the Roundcube Larry look the application has
today, with a navy `#1B3A6B` header and primary action, square corners and the `"Segoe UI"` font stack.

#### Scenario: Choose Larry
- **WHEN** the user chooses "Larry"
- **THEN** the header is navy with white text, the primary button is navy, every corner is square and the selected
  message has the 3 px navy bar on its left

### Requirement: Outlook theme
The registry SHALL include `outlook` ("Outlook", light), inspired by Outlook on the web: a brand-blue `#0F6CBD`
header with white text, white surfaces on a light gray page, blue primary actions, 4 px corners on controls and
surfaces, and the `"Segoe UI"` font stack. Normal text reaches 4.5:1.

#### Scenario: Choose Outlook
- **WHEN** the user chooses "Outlook"
- **THEN** the header is blue `#0F6CBD` with white text, buttons and fields have 4 px corners, and the selected
  message row is a light blue with dark text

#### Scenario: Unread message
- **WHEN** the Outlook theme is active and the list shows an unread message
- **THEN** its subject is bold and its unread mark is brand blue

### Requirement: Outlook dark theme
The registry SHALL include `outlook-dark` ("Outlook (dark)", dark): the Outlook look on dark gray surfaces
(page `#1F1F1F`, surfaces `#292929`), light text, a dark header and a lighter blue `#479EF5` for links, marks and
focus. Normal text reaches 4.5:1.

#### Scenario: Choose Outlook dark
- **WHEN** the user chooses "Outlook (dark)"
- **THEN** the mail view, the sidebar, the dialogs and the login screen have dark surfaces with light text and no
  white area except the message paper

### Requirement: Apple Mail theme
The registry SHALL include `apple-mail` ("Apple Mail", light), inspired by Mail on macOS: a light toolbar header with
dark text, a light gray sidebar, white lists, the system blue as the only accent, 6 px controls, 10 px surfaces, and
the `-apple-system` font stack. Normal text reaches 4.5:1.

#### Scenario: Choose Apple Mail
- **WHEN** the user chooses "Apple Mail"
- **THEN** the header is light gray with dark text and a hairline below, and dialogs have 10 px corners

#### Scenario: Selected message
- **WHEN** the Apple Mail theme is active and the user selects a message
- **THEN** the row is a blue rounded rectangle inset from the list's edges, with white text and no bar on its left

### Requirement: Apple Mail dark theme
The registry SHALL include `apple-mail-dark` ("Apple Mail (dark)", dark): the Apple Mail look on near-black surfaces
(page `#1E1E1E`), translucent white text and the dark-mode system blue `#0A84FF` as the accent. Normal text reaches
4.5:1.

#### Scenario: Choose Apple Mail dark
- **WHEN** the user chooses "Apple Mail (dark)"
- **THEN** the toolbar header, the sidebar and the lists are dark with light text, and the selected message keeps the
  inset blue rounded rectangle

### Requirement: Breeze themes
The registry SHALL include `breeze` ("Breeze", light) and `breeze-dark` ("Breeze (dark)", dark), ported from
ad-manager's KDE Breeze themes: the Breeze window and view grays with the Breeze blue `#3DAEE9` as the selection
and accent family. Normal text reaches 4.5:1 in both.

#### Scenario: Choose Breeze
- **WHEN** the user chooses "Breeze"
- **THEN** the page is the light Breeze gray, surfaces are white and the selected message uses the Breeze blue family

#### Scenario: Choose Breeze dark
- **WHEN** the user chooses "Breeze (dark)"
- **THEN** the interface is dark with light text and keeps the Breeze blue family for selection and focus

### Requirement: Graphite theme
The registry SHALL include `graphite` ("Graphite (dark)", dark), ported from ad-manager's Grafana-based theme:
canvas `#111217`, surfaces `#181B1F`, text `#CCCCDC`, blue `#3D71D9` actions and an orange `#FF8833` accent.
Normal text reaches 4.5:1.

#### Scenario: Choose Graphite
- **WHEN** the user chooses "Graphite (dark)"
- **THEN** the interface is dark, the primary button is blue with white text and the current tab's mark is orange

### Requirement: High contrast theme
The registry SHALL include `high-contrast` ("High contrast (dark)", dark), an accessibility theme: black surfaces,
white text, yellow `#FFFF00` as the only accent, no shadows used for depth and a solid 3 px focus ring. It SHALL
meet the enhanced contrast level: 7:1 for normal text, 4.5:1 for large text and 3:1 for interface components.

#### Scenario: Choose High contrast
- **WHEN** the user chooses "High contrast (dark)"
- **THEN** every surface is black with white text, and the primary button, links, the selected message (with black
  text) and the current tab use yellow

#### Scenario: Keyboard focus
- **WHEN** the High contrast theme is active and a control receives keyboard focus
- **THEN** it shows a solid 3 px yellow ring

#### Scenario: Enhanced contrast
- **WHEN** the contrast test runs for `high-contrast`
- **THEN** every normal-text pairing reaches 7:1, every large-text pairing 4.5:1 and every component pairing 3:1

### Requirement: No third-party brand assets
Themes inspired by other products SHALL use only colors and system font stacks. They SHALL NOT add that product's
logo, icons, bundled fonts or name anywhere except the theme's label.

#### Scenario: Brand mark is unchanged
- **WHEN** the Outlook or the Apple Mail theme is active
- **THEN** the application shows its own brand mark and name in the header and on the login screen, and no font file
  is downloaded for the theme
