# Documentation Index

This directory contains the main technical and contributor documentation for **go-cubemail-vue**.

## Core Documentation

| Document | Description |
|----------|-------------|
| [Software Design Document (SDD)](SDD.md) | Complete architecture, design decisions, data models, and implementation details of the project. |
| [Development Guide](DEVELOPMENT.md) | How to set up a local development environment, build the project, run in dev mode, and useful commands. |
| [Contributing Guide](CONTRIBUTING.md) | Guidelines for contributing code, reporting issues, pull request process, and coding standards. |
| [Code Audit & Improvements Report](CODE_AUDIT_AND_IMPROVEMENTS.md) | Results of a comprehensive code and documentation quality audit (June 2026). |
| [Releasing Guide](RELEASING.md) | How to create version tags and GitHub Releases, including the process for writing polished, structured release notes that follow the project pattern. |

## Implementation Guides

| Document | Description |
|----------|-------------|
| [Calendar Implementation Guide](CALENDAR_IMPLEMENTATION.md) | Step-by-step plan to build the calendar module (Echo + Vue 3), based on SOGo's calendar architecture. |
| [Calendar Go Reference](CALENDAR_GO_REFERENCE.md) | Documentation of all Go types and functions in the calendar backend packages. |
| [Calendar cURL Testing Guide](CALENDAR_CURL_TESTING.md) | How to test every calendar API endpoint with curl (auth, CRUD, ICS import/export). |
| [Calendar API Testing Guide](CALENDAR_API_TESTING.md) | Legacy API testing guide (superseded by cURL guide for curl-specific workflows). |
| [ActiveSync Implementation Guide](ACTIVESYNC_IMPLEMENTATION.md) | Step-by-step plan to build a Microsoft Exchange ActiveSync (EAS) server in Go for mobile mail/calendar/contacts sync. |
| [ActiveSync Go Reference](ACTIVESYNC_GO_REFERENCE.md) | Documentation of all Go types and functions in the ActiveSync backend packages. |
| [ActiveSync cURL Testing Guide](ACTIVESYNC_CURL_TESTING.md) | Test EAS endpoints with curl (OPTIONS, Provision, FolderSync, Ping, Autodiscover). |
| [ActiveSync Status, Roadmap & PHP Reference](ACTIVESYNC_STATUS_ROADMAP_AND_PHP_REFERENCE.md) | Complete status of what was built, remaining work, Go test client guide, and Horde PHP ActiveSync analysis. |
| [ActiveSync Go Server from Horde (PHP)](ACTIVESYNC_GO_SERVER_FROM_HORDE.md) | Phased implementation guide: Horde ActiveSync architecture analysis, PHP→Go layer mapping, Driver design, and acceptance criteria per phase. |

## Other Important Documentation

- **[English Only Policy](../specs/english_only.md)** — Strict project rule requiring all code, comments, logs, and documentation to be written in English.
- **[Production Setup Guide](../setup/README.md)** — Step-by-step guide for installing and running the application in production (Ubuntu + MariaDB example).
- **[Main Project README](../../README.md)** — General overview, quick start, and features for end users.
- **[Frontend README](../../frontend/README.md)** — Frontend-specific development guide (Vue 3 + TypeScript).

> **Note:** For a higher-level overview of all documentation, see the [Documentation Index](../README.md) at the root of the `DOCUMENTS/` directory.

## Quick Links

- **For new contributors**: Start with [Contributing Guide](CONTRIBUTING.md) and [Development Guide](DEVELOPMENT.md).
- **For understanding the architecture**: Read the [Software Design Document (SDD)](SDD.md).
- **For production deployment**: See the [Production Setup Guide](../setup/README.md).

---

If you find any documentation outdated or missing, please open an issue or submit a pull request. We appreciate contributions that help keep the docs accurate and helpful!