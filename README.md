# parley

parley is a terminal API client: a Postman/Insomnia/Bruno-style workflow
(collections, environments, auth, scripting, import/export) without leaving
the terminal or running an Electron app.

Status: early stage, pre-1.0. Interfaces and storage format may still
change.

## why not just curl

curl handles the request. It does not handle the things that make API work
sustainable past a single one-off call:

- persistent, organized collections instead of shell history or a folder of
  saved curl commands
- environments (dev/staging/prod) with `{{variable}}` substitution, instead
  of hand-editing flags every time
- auth flows with token capture and refresh, instead of copy-pasting a
  bearer token out of one response into the next request
- pre-request and test scripting, instead of a wrapper shell script

parley isn't competing with curl. It imports curl commands directly and can
generate them back out, so it fits alongside curl rather than replacing it.

## why not just postman or insomnia

Those already solve the problems above; parley targets the same job with a
different interface: a single static binary, keyboard-driven, no browser or
Electron runtime, and plain files on disk instead of a proprietary or
cloud-synced store. `parley run` also means the same binary that edits a
collection interactively can execute it in CI, without a separate Node.js
tool.

## features

- workspaces: multiple independent sets of collections, environments, and
  history, switchable without restarting
- collections: folders and requests as plain files on disk, reordered and
  renamed from the sidebar
- environments and globals: named variable sets with `{{key}}` substitution
  into the URL, params, headers, and body
- auth: bearer/token auth with automatic capture (pull a token out of a
  response by JSON path) and a configurable refresh flow, without writing a
  script for the common case
- body editor: raw text, URL-encoded, multipart, binary, and GraphQL
  (separate query and variables) as first-class modes
- scripting: a Postman-compatible `pm.*` API (`pm.test`, `pm.expect`,
  `pm.environment`, `pm.globals`, `pm.variables`, `pm.request`,
  `pm.response`) for pre-request and test scripts, run in an embedded JS
  runtime
- import: curl commands, Postman Collection v2.1, and OpenAPI 3 (JSON or
  YAML), with format auto-detection
- code generation: turn a saved request back into a curl command or Go code
- history: the last 50 requests sent, restorable from a modal
- `parley run`: execute a saved collection or request headlessly, for CI
  (see below)

## running collections in CI: `parley run`

`parley run` reuses the exact same send, pre-request-script, and
test-script engine the TUI uses, without opening the TUI:

    parley run [flags] <path>

`<path>` is a folder or request name from the collection, using its
display name rather than the file it's stored as (for example `users/get`,
or `users` to run every request in that folder, or nothing to run the
whole collection).

Flags:

- `--workspace <name>`: run against a workspace other than the active one
- `--env <name>`: environment to resolve `{{variables}}` from (globals
  apply either way; omit `--env` to run with no named environment)
- `--reporter text|json|junit`: output format (default `text`)
- `--out <file>`: write the report to a file instead of stdout

Exit codes: `0` if every request sent successfully and every test
assertion passed, `1` if the run completed but something failed, `2` if
the run couldn't start at all (bad path, bad reporter, unknown
environment or workspace).

The `junit` reporter emits one testsuite per run and one testcase per
`pm.test` assertion (a request with no test script still gets one testcase,
pass or fail on whether it sent successfully), so it drops into existing
CI test-result dashboards.

## sharing collections with a team

Because collections are plain files on disk, a team can share and co-edit them
through an ordinary git repository - pull teammates' changes, push your own, no
server involved. Secret-holding and per-user files stay local via a git-ignore
template. See [docs/collaboration.md](docs/collaboration.md) for the full
workflow.

## install

macOS and Linux have prebuilt binaries; anything else uses Go.

curl (macOS / Linux, no Go needed):

    curl -fsSL https://raw.githubusercontent.com/dhitalkamal/parley/main/install.sh | sh

Homebrew (macOS / Linux):

    brew install dhitalkamal/tap/parley

Go (any platform, needs Go 1.26.5+; installs into `$(go env GOPATH)/bin`):

    go install github.com/dhitalkamal/parley/cmd/parley@latest

Or build from a local clone:

    go build -o parley ./cmd/parley

## quickstart

    parley

launches the TUI. It creates a "Personal" workspace on first run, stored
under `$PARLEY_HOME`, else `$XDG_CONFIG_HOME/parley`, else
`~/.config/parley`.

    parley run <path>

runs a saved request or folder from the active workspace headlessly - see
above. Omit `<path>` to run the whole collection.

    parley version    print the version
    parley help       show the top-level command summary

Inside the TUI, press `?` for the keybindings overlay or `ctrl+k` for the
command palette. A full key reference lives in [CHEATSHEET.md](CHEATSHEET.md).

## license

MIT - see [LICENSE](LICENSE).
