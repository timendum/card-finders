mod backend
mod frontend

default: help

help:
  @just --list --unsorted

# dev: Go backend (:8080) + Vite dev server (proxies /api and /ws to :8080) together
[script('pwsh', '-NoProfile', '-Command')]
dev:
  Write-Host "==> dev (backend :8080 + vite)"
  $backend = Start-Process -PassThru -NoNewWindow just -ArgumentList 'backend','dev'
  $frontend = Start-Process -PassThru -NoNewWindow just -ArgumentList 'frontend','dev'
  try {
    Wait-Process -Id $backend.Id, $frontend.Id -ErrorAction SilentlyContinue
  } finally {
    foreach ($p in @($backend, $frontend)) {
      if ($p -and -not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
    }
  }

# run: build the frontend bundle, then serve it from the Go backend (single process, :8080)
# Paths are repo-root-relative so the static bundle resolves; pass extra server flags via ARGS.
run *ARGS:
  @just frontend build
  @echo "==> run (backend serving frontend/dist)"
  go -C backend run ./cmd/server --static-dir ../frontend/dist {{ARGS}}

# install deps in both projects
install:
  @just backend install
  @just frontend install

# build both projects
build:
  @just backend build
  @just frontend build

generate:
  @just backend generate

# test both projects
test:
  @just backend test
  @just frontend test

# lint both projects
lint:
  @just backend lint
  @just frontend lint

# format both projects
fmt:
  @just backend fmt
  @just frontend fmt

format: fmt

# verify formatting in both projects
fmt-check:
  @just backend fmt-check
  @just frontend fmt-check

# sanity checks across both projects
checks:
  @just backend checks
  @just frontend checks

# clean both projects
clean:
  @just backend clean
  @just frontend clean
