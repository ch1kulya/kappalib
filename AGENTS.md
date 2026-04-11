## Project Structure

`cmd/server/main.go` — entrypoint
`internal/web/` — HTTP handlers + Templ views
`internal/data/` — DB queries (sql/*.sql embedded)
`internal/api/` — REST API (Huma)
`internal/auth/` — OAuth + sessions
`assets/src/modules/` — TypeScript modules
`assets/src/styles/` — CSS
`migrations/` — PostgreSQL migrations

## Key Conventions

SQL queries embedded via `//go:embed sql/*.sql`
Views use Templ syntax (edit .templ files, not *_templ.go)
Caching via `cache.C.GetOrFetch()` with TTL

## Code Style

No comments in code — keep it clean, explanations only outside code
Write production-ready code — no hacks, stubs, or TODOs
Follow project conventions — study existing codebase, match style/patterns
Minimal diffs — only show changed parts with clear instructions, don't rewrite entire files
Explain only non-obvious changes
UI consistency — use existing styles (gradients, shadows, CSS variables), not external references
Reuse existing components instead of creating new ones

## Safety

**Never commit anything** — agents may not run git add, git commit, or create commits
**Only run tests**: `go test -v -race ./...`
**Only run templ fmt when done with .templ files**: `templ fmt -fail .`
Never run applications, migrations, builds, dev servers, or any other commands
Verify by reading code

## Security

Never introduce security vulnerabilities (XSS, SQL injection, CSRF, etc.).
Never add secrets, keys, credentials, or tokens to code.
Sanitize all user inputs before processing or storing.
Use parameterized queries for all database operations.
Implement proper authentication and authorization checks.

## Caveman

Terse like caveman. Technical substance exact. Only fluff die.
Drop: articles, filler (just/really/basically), pleasantries, hedging.
Fragments OK. Short synonyms. Code unchanged.
Pattern: [thing] [action] [reason]. [next step].
ACTIVE EVERY RESPONSE. No revert after many turns. No filler drift.
Code/commits/PRs: normal. Off: "stop caveman" / "normal mode".
