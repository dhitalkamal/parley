# parley 1.0 readiness

Status of parley toward a 1.0 release. Produced from a QA program in
2026-09: an automated multi-angle review, a live TUI QA pass, and an
adversarially-verified fix round. Living doc - update as items close.

## done

- full codebase review: architecture map + bug/security/perf/maintainability
  review, adversarially verified, plus a gap critic. 22 findings.
- live TUI QA (tmux) across terminal sizes down to 20x8: no crashes, layout
  degrades gracefully. render, help overlay, palette, tree nav, tab switch,
  body editor, dashboard all work.
- 22 findings fixed and consolidated (PR: qa-remediation), each with a test.
- dead code eliminated: deadcode ./... reports zero unreachable functions.
- verified: go build, go vet, go test, go test -race all pass.
- CLI fixes shipped separately: no-path `parley run` runs the whole
  collection, `parley version`/`help`, version stamping via ldflags.

## must-do before 1.0

### release safety
- [x] add CI: go build, go vet, go test -race, and a deadcode gate run on
      every push to main and every PR (.github/workflows/ci.yml). verified
      green on GitHub before merge.
- [x] add a LICENSE file. MIT, added at repo root with a README section.
- [x] confirm the release build injects the version ldflag. v0.2.0 was cut
      from .goreleaser.yaml and `parley version` reports 0.2.0. note:
      goreleaser v2.16 fully deprecates the `brews` section used for the
      homebrew formula; it keeps working until goreleaser v3 (see
      nice-to-have).

### storage format
- [ ] decide and document whether the on-disk format (collections/,
      environments/, workspaces.json, jsonl histories) is frozen for 1.0.
      README says it "may still change" - 1.0 should commit to it or version
      it. deferred: a markdown-based request format is under consideration,
      so the freeze waits for that decision.
- [x] migration story for the file-permission tightening (0644 -> 0600):
      every secret-file writer now enforces 0600 on existing files too, and a
      best-effort startup sweep re-chmods environments/*.json, globals.json,
      active_environment, history.jsonl, last_responses.json and runs.jsonl
      in every registered workspace (collections/ is left alone). a failed
      sweep shows one warning and never blocks startup. (v0.3.0, #12)

### perf follow-ups (partials that were improved but not fully closed)
- [ ] websocket transcript render is bounded (cap 5000 + block cache) but
      each refresh is still O(n) string-building. windowing to the visible
      viewport would make it O(visible). acceptable for 1.0, revisit if a
      high-rate socket lags.
- [ ] dashboard overview is now cached; the run-history filter/scan and the
      performance section still scan on filter changes. fine at current
      scale, revisit if run history grows large.
- [x] history.jsonl is now capped on disk (maxHistoryEntries, oldest evicted
      on append via an atomic rewrite), which also bounds ListHistory's parse
      cost. append is per-send, not a hot path, so trimming there is cheap.

### test coverage gaps (foundational, thin)
- [x] internal/platform/fsstore raised from ~17 to 100 percent - round-trip
      tests for RequestToFile/RequestFromFile (HTTP, WebSocket, zero-value,
      disabled auth/refresh gating, timeout conversion).
- [x] internal/netcheck raised from ~43 to ~91 percent (defaultListInterfaces
      and hasRoutableAddr covered against the real loopback interface). the
      remaining gap is the routable-IP-present branch, which needs a real
      routed interface the test host may not have.
- [x] cmd/parley launch path raised from ~45 to ~86 percent (dataHome,
      buildModel, runCLICommand, workspace selection, the permission sweep).
      main() and tea.NewProgram still need a tty. (v0.3.0, #12, #14)

### error-path UX
- [x] audit user-facing error messages for actionability: `parley run`
      workspace, environment, path and --out errors list the valid choices or
      the next step, corrupt JSON errors name the file, and import errors say
      which format failed. (v0.3.0, #15)
- [x] responsive sidebar (DECISION, 2026-11-14): keep the current behavior.
      the sidebar drops out below minWidthForSidebar (60 cols) and stays
      fixed-width above it; adaptive sizing in the 60-90 col band moves to
      nice-to-have.

### secrets
- [x] the Cookies tab now masks cookie values by default (session tokens are
      inherently sensitive), revealable with the same ctrl+u toggle. Path/
      Domain/flags stay visible.
- [x] at-rest secrets (DECISION, 2026-11-14): history.jsonl stores each
      request before {{variable}} substitution, so secrets that come from
      environments never reach it, and a history re-run resolves current
      values. values typed directly into a request are stored as typed, and
      entries written by older versions roll off after 50 sends.
      last_responses.json keeps full responses (login responses can carry
      tokens) and stays owner-only. (v0.3.0, #13)

## nice-to-have (post-1.0)

- [ ] responsive/adaptive layouts beyond the current graceful floor,
      including sidebar sizing in the 60-90 col band.
- [ ] configurable transcript/history caps.
- [ ] a `parley --help` that also documents PARLEY_HOME / XDG_CONFIG_HOME.
- [ ] cap runs.jsonl the way history.jsonl is capped; it grows without bound.
- [ ] tui Model.status is written in many places but never rendered; render
      it or drop the writes.
- [ ] a request with an empty URL shows Go's raw `unsupported protocol
      scheme ""` error.
- [ ] `parley run` checks the path before --reporter, so a bad path hides a
      bad reporter value.
- [ ] move the homebrew formula from goreleaser `brews` to `homebrew_casks`,
      tap-wide together with mdo; unsigned binaries need a quarantine step.
