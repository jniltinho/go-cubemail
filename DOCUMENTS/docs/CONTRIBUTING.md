# Contributing to Go CubeMail

Thank you for considering contributing to **go-cubemail-vue**! This document outlines the guidelines and processes for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment. Please be kind and constructive in all interactions.

## Getting Started

Before contributing, please read:

- [Development Guide](DEVELOPMENT.md) — How to set up your environment
- [Software Design Document (SDD)](SDD.md) — Architecture and design decisions
- [English Only Policy](../specs/english_only.md) — **Strict requirement** for all contributions

## How Can I Contribute?

### Reporting Bugs

- Use the [GitHub Issues](https://github.com/jniltinho/go-cubemail-vue/issues) page.
- Provide a clear and descriptive title.
- Include steps to reproduce the issue.
- Mention your environment (OS, Go version, Node version, database).
- If possible, include logs or screenshots.

### Suggesting Enhancements

- Open an issue with the label `enhancement`.
- Explain the problem you are trying to solve.
- Describe the solution you have in mind and why it would be useful.

### Pull Requests

We welcome pull requests! Please follow these guidelines:

1. **Fork the repository** and create your branch from `main`.
2. **Make your changes** following the project's style and the English-only policy.
3. **Test your changes** locally before submitting.
4. **Write clear commit messages** (see below).
5. **Open a Pull Request** against the `main` branch.

## Development Guidelines

### English Only Policy

This is a **hard requirement**. All contributions must follow the rules defined in:

→ [DOCUMENTS/specs/english_only.md](../specs/english_only.md)

This includes:
- Code (variables, functions, comments, error messages, logs)
- Documentation (README, docs, commit messages, PR descriptions)
- User-facing strings

Pull requests that do not comply will be requested to be updated.

### Code Style

- **Go**: Follow `gofmt` and `goimports`. Run `go fmt ./...` before committing.
- **Frontend**: Follow the existing Prettier + ESLint configuration.
- Keep functions small and focused.
- Add comments for complex logic.

### Commit Messages

We prefer clear, descriptive commit messages. Recommended format:

```
type(scope): short description

Longer explanation if needed.

Fixes #123
```

Common types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only changes
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests
- `chore`: Maintenance tasks

Examples:
- `feat: add support for custom IMAP ports in init command`
- `fix: prevent race condition in session cleanup`
- `docs: improve DEVELOPMENT.md setup instructions`

### Pull Request Process

1. Ensure your branch is up to date with `main`.
2. Make sure all tests pass and the project builds.
3. Fill out the PR template (if available).
4. Request review from maintainers.
5. Be responsive to feedback.

Large changes should be discussed in an issue first.

## Project Documentation

When contributing, please keep documentation up to date:

- Update `README.md` for user-facing changes.
- Update `DEVELOPMENT.md` for developer workflow changes.
- Update `SDD.md` for significant architectural decisions.
- Add or update entries in `DOCUMENTS/docs/` when appropriate.

## Questions?

If you have questions or need clarification:

- Open a [GitHub Discussion](https://github.com/jniltinho/go-cubemail-vue/discussions)
- Or open an issue with the `question` label

We appreciate your time and effort in helping improve Go CubeMail!

---

Thank you for contributing! 🚀
