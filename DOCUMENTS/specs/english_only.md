# English Only Policy

**Description:** 
This skill enforces that the entire `go-cubemail` project must be written in English. This is a strict rule to ensure consistency and maintainability across the codebase.

## Rules & Guidelines

When interacting with this repository or writing any code on behalf of the user, AI agents must adhere to the following rules:

1. **Source Code:**
   - All variable names, function names, structs, interfaces, and package names MUST be in English.
   - Example: Use `GetSession` instead of `ObterSessao`.

2. **Comments and Documentation:**
   - All code comments (`//` or `/* */`), docstrings, and markdown documentation files MUST be written in English.
   - Example: `// InitDB loads sessions from the database into memory.`

3. **Log Messages and Error Handling:**
   - All log messages (e.g., `slog.Info`, `slog.Error`) and error strings (`fmt.Errorf()`, `errors.New()`) MUST be in English.
   - Example: `slog.Error("Failed to connect to database")` instead of `"Falha ao conectar"`.

4. **API Responses and User Interface:**
   - Any JSON responses from the backend or internal API endpoints MUST use English keys and values (unless localization is explicitly requested for the UI layer).

**Note to AI Agents:** Before applying any changes or writing new features, ensure that you translate any Portuguese context provided by the user into English within the codebase.
