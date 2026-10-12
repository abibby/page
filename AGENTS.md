# Repository Guidelines

## Project Structure & Module Organization

Salusa documentation is available at [gosalusa.com/llms.txt](https://gosalusa.com/llms.txt).

Page combines a Go/Salusa backend with a React/TypeScript UI. `main.go` starts the kernel in `app/kernel.go`. HTTP handlers, dependency providers, models, and background jobs live under `app/`; routes are registered in `routes/routes.go`. Keep library storage, metadata extraction, and external integrations in `services/`. Configuration lives in `config/`, and versioned database changes belong in `migrations/`.

Frontend pages, components, hooks, and API types live in `ui/src/`. Vite produces `ui/dist/`, which the production Go binary embeds. Server templates live in `resources/`. Go unit tests sit beside their implementations; `tests/integration/` uses helpers from `test/`.

## Build, Test, and Development Commands

Run commands from the repository root unless noted:

- `npm --prefix ui ci`: install locked frontend dependencies.
- `npm --prefix ui run build`: type-check TypeScript and build assets; run before ordinary Go builds or tests on a fresh checkout.
- `go run .`: start the backend with the embedded UI.
- `npm --prefix ui run dev`: start Vite; set `VITE_PROXY_HOST` to the backend address if it differs from `http://localhost:8010`.
- `go test ./...`: run Go unit and integration tests.
- `go test -race ./services/library`: check concurrent library operations.
- `golangci-lint run ./...` and `npm --prefix ui run lint`: run backend and frontend lint checks.
- `CGO_ENABLED=0 go build .`: verify the static build used by Docker.

## Coding Style & Naming Conventions

Format Go with `gofmt` and use idiomatic exported names and lowercase package names. Match the UI's two-space indentation, PascalCase components, and kebab-case filenames such as `upload-button.tsx`. Keep API types aligned with handler responses. Add timestamped migrations such as `20261012_000000-Library.go`; do not rewrite applied migrations.

## Testing Guidelines

Use Go's `testing` package and Testify assertions. Name files `*_test.go` and tests `TestBehavior`. Use temporary databases and directories. Cover migration rollback, path preservation, and duplicate imports when changing storage. No numeric coverage threshold or frontend test runner is configured.

## Commit & Pull Request Guidelines

History uses short imperative subjects such as `fix ts errors`; Conventional Commits are not established. Keep commits focused. PRs should explain behavior changes, list validation commands, link relevant issues, and include screenshots for UI changes.

## Configuration & Data Compatibility

Copy `.env.example` to `.env`; never commit secrets. Keep `DATABASE_PATH` separate from Calibre's `metadata.db`. Preserve existing author paths and the `CALIBRE_LIBRARY` fallback; new author folders use names without IDs.
