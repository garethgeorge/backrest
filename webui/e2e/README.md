# WebUI end-to-end tests (Playwright)

Full-stack browser tests: each test boots a real `backrest` binary (API +
embedded SPA on one port) with its own temporary data directory, and drives
the UI with Playwright.

## Running locally

```sh
cd webui
pnpm run e2e          # headless
pnpm run e2e:ui       # Playwright UI mode
```

### NixOS

Stock Playwright browser downloads do not run on NixOS. Point Playwright at
the prebuilt nix browser bundle instead (pinned `@playwright/test` matches the
nixpkgs `playwright-driver` version):

```sh
export PLAYWRIGHT_BROWSERS_PATH=$(nix build 'nixpkgs#playwright-driver.browsers' --print-out-paths --no-link)
export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true
cd webui && pnpm run e2e
```

On other platforms (and CI), a standard `pnpm exec playwright install chromium`
works and no env vars are needed.

## What global setup does (e2e/global-setup.ts)

1. Builds `webui/dist` with `pnpm run build` — required because the Go binary
   embeds it (`//go:embed dist`). Skipped when an mtime marker in
   `e2e/.cache/` says dist is fresh, or when `E2E_SKIP_WEBUI_BUILD=1`.
2. Builds the backrest binary to `e2e/.cache/backrest`
   (`go build ../cmd/backrest`).
3. Provisions restic: backrest auto-downloads its pinned restic version on
   first startup, so setup boots a throwaway instance once, caches the binary
   at `e2e/.cache/restic-data/restic`, and exports it as
   `BACKREST_RESTIC_COMMAND` for all test instances.

### Env knobs

| Variable                  | Effect                                                 |
| ------------------------- | ------------------------------------------------------ |
| `E2E_SKIP_WEBUI_BUILD=1`  | Never rebuild `webui/dist` (it must already exist)     |
| `E2E_BACKREST_BIN=<path>` | Use this backrest binary instead of running `go build` |
| `BACKREST_RESTIC_COMMAND` | Use this restic binary instead of provisioning one     |

## Writing specs (conventions)

- Import from the harness, not `@playwright/test` directly:
  `import { test, expect } from "../harness/fixtures";`
- The `backrest` fixture gives every test a **fresh instance** (empty config,
  own port and temp data dir). Nothing is shared between tests; there is no
  `baseURL` — navigate with `page.goto(backrest.url)`. On failure the
  instance's stdout/stderr are attached to the report.
- Seed state through the API, not the UI, unless the UI flow _is_ the thing
  under test: `seedInstance(backrest)` (name + disable auth — without it the
  first-run Settings modal auto-opens), `seedRepo(backrest)` (local restic
  repo at `backrest.repoPath(id)`), `seedPlan(backrest, id, repoId, paths)`
  (schedule disabled, so backups only run when triggered).
  `backrest.makeTestData({...})` writes files to back up and returns the
  directory path.
- Assert **persistent state** (operation rows, config round-trips, sidebar
  entries), not toasts — toasts are transient and flaky.
- Modals render in portals; scope queries with `page.getByRole("dialog")`.
- The SPA uses a HashRouter: deep links look like `${backrest.url}/#/plan/my-plan`.
- English strings live in `webui/messages/en.json`; the config forces
  `locale: "en-US"`.

### data-testid inventory

| testid                                               | where                                                                                                          |
| ---------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `settings-instance-id`                               | Settings modal: instance id input                                                                              |
| `settings-disable-auth`                              | Settings modal: disable-auth toggle (clickable label)                                                          |
| `settings-submit`                                    | Settings modal: save button                                                                                    |
| `add-repo-name`                                      | Add repo modal: name input                                                                                     |
| `add-repo-uri`                                       | Add repo modal: URI input                                                                                      |
| `add-repo-password`                                  | Add repo modal: password input                                                                                 |
| `add-repo-test-config`                               | Add repo modal: "Test configuration" button                                                                    |
| `add-repo-submit`                                    | Add repo modal: submit button                                                                                  |
| `add-plan-name`                                      | Add plan modal: name input                                                                                     |
| `add-plan-repo-select`                               | Add plan modal: repository select trigger                                                                      |
| `add-plan-path`                                      | Add plan modal: paths list container                                                                           |
| `add-plan-path-input`                                | Add plan modal: each path input (repeats; use `.nth()`/`.last()`)                                              |
| `add-plan-path-add`                                  | Add plan modal: "add path" button                                                                              |
| `add-plan-submit`                                    | Add plan modal: submit button                                                                                  |
| `sidebar-add-plan`                                   | Sidebar: "Add plan" button                                                                                     |
| `sidebar-add-repo`                                   | Sidebar: "Add repo" button                                                                                     |
| `sidebar-item-plan-${id}`                            | Sidebar: plan row                                                                                              |
| `sidebar-item-repo-${id}`                            | Sidebar: repo row                                                                                              |
| `plan-backup-now`                                    | Plan view: "Backup now" button                                                                                 |
| `operation-row`                                      | Operation list/tree rows; carries `data-op-type` (e.g. "Backup") and `data-status` (e.g. "success") attributes |
| `login-username` / `login-password` / `login-submit` | Login modal                                                                                                    |
| `snapshot-browser-entry`                             | Snapshot browser: file/dir row                                                                                 |
| `snapshot-restore`                                   | Snapshot browser: "Restore to path" menu item                                                                  |

`data-op-type` / `data-status` values come from the UI's display helpers
(`displayTypeToString` / `nameForStatus`) and are the (en-US) display strings:
op types `Backup`, `Dry Run Backup`, `Snapshot`, `Forget`, `Prune`, `Check`,
`Restore`, `Stats`, `Run Hook`, `Run Command`, `Unknown`; statuses `pending`,
`in progress`, `error`, `warning`, `success`, `cancelled`, `Unknown`
(statuses are lowercase in en.json).
