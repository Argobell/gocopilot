# Repository Guidelines

## Project Structure & Module Organization
`cmd/gocopilot` holds the CLI entry point and is the place to wire new commands or flags. `internal/agent` encapsulates agent orchestration and shared state, while `internal/tools` contains tool integrations and reusable helpers. Keep configuration files such as `.env` and `.env.example` in the repo root; store secrets only in your local `.env`.

## Build, Test, and Development Commands
Use `go run ./cmd/gocopilot` for quick manual verification during development. `go build ./cmd/gocopilot` generates the binary that CI and releases consume. Run `go test ./...` before every commit; add `-run` filters for focused suites and `-cover` when you need coverage details. `go mod tidy` keeps module metadata synchronized after dependency edits.

## Coding Style & Naming Conventions
Format code with `gofmt` (tabs for indentation, gofmt defaults) and organize imports via `goimports` if installed. Name packages with concise lowercase nouns, export only what you intend external callers to use, and document exported identifiers with complete sentences. Keep file names `snake_case.go` and mirror the package or feature they implement.

## Testing Guidelines
Locate all Go tests inside the top-level `tests/` directory, mirroring the package layout and naming files with the `_test.go` suffix. Prefer table-driven cases for clarity, use subtests with `t.Run`, and call `t.Parallel()` when the code under test is concurrency-safe. Mock tool integrations at interface boundaries so agent behavior remains deterministic; stash fixtures under `tests/testdata` when sample payloads are required.

## Commit & Pull Request Guidelines
Follow Conventional Commits (e.g., `feat:`, `refactor:`) as seen in the history, and keep subject lines under 72 characters. Provide bilingual context in the body when helpful, mirroring earlier commits. Pull requests should link related issues, summarize behavior changes, note config updates, and include the latest `go test ./...` output or other validation steps.
