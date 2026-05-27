# Releasing

This document explains the complete process for creating new version tags and GitHub Releases in the **go-cubemail-vue** project.

It covers both the automated path and the recommended approach for producing the high-quality, human-written release notes that follow the established project pattern (used for v0.0.21, v0.0.23, etc.).

## Versioning Scheme

The project follows a lightweight `v0.0.x` versioning scheme:

- `v0.0.21`, `v0.0.22`, `v0.0.23`, etc.
- Tags are **always** prefixed with `v`.
- The release title on GitHub must also use the `v` prefix (e.g. `v0.0.23`).

## Prerequisites

Before creating a release you need:

- Write access to the repository
- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated with the `repo` scope:
  ```bash
  gh auth status
  ```
- A clean working tree on the latest `main` branch:
  ```bash
  git status
  git fetch origin
  git checkout main
  git pull --ff-only
  ```

## Two Release Approaches

| Approach                        | Best For                          | Release Notes Quality                  | Recommended |
|---------------------------------|-----------------------------------|----------------------------------------|-------------|
| Automated (just tag + push)     | Hotfixes, CI-only changes         | Minimal (auto-generated compare link)  | No          |
| Polished notes (tag + manual edit) | Regular user-facing releases   | Structured sections with emojis        | **Yes**     |

## Recommended Process: Release with Polished Notes

This is the process used for the majority of releases and produces the clean, well-organized notes visible on the project's GitHub Releases page.

### 1. Review Changes Since the Last Release

```bash
git log v0.0.22..HEAD --oneline
```

Or with more detail:

```bash
git log v0.0.22..HEAD --stat
```

### 2. Categorize the Changes

Group commits into the following sections (use this exact order and emoji style):

- **✨ New Features** — New user-visible functionality
- **🔧 Improvements** — Enhancements, refactors, performance, CI improvements, developer experience
- **🧹 Cleanup** — Removal of dead code, unused assets, dependency cleanup
- **📚 Documentation** — Updates to README, SDD, guides, comments, etc.

If a category has no items, simply omit the section.

### 3. Create an Annotated Tag

Always use an annotated tag:

```bash
git tag -a v0.0.23 -m "Release v0.0.23"
```

### 4. Push the Tag

Pushing the tag triggers the Release workflow:

```bash
git push origin v0.0.23
```

### 5. Wait for the CI Workflow to Complete

The workflow (`release.yml`) will:

- Build the frontend and Go binary
- Package the Linux amd64 tarball
- Create the GitHub Release and attach the artifact

You can monitor progress in the **Actions** tab or poll with:

```bash
gh release view v0.0.23
```

Wait until the asset (e.g. `go-cubemail_0.0.23_linux_amd64.tar.gz`) appears.

### 6. Add Polished Release Notes

This is the key step that differentiates a great release from a basic one.

#### Option A — Using GitHub CLI (recommended)

```bash
gh release edit v0.0.23 \
  --title "v0.0.23" \
  --notes-file /tmp/release-notes-v0.0.23.md
```

#### Option B — GitHub Web UI

1. Go to the release page
2. Click the edit pencil icon
3. Paste the markdown in the description field
4. Save

### 7. Verify the Release

After editing, run:

```bash
gh release view v0.0.23
```

Check that:

- The title is `v0.0.23`
- The structured sections are present
- The binary asset is attached
- The **Full Changelog** link at the bottom is correct

## Release Notes Template

Copy and adapt the following template:

```markdown
# Release v0.0.23

## ✨ New Features

- ...

## 🔧 Improvements

- ...

## 🧹 Cleanup

- ...

## 📚 Documentation

- ...

**Full Changelog**: https://github.com/jniltinho/go-cubemail-vue/compare/v0.0.22...v0.0.23
```

### Example of Real Notes

See the actual releases for reference:

- [v0.0.23](https://github.com/jniltinho/go-cubemail-vue/releases/tag/v0.0.23)
- [v0.0.21](https://github.com/jniltinho/go-cubemail-vue/releases/tag/v0.0.21)

## Commit Categorization Guidelines

| Conventional Commit Prefix | Typical Section     | Example |
|----------------------------|---------------------|---------|
| `feat:`                    | New Features        | `feat(editor): add new composer button` |
| `fix:`, `perf:`            | Improvements        | `fix(imap): handle large attachments` |
| `refactor:`, `ci:`, `chore: (build)` | Improvements | `ci: fix release workflow prefix handling` |
| `chore:`, `cleanup:`       | Cleanup             | `chore: remove unused hero.png asset` |
| `docs:`, `doc:`            | Documentation       | `docs: update SDD after TipTap migration` |

When a single commit touches multiple areas, choose the most relevant section or split the description across bullets.

## Current Workflow Limitations

The release workflow (`.github/workflows/release.yml`) currently:

- Only builds for `linux/amd64`
- Produces a single `.tar.gz` artifact
- Does **not** automatically generate nice release notes (body is left empty)
- Deb and RPM packaging steps are commented out

These are intentional simplifications. If you need additional architectures or package formats, update the workflow and this document.

## Related Documentation

- [Release Workflow](https://github.com/jniltinho/go-cubemail-vue/blob/main/.github/workflows/release.yml) — The actual GitHub Action definition
- [Development Guide](DEVELOPMENT.md) — Short releasing overview + local development commands
- [Software Design Document (SDD)](SDD.md) — Project architecture

## Quick Command Reference

```bash
# 1. Create tag
git tag -a v0.0.24 -m "Release v0.0.24"

# 2. Push (triggers CI)
git push origin v0.0.24

# 3. Edit notes after CI finishes
gh release edit v0.0.24 --notes-file notes.md --title "v0.0.24"

# 4. Verify
gh release view v0.0.24
```

---

If you improve the release process or the workflow, please update this document accordingly. Contributions that make releasing easier and more consistent are very welcome!