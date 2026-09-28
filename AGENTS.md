## Project Structure

- `cmd/server/main.go` — entrypoint
- `internal/web/` — HTTP handlers + Templ views
- `internal/data/` — DB queries (sql/*.sql embedded)
- `internal/api/` — REST API (Huma)
- `internal/auth/` — OAuth + sessions
- `assets/src/modules/` — TypeScript modules
- `assets/src/styles/` — CSS
- `migrations/` — PostgreSQL migrations

## Key Conventions

SQL queries embedded via `//go:embed sql/*.sql`.
Views use Templ syntax (edit .templ files, not *_templ.go).
Caching via `cache.C.GetOrFetch()` with TTL.

## Code Style

No comments in code — keep it clean, explanations only outside code.
Write production-ready code — no hacks, stubs, or TODOs.
Follow project conventions — study existing codebase, match style/patterns.
Minimal diffs — only show changed parts with clear instructions, don't rewrite entire files.
Explain only non-obvious changes.
UI consistency — use existing styles (gradients, shadows, CSS variables), not external references.
Try to reuse existing components instead of creating new ones.

## Safety

### Allowed Commands

**Read-only inspection**: `git status/diff/log/show/blame/branch`, `git fetch`, `rg`, `grep`, `find`, `ls`, `cat`.
**Codegen**: `templ generate` (writes only gitignored `*_templ.go`).
**Formatting**: `dprint fmt` and `dprint check` — pass changed file paths to keep diffs minimal.
**Checks**: `go vet ./...`, `go build ./...` (no `-o`, no artifacts), `tsc --noEmit -p .` if installed globally (never `npx`, never `npm install`).
**Tests**: `go test -v -race ./...`, targeted runs with `-run`.
**Read-only `gh`**: `gh issue list/view`, `gh pr list/view/diff/checks`, `gh run list/view`, `gh api` GET requests.

### Dev Environment

Follow CONTRIBUTING.md using only these commands:

- `docker compose -f docker-compose.dev.yml up -d/ps/logs/stop`
- `go run cmd/seed/main.go`
- `air` — run in background
- `curl` to `localhost` only

Before seed or `air`: confirm `DATABASE_URL` and `DEV_DATABASE_URL` hosts are `localhost` or `127.0.0.1` — extract host only, never print full values. Not local or missing — stop and ask.
If `.env` or `.env.dev` is missing — ask the owner, never create or edit them.
Stop processes you started when done.

### Git & Pull Requests

**`main` is off-limits** — no checkout, commit, push, merge, rebase, or PR from or into `main`.
**`dev` is protected** — no direct commits or pushes. PRs into `dev`: open only, never merge.
Branch from `origin/dev`. Commit and push only to feature branches.
Never push to branches another agent or person is working on unless asked.
PRs between feature branches: open and merge allowed.
**Allowed `gh` writes**: `gh pr create`, `gh pr merge`, `gh pr edit`, `gh pr comment` — within the rules above.
No force push, no `--no-verify`, no rewriting pushed history, no tags or releases.
Stage explicit paths only — no `git add -A` or `git add .`.
Before commit: `templ generate`, `dprint check`, `go vet ./...`, `go test -v -race ./...` must pass.
Commit messages follow Conventional Commits (`feat:`, `fix:`, `refactor:`, `style:`, `chore:`).

### Forbidden

Never read, print, or copy `.env*` contents or other secrets — only the host check above.
Never run Docker beyond the dev compose commands above, never run seed, migrations, or the server against a non-local DB.
Never change dependencies (`go get`, `go mod tidy`) or install tools unless the task explicitly requires it.
Never modify `.github/workflows/` unless the task explicitly requires it.
No other `gh` write actions — issues, releases, labels, secrets, repo settings, non-GET `gh api`.
Any command not listed above — ask first.
Verify by reading code and running allowed checks.

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
