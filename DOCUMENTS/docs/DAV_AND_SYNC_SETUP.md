# DAV & Sync Setup Guide

> **Project:** go-cubemail-vue  
> **Scope:** CalDAV, CardDAV, and Exchange ActiveSync (EAS) — configuration and Thunderbird setup  
> **Status:** All three protocols are implemented and active when `[activesync] enabled = true`

---

## Table of Contents

1. [Overview](#1-overview)
2. [Server Endpoints](#2-server-endpoints)
3. [config.toml Reference](#3-configtoml-reference)
4. [CalDAV — Calendar Sync](#4-caldav--calendar-sync)
5. [CardDAV — Contacts Sync](#5-carddav--contacts-sync)
6. [Thunderbird Setup (Step by Step)](#6-thunderbird-setup-step-by-step)
7. [Exchange ActiveSync (EAS)](#7-exchange-activesync-eas)
8. [Web Push Notifications](#8-web-push-notifications)
9. [Troubleshooting](#9-troubleshooting)

---

## 1. Overview

go-cubemail-vue exposes three open-standard sync protocols alongside its webmail UI:

| Protocol | Purpose | Clients |
|----------|---------|---------|
| **CalDAV** (RFC 4791) | Calendar event sync | Thunderbird + TbSync, Apple Calendar, Evolution, GNOME Calendar |
| **CardDAV** (RFC 6352) | Contacts sync | Thunderbird + TbSync, Apple Contacts, Evolution, KAddressBook |
| **EAS** (Exchange ActiveSync) | Mail + calendar + contacts for mobile | iOS Mail, Android Gmail/Outlook, Outlook desktop |

All three share the same HTTP Basic Auth backend: credentials are validated against your IMAP server, so users log in with the **same username and password** they use in the webmail.

---

## 2. Server Endpoints

Replace `https://mail.example.com` with your `server.base_url` value from `config.toml`.

### CalDAV

| Purpose | Method | URL |
|---------|--------|-----|
| Well-known discovery | `GET` / `PROPFIND` | `https://mail.example.com/.well-known/caldav` |
| User principal | `PROPFIND` | `https://mail.example.com/dav/{username}/` |
| Calendar list | `PROPFIND` | `https://mail.example.com/dav/{username}/calendars/` |
| Calendar collection | `PROPFIND` / `REPORT` | `https://mail.example.com/dav/{username}/calendars/{cal}/` |
| Individual event | `GET` / `PUT` / `DELETE` | `https://mail.example.com/dav/{username}/calendars/{cal}/{uid}.ics` |

### CardDAV

| Purpose | Method | URL |
|---------|--------|-----|
| Well-known discovery | `GET` / `PROPFIND` | `https://mail.example.com/.well-known/carddav` |
| Address book home | `PROPFIND` | `https://mail.example.com/dav/{username}/contacts/` |
| Address book | `PROPFIND` / `REPORT` | `https://mail.example.com/dav/{username}/contacts/default/` |
| Individual contact | `GET` / `PUT` / `DELETE` | `https://mail.example.com/dav/{username}/contacts/default/{uid}.vcf` |

### Exchange ActiveSync

| Purpose | Method | URL |
|---------|--------|-----|
| Autodiscover | `POST` / `GET` | `https://mail.example.com/autodiscover/autodiscover.xml` |
| Autodiscover (well-known) | `GET` | `https://mail.example.com/.well-known/autodiscover/autodiscover.xml` |
| EAS commands | `POST` | `https://mail.example.com/Microsoft-Server-ActiveSync` |

---

## 3. config.toml Reference

### ActiveSync section

```toml
[activesync]
enabled               = true        # set false to disable EAS entirely
debug                 = false        # log full EAS XML request/response
max_ping_interval_sec = 30          # max polling interval for PING command
max_sync_window_size  = 100         # max items per Sync response
protocol_version      = "16.1"      # EAS version advertised to clients
```

### Web Push section (optional — for browser notifications)

```toml
[push]
# Generate keys once with:  npx web-push generate-vapid-keys
vapid_public_key  = "BExxxxxxxxxxxxxxxx…"
vapid_private_key = "Dpxxxxxxxxxxxxxxxx…"
vapid_contact     = "mailto:webmail@example.com"
```

After changing `[push]` keys, run `go-cubemail migrate` to store the public key in the database.

---

## 4. CalDAV — Calendar Sync

### How the discovery chain works

```
Client
  │
  ├─1─► GET /.well-known/caldav
  │       └── 301 → /dav/{user}/
  │
  ├─2─► PROPFIND /dav/{user}/
  │       └── current-user-principal, calendar-home-set
  │
  ├─3─► PROPFIND /dav/{user}/calendars/  Depth:1
  │       └── calendar list (name, colour, ctag)
  │
  ├─4─► PROPFIND /dav/{user}/calendars/{cal}/  Depth:1
  │       └── event list (getetag, getcontenttype)
  │
  ├─5─► REPORT /dav/{user}/calendars/{cal}/
  │       └── calendar-query / calendar-multiget with iCalendar bodies
  │
  └─6─► GET|PUT|DELETE /dav/{user}/calendars/{cal}/{uid}.ics
          └── individual event read/write/delete
```

### Supported features

- Multiple personal calendars per user (create, rename, colour, delete), each at
  its own stable URL — renaming a calendar does not break client sync state
- Full CRUD for events; the iCalendar payload is stored and returned byte for
  byte, so VTIMEZONE, VALARM and `X-*` properties survive a round trip
- Recurring events (RRULE: DAILY, WEEKLY, MONTHLY, YEARLY)
- Standard discovery via `.well-known/caldav` redirect
- `sync-collection` REPORT (RFC 6578) — delta sync including deletions
- `calendar-query` and `calendar-multiget` REPORTs
- `MKCALENDAR` and `PROPPATCH`, so clients can create and rename calendars
- Conditional writes via `If-Match` / `If-None-Match`, answering **412** on a
  conflict so two clients cannot silently overwrite each other

---

## 5. CardDAV — Contacts Sync

### How the discovery chain works

```
Client
  │
  ├─1─► GET /.well-known/carddav
  │       └── 301 → /dav/{user}/
  │
  ├─2─► PROPFIND /dav/{user}/
  │       └── addressbook-home-set
  │
  ├─3─► PROPFIND /dav/{user}/contacts/  Depth:1
  │       └── address-book collection (displayname, resourcetype)
  │
  ├─4─► PROPFIND /dav/{user}/contacts/default/  Depth:1
  │       └── vCard list with getetag
  │
  ├─5─► REPORT /dav/{user}/contacts/default/
  │       └── addressbook-query / addressbook-multiget
  │
  └─6─► GET|PUT|DELETE /dav/{user}/contacts/default/{uid}.vcf
          └── individual vCard read/write/delete
```

### Supported features

- Multiple address books per user, provisioned as `default` on first access and
  creatable with `MKCOL`
- vCard 3.0 and 4.0: the card is stored exactly as the client sent it, so
  addresses, photos, birthdays, extra e-mails and phone numbers and `X-*`
  extensions are never lost — including when the contact is edited from the
  web UI, which patches the stored card instead of regenerating it
- `sync-collection` REPORT (RFC 6578) — delta sync including deletions
- `addressbook-query` with `prop-filter` text matching, and `addressbook-multiget`
- Conditional writes via `If-Match` / `If-None-Match` (**412** on conflict)
- `DAV: 1, 2, 3, access-control, calendar-access, addressbook` on OPTIONS

> Contact groups (`KIND:group`), the global address list and automatically
> collected contacts are not implemented yet — see
> [§6 of the implementation guide](DAV_IMPLEMENTATION.md#6-remaining-work).

---

## 6. Thunderbird Setup (Step by Step)

Thunderbird requires the **TbSync** extension plus the **Provider for CalDAV & CardDAV** extension.

### 6.1 Install extensions

1. Open Thunderbird → **Tools → Add-ons and Themes**
2. Search for **TbSync** → Install
3. Search for **Provider for CalDAV & CardDAV** → Install
4. Restart Thunderbird when prompted

### 6.2 Add the account in TbSync

1. Open **Tools → TbSync**
2. Click **"Account actions" → "Add new account"**
3. Select **"CalDAV & CardDAV"**
4. Fill in the form:

   | Field | Value |
   |-------|-------|
   | **Account name** | Any label (e.g. `My Webmail`) |
   | **User name** | Your IMAP username (e.g. `alice@example.com`) |
   | **Password** | Your IMAP/webmail password |
   | **CalDAV server URL** | `https://mail.example.com/.well-known/caldav` |
   | **CardDAV server URL** | `https://mail.example.com/.well-known/carddav` |

   > **Tip:** You can also enter just `https://mail.example.com` — TbSync will discover both `.well-known` endpoints automatically.

5. Click **"Save & connect"**
6. TbSync will discover your calendars and address book.

### 6.3 Subscribe to resources

After connecting:

1. In the TbSync account pane you will see:
   - **Calendars** — one entry per personal calendar
   - **Address Books** — `default` (your contacts)

2. Check the box next to each resource you want to sync.
3. Click **"Synchronize now"** (the circular arrow button).

### 6.4 Thunderbird Calendar (Lightning)

Once synced via TbSync, your calendars appear directly in the **Thunderbird Calendar** tab (built-in since Thunderbird 78).

- Events created in Thunderbird are pushed to go-cubemail-vue immediately on save.
- Events created in the webmail appear in Thunderbird on the next sync cycle.
- Sync interval: configured in TbSync → Account → "Sync interval" (default: 30 min; you can set it to 5 min or manual).

### 6.5 Thunderbird Contacts (Address Book)

After TbSync sync, a new address book named **"My Webmail — default"** (or whatever you labeled the account) appears in **Tools → Address Book**.

- Contacts are bidirectionally synced with the webmail contacts list.
- Changes are uploaded on the next sync cycle; you can trigger it manually with **"Synchronize now"** in TbSync.

### 6.6 Thunderbird Email (IMAP/SMTP) — separate from TbSync

CalDAV/CardDAV sync is independent from email. To also configure Thunderbird for IMAP email access:

1. **File → New → Existing Mail Account…**
2. Enter your name, email, and password.
3. Thunderbird will auto-detect IMAP and SMTP settings from your mail server.
4. Use the same credentials as for CalDAV/CardDAV.

---

## 7. Exchange ActiveSync (EAS)

EAS is primarily intended for **mobile devices** (iOS, Android) and **Outlook desktop**. It bundles mail + calendar + contacts in a single protocol.

### 7.1 Enable EAS in config.toml

```toml
[activesync]
enabled = true
```

Restart go-cubemail-vue after any config change.

### 7.2 iOS Setup

1. **Settings → Mail → Accounts → Add Account → Microsoft Exchange**
2. Fill in:
   - **Email:** your full email address
   - **Password:** your webmail password
   - **Description:** any label
3. Tap **Next** — iOS will try Autodiscover at `https://mail.example.com/autodiscover/autodiscover.xml` automatically.
4. If Autodiscover succeeds, iOS fills in the server URL for you.
5. If not (e.g. self-signed cert), tap **Configure Manually** and enter:
   - **Server:** `mail.example.com`
   - **Domain:** leave blank
6. Choose which resources to enable: Mail, Contacts, Calendars.

### 7.3 Android (Samsung Mail / Outlook)

1. Open your mail app → **Add Account → Exchange / Office 365**
2. Enter email and password.
3. The app attempts Autodiscover. If it fails, enter the server manually:
   - **Server:** `mail.example.com`
   - **Port:** `443` (HTTPS) or `80` (HTTP dev)
   - **Use SSL:** match your TLS config

### 7.4 Outlook Desktop

1. **File → Add Account**
2. Enter your email address and click **Connect**
3. Select **Exchange** when asked for account type
4. Enter your password — Outlook runs Autodiscover against `https://mail.example.com`

> **Note:** EAS protocol version `16.1` is advertised. Outlook 2016+ and iOS 14+ support this version.

### 7.5 EAS Autodiscover

When a client POSTs to `/autodiscover/autodiscover.xml`, go-cubemail-vue responds with:

```xml
<Autodiscover>
  <Response>
    <Culture>en:us</Culture>
    <User>
      <DisplayName>alice@example.com</DisplayName>
      <EMailAddress>alice@example.com</EMailAddress>
    </User>
    <Action>
      <Settings>
        <Server>
          <Type>MobileSync</Type>
          <Url>https://mail.example.com/Microsoft-Server-ActiveSync</Url>
          <Name>mail.example.com</Name>
        </Server>
      </Settings>
    </Action>
  </Response>
</Autodiscover>
```

---

## 8. Web Push Notifications

Web Push delivers browser notifications for new email without keeping the browser tab open.

### 8.1 Generate VAPID keys

```bash
npx web-push generate-vapid-keys
```

Output example:

```
Public Key:
BEGMWCamtk...

Private Key:
Dp6REHty66...
```

### 8.2 Configure config.toml

```toml
[push]
vapid_public_key  = "BEGMWCamtk…"          # paste Public Key here
vapid_private_key = "Dp6REHty66…"          # paste Private Key here
vapid_contact     = "mailto:admin@example.com"
```

### 8.3 Apply the change

```bash
./bin/go-cubemail migrate    # stores public key in database
./bin/go-cubemail serve      # restart to pick up new config
```

### 8.4 Enable notifications in the webmail

1. Log in to go-cubemail-vue.
2. Go to **Settings → Preferences**.
3. Under **Desktop & Push Notifications**, click **"Enable notifications"**.
4. Accept the browser permission prompt.
5. Push subscriptions are registered for your browser/device; you receive a notification whenever a new message arrives, even with the tab in the background.

> **HTTPS required:** Browsers only allow Web Push (and notification permission prompts) on HTTPS origins. For local development over HTTP, use Firefox with the `dom.push.testing.allowInsecure` pref set to `true`.

---

## 9. Troubleshooting

### "Authentication failed" on CalDAV/CardDAV

- The DAV endpoints use **HTTP Basic Auth** validated against IMAP.
- Ensure the username is the full email address (e.g. `alice@example.com`), not just `alice`.
- If your IMAP server requires a specific port or TLS setting, those are inherited from the `[imap]` section in `config.toml`.

### TbSync shows "Server not found"

1. Verify `server.base_url` in `config.toml` is reachable from the Thunderbird machine.
2. Check that port 8080 (or your configured port) is open in the firewall.
3. Try accessing `https://mail.example.com/.well-known/caldav` directly in a browser — you should get a redirect (HTTP 301) to `/dav/{username}/`.

### Thunderbird shows duplicate address books after re-syncing

Remove the TbSync account and re-add it. Duplicates happen when TbSync is removed without first deactivating the resources.

### iOS / Android Autodiscover fails

1. Confirm `[activesync] enabled = true` in `config.toml` and that the server was restarted.
2. Test Autodiscover manually:
   ```bash
   curl -s -u user@example.com \
     -X POST https://mail.example.com/autodiscover/autodiscover.xml \
     -H "Content-Type: text/xml" \
     -d '<?xml version="1.0"?><Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/mobilesync/requestschema/2006"><Request><EMailAddress>user@example.com</EMailAddress><AcceptableResponseSchema>http://schemas.microsoft.com/exchange/autodiscover/mobilesync/responseschema/2006</AcceptableResponseSchema></Request></Autodiscover>'
   ```
3. A valid response contains `<Url>…/Microsoft-Server-ActiveSync</Url>`.

### Web Push notifications not arriving

1. Check that `[push] vapid_public_key` and `vapid_private_key` are set and non-empty.
2. Run `go-cubemail migrate` after adding the keys.
3. Ensure the site is served over HTTPS — Web Push does not work on plain HTTP in production browsers.
4. Check browser console (F12 → Console) for subscription errors after clicking "Enable notifications".
